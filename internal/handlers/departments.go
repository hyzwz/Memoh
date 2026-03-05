package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// DepartmentDTO is the API representation of a department.
type DepartmentDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ParentID    string `json:"parent_id,omitempty"`
}

// CreateDepartmentRequest is the body for creating a department.
type CreateDepartmentRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ParentID    string `json:"parent_id,omitempty"`
}

// DepartmentServiceInterface abstracts department operations for the handler.
type DepartmentServiceInterface interface {
	ListDepartments(ctx context.Context, botID string) ([]DepartmentDTO, error)
	CreateDepartment(ctx context.Context, botID string, req CreateDepartmentRequest) (*DepartmentDTO, error)
}

// DepartmentHandler handles department API endpoints.
type DepartmentHandler struct {
	svc DepartmentServiceInterface
}

// NewDepartmentHandler creates a new department handler.
func NewDepartmentHandler(svc DepartmentServiceInterface) *DepartmentHandler {
	return &DepartmentHandler{svc: svc}
}

// Register registers department routes.
func (h *DepartmentHandler) Register(e *echo.Echo) {
	g := e.Group("/bots/:bot_id/departments")
	g.GET("", h.ListDepartments)
	g.POST("", h.CreateDepartment)
}

// ListDepartments godoc
// @Summary List departments for a bot
// @Tags departments
// @Param bot_id path string true "Bot ID"
// @Success 200 {array} DepartmentDTO
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/departments [get]
func (h *DepartmentHandler) ListDepartments(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	deps, err := h.svc.ListDepartments(c.Request().Context(), botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list departments")
	}
	return c.JSON(http.StatusOK, deps)
}

// CreateDepartment godoc
// @Summary Create a department
// @Tags departments
// @Param bot_id path string true "Bot ID"
// @Param body body CreateDepartmentRequest true "Department data"
// @Success 201 {object} DepartmentDTO
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/departments [post]
func (h *DepartmentHandler) CreateDepartment(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	var req CreateDepartmentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if strings.TrimSpace(req.Name) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	dept, err := h.svc.CreateDepartment(c.Request().Context(), botID, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create department")
	}
	return c.JSON(http.StatusCreated, dept)
}
