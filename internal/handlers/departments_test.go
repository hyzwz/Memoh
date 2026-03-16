package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

type fakeDepartmentService struct {
	departments    []DepartmentDTO
	skillTemplates []SkillTemplateBriefDTO
	dirTemplates   []string
	syncResult     *SyncResultDTO
	bots           []BotBriefDTO
	err            error
}

func (f *fakeDepartmentService) ListAllDepartments(_ context.Context) ([]DepartmentDTO, error) {
	return f.departments, f.err
}

func (f *fakeDepartmentService) GetDepartment(_ context.Context, _ string) (*DepartmentDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	if len(f.departments) > 0 {
		return &f.departments[0], nil
	}
	return &DepartmentDTO{ID: "dept-1", Name: "Engineering"}, nil
}

func (f *fakeDepartmentService) CreateDepartment(_ context.Context, _ CreateDepartmentRequest) (*DepartmentDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &DepartmentDTO{ID: "dept-1", Name: "Engineering"}, nil
}

func (f *fakeDepartmentService) UpdateDepartment(_ context.Context, _ string, _ UpdateDepartmentRequest) (*DepartmentDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &DepartmentDTO{ID: "dept-1", Name: "Engineering Updated"}, nil
}

func (f *fakeDepartmentService) DeleteDepartment(_ context.Context, _ string) error {
	return f.err
}

func (f *fakeDepartmentService) ListBotDepartments(_ context.Context, _ string) ([]DepartmentDTO, error) {
	return f.departments, f.err
}

func (f *fakeDepartmentService) SetBotDepartmentAccess(_ context.Context, _, _ string) error {
	return f.err
}

func (f *fakeDepartmentService) RemoveBotDepartmentAccess(_ context.Context, _, _ string) error {
	return f.err
}

func (f *fakeDepartmentService) ListDepartmentBots(_ context.Context, _ string) ([]BotBriefDTO, error) {
	return f.bots, f.err
}

func (f *fakeDepartmentService) AddSkillTemplate(_ context.Context, _, _ string) error {
	return f.err
}

func (f *fakeDepartmentService) RemoveSkillTemplate(_ context.Context, _, _ string) error {
	return f.err
}

func (f *fakeDepartmentService) ListSkillTemplates(_ context.Context, _ string) ([]SkillTemplateBriefDTO, error) {
	return f.skillTemplates, f.err
}

func (f *fakeDepartmentService) GetDirectoryTemplates(_ context.Context, _ string) ([]string, error) {
	return f.dirTemplates, f.err
}

func (f *fakeDepartmentService) UpdateDirectoryTemplates(_ context.Context, _ string, paths []string) error {
	if f.err != nil {
		return f.err
	}
	f.dirTemplates = paths
	return nil
}

func (f *fakeDepartmentService) SyncSkills(_ context.Context, _ string) (*SyncResultDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.syncResult, nil
}

func (f *fakeDepartmentService) SyncDirectories(_ context.Context, _ string) (*SyncResultDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.syncResult, nil
}

// --- Global department tests ---

func TestDepartmentHandlerListAll(t *testing.T) {
	svc := &fakeDepartmentService{
		departments: []DepartmentDTO{
			{ID: "dept-1", Name: "Engineering"},
			{ID: "dept-2", Name: "Design"},
		},
	}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/departments", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.ListAllDepartments(c); err != nil {
		t.Fatalf("ListAllDepartments: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Engineering") {
		t.Fatalf("body missing Engineering: %s", body)
	}
}

func TestDepartmentHandlerCreate(t *testing.T) {
	svc := &fakeDepartmentService{}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	reqBody := `{"name":"Engineering","description":"Eng team"}`
	req := httptest.NewRequest(http.MethodPost, "/departments", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.CreateDepartment(c); err != nil {
		t.Fatalf("CreateDepartment: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}
}

// --- Bot-scoped department tests ---

func TestDepartmentHandlerListBotDepartments(t *testing.T) {
	svc := &fakeDepartmentService{
		departments: []DepartmentDTO{
			{ID: "dept-1", Name: "Engineering"},
		},
	}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bots/bot-1/departments", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")

	if err := h.ListBotDepartments(c); err != nil {
		t.Fatalf("ListBotDepartments: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestDepartmentHandlerMissingBotID(t *testing.T) {
	svc := &fakeDepartmentService{}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bots//departments", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("")

	err := h.ListBotDepartments(c)
	if err == nil {
		t.Fatal("expected error for missing bot_id")
	}
}

func TestAssignBotDepartment(t *testing.T) {
	svc := &fakeDepartmentService{}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	body := `{"department_id":"dept-1"}`
	req := httptest.NewRequest(http.MethodPost, "/bots/bot-1/departments", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")

	if err := h.AssignBotDepartment(c); err != nil {
		t.Fatalf("AssignBotDepartment: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestRemoveBotDepartment(t *testing.T) {
	svc := &fakeDepartmentService{}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/bots/bot-1/departments/dept-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id", "department_id")
	c.SetParamValues("bot-1", "dept-1")

	if err := h.RemoveBotDepartment(c); err != nil {
		t.Fatalf("RemoveBotDepartment: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

// --- Skill template handler tests ---

func TestListDepartmentSkillTemplates(t *testing.T) {
	svc := &fakeDepartmentService{
		skillTemplates: []SkillTemplateBriefDTO{
			{ID: "tpl-1", Name: "Greeting", Description: "Says hello", Version: 1},
			{ID: "tpl-2", Name: "Summary", Description: "Summarizes text", Version: 2},
		},
	}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/departments/dept-1/skill-templates", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("department_id")
	c.SetParamValues("dept-1")

	if err := h.ListDepartmentSkillTemplates(c); err != nil {
		t.Fatalf("ListDepartmentSkillTemplates: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Greeting") || !strings.Contains(body, "Summary") {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestListDepartmentSkillTemplates_MissingID(t *testing.T) {
	h := NewDepartmentHandler(&fakeDepartmentService{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/departments//skill-templates", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("department_id")
	c.SetParamValues("")

	if err := h.ListDepartmentSkillTemplates(c); err == nil {
		t.Fatal("expected error for missing department_id")
	}
}

func TestAddDepartmentSkillTemplate(t *testing.T) {
	svc := &fakeDepartmentService{}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	body := `{"template_id":"tpl-123"}`
	req := httptest.NewRequest(http.MethodPost, "/departments/dept-1/skill-templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("department_id")
	c.SetParamValues("dept-1")

	if err := h.AddDepartmentSkillTemplate(c); err != nil {
		t.Fatalf("AddDepartmentSkillTemplate: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestAddDepartmentSkillTemplate_MissingTemplateID(t *testing.T) {
	h := NewDepartmentHandler(&fakeDepartmentService{})
	e := echo.New()
	body := `{"template_id":""}`
	req := httptest.NewRequest(http.MethodPost, "/departments/dept-1/skill-templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("department_id")
	c.SetParamValues("dept-1")

	if err := h.AddDepartmentSkillTemplate(c); err == nil {
		t.Fatal("expected error for missing template_id")
	}
}

func TestAddDepartmentSkillTemplate_ServiceError(t *testing.T) {
	svc := &fakeDepartmentService{err: errors.New("db error")}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	body := `{"template_id":"tpl-123"}`
	req := httptest.NewRequest(http.MethodPost, "/departments/dept-1/skill-templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("department_id")
	c.SetParamValues("dept-1")

	err := h.AddDepartmentSkillTemplate(c)
	if err == nil {
		t.Fatal("expected error from service")
	}
}

func TestRemoveDepartmentSkillTemplate(t *testing.T) {
	svc := &fakeDepartmentService{}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/departments/dept-1/skill-templates/tpl-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("department_id", "template_id")
	c.SetParamValues("dept-1", "tpl-1")

	if err := h.RemoveDepartmentSkillTemplate(c); err != nil {
		t.Fatalf("RemoveDepartmentSkillTemplate: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestRemoveDepartmentSkillTemplate_MissingTemplateID(t *testing.T) {
	h := NewDepartmentHandler(&fakeDepartmentService{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/departments/dept-1/skill-templates/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("department_id", "template_id")
	c.SetParamValues("dept-1", "")

	if err := h.RemoveDepartmentSkillTemplate(c); err == nil {
		t.Fatal("expected error for missing template_id")
	}
}

// --- Directory template handler tests ---

func TestGetDirectoryTemplates(t *testing.T) {
	svc := &fakeDepartmentService{
		dirTemplates: []string{"/data/reports", "/data/exports"},
	}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/departments/dept-1/directory-templates", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("department_id")
	c.SetParamValues("dept-1")

	if err := h.GetDirectoryTemplates(c); err != nil {
		t.Fatalf("GetDirectoryTemplates: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "/data/reports") {
		t.Fatalf("body missing /data/reports: %s", body)
	}
}

func TestUpdateDirectoryTemplates(t *testing.T) {
	svc := &fakeDepartmentService{}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	body := `{"paths":["/data/reports","/data/docs"]}`
	req := httptest.NewRequest(http.MethodPut, "/departments/dept-1/directory-templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("department_id")
	c.SetParamValues("dept-1")

	if err := h.UpdateDirectoryTemplates(c); err != nil {
		t.Fatalf("UpdateDirectoryTemplates: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestUpdateDirectoryTemplates_ValidationError(t *testing.T) {
	svc := &fakeDepartmentService{err: errors.New("path must start with /data/: /etc/passwd")}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	body := `{"paths":["/etc/passwd"]}`
	req := httptest.NewRequest(http.MethodPut, "/departments/dept-1/directory-templates", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("department_id")
	c.SetParamValues("dept-1")

	err := h.UpdateDirectoryTemplates(c)
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

// --- Sync handler tests ---

func TestSyncSkills(t *testing.T) {
	svc := &fakeDepartmentService{
		syncResult: &SyncResultDTO{
			TotalBots: 3,
			Installed: 5,
			Skipped:   2,
		},
	}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/departments/dept-1/sync-skills", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("department_id")
	c.SetParamValues("dept-1")

	if err := h.SyncSkills(c); err != nil {
		t.Fatalf("SyncSkills: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"total_bots":3`) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestSyncSkills_MissingDepartmentID(t *testing.T) {
	h := NewDepartmentHandler(&fakeDepartmentService{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/departments//sync-skills", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("department_id")
	c.SetParamValues("")

	if err := h.SyncSkills(c); err == nil {
		t.Fatal("expected error for missing department_id")
	}
}

func TestSyncDirectories(t *testing.T) {
	svc := &fakeDepartmentService{
		syncResult: &SyncResultDTO{
			TotalBots: 2,
			Installed: 4,
		},
	}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/departments/dept-1/sync-directories", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("department_id")
	c.SetParamValues("dept-1")

	if err := h.SyncDirectories(c); err != nil {
		t.Fatalf("SyncDirectories: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"total_bots":2`) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestSyncDirectories_ServiceError(t *testing.T) {
	svc := &fakeDepartmentService{err: errors.New("sync failed")}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/departments/dept-1/sync-directories", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("department_id")
	c.SetParamValues("dept-1")

	err := h.SyncDirectories(c)
	if err == nil {
		t.Fatal("expected error from service")
	}
}

// --- Department bots tests ---

func TestListDepartmentBots(t *testing.T) {
	svc := &fakeDepartmentService{
		bots: []BotBriefDTO{
			{ID: "bot-1", DisplayName: "Bot One"},
		},
	}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/departments/dept-1/bots", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("department_id")
	c.SetParamValues("dept-1")

	if err := h.ListDepartmentBots(c); err != nil {
		t.Fatalf("ListDepartmentBots: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Bot One") {
		t.Fatalf("body missing Bot One: %s", body)
	}
}
