package dingtalk

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/memohai/memoh/internal/channel"
	routepkg "github.com/memohai/memoh/internal/channel/route"
)

const defaultSessionResetConfirmation = "Started a new session."

type routeResetStore interface {
	Find(ctx context.Context, botID, platform, conversationID, threadID string) (routepkg.Route, error)
	Delete(ctx context.Context, routeID string) error
}

type RouteResetHandler struct {
	routes       routeResetStore
	confirmation string
}

func NewRouteResetHandler(routes routeResetStore) *RouteResetHandler {
	return &RouteResetHandler{
		routes:       routes,
		confirmation: defaultSessionResetConfirmation,
	}
}

func (h *RouteResetHandler) HandleReset(ctx context.Context, cfg channel.ChannelConfig, msg channel.InboundMessage, _ SessionCommand) (string, error) {
	if h == nil || h.routes == nil {
		return "", errors.New("dingtalk route reset handler not configured")
	}

	route, err := h.routes.Find(ctx,
		strings.TrimSpace(cfg.BotID),
		string(Type),
		strings.TrimSpace(msg.Conversation.ID),
		resetThreadID(msg),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return h.confirmationText(), nil
		}
		return "", err
	}
	if routeID := strings.TrimSpace(route.ID); routeID != "" {
		if err := h.routes.Delete(ctx, routeID); err != nil {
			return "", err
		}
	}

	return h.confirmationText(), nil
}

func (h *RouteResetHandler) confirmationText() string {
	if h == nil || strings.TrimSpace(h.confirmation) == "" {
		return defaultSessionResetConfirmation
	}
	return strings.TrimSpace(h.confirmation)
}

func resetThreadID(msg channel.InboundMessage) string {
	if msg.Message.Thread != nil && strings.TrimSpace(msg.Message.Thread.ID) != "" {
		return strings.TrimSpace(msg.Message.Thread.ID)
	}
	return strings.TrimSpace(msg.Conversation.ThreadID)
}
