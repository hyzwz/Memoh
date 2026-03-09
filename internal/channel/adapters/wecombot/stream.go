package wecombot

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/memohai/memoh/internal/channel"
)

type replyClient interface {
	Connected() bool
	SendMarkdown(ctx context.Context, chatID, content string) error
	ReplyStream(ctx context.Context, sourceReqID, streamID, content string, finish bool) error
}

type outboundStream struct {
	client             replyClient
	targetType         targetType
	targetID           string
	sourceReqID        string
	streamID           string
	sendThinkingPrompt bool

	closed atomic.Bool
	mu     sync.Mutex
	buffer strings.Builder
}

func newOutboundStream(client replyClient, targetType targetType, targetID string, sourceReqID string, sendThinkingPrompt bool) *outboundStream {
	return &outboundStream{
		client:             client,
		targetType:         targetType,
		targetID:           targetID,
		sourceReqID:        strings.TrimSpace(sourceReqID),
		streamID:           generateReqID("stream"),
		sendThinkingPrompt: sendThinkingPrompt,
	}
}

func (s *outboundStream) Push(ctx context.Context, event channel.StreamEvent) error {
	if s == nil || s.client == nil {
		return errors.New("wecom ai bot stream not configured")
	}
	if s.closed.Load() {
		return errors.New("wecom ai bot stream is closed")
	}
	if !s.client.Connected() {
		return errors.New("wecom ai bot websocket is not connected")
	}
	switch event.Type {
	case channel.StreamEventStatus:
		if event.Status == channel.StreamStatusStarted && s.sourceReqID != "" && s.sendThinkingPrompt {
			return s.client.ReplyStream(ctx, s.sourceReqID, s.streamID, thinkingPlaceholder, false)
		}
		return nil
	case channel.StreamEventDelta:
		if event.Delta == "" || event.Phase == channel.StreamPhaseReasoning {
			return nil
		}
		if s.sourceReqID == "" {
			s.mu.Lock()
			s.buffer.WriteString(event.Delta)
			s.mu.Unlock()
			return nil
		}
		s.mu.Lock()
		s.buffer.WriteString(event.Delta)
		content := s.buffer.String()
		s.mu.Unlock()
		return s.client.ReplyStream(ctx, s.sourceReqID, s.streamID, content, false)
	case channel.StreamEventFinal:
		if event.Final == nil {
			return errors.New("wecom ai bot stream final payload is required")
		}
		return s.flush(ctx, event.Final.Message, true)
	case channel.StreamEventError:
		errText := strings.TrimSpace(event.Error)
		if errText == "" {
			return nil
		}
		return s.flush(ctx, channel.Message{Text: "Error: " + errText}, true)
	case channel.StreamEventAttachment,
		channel.StreamEventPhaseStart,
		channel.StreamEventPhaseEnd,
		channel.StreamEventToolCallStart,
		channel.StreamEventToolCallEnd,
		channel.StreamEventAgentStart,
		channel.StreamEventAgentEnd,
		channel.StreamEventProcessingStarted,
		channel.StreamEventProcessingCompleted,
		channel.StreamEventProcessingFailed:
		return nil
	default:
		return nil
	}
}

func (s *outboundStream) Close(ctx context.Context) error {
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

func (s *outboundStream) flush(ctx context.Context, msg channel.Message, finish bool) error {
	if len(msg.Attachments) > 0 {
		return errors.New("wecom ai bot outbound attachments are not supported yet")
	}
	s.mu.Lock()
	buffered := strings.TrimSpace(s.buffer.String())
	s.buffer.Reset()
	s.mu.Unlock()
	content := buffered
	if content == "" {
		content = strings.TrimSpace(msg.PlainText())
	}
	if content == "" {
		return nil
	}
	if s.sourceReqID != "" {
		return s.client.ReplyStream(ctx, s.sourceReqID, s.streamID, content, finish)
	}
	if s.targetType != targetChatID {
		return errors.New("wecom ai bot proactive send requires chatid target")
	}
	return s.client.SendMarkdown(ctx, s.targetID, content)
}
