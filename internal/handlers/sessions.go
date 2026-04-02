package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/memohai/memoh/internal/accounts"
	"github.com/memohai/memoh/internal/bots"
	"github.com/memohai/memoh/internal/conversation"
)

type SessionsService interface {
	ListByBotAndChannelIdentity(ctx context.Context, botID, channelIdentityID string) ([]conversation.ConversationListItem, error)
	Create(ctx context.Context, botID, channelIdentityID string, req conversation.CreateRequest) (conversation.Conversation, error)
	Delete(ctx context.Context, conversationID string) error
}

type SessionsHandler struct {
	service        SessionsService
	botService     *bots.Service
	accountService *accounts.Service
	logger         *slog.Logger
	authorize      func(ctx context.Context, channelIdentityID, botID string) (bots.Bot, error)
}

func NewSessionsHandler(log *slog.Logger, service *conversation.Service, botService *bots.Service, accountService *accounts.Service) *SessionsHandler {
	return &SessionsHandler{
		service:        service,
		botService:     botService,
		accountService: accountService,
		logger:         log.With(slog.String("handler", "sessions")),
	}
}

func (h *SessionsHandler) Register(e *echo.Echo) {
	group := e.Group("/bots/:bot_id/sessions")
	group.GET("", h.ListSessions)
	group.POST("", h.CreateSession)
	group.DELETE("/:id", h.DeleteSession)
}

// ListSessions godoc
// @Summary List bot sessions
// @Description List visible sessions for a bot and current channel identity
// @Tags sessions
// @Produce json
// @Param bot_id path string true "Bot ID"
// @Success 200 {object} map[string][]conversation.ConversationListItem
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/sessions [get]
func (h *SessionsHandler) ListSessions(c echo.Context) error {
	channelIdentityID, err := RequireChannelIdentityID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), channelIdentityID, botID); err != nil {
		return err
	}
	items, err := h.service.ListByBotAndChannelIdentity(c.Request().Context(), botID, channelIdentityID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// CreateSession godoc
// @Summary Create bot session
// @Description Create a new session for a bot and current channel identity
// @Tags sessions
// @Accept json
// @Produce json
// @Param bot_id path string true "Bot ID"
// @Param payload body conversation.CreateRequest true "Session payload"
// @Success 201 {object} conversation.Conversation
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/sessions [post]
func (h *SessionsHandler) CreateSession(c echo.Context) error {
	channelIdentityID, err := RequireChannelIdentityID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), channelIdentityID, botID); err != nil {
		return err
	}
	var req conversation.CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	item, err := h.service.Create(c.Request().Context(), botID, channelIdentityID, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, item)
}

// DeleteSession godoc
// @Summary Delete bot session
// @Description Delete a bot session by ID
// @Tags sessions
// @Param bot_id path string true "Bot ID"
// @Param id path string true "Session ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/sessions/{id} [delete]
func (h *SessionsHandler) DeleteSession(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	channelIdentityID, err := RequireChannelIdentityID(c)
	if err != nil {
		return err
	}
	if _, err := h.authorizeBotAccess(c.Request().Context(), channelIdentityID, botID); err != nil {
		return err
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "id is required")
	}
	if err := h.service.Delete(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *SessionsHandler) authorizeBotAccess(ctx context.Context, channelIdentityID, botID string) (bots.Bot, error) {
	if h.authorize != nil {
		return h.authorize(ctx, channelIdentityID, botID)
	}
	return AuthorizeBotAccess(ctx, h.botService, h.accountService, channelIdentityID, botID, bots.AccessPolicy{AllowPublicMember: true})
}
