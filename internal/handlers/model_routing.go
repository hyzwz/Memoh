package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// ModelRouteDTO is the API representation of a model route.
type ModelRouteDTO struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	ModelID        string         `json:"model_id"`
	Priority       int            `json:"priority"`
	ComplexityTier string         `json:"complexity_tier"`
	IsEnabled      bool           `json:"is_enabled"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// CreateModelRouteRequest is the body for creating a model route.
type CreateModelRouteRequest struct {
	Name           string         `json:"name"`
	ModelID        string         `json:"model_id"`
	Priority       int            `json:"priority"`
	ComplexityTier string         `json:"complexity_tier"`
	IsEnabled      *bool          `json:"is_enabled,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// ModelRoutingServiceInterface abstracts model routing operations.
type ModelRoutingServiceInterface interface {
	ListRoutes(ctx context.Context, botID string) ([]ModelRouteDTO, error)
	CreateRoute(ctx context.Context, botID string, req CreateModelRouteRequest) (*ModelRouteDTO, error)
	DeleteRoute(ctx context.Context, botID, id string) error
}

// ModelRoutingHandlerOption configures optional dependencies.
type ModelRoutingHandlerOption func(*ModelRoutingHandler)

// WithModelRoutingAudit sets the audit logger for route mutations.
func WithModelRoutingAudit(al AuditLoggerInterface) ModelRoutingHandlerOption {
	return func(h *ModelRoutingHandler) { h.audit = al }
}

// ModelRoutingHandler handles model routing API endpoints.
type ModelRoutingHandler struct {
	svc        ModelRoutingServiceInterface
	audit      AuditLoggerInterface
	middleware []echo.MiddlewareFunc
}

// NewModelRoutingHandler creates a new model routing handler.
func NewModelRoutingHandler(svc ModelRoutingServiceInterface, opts ...any) *ModelRoutingHandler {
	h := &ModelRoutingHandler{svc: svc, audit: noopAuditLogger{}}
	for _, o := range opts {
		switch v := o.(type) {
		case echo.MiddlewareFunc:
			h.middleware = append(h.middleware, v)
		case ModelRoutingHandlerOption:
			v(h)
		}
	}
	return h
}

// Register registers model routing routes.
func (h *ModelRoutingHandler) Register(e *echo.Echo) {
	g := e.Group("/bots/:bot_id/model-routes", h.middleware...)
	g.GET("", h.ListRoutes)
	g.POST("", h.CreateRoute)
	g.DELETE("/:route_id", h.DeleteRoute)
}

// ListRoutes godoc
// @Summary List model routes for a bot
// @Tags model-routing
// @Param bot_id path string true "Bot ID"
// @Success 200 {array} ModelRouteDTO
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/model-routes [get]
func (h *ModelRoutingHandler) ListRoutes(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	routes, err := h.svc.ListRoutes(c.Request().Context(), botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list model routes")
	}
	return c.JSON(http.StatusOK, routes)
}

// CreateRoute godoc
// @Summary Create a model route
// @Tags model-routing
// @Param bot_id path string true "Bot ID"
// @Param body body CreateModelRouteRequest true "Route data"
// @Success 201 {object} ModelRouteDTO
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/model-routes [post]
func (h *ModelRoutingHandler) CreateRoute(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	var req CreateModelRouteRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if strings.TrimSpace(req.Name) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}
	if strings.TrimSpace(req.ModelID) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "model_id is required")
	}

	route, err := h.svc.CreateRoute(c.Request().Context(), botID, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create model route")
	}
	h.audit.Log(c.Request().Context(), "", botID, "create", "model_route", route.ID, c.RealIP(), c.Request().UserAgent(), nil)
	return c.JSON(http.StatusCreated, route)
}

// DeleteRoute godoc
// @Summary Delete a model route
// @Tags model-routing
// @Param bot_id path string true "Bot ID"
// @Param route_id path string true "Route ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/model-routes/{route_id} [delete]
func (h *ModelRoutingHandler) DeleteRoute(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	routeID := strings.TrimSpace(c.Param("route_id"))
	if routeID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "route_id is required")
	}

	if err := h.svc.DeleteRoute(c.Request().Context(), botID, routeID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete model route")
	}
	h.audit.Log(c.Request().Context(), "", botID, "delete", "model_route", routeID, c.RealIP(), c.Request().UserAgent(), nil)
	return c.NoContent(http.StatusNoContent)
}
