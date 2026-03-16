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

// UpdateDepartmentRequest is the body for updating a department.
type UpdateDepartmentRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ParentID    string `json:"parent_id,omitempty"`
}

// BotDepartmentAssignRequest is the body for assigning a bot to a department.
type BotDepartmentAssignRequest struct {
	DepartmentID string `json:"department_id"`
}

// BotBriefDTO is a compact representation of a bot.
type BotBriefDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
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
	ListAllDepartments(ctx context.Context) ([]DepartmentDTO, error)
	GetDepartment(ctx context.Context, departmentID string) (*DepartmentDTO, error)
	CreateDepartment(ctx context.Context, req CreateDepartmentRequest) (*DepartmentDTO, error)
	UpdateDepartment(ctx context.Context, departmentID string, req UpdateDepartmentRequest) (*DepartmentDTO, error)
	DeleteDepartment(ctx context.Context, departmentID string) error
	ListBotDepartments(ctx context.Context, botID string) ([]DepartmentDTO, error)
	SetBotDepartmentAccess(ctx context.Context, botID, departmentID string) error
	RemoveBotDepartmentAccess(ctx context.Context, botID, departmentID string) error
	ListDepartmentBots(ctx context.Context, departmentID string) ([]BotBriefDTO, error)
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

// WithDepartmentGlobalMiddleware sets middleware for global /departments routes.
func WithDepartmentGlobalMiddleware(mw echo.MiddlewareFunc) DepartmentHandlerOption {
	return func(h *DepartmentHandler) { h.globalMiddleware = append(h.globalMiddleware, mw) }
}

// DepartmentHandler handles department API endpoints.
type DepartmentHandler struct {
	svc              DepartmentServiceInterface
	audit            AuditLoggerInterface
	botMiddleware    []echo.MiddlewareFunc
	globalMiddleware []echo.MiddlewareFunc
}

// NewDepartmentHandler creates a new department handler.
func NewDepartmentHandler(svc DepartmentServiceInterface, opts ...any) *DepartmentHandler {
	h := &DepartmentHandler{svc: svc, audit: noopAuditLogger{}}
	for _, o := range opts {
		switch v := o.(type) {
		case echo.MiddlewareFunc:
			h.botMiddleware = append(h.botMiddleware, v)
		case DepartmentHandlerOption:
			v(h)
		}
	}
	return h
}

// Register registers department routes.
func (h *DepartmentHandler) Register(e *echo.Echo) {
	// Global department management (no bot_id, requires role-based auth)
	g := e.Group("/departments", h.globalMiddleware...)
	g.GET("", h.ListAllDepartments)
	g.POST("", h.CreateDepartment)

	d := g.Group("/:department_id")
	d.GET("", h.GetDepartment)
	d.PUT("", h.UpdateDepartment)
	d.DELETE("", h.DeleteDepartment)
	d.GET("/skill-templates", h.ListDepartmentSkillTemplates)
	d.POST("/skill-templates", h.AddDepartmentSkillTemplate)
	d.DELETE("/skill-templates/:template_id", h.RemoveDepartmentSkillTemplate)
	d.GET("/directory-templates", h.GetDirectoryTemplates)
	d.PUT("/directory-templates", h.UpdateDirectoryTemplates)
	d.POST("/sync-skills", h.SyncSkills)
	d.POST("/sync-directories", h.SyncDirectories)
	d.GET("/bots", h.ListDepartmentBots)

	// Bot-scoped department association (requires bot permission)
	b := e.Group("/bots/:bot_id/departments", h.botMiddleware...)
	b.GET("", h.ListBotDepartments)
	b.POST("", h.AssignBotDepartment)
	b.DELETE("/:department_id", h.RemoveBotDepartment)
}

// ─── Global department handlers ─────────────────────────────────

// ListAllDepartments godoc
// @Summary List all departments
// @Tags departments
// @Success 200 {array} DepartmentDTO
// @Router /departments [get]
func (h *DepartmentHandler) ListAllDepartments(c echo.Context) error {
	deps, err := h.svc.ListAllDepartments(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list departments")
	}
	return c.JSON(http.StatusOK, deps)
}

// GetDepartment godoc
// @Summary Get a department by ID
// @Tags departments
// @Param department_id path string true "Department ID"
// @Success 200 {object} DepartmentDTO
// @Failure 400 {object} ErrorResponse
// @Router /departments/{department_id} [get]
func (h *DepartmentHandler) GetDepartment(c echo.Context) error {
	departmentID := strings.TrimSpace(c.Param("department_id"))
	if departmentID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "department_id is required")
	}

	dept, err := h.svc.GetDepartment(c.Request().Context(), departmentID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get department")
	}
	return c.JSON(http.StatusOK, dept)
}

// CreateDepartment godoc
// @Summary Create a department
// @Tags departments
// @Param body body CreateDepartmentRequest true "Department data"
// @Success 201 {object} DepartmentDTO
// @Failure 400 {object} ErrorResponse
// @Router /departments [post]
func (h *DepartmentHandler) CreateDepartment(c echo.Context) error {
	var req CreateDepartmentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if strings.TrimSpace(req.Name) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	dept, err := h.svc.CreateDepartment(c.Request().Context(), req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create department")
	}
	h.audit.Log(c.Request().Context(), auditUserID(c), "", "create", "department", dept.ID, c.RealIP(), c.Request().UserAgent(), nil)
	return c.JSON(http.StatusCreated, dept)
}

// UpdateDepartment godoc
// @Summary Update a department
// @Tags departments
// @Param department_id path string true "Department ID"
// @Param body body UpdateDepartmentRequest true "Department data"
// @Success 200 {object} DepartmentDTO
// @Failure 400 {object} ErrorResponse
// @Router /departments/{department_id} [put]
func (h *DepartmentHandler) UpdateDepartment(c echo.Context) error {
	departmentID := strings.TrimSpace(c.Param("department_id"))
	if departmentID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "department_id is required")
	}

	var req UpdateDepartmentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if strings.TrimSpace(req.Name) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	dept, err := h.svc.UpdateDepartment(c.Request().Context(), departmentID, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update department")
	}
	h.audit.Log(c.Request().Context(), auditUserID(c), "", "update", "department", dept.ID, c.RealIP(), c.Request().UserAgent(), nil)
	return c.JSON(http.StatusOK, dept)
}

// DeleteDepartment godoc
// @Summary Delete a department
// @Tags departments
// @Param department_id path string true "Department ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Router /departments/{department_id} [delete]
func (h *DepartmentHandler) DeleteDepartment(c echo.Context) error {
	departmentID := strings.TrimSpace(c.Param("department_id"))
	if departmentID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "department_id is required")
	}

	if err := h.svc.DeleteDepartment(c.Request().Context(), departmentID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete department")
	}
	h.audit.Log(c.Request().Context(), auditUserID(c), "", "delete", "department", departmentID, c.RealIP(), c.Request().UserAgent(), nil)
	return c.NoContent(http.StatusNoContent)
}

// ListDepartmentBots godoc
// @Summary List bots in a department
// @Tags departments
// @Param department_id path string true "Department ID"
// @Success 200 {array} BotBriefDTO
// @Failure 400 {object} ErrorResponse
// @Router /departments/{department_id}/bots [get]
func (h *DepartmentHandler) ListDepartmentBots(c echo.Context) error {
	departmentID := strings.TrimSpace(c.Param("department_id"))
	if departmentID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "department_id is required")
	}

	bots, err := h.svc.ListDepartmentBots(c.Request().Context(), departmentID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list department bots")
	}
	return c.JSON(http.StatusOK, bots)
}

// ─── Bot-scoped department handlers ─────────────────────────────

// ListBotDepartments godoc
// @Summary List departments for a bot
// @Tags departments
// @Param bot_id path string true "Bot ID"
// @Success 200 {array} DepartmentDTO
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/departments [get]
func (h *DepartmentHandler) ListBotDepartments(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	deps, err := h.svc.ListBotDepartments(c.Request().Context(), botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list departments")
	}
	return c.JSON(http.StatusOK, deps)
}

// AssignBotDepartment godoc
// @Summary Assign a bot to a department
// @Tags departments
// @Param bot_id path string true "Bot ID"
// @Param body body BotDepartmentAssignRequest true "Department to assign"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/departments [post]
func (h *DepartmentHandler) AssignBotDepartment(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	var req BotDepartmentAssignRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if strings.TrimSpace(req.DepartmentID) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "department_id is required")
	}

	if err := h.svc.SetBotDepartmentAccess(c.Request().Context(), botID, req.DepartmentID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to assign bot to department")
	}
	h.audit.Log(c.Request().Context(), auditUserID(c), botID, "assign", "bot_department", req.DepartmentID, c.RealIP(), c.Request().UserAgent(), nil)
	return c.NoContent(http.StatusNoContent)
}

// RemoveBotDepartment godoc
// @Summary Remove a bot from a department
// @Tags departments
// @Param bot_id path string true "Bot ID"
// @Param department_id path string true "Department ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/departments/{department_id} [delete]
func (h *DepartmentHandler) RemoveBotDepartment(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}
	departmentID := strings.TrimSpace(c.Param("department_id"))
	if departmentID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "department_id is required")
	}

	if err := h.svc.RemoveBotDepartmentAccess(c.Request().Context(), botID, departmentID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to remove bot from department")
	}
	h.audit.Log(c.Request().Context(), auditUserID(c), botID, "unassign", "bot_department", departmentID, c.RealIP(), c.Request().UserAgent(), nil)
	return c.NoContent(http.StatusNoContent)
}

// ─── Department detail handlers (skill templates, directory templates, sync) ──

// ListDepartmentSkillTemplates godoc
// @Summary List skill templates assigned to a department
// @Tags departments
// @Param department_id path string true "Department ID"
// @Success 200 {array} SkillTemplateBriefDTO
// @Failure 400 {object} ErrorResponse
// @Router /departments/{department_id}/skill-templates [get]
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
// @Param department_id path string true "Department ID"
// @Param body body AddSkillTemplateRequest true "Skill template to add"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Router /departments/{department_id}/skill-templates [post]
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
// @Param department_id path string true "Department ID"
// @Param template_id path string true "Template ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Router /departments/{department_id}/skill-templates/{template_id} [delete]
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
// @Param department_id path string true "Department ID"
// @Success 200 {object} DirectoryTemplatesRequest
// @Failure 400 {object} ErrorResponse
// @Router /departments/{department_id}/directory-templates [get]
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
// @Param department_id path string true "Department ID"
// @Param body body DirectoryTemplatesRequest true "Directory templates"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Router /departments/{department_id}/directory-templates [put]
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
// @Param department_id path string true "Department ID"
// @Success 200 {object} SyncResultDTO
// @Failure 400 {object} ErrorResponse
// @Router /departments/{department_id}/sync-skills [post]
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
// @Param department_id path string true "Department ID"
// @Success 200 {object} SyncResultDTO
// @Failure 400 {object} ErrorResponse
// @Router /departments/{department_id}/sync-directories [post]
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
