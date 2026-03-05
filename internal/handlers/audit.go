package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// AuditLogDTO is the API representation of an audit log entry.
type AuditLogDTO struct {
	ID        string         `json:"id"`
	Action    string         `json:"action"`
	UserID    string         `json:"user_id"`
	BotID     string         `json:"bot_id,omitempty"`
	Detail    map[string]any `json:"detail,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// AuditLogQuery holds filter parameters for listing audit logs.
type AuditLogQuery struct {
	BotID  string
	UserID string
	Action string
	Limit  int
	Offset int
}

// AuditQueryServiceInterface abstracts audit log querying.
type AuditQueryServiceInterface interface {
	ListAuditLogs(ctx context.Context, query AuditLogQuery) ([]AuditLogDTO, int64, error)
}

// AuditHandler handles audit log API endpoints.
type AuditHandler struct {
	svc AuditQueryServiceInterface
}

// NewAuditHandler creates a new audit handler.
func NewAuditHandler(svc AuditQueryServiceInterface) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// Register registers audit log routes.
func (h *AuditHandler) Register(e *echo.Echo) {
	e.GET("/bots/:bot_id/audit-logs", h.ListAuditLogs)
}

// ListAuditLogs godoc
// @Summary List audit logs for a bot
// @Tags audit
// @Param bot_id path string true "Bot ID"
// @Param user_id query string false "Filter by user"
// @Param action query string false "Filter by action"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]any
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/audit-logs [get]
func (h *AuditHandler) ListAuditLogs(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	limit := 50
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	offset := 0
	if v := c.QueryParam("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	query := AuditLogQuery{
		BotID:  botID,
		UserID: strings.TrimSpace(c.QueryParam("user_id")),
		Action: strings.TrimSpace(c.QueryParam("action")),
		Limit:  limit,
		Offset: offset,
	}

	logs, total, err := h.svc.ListAuditLogs(c.Request().Context(), query)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list audit logs")
	}

	return c.JSON(http.StatusOK, map[string]any{
		"logs":  logs,
		"total": total,
	})
}
