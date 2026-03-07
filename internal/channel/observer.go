package channel

import "context"

// StreamObserver receives copies of stream events for cross-channel broadcasting
// or external notification (e.g. webhooks). Implementations must be safe for
// concurrent use and should not block.
type StreamObserver interface {
	// OnStreamEvent is called for every event pushed to an outbound stream.
	// botID identifies the bot that owns the conversation.
	// source is the channel type that originated the event (e.g. "telegram").
	OnStreamEvent(ctx context.Context, botID string, source ChannelType, event StreamEvent)
}

// teeStream wraps an OutboundStream and mirrors every Push call to an observer.
type teeStream struct {
	primary  OutboundStream
	observer StreamObserver
	botID    string
	source   ChannelType
	metadata map[string]any
}

// NewTeeStream wraps primary so that every Push also notifies observer.
// metadata carries stream-scoped fields (e.g. context_user_id) that are
// injected into every observed event for downstream filtering.
func NewTeeStream(primary OutboundStream, observer StreamObserver, botID string, source ChannelType, metadata ...map[string]any) OutboundStream {
	if observer == nil {
		return primary
	}
	var meta map[string]any
	if len(metadata) > 0 {
		meta = metadata[0]
	}
	return &teeStream{
		primary:  primary,
		observer: observer,
		botID:    botID,
		source:   source,
		metadata: meta,
	}
}

func (t *teeStream) Push(ctx context.Context, event StreamEvent) error {
	err := t.primary.Push(ctx, event)
	// Notify observer regardless of push error — the event was produced and
	// should still be visible in monitoring/WebUI even if the primary channel
	// delivery failed (e.g. Telegram rate-limit).
	observed := event
	if len(t.metadata) > 0 {
		merged := make(map[string]any, len(event.Metadata)+len(t.metadata))
		for k, v := range event.Metadata {
			merged[k] = v
		}
		// Stream-scoped metadata wins to prevent spoofing.
		for k, v := range t.metadata {
			merged[k] = v
		}
		observed.Metadata = merged
	}
	t.observer.OnStreamEvent(ctx, t.botID, t.source, observed)
	return err
}

func (t *teeStream) Close(ctx context.Context) error {
	return t.primary.Close(ctx)
}
