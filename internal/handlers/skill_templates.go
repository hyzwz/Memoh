package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

	"github.com/memohai/memoh/internal/accounts"
	"github.com/memohai/memoh/internal/bots"
	dbsqlc "github.com/memohai/memoh/internal/db/sqlc"
	"github.com/memohai/memoh/internal/mcp"
)

type SkillTemplateHandler struct {
	logger         *slog.Logger
	queries        *dbsqlc.Queries
	manager        *mcp.Manager
	botService     *bots.Service
	accountService *accounts.Service
}

func NewSkillTemplateHandler(
	log *slog.Logger,
	queries *dbsqlc.Queries,
	manager *mcp.Manager,
	botService *bots.Service,
	accountService *accounts.Service,
) *SkillTemplateHandler {
	return &SkillTemplateHandler{
		logger:         log.With(slog.String("handler", "skill_templates")),
		queries:        queries,
		manager:        manager,
		botService:     botService,
		accountService: accountService,
	}
}

func (h *SkillTemplateHandler) Register(e *echo.Echo) {
	tpl := e.Group("/skill-templates")
	tpl.GET("", h.List)
	tpl.POST("", h.Create)
	tpl.GET("/:id", h.GetByID)
	tpl.PUT("/:id", h.Update)
	tpl.DELETE("/:id", h.Delete)

	store := e.Group("/bots/:bot_id/skill-store")
	store.GET("", h.BrowseStore)
	store.POST("/install", h.Install)
	store.POST("/uninstall", h.Uninstall)
	store.POST("/update", h.UpdateInstall)
}

// --- request/response types ---

type SkillTemplateRequest struct {
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Content     string   `json:"content"`
	Author      string   `json:"author"`
	Icon        string   `json:"icon"`
	Tags        []string `json:"tags"`
	IsPublished bool     `json:"is_published"`
}

type SkillTemplateResponse struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Slug         string   `json:"slug"`
	Description  string   `json:"description"`
	Category     string   `json:"category"`
	Content      string   `json:"content"`
	Version      int32    `json:"version"`
	Author       string   `json:"author"`
	Icon         string   `json:"icon"`
	Tags         []string `json:"tags"`
	IsPublished  bool     `json:"is_published"`
	InstallCount int32    `json:"install_count"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

type StoreTemplateResponse struct {
	SkillTemplateResponse
	Installed        bool  `json:"installed"`
	InstalledVersion int32 `json:"installed_version,omitempty"`
	UpdateAvailable  bool  `json:"update_available"`
	Customized       bool  `json:"customized"`
}

type InstallRequest struct {
	TemplateID string `json:"template_id"`
}

type UninstallRequest struct {
	TemplateID string `json:"template_id"`
}

type UpdateInstallRequest struct {
	TemplateID string `json:"template_id"`
}

func templateToResponse(t dbsqlc.SkillTemplate) SkillTemplateResponse {
	tags := t.Tags
	if tags == nil {
		tags = []string{}
	}
	return SkillTemplateResponse{
		ID:           uuidToString(t.ID),
		Name:         t.Name,
		Slug:         t.Slug,
		Description:  t.Description,
		Category:     t.Category,
		Content:      t.Content,
		Version:      t.Version,
		Author:       t.Author,
		Icon:         t.Icon,
		Tags:         tags,
		IsPublished:  t.IsPublished,
		InstallCount: t.InstallCount,
		CreatedAt:    t.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    t.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return u, fmt.Errorf("invalid UUID: %s", s)
	}
	return u, nil
}

// --- template CRUD ---

// List godoc
// @Summary List skill templates
// @Tags skill-templates
// @Param category query string false "Filter by category"
// @Success 200 {array} SkillTemplateResponse
// @Failure 500 {object} ErrorResponse
// @Router /skill-templates [get].
func (h *SkillTemplateHandler) List(c echo.Context) error {
	ctx := c.Request().Context()
	category := strings.TrimSpace(c.QueryParam("category"))

	var templates []dbsqlc.SkillTemplate
	var err error

	if category != "" {
		templates, err = h.queries.ListSkillTemplatesByCategory(ctx, category)
	} else {
		templates, err = h.queries.ListSkillTemplates(ctx)
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resp := make([]SkillTemplateResponse, len(templates))
	for i, t := range templates {
		resp[i] = templateToResponse(t)
	}
	return c.JSON(http.StatusOK, resp)
}

// Create godoc
// @Summary Create a skill template
// @Tags skill-templates
// @Param payload body SkillTemplateRequest true "Template data"
// @Success 201 {object} SkillTemplateResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /skill-templates [post].
func (h *SkillTemplateHandler) Create(c echo.Context) error {
	var req SkillTemplateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Content) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name and content are required")
	}
	if strings.TrimSpace(req.Slug) == "" {
		req.Slug = slugify(req.Name)
	}
	if req.Category == "" {
		req.Category = "general"
	}
	if req.Tags == nil {
		req.Tags = []string{}
	}

	t, err := h.queries.CreateSkillTemplate(c.Request().Context(), dbsqlc.CreateSkillTemplateParams{
		Name:        strings.TrimSpace(req.Name),
		Slug:        strings.TrimSpace(req.Slug),
		Description: strings.TrimSpace(req.Description),
		Category:    req.Category,
		Content:     req.Content,
		Author:      strings.TrimSpace(req.Author),
		Icon:        strings.TrimSpace(req.Icon),
		Tags:        req.Tags,
		IsPublished: req.IsPublished,
	})
	if err != nil {
		if strings.Contains(err.Error(), "skill_templates_slug_unique") {
			return echo.NewHTTPError(http.StatusConflict, "slug already exists")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, templateToResponse(t))
}

// GetByID godoc
// @Summary Get a skill template by ID
// @Tags skill-templates
// @Param id path string true "Template ID"
// @Success 200 {object} SkillTemplateResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /skill-templates/{id} [get].
func (h *SkillTemplateHandler) GetByID(c echo.Context) error {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	t, err := h.queries.GetSkillTemplateByID(c.Request().Context(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "template not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, templateToResponse(t))
}

// Update godoc
// @Summary Update a skill template
// @Tags skill-templates
// @Param id path string true "Template ID"
// @Param payload body SkillTemplateRequest true "Template data"
// @Success 200 {object} SkillTemplateResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /skill-templates/{id} [put].
func (h *SkillTemplateHandler) Update(c echo.Context) error {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	var req SkillTemplateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Content) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name and content are required")
	}
	if strings.TrimSpace(req.Slug) == "" {
		req.Slug = slugify(req.Name)
	}
	if req.Tags == nil {
		req.Tags = []string{}
	}

	t, err := h.queries.UpdateSkillTemplate(c.Request().Context(), dbsqlc.UpdateSkillTemplateParams{
		ID:          id,
		Name:        strings.TrimSpace(req.Name),
		Slug:        strings.TrimSpace(req.Slug),
		Description: strings.TrimSpace(req.Description),
		Category:    req.Category,
		Content:     req.Content,
		Author:      strings.TrimSpace(req.Author),
		Icon:        strings.TrimSpace(req.Icon),
		Tags:        req.Tags,
		IsPublished: req.IsPublished,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "template not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, templateToResponse(t))
}

// Delete godoc
// @Summary Delete a skill template
// @Tags skill-templates
// @Param id path string true "Template ID"
// @Success 204
// @Failure 500 {object} ErrorResponse
// @Router /skill-templates/{id} [delete].
func (h *SkillTemplateHandler) Delete(c echo.Context) error {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := h.queries.DeleteSkillTemplate(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

// --- bot skill store ---

// BrowseStore godoc
// @Summary Browse skill template store for a bot
// @Tags skill-store
// @Param bot_id path string true "Bot ID"
// @Param category query string false "Filter by category"
// @Success 200 {array} StoreTemplateResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/skill-store [get].
func (h *SkillTemplateHandler) BrowseStore(c echo.Context) error {
	botID, err := h.requireBotAccess(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	category := strings.TrimSpace(c.QueryParam("category"))

	var templates []dbsqlc.SkillTemplate
	if category != "" {
		templates, err = h.queries.ListSkillTemplatesByCategory(ctx, category)
	} else {
		templates, err = h.queries.ListPublishedSkillTemplates(ctx)
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	botUUID, _ := parseUUID(botID)
	installs, err := h.queries.ListBotSkillInstalls(ctx, botUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	installMap := make(map[string]dbsqlc.ListBotSkillInstallsRow)
	for _, inst := range installs {
		installMap[uuidToString(inst.TemplateID)] = inst
	}

	resp := make([]StoreTemplateResponse, len(templates))
	for i, t := range templates {
		tID := uuidToString(t.ID)
		item := StoreTemplateResponse{
			SkillTemplateResponse: templateToResponse(t),
		}
		if inst, ok := installMap[tID]; ok {
			item.Installed = true
			item.InstalledVersion = inst.InstalledVersion
			item.UpdateAvailable = t.Version > inst.InstalledVersion
			item.Customized = inst.Customized
		}
		resp[i] = item
	}
	return c.JSON(http.StatusOK, resp)
}

// Install godoc
// @Summary Install a skill template to a bot
// @Tags skill-store
// @Param bot_id path string true "Bot ID"
// @Param payload body InstallRequest true "Install request"
// @Success 200 {object} skillsOpResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/skill-store/install [post].
func (h *SkillTemplateHandler) Install(c echo.Context) error {
	botID, err := h.requireBotAccess(c)
	if err != nil {
		return err
	}

	var req InstallRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	templateUUID, err := parseUUID(req.TemplateID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid template_id")
	}

	ctx := c.Request().Context()

	tmpl, err := h.queries.GetSkillTemplateByID(ctx, templateUUID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "template not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	parsed := parseSkillFile(tmpl.Content, tmpl.Slug)
	if !isValidSkillName(parsed.Name) {
		parsed.Name = tmpl.Slug
	}

	client, err := h.manager.MCPClient(ctx, botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("container not reachable: %v", err))
	}

	dirPath := path.Join(skillsDirPath, parsed.Name)
	if err := client.Mkdir(ctx, dirPath); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("mkdir failed: %v", err))
	}
	filePath := path.Join(dirPath, "SKILL.md")
	if err := client.WriteFile(ctx, filePath, []byte(tmpl.Content)); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("write failed: %v", err))
	}

	botUUID, _ := parseUUID(botID)
	_, err = h.queries.CreateBotSkillInstall(ctx, dbsqlc.CreateBotSkillInstallParams{
		BotID:            botUUID,
		TemplateID:       templateUUID,
		InstalledVersion: tmpl.Version,
		SkillName:        parsed.Name,
	})
	if err != nil {
		if strings.Contains(err.Error(), "bot_skill_installs_bot_template") {
			return echo.NewHTTPError(http.StatusConflict, "template already installed")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	_ = h.queries.IncrementInstallCount(ctx, templateUUID)

	return c.JSON(http.StatusOK, skillsOpResponse{OK: true})
}

// Uninstall godoc
// @Summary Uninstall a skill template from a bot
// @Tags skill-store
// @Param bot_id path string true "Bot ID"
// @Param payload body UninstallRequest true "Uninstall request"
// @Success 200 {object} skillsOpResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/skill-store/uninstall [post].
func (h *SkillTemplateHandler) Uninstall(c echo.Context) error {
	botID, err := h.requireBotAccess(c)
	if err != nil {
		return err
	}

	var req UninstallRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	templateUUID, err := parseUUID(req.TemplateID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid template_id")
	}

	ctx := c.Request().Context()
	botUUID, _ := parseUUID(botID)

	install, err := h.queries.GetBotSkillInstall(ctx, dbsqlc.GetBotSkillInstallParams{
		BotID:      botUUID,
		TemplateID: templateUUID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "install not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	client, err := h.manager.MCPClient(ctx, botID)
	if err == nil {
		_ = client.DeleteFile(ctx, path.Join(skillsDirPath, install.SkillName), true)
	}

	if err := h.queries.DeleteBotSkillInstall(ctx, dbsqlc.DeleteBotSkillInstallParams{
		BotID:      botUUID,
		TemplateID: templateUUID,
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	_ = h.queries.DecrementInstallCount(ctx, templateUUID)

	return c.JSON(http.StatusOK, skillsOpResponse{OK: true})
}

// UpdateInstall godoc
// @Summary Update an installed skill to the latest template version
// @Tags skill-store
// @Param bot_id path string true "Bot ID"
// @Param payload body UpdateInstallRequest true "Update request"
// @Success 200 {object} skillsOpResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/skill-store/update [post].
func (h *SkillTemplateHandler) UpdateInstall(c echo.Context) error {
	botID, err := h.requireBotAccess(c)
	if err != nil {
		return err
	}

	var req UpdateInstallRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	templateUUID, err := parseUUID(req.TemplateID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid template_id")
	}

	ctx := c.Request().Context()
	botUUID, _ := parseUUID(botID)

	install, err := h.queries.GetBotSkillInstall(ctx, dbsqlc.GetBotSkillInstallParams{
		BotID:      botUUID,
		TemplateID: templateUUID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "install not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	tmpl, err := h.queries.GetSkillTemplateByID(ctx, templateUUID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "template not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	client, err := h.manager.MCPClient(ctx, botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("container not reachable: %v", err))
	}

	dirPath := path.Join(skillsDirPath, install.SkillName)
	if err := client.Mkdir(ctx, dirPath); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("mkdir failed: %v", err))
	}
	filePath := path.Join(dirPath, "SKILL.md")
	if err := client.WriteFile(ctx, filePath, []byte(tmpl.Content)); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("write failed: %v", err))
	}

	_, err = h.queries.UpdateBotSkillInstallVersion(ctx, dbsqlc.UpdateBotSkillInstallVersionParams{
		InstalledVersion: tmpl.Version,
		BotID:            botUUID,
		TemplateID:       templateUUID,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, skillsOpResponse{OK: true})
}

// --- helpers ---

func (h *SkillTemplateHandler) requireBotAccess(c echo.Context) (string, error) {
	channelIdentityID, err := RequireChannelIdentityID(c)
	if err != nil {
		return "", err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return "", echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	if _, err := AuthorizeBotAccess(c.Request().Context(), h.botService, h.accountService, channelIdentityID, botID, bots.AccessPolicy{AllowPublicMember: false}); err != nil {
		return "", err
	}
	return botID, nil
}

func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r == ' ' || r == '_' || r == '-' {
			return '-'
		}
		return -1
	}, s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}
