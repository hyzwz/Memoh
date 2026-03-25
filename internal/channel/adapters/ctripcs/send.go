package ctripcs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/memohai/memoh/internal/channel"
)

var (
	errCtripOutboundTextRequired       = errors.New("ctrip_cs outbound text is required")
	errCtripOutboundAttachments        = errors.New("ctrip_cs outbound attachments are not supported yet")
	errCtripOutboundReplyTargetNeeded  = errors.New("ctrip_cs reply target is required")
	errCtripOutboundScriptEmptySession = errors.New("ctrip_cs outbound session id is required")
)

type ctripOutboundSendResult struct {
	Success bool   `json:"success"`
	Focused bool   `json:"focused,omitempty"`
	Cleared bool   `json:"cleared,omitempty"`
	Error   string `json:"error,omitempty"`
}

type ctripOutboundStream struct {
	adapter *Adapter
	cfg     channel.ChannelConfig
	target  string
	reply   *channel.ReplyRef

	mu          sync.Mutex
	deltaBuffer strings.Builder
	finalText   string
	closed      atomic.Bool
}

func (a *Adapter) Send(ctx context.Context, cfg channel.ChannelConfig, msg channel.OutboundMessage) error {
	parsedCfg, err := parseConfig(cfg.Credentials)
	if err != nil {
		return err
	}
	if a == nil || a.browserGateway == nil {
		return errors.New("ctrip_cs browser gateway is not configured")
	}

	target, err := resolveOutboundTarget(msg)
	if err != nil {
		return err
	}
	sessionID, err := parseSessionTarget(target)
	if err != nil {
		return err
	}

	text := strings.TrimSpace(msg.Message.PlainText())
	if text == "" {
		return errCtripOutboundTextRequired
	}
	if len(msg.Message.Attachments) > 0 {
		return errCtripOutboundAttachments
	}

	contextID := strings.TrimSpace(parsedCfg.BrowserContextID)
	exists, err := a.browserGateway.Exists(ctx, contextID)
	if err != nil {
		return fmt.Errorf("ctrip_cs check browser context: %w", err)
	}
	if !exists {
		return errMissingBrowserContext
	}

	if err := a.browserGateway.Navigate(ctx, contextID, parsedCfg.InboxPageURL); err != nil {
		return fmt.Errorf("ctrip_cs navigate inbox page: %w", err)
	}

	raw, err := a.browserGateway.Evaluate(ctx, contextID, buildOutboundSendScript(sessionID, text))
	if err != nil {
		return fmt.Errorf("ctrip_cs outbound send: %w", err)
	}

	var result ctripOutboundSendResult
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &result); err != nil {
			return fmt.Errorf("ctrip_cs decode outbound send result: %w", err)
		}
	}
	if !result.Success {
		if trimmed := strings.TrimSpace(result.Error); trimmed != "" {
			return errors.New(trimmed)
		}
		return errors.New("ctrip_cs outbound send failed")
	}
	return nil
}

func (a *Adapter) OpenStream(ctx context.Context, cfg channel.ChannelConfig, target string, opts channel.StreamOptions) (channel.OutboundStream, error) {
	if a == nil || a.browserGateway == nil {
		return nil, errors.New("ctrip_cs browser gateway is not configured")
	}
	if _, err := parseConfig(cfg.Credentials); err != nil {
		return nil, err
	}
	resolvedTarget, err := resolveStreamTarget(target, opts.Reply)
	if err != nil {
		return nil, err
	}
	_ = ctx
	return &ctripOutboundStream{
		adapter: a,
		cfg:     cfg,
		target:  resolvedTarget,
		reply:   opts.Reply,
	}, nil
}

func (s *ctripOutboundStream) Push(ctx context.Context, event channel.StreamEvent) error {
	if s == nil || s.adapter == nil {
		return errors.New("ctrip_cs outbound stream is not configured")
	}
	if s.closed.Load() {
		return errors.New("ctrip_cs outbound stream is closed")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	switch event.Type {
	case channel.StreamEventDelta:
		if event.Phase == channel.StreamPhaseReasoning || strings.TrimSpace(event.Delta) == "" {
			return nil
		}
		s.mu.Lock()
		s.deltaBuffer.WriteString(event.Delta)
		s.mu.Unlock()
		return nil
	case channel.StreamEventFinal:
		if event.Final == nil {
			return errors.New("ctrip_cs stream final payload is required")
		}
		finalText := strings.TrimSpace(event.Final.Message.PlainText())
		if finalText == "" {
			return nil
		}
		s.mu.Lock()
		s.finalText = finalText
		s.mu.Unlock()
		return nil
	default:
		return nil
	}
}

func (s *ctripOutboundStream) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if s.closed.Swap(true) {
		return nil
	}

	content := s.snapshotContent()
	if content == "" {
		return nil
	}
	msg := channel.OutboundMessage{
		Target: s.target,
		Message: channel.Message{
			Text:  content,
			Reply: s.reply,
		},
	}
	return s.adapter.Send(ctx, s.cfg, msg)
}

func (s *ctripOutboundStream) snapshotContent() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if trimmed := strings.TrimSpace(s.finalText); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(s.deltaBuffer.String())
}

func resolveOutboundTarget(msg channel.OutboundMessage) (string, error) {
	if msg.Message.Reply != nil {
		if target := strings.TrimSpace(msg.Message.Reply.Target); target != "" {
			return target, nil
		}
	}
	if target := strings.TrimSpace(msg.Target); target != "" {
		return target, nil
	}
	return "", errCtripOutboundReplyTargetNeeded
}

func resolveStreamTarget(target string, reply *channel.ReplyRef) (string, error) {
	if trimmed := strings.TrimSpace(target); trimmed != "" {
		return trimmed, nil
	}
	if reply != nil {
		if trimmed := strings.TrimSpace(reply.Target); trimmed != "" {
			return trimmed, nil
		}
	}
	return "", errCtripOutboundReplyTargetNeeded
}

func parseSessionTarget(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errCtripOutboundScriptEmptySession
	}
	if !strings.HasPrefix(trimmed, "session:") {
		return "", fmt.Errorf("ctrip_cs target must use session:<conversation_id>, got %q", trimmed)
	}
	sessionID := strings.TrimSpace(strings.TrimPrefix(trimmed, "session:"))
	if sessionID == "" {
		return "", errCtripOutboundScriptEmptySession
	}
	return sessionID, nil
}

func buildOutboundSendScript(sessionID, text string) string {
	return fmt.Sprintf(`(() => {
  const sessionId = %s;
  const messageText = %s;

  const isVisible = (el) => {
    if (!el || typeof el !== 'object') return false;
    const node = /** @type {HTMLElement} */ (el);
    const view = globalThis.getComputedStyle ? getComputedStyle(node) : null;
    if (!view) return true;
    return view.display !== 'none' && view.visibility !== 'hidden' && view.opacity !== '0';
  };

  const textOf = (value) => {
    if (typeof value !== 'string') return '';
    return value.trim();
  };

  const findSession = () => {
    const selectors = [
      '[data-session-id="' + sessionId + '"]',
      '[data-conversation-id="' + sessionId + '"]',
      '[data-thread-id="' + sessionId + '"]',
      '[href*="' + sessionId + '"]',
    ];
    for (const selector of selectors) {
      const el = document.querySelector(selector);
      if (el && isVisible(el)) return el;
    }
    const candidates = document.querySelectorAll('button,a,[role="button"],[role="link"],li,div,span');
    for (const el of candidates) {
      if (!isVisible(el)) continue;
      if (textOf(el.textContent).includes(sessionId)) return el;
    }
    return null;
  };

  const findComposer = () => {
    const selectors = [
      'textarea:not([disabled])',
      'input[type="text"]:not([disabled])',
      '[contenteditable="true"]',
      '[role="textbox"]',
    ];
    for (const selector of selectors) {
      const el = document.querySelector(selector);
      if (el && isVisible(el)) return el;
    }
    return null;
  };

  const sessionEl = findSession();
  if (!sessionEl) {
    throw new Error('ctrip_cs session not found for ' + sessionId);
  }
  if (typeof sessionEl.click === 'function') {
    sessionEl.click();
  }

  const composer = findComposer();
  if (!composer) {
    throw new Error('ctrip_cs composer not found');
  }

  const isInput = composer instanceof HTMLInputElement || composer instanceof HTMLTextAreaElement;
  if (typeof composer.focus === 'function') {
    composer.focus();
  }
  if (isInput) {
    composer.value = messageText;
    composer.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText', data: messageText }));
    composer.dispatchEvent(new Event('change', { bubbles: true }));
  } else {
    composer.textContent = messageText;
    composer.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText', data: messageText }));
  }

  composer.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', code: 'Enter', bubbles: true, cancelable: true }));
  composer.dispatchEvent(new KeyboardEvent('keyup', { key: 'Enter', code: 'Enter', bubbles: true, cancelable: true }));

  const currentValue = isInput ? composer.value : composer.textContent;
  const cleared = textOf(currentValue) === '';
  return {
    success: cleared,
    focused: true,
    cleared,
  };
})()`, jsString(sessionID), jsString(text))
}

func jsString(value string) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}
