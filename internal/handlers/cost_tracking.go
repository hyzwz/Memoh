package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// BudgetDTO is the API representation of a budget.
type BudgetDTO struct {
	ID             string  `json:"id"`
	ScopeType      string  `json:"scope_type"`
	ScopeID        string  `json:"scope_id"`
	Period         string  `json:"period"`
	LimitAmount    float64 `json:"limit_amount"`
	AlertThreshold float64 `json:"alert_threshold"`
	ActionOnExceed string  `json:"action_on_exceed"`
	IsEnabled      bool    `json:"is_enabled"`
}

// BudgetCheckDTO is the API representation of a budget check result.
type BudgetCheckDTO struct {
	Allowed    bool    `json:"allowed"`
	Alert      bool    `json:"alert"`
	Spent      float64 `json:"spent"`
	Limit      float64 `json:"limit"`
	Percentage float64 `json:"percentage"`
	Action     string  `json:"action,omitempty"`
}

// CreateBudgetRequest is the body for creating a budget.
type CreateBudgetRequest struct {
	ScopeType      string  `json:"scope_type"`
	ScopeID        string  `json:"scope_id"`
	Period         string  `json:"period"`
	LimitAmount    float64 `json:"limit_amount"`
	AlertThreshold float64 `json:"alert_threshold"`
	ActionOnExceed string  `json:"action_on_exceed"`
}

// CostTrackingServiceInterface abstracts cost tracking operations.
type CostTrackingServiceInterface interface {
	ListBudgets(ctx context.Context, scopeType string) ([]BudgetDTO, error)
	CreateBudget(ctx context.Context, req CreateBudgetRequest) (*BudgetDTO, error)
	DeleteBudget(ctx context.Context, id string) error
	CheckBudget(ctx context.Context, scopeType, scopeID string) (*BudgetCheckDTO, error)
}

// CostTrackingHandlerOption configures optional dependencies.
type CostTrackingHandlerOption func(*CostTrackingHandler)

// WithCostTrackingAudit sets the audit logger for budget mutations.
func WithCostTrackingAudit(al AuditLoggerInterface) CostTrackingHandlerOption {
	return func(h *CostTrackingHandler) { h.audit = al }
}

// CostTrackingHandler handles cost tracking and budget API endpoints.
type CostTrackingHandler struct {
	svc        CostTrackingServiceInterface
	audit      AuditLoggerInterface
	middleware []echo.MiddlewareFunc
}

// NewCostTrackingHandler creates a new cost tracking handler.
func NewCostTrackingHandler(svc CostTrackingServiceInterface, opts ...any) *CostTrackingHandler {
	h := &CostTrackingHandler{svc: svc, audit: noopAuditLogger{}}
	for _, o := range opts {
		switch v := o.(type) {
		case echo.MiddlewareFunc:
			h.middleware = append(h.middleware, v)
		case CostTrackingHandlerOption:
			v(h)
		}
	}
	return h
}

// Register registers cost tracking routes.
func (h *CostTrackingHandler) Register(e *echo.Echo) {
	g := e.Group("/budgets", h.middleware...)
	g.GET("", h.ListBudgets)
	g.POST("", h.CreateBudget)
	g.DELETE("/:budget_id", h.DeleteBudget)
	g.GET("/check", h.CheckBudget)
}

// ListBudgets godoc
// @Summary List budgets
// @Tags cost
// @Param scope_type query string false "Filter by scope type"
// @Success 200 {array} BudgetDTO
// @Router /budgets [get]
func (h *CostTrackingHandler) ListBudgets(c echo.Context) error {
	scopeType := strings.TrimSpace(c.QueryParam("scope_type"))

	budgets, err := h.svc.ListBudgets(c.Request().Context(), scopeType)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list budgets")
	}
	return c.JSON(http.StatusOK, budgets)
}

// CreateBudget godoc
// @Summary Create a budget
// @Tags cost
// @Param body body CreateBudgetRequest true "Budget data"
// @Success 201 {object} BudgetDTO
// @Failure 400 {object} ErrorResponse
// @Router /budgets [post]
func (h *CostTrackingHandler) CreateBudget(c echo.Context) error {
	var req CreateBudgetRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	req.ScopeType = strings.ToLower(strings.TrimSpace(req.ScopeType))
	if req.ScopeType == "global" {
		req.ScopeType = "system"
	}
	if strings.TrimSpace(req.ScopeType) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "scope_type is required")
	}
	if strings.TrimSpace(req.ScopeID) == "" && req.ScopeType != "system" {
		return echo.NewHTTPError(http.StatusBadRequest, "scope_id is required")
	}
	if req.ScopeType == "system" && strings.TrimSpace(req.ScopeID) == "" {
		req.ScopeID = "system"
	}
	if strings.TrimSpace(req.Period) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "period is required")
	}
	if req.LimitAmount <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "limit_amount must be positive")
	}

	// Validate allowed values.
	validScopeTypes := map[string]bool{"bot": true, "user": true, "department": true, "system": true}
	if !validScopeTypes[req.ScopeType] {
		return echo.NewHTTPError(http.StatusBadRequest, "scope_type must be one of: bot, user, department, system")
	}
	validPeriods := map[string]bool{"daily": true, "weekly": true, "monthly": true}
	if !validPeriods[req.Period] {
		return echo.NewHTTPError(http.StatusBadRequest, "period must be one of: daily, weekly, monthly")
	}
	validActions := map[string]bool{"warn": true, "block": true, "downgrade": true}
	if req.ActionOnExceed != "" && !validActions[req.ActionOnExceed] {
		return echo.NewHTTPError(http.StatusBadRequest, "action_on_exceed must be one of: warn, block, downgrade")
	}
	if req.AlertThreshold < 0 || req.AlertThreshold > 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "alert_threshold must be between 0 and 1")
	}

	budget, err := h.svc.CreateBudget(c.Request().Context(), req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create budget")
	}
	h.audit.Log(c.Request().Context(), auditUserID(c), "", "create", "budget", budget.ID, c.RealIP(), c.Request().UserAgent(), nil)
	return c.JSON(http.StatusCreated, budget)
}

// DeleteBudget godoc
// @Summary Delete a budget
// @Tags cost
// @Param budget_id path string true "Budget ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Router /budgets/{budget_id} [delete]
func (h *CostTrackingHandler) DeleteBudget(c echo.Context) error {
	budgetID := strings.TrimSpace(c.Param("budget_id"))
	if budgetID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "budget_id is required")
	}

	if err := h.svc.DeleteBudget(c.Request().Context(), budgetID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete budget")
	}
	h.audit.Log(c.Request().Context(), auditUserID(c), "", "delete", "budget", budgetID, c.RealIP(), c.Request().UserAgent(), nil)
	return c.NoContent(http.StatusNoContent)
}

// CheckBudget godoc
// @Summary Check budget status
// @Tags cost
// @Param scope_type query string true "Scope type"
// @Param scope_id query string true "Scope ID"
// @Success 200 {object} BudgetCheckDTO
// @Failure 400 {object} ErrorResponse
// @Router /budgets/check [get]
func (h *CostTrackingHandler) CheckBudget(c echo.Context) error {
	scopeType := strings.ToLower(strings.TrimSpace(c.QueryParam("scope_type")))
	scopeID := strings.TrimSpace(c.QueryParam("scope_id"))
	if scopeType == "global" {
		scopeType = "system"
	}
	if scopeType == "system" && scopeID == "" {
		scopeID = "system"
	}

	if scopeType == "" || scopeID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "scope_type and scope_id are required")
	}

	result, err := h.svc.CheckBudget(c.Request().Context(), scopeType, scopeID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to check budget")
	}
	return c.JSON(http.StatusOK, result)
}
