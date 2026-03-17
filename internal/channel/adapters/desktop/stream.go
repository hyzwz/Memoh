package desktop

import (
	"context"
	"log/slog"

	"github.com/memohai/memoh/internal/channel"
)

// desktopStream implements channel.OutboundStream for desktop clients.
type desktopStream struct {
	logger *slog.Logger
}

// Push sends a stream event to the desktop client.
func (s *desktopStream) Push(ctx context.Context, event channel.StreamEvent) error {
	// Stream events are forwarded via the local WebSocket handler.
	s.logger.Debug("desktop stream push", slog.String("type", string(event.Type)))
	return nil
}

// Close finalizes the stream session.
func (s *desktopStream) Close(ctx context.Context) error {
	s.logger.Debug("desktop stream close")
	return nil
}
