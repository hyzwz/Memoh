package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// ErrHandNotFound is returned when a hand is not found.
var ErrHandNotFound = errors.New("hand not found")

// ErrRouteNotFound is returned when a model route is not found or doesn't belong to the bot.
var ErrRouteNotFound = errors.New("route not found")

// HandDTO is the API representation of a hand.
type HandDTO struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Type        string         `json:"type"`
	Content     string         `json:"content,omitempty"`
	IsEnabled   bool           `json:"is_enabled"`
	Schedule    map[string]any `json:"schedule,omitempty"`
	Triggers    []any          `json:"triggers,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// HandExecutionResultDTO is the API representation of an execution result.
type HandExecutionResultDTO struct {
	Status       string    `json:"status"`
	ResultText   string    `json:"result_text,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at"`
}

// CreateHandRequest is the body for creating a hand from markdown.
type CreateHandRequest struct {
	Markdown string `json:"markdown"`
}

// HandsServiceInterface abstracts hand operations for the handler.
// All methods that access a specific hand take botID for scoping.
type HandsServiceInterface interface {
	List(ctx context.Context, botID string) ([]HandDTO, error)
	Get(ctx context.Context, botID, id string) (*HandDTO, error)
	CreateFromMarkdown(ctx context.Context, botID, markdown string) (*HandDTO, error)
	Delete(ctx context.Context, botID, id string) error
	Execute(ctx context.Context, botID, handID, triggerType string) (*HandExecutionResultDTO, error)
}

// HandsHandlerOption configures optional dependencies for HandsHandler.
type HandsHandlerOption func(*HandsHandler)

// WithAuditLogger sets the audit logger for hand mutations.
func WithAuditLogger(al AuditLoggerInterface) HandsHandlerOption {
	return func(h *HandsHandler) { h.audit = al }
}

// HandsHandler handles hand CRUD and execution API endpoints.
type HandsHandler struct {
	svc        HandsServiceInterface
	audit      AuditLoggerInterface
	middleware []echo.MiddlewareFunc
}

// NewHandsHandler creates a new hands handler.
func NewHandsHandler(svc HandsServiceInterface, opts ...any) *HandsHandler {
	h := &HandsHandler{svc: svc, audit: noopAuditLogger{}}
	for _, opt := range opts {
		switch v := opt.(type) {
		case echo.MiddlewareFunc:
			h.middleware = append(h.middleware, v)
		case HandsHandlerOption:
			v(h)
		}
	}
	return h
}

// Register registers hand routes.
func (h *HandsHandler) Register(e *echo.Echo) {
	g := e.Group("/bots/:bot_id/hands", h.middleware...)
	g.GET("", h.ListHands)
	g.POST("", h.CreateHand)
	g.GET("/:hand_id", h.GetHand)
	g.DELETE("/:hand_id", h.DeleteHand)
	g.POST("/:hand_id/execute", h.ExecuteHand)
}

// ListHands godoc
// @Summary List hands for a bot
// @Tags hands
// @Param bot_id path string true "Bot ID"
// @Success 200 {array} HandDTO
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/hands [get]
func (h *HandsHandler) ListHands(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	hands, err := h.svc.List(c.Request().Context(), botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list hands")
	}
	return c.JSON(http.StatusOK, hands)
}

// GetHand godoc
// @Summary Get a hand by ID
// @Tags hands
// @Param bot_id path string true "Bot ID"
// @Param hand_id path string true "Hand ID"
// @Success 200 {object} HandDTO
// @Failure 404 {object} ErrorResponse
// @Router /bots/{bot_id}/hands/{hand_id} [get]
func (h *HandsHandler) GetHand(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	handID := strings.TrimSpace(c.Param("hand_id"))
	if handID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "hand_id is required")
	}

	hand, err := h.svc.Get(c.Request().Context(), botID, handID)
	if err != nil {
		if errors.Is(err, ErrHandNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "hand not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get hand")
	}
	return c.JSON(http.StatusOK, hand)
}

// CreateHand godoc
// @Summary Create a hand from markdown
// @Tags hands
// @Param bot_id path string true "Bot ID"
// @Param body body CreateHandRequest true "Hand markdown"
// @Success 201 {object} HandDTO
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/hands [post]
func (h *HandsHandler) CreateHand(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	var req CreateHandRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if strings.TrimSpace(req.Markdown) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "markdown is required")
	}

	hand, err := h.svc.CreateFromMarkdown(c.Request().Context(), botID, req.Markdown)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to create hand")
	}
	h.audit.Log(c.Request().Context(), auditUserID(c), botID, "create", "hand", hand.ID, c.RealIP(), c.Request().UserAgent(), nil)
	return c.JSON(http.StatusCreated, hand)
}

// DeleteHand godoc
// @Summary Delete a hand
// @Tags hands
// @Param bot_id path string true "Bot ID"
// @Param hand_id path string true "Hand ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/hands/{hand_id} [delete]
func (h *HandsHandler) DeleteHand(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	handID := strings.TrimSpace(c.Param("hand_id"))
	if handID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "hand_id is required")
	}

	if err := h.svc.Delete(c.Request().Context(), botID, handID); err != nil {
		if errors.Is(err, ErrHandNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "hand not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete hand")
	}
	h.audit.Log(c.Request().Context(), auditUserID(c), botID, "delete", "hand", handID, c.RealIP(), c.Request().UserAgent(), nil)
	return c.NoContent(http.StatusNoContent)
}

// ExecuteHand godoc
// @Summary Manually execute a hand
// @Tags hands
// @Param bot_id path string true "Bot ID"
// @Param hand_id path string true "Hand ID"
// @Success 200 {object} HandExecutionResultDTO
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/hands/{hand_id}/execute [post]
func (h *HandsHandler) ExecuteHand(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	handID := strings.TrimSpace(c.Param("hand_id"))
	if handID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "hand_id is required")
	}

	result, err := h.svc.Execute(c.Request().Context(), botID, handID, "manual")
	if err != nil {
		if errors.Is(err, ErrHandNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "hand not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to execute hand")
	}
	h.audit.Log(c.Request().Context(), auditUserID(c), botID, "execute", "hand", handID, c.RealIP(), c.Request().UserAgent(), map[string]string{"status": result.Status})
	return c.JSON(http.StatusOK, result)
}
