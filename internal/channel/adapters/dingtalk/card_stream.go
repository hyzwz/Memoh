package dingtalk

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/memohai/memoh/internal/channel"
)

const defaultCardTitle = "Memoh"

type cardStream struct {
	adapter         *Adapter
	cfg             channel.ChannelConfig
	target          Target
	reply           *channel.ReplyRef
	sourceMessageID string
	patchInterval   time.Duration
	now             func() time.Time

	mu          sync.Mutex
	buffer      strings.Builder
	lastPatched string
	lastUpdate  time.Time
	cardID      string
	degraded    bool
	closed      atomic.Bool
}

func (a *Adapter) OpenStream(ctx context.Context, cfg channel.ChannelConfig, target string, opts channel.StreamOptions) (channel.OutboundStream, error) {
	parsedCfg, err := parseConfig(cfg.Credentials)
	if err != nil {
		return nil, err
	}
	parsedTarget, err := parseTarget(target)
	if err != nil {
		return nil, err
	}
	switch parsedTarget.Family {
	case targetFamilyWebhook, targetFamilyConversation:
	default:
		return nil, fmt.Errorf("dingtalk ai card streaming does not support %s targets", parsedTarget.Family)
	}
	if !parsedCfg.EnableAICard {
		return a.newPlainTextOutboundStream(ctx, cfg, parsedTarget, opts)
	}

	interval := time.Duration(parsedCfg.CardUpdateThrottleMS) * time.Millisecond
	if interval < 0 {
		interval = 0
	}
	stream := &cardStream{
		adapter:         a,
		cfg:             cfg,
		target:          parsedTarget,
		reply:           opts.Reply,
		sourceMessageID: strings.TrimSpace(opts.SourceMessageID),
		patchInterval:   interval,
		now:             time.Now,
	}
	if err := stream.bootstrap(ctx); err != nil {
		return nil, err
	}
	return stream, nil
}

func (a *Adapter) newPlainTextOutboundStream(_ context.Context, cfg channel.ChannelConfig, target Target, opts channel.StreamOptions) (channel.OutboundStream, error) {
	return &cardStream{
		adapter:         a,
		cfg:             cfg,
		target:          target,
		reply:           opts.Reply,
		sourceMessageID: strings.TrimSpace(opts.SourceMessageID),
		patchInterval:   0,
		now:             time.Now,
		degraded:        true,
	}, nil
}

func (s *cardStream) bootstrap(ctx context.Context) error {
	if s == nil || s.degraded {
		return nil
	}
	return s.ensureCard(ctx, defaultCardTitle)
}

func (s *cardStream) Push(ctx context.Context, event channel.StreamEvent) error {
	if s == nil || s.adapter == nil {
		return errors.New("dingtalk card stream not configured")
	}
	if s.closed.Load() {
		return errors.New("dingtalk card stream is closed")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	switch event.Type {
	case channel.StreamEventStatus,
		channel.StreamEventPhaseStart,
		channel.StreamEventPhaseEnd,
		channel.StreamEventToolCallStart,
		channel.StreamEventToolCallEnd:
		return s.ensureCard(ctx, "Thinking...")
	case channel.StreamEventDelta:
		if event.Phase == channel.StreamPhaseReasoning || strings.TrimSpace(event.Delta) == "" {
			return nil
		}
		s.mu.Lock()
		s.buffer.WriteString(event.Delta)
		buffered := s.buffer.String()
		lastPatched := s.lastPatched
		lastUpdate := s.lastUpdate
		s.mu.Unlock()
		if s.degraded {
			return nil
		}
		if buffered == lastPatched {
			return nil
		}
		if !lastUpdate.IsZero() && s.now().Sub(lastUpdate) < s.patchInterval {
			return nil
		}
		return s.patchCard(ctx, buffered, false)
	case channel.StreamEventFinal:
		if event.Final == nil {
			return errors.New("dingtalk card stream final payload is required")
		}
		return s.flush(ctx, event.Final.Message, true)
	case channel.StreamEventError:
		errText := strings.TrimSpace(event.Error)
		if errText == "" {
			return nil
		}
		return s.fallback(ctx, channel.Message{Text: "Error: " + errText})
	case channel.StreamEventAttachment:
		return nil
	default:
		return nil
	}
}

func (s *cardStream) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.closed.Store(true)
	return nil
}

func (s *cardStream) ensureCard(ctx context.Context, title string) error {
	if s.degraded {
		return nil
	}
	s.mu.Lock()
	existing := strings.TrimSpace(s.cardID)
	buffered := s.buffer.String()
	s.mu.Unlock()
	if existing != "" {
		return nil
	}

	content := strings.TrimSpace(buffered)
	if content == "" {
		content = strings.TrimSpace(title)
	}
	cfg := parseConfigOrDefault(s.cfg)
	cardID, err := s.adapter.getClient(cfg).createAICard(ctx, cfg, s.target, s.sessionWebhookRef(), s.sourceMessageID, title, content)
	if err != nil {
		return s.fallback(ctx, channel.Message{Text: content})
	}
	s.mu.Lock()
	s.cardID = cardID
	s.lastPatched = content
	s.lastUpdate = s.now()
	s.mu.Unlock()
	return nil
}

func (s *cardStream) patchCard(ctx context.Context, content string, final bool) error {
	if s.degraded {
		return nil
	}
	cardID := s.cardIDValue()
	if cardID == "" {
		if err := s.ensureCard(ctx, defaultCardTitle); err != nil {
			return err
		}
		cardID = s.cardIDValue()
	}
	if cardID == "" {
		return nil
	}
	if !final {
		s.mu.Lock()
		lastUpdate := s.lastUpdate
		s.mu.Unlock()
		if !lastUpdate.IsZero() && s.now().Sub(lastUpdate) < s.patchInterval {
			return nil
		}
	}
	cfg := parseConfigOrDefault(s.cfg)
	if err := s.adapter.getClient(cfg).updateAICard(ctx, cfg, cardID, s.sourceMessageID, content, final); err != nil {
		return s.fallback(ctx, channel.Message{Text: content})
	}
	s.mu.Lock()
	s.lastPatched = content
	s.lastUpdate = s.now()
	s.mu.Unlock()
	return nil
}

func (s *cardStream) flush(ctx context.Context, msg channel.Message, final bool) error {
	content := strings.TrimSpace(msg.PlainText())
	if s.degraded {
		if content == "" {
			s.mu.Lock()
			content = strings.TrimSpace(s.buffer.String())
			s.mu.Unlock()
		}
		if content == "" && len(msg.Attachments) == 0 {
			return nil
		}
		fallbackMsg := msg
		if strings.TrimSpace(fallbackMsg.PlainText()) == "" && content != "" {
			fallbackMsg.Text = content
		}
		return s.fallback(ctx, fallbackMsg)
	}
	if content == "" {
		s.mu.Lock()
		content = strings.TrimSpace(s.buffer.String())
		s.mu.Unlock()
	}
	if content == "" && len(msg.Attachments) == 0 {
		return nil
	}
	if err := s.patchCard(ctx, content, final); err != nil {
		return err
	}
	return nil
}

func (s *cardStream) fallback(ctx context.Context, msg channel.Message) error {
	s.degraded = true
	msg.Reply = nil
	fallbackTarget := s.fallbackTarget()
	if fallbackTarget == "" {
		fallbackTarget = targetString(s.target)
	}
	return s.adapter.Send(ctx, s.cfg, channel.OutboundMessage{
		Target:  fallbackTarget,
		Message: msg,
	})
}

func (s *cardStream) cardIDValue() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.TrimSpace(s.cardID)
}

func (s *cardStream) sessionWebhookRef() string {
	if s != nil && s.reply != nil {
		if target := strings.TrimSpace(s.reply.Target); target != "" {
			if parsed, err := parseTarget(target); err == nil && parsed.Family == targetFamilyWebhook {
				return parsed.Value
			}
			return target
		}
	}
	if s != nil && s.target.Family == targetFamilyWebhook {
		return strings.TrimSpace(s.target.Value)
	}
	return ""
}

func (s *cardStream) fallbackTarget() string {
	if s != nil && s.reply != nil {
		if target := strings.TrimSpace(s.reply.Target); target != "" {
			return target
		}
	}
	if s != nil && s.target.Family == targetFamilyWebhook {
		return targetString(s.target)
	}
	return targetString(s.target)
}

func targetString(target Target) string {
	if target.Family == targetFamilyUser {
		return fmt.Sprintf("%s:%s:%s", target.Family, target.UserSubtype, target.Value)
	}
	return fmt.Sprintf("%s:%s", target.Family, target.Value)
}

func parseConfigOrDefault(cfg channel.ChannelConfig) Config {
	parsed, err := parseConfig(cfg.Credentials)
	if err != nil {
		return Config{}
	}
	if strings.TrimSpace(parsed.RobotCode) == "" {
		parsed.RobotCode = selfIdentityRobotCode(cfg.SelfIdentity)
	}
	return parsed
}
