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

// SkillTemplateBriefDTO is a compact representation of a skill template.
type SkillTemplateBriefDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     int32  `json:"version"`
}

// AddSkillTemplateRequest is the body for adding a skill template to a department.
type AddSkillTemplateRequest struct {
	TemplateID string `json:"template_id"`
}

// DirectoryTemplatesRequest is the body for updating directory templates.
type DirectoryTemplatesRequest struct {
	Paths []string `json:"paths"`
}

// SyncResultDTO is the result of a sync operation.
type SyncResultDTO struct {
	TotalBots int            `json:"total_bots"`
	Installed int            `json:"installed"`
	Skipped   int            `json:"skipped"`
	Errors    []BotSyncError `json:"errors,omitempty"`
}

// BotSyncError describes a sync error for a single bot.
type BotSyncError struct {
	BotID   string `json:"bot_id"`
	Message string `json:"message"`
}

// DepartmentServiceInterface abstracts department operations for the handler.
type DepartmentServiceInterface interface {
	ListDepartments(ctx context.Context, botID string) ([]DepartmentDTO, error)
	CreateDepartment(ctx context.Context, botID string, req CreateDepartmentRequest) (*DepartmentDTO, error)
	AddSkillTemplate(ctx context.Context, departmentID, templateID string) error
	RemoveSkillTemplate(ctx context.Context, departmentID, templateID string) error
	ListSkillTemplates(ctx context.Context, departmentID string) ([]SkillTemplateBriefDTO, error)
	GetDirectoryTemplates(ctx context.Context, departmentID string) ([]string, error)
	UpdateDirectoryTemplates(ctx context.Context, departmentID string, paths []string) error
	SyncSkills(ctx context.Context, departmentID string) (*SyncResultDTO, error)
	SyncDirectories(ctx context.Context, departmentID string) (*SyncResultDTO, error)
}

// DepartmentHandlerOption configures optional dependencies.
type DepartmentHandlerOption func(*DepartmentHandler)

// WithDepartmentAudit sets the audit logger for department mutations.
func WithDepartmentAudit(al AuditLoggerInterface) DepartmentHandlerOption {
	return func(h *DepartmentHandler) { h.audit = al }
}

// DepartmentHandler handles department API endpoints.
type DepartmentHandler struct {
	svc        DepartmentServiceInterface
	audit      AuditLoggerInterface
	middleware []echo.MiddlewareFunc
}

// NewDepartmentHandler creates a new department handler.
func NewDepartmentHandler(svc DepartmentServiceInterface, opts ...any) *DepartmentHandler {
	h := &DepartmentHandler{svc: svc, audit: noopAuditLogger{}}
	for _, o := range opts {
		switch v := o.(type) {
		case echo.MiddlewareFunc:
			h.middleware = append(h.middleware, v)
		case DepartmentHandlerOption:
			v(h)
		}
	}
	return h
}

// Register registers department routes.
func (h *DepartmentHandler) Register(e *echo.Echo) {
	g := e.Group("/bots/:bot_id/departments", h.middleware...)
	g.GET("", h.ListDepartments)
	g.POST("", h.CreateDepartment)

	// Department detail routes are nested under /bots/:bot_id so that
	// RequireBotPermission middleware can extract bot_id for authorization.
	d := g.Group("/:department_id")
	d.GET("/skill-templates", h.ListDepartmentSkillTemplates)
	d.POST("/skill-templates", h.AddDepartmentSkillTemplate)
	d.DELETE("/skill-templates/:template_id", h.RemoveDepartmentSkillTemplate)
	d.GET("/directory-templates", h.GetDirectoryTemplates)
	d.PUT("/directory-templates", h.UpdateDirectoryTemplates)
	d.POST("/sync-skills", h.SyncSkills)
	d.POST("/sync-directories", h.SyncDirectories)
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
	h.audit.Log(c.Request().Context(), auditUserID(c), botID, "create", "department", dept.ID, c.RealIP(), c.Request().UserAgent(), nil)
	return c.JSON(http.StatusCreated, dept)
}

// ListDepartmentSkillTemplates godoc
// @Summary List skill templates assigned to a department
// @Tags departments
// @Param bot_id path string true "Bot ID"
// @Param department_id path string true "Department ID"
// @Success 200 {array} SkillTemplateBriefDTO
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/departments/{department_id}/skill-templates [get]
func (h *DepartmentHandler) ListDepartmentSkillTemplates(c echo.Context) error {
	departmentID := strings.TrimSpace(c.Param("department_id"))
	if departmentID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "department_id is required")
	}

	templates, err := h.svc.ListSkillTemplates(c.Request().Context(), departmentID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list skill templates")
	}
	return c.JSON(http.StatusOK, templates)
}

// AddDepartmentSkillTemplate godoc
// @Summary Add a skill template to a department
// @Tags departments
// @Param bot_id path string true "Bot ID"
// @Param department_id path string true "Department ID"
// @Param body body AddSkillTemplateRequest true "Skill template to add"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/departments/{department_id}/skill-templates [post]
func (h *DepartmentHandler) AddDepartmentSkillTemplate(c echo.Context) error {
	departmentID := strings.TrimSpace(c.Param("department_id"))
	if departmentID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "department_id is required")
	}

	var req AddSkillTemplateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if strings.TrimSpace(req.TemplateID) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "template_id is required")
	}

	if err := h.svc.AddSkillTemplate(c.Request().Context(), departmentID, req.TemplateID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to add skill template")
	}
	return c.NoContent(http.StatusNoContent)
}

// RemoveDepartmentSkillTemplate godoc
// @Summary Remove a skill template from a department
// @Tags departments
// @Param bot_id path string true "Bot ID"
// @Param department_id path string true "Department ID"
// @Param template_id path string true "Template ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/departments/{department_id}/skill-templates/{template_id} [delete]
func (h *DepartmentHandler) RemoveDepartmentSkillTemplate(c echo.Context) error {
	departmentID := strings.TrimSpace(c.Param("department_id"))
	if departmentID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "department_id is required")
	}
	templateID := strings.TrimSpace(c.Param("template_id"))
	if templateID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "template_id is required")
	}

	if err := h.svc.RemoveSkillTemplate(c.Request().Context(), departmentID, templateID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to remove skill template")
	}
	return c.NoContent(http.StatusNoContent)
}

// GetDirectoryTemplates godoc
// @Summary Get directory templates for a department
// @Tags departments
// @Param bot_id path string true "Bot ID"
// @Param department_id path string true "Department ID"
// @Success 200 {object} DirectoryTemplatesRequest
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/departments/{department_id}/directory-templates [get]
func (h *DepartmentHandler) GetDirectoryTemplates(c echo.Context) error {
	departmentID := strings.TrimSpace(c.Param("department_id"))
	if departmentID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "department_id is required")
	}

	paths, err := h.svc.GetDirectoryTemplates(c.Request().Context(), departmentID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get directory templates")
	}
	return c.JSON(http.StatusOK, DirectoryTemplatesRequest{Paths: paths})
}

// UpdateDirectoryTemplates godoc
// @Summary Update directory templates for a department
// @Tags departments
// @Param bot_id path string true "Bot ID"
// @Param department_id path string true "Department ID"
// @Param body body DirectoryTemplatesRequest true "Directory templates"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/departments/{department_id}/directory-templates [put]
func (h *DepartmentHandler) UpdateDirectoryTemplates(c echo.Context) error {
	departmentID := strings.TrimSpace(c.Param("department_id"))
	if departmentID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "department_id is required")
	}

	var req DirectoryTemplatesRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := h.svc.UpdateDirectoryTemplates(c.Request().Context(), departmentID, req.Paths); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

// SyncSkills godoc
// @Summary Sync skill templates to all bots in a department
// @Tags departments
// @Param bot_id path string true "Bot ID"
// @Param department_id path string true "Department ID"
// @Success 200 {object} SyncResultDTO
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/departments/{department_id}/sync-skills [post]
func (h *DepartmentHandler) SyncSkills(c echo.Context) error {
	departmentID := strings.TrimSpace(c.Param("department_id"))
	if departmentID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "department_id is required")
	}

	result, err := h.svc.SyncSkills(c.Request().Context(), departmentID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to sync skills")
	}
	return c.JSON(http.StatusOK, result)
}

// SyncDirectories godoc
// @Summary Sync directory templates to all bots in a department
// @Tags departments
// @Param bot_id path string true "Bot ID"
// @Param department_id path string true "Department ID"
// @Success 200 {object} SyncResultDTO
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/departments/{department_id}/sync-directories [post]
func (h *DepartmentHandler) SyncDirectories(c echo.Context) error {
	departmentID := strings.TrimSpace(c.Param("department_id"))
	if departmentID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "department_id is required")
	}

	result, err := h.svc.SyncDirectories(c.Request().Context(), departmentID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to sync directories")
	}
	return c.JSON(http.StatusOK, result)
}
