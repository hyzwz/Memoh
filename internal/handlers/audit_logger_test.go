package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

// fakeAuditLogger records audit log calls for test assertions.
type fakeAuditLogger struct {
	mu   sync.Mutex
	logs []auditLogEntry
}

type auditLogEntry struct {
	Action       string
	ResourceType string
	ResourceID   string
	BotID        string
}

func (f *fakeAuditLogger) Log(_ context.Context, _, botID, action, resourceType, resourceID, _, _ string, _ any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.logs = append(f.logs, auditLogEntry{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		BotID:        botID,
	})
}

func (f *fakeAuditLogger) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.logs)
}

func (f *fakeAuditLogger) last() auditLogEntry {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.logs[len(f.logs)-1]
}

// --- HandsHandler audit tests ---

func TestHandsHandlerCreateAuditLog(t *testing.T) {
	svc := &fakeHandsService{
		created: &HandDTO{ID: "h-new", Name: "new-hand", Type: "hand"},
	}
	audit := &fakeAuditLogger{}
	h := NewHandsHandler(svc, WithAuditLogger(audit))

	e := echo.New()
	body := `{"markdown":"---\nname: new-hand\ntype: hand\n---\nDo something."}`
	req := httptest.NewRequest(http.MethodPost, "/bots/bot-1/hands", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")

	if err := h.CreateHand(c); err != nil {
		t.Fatalf("CreateHand: %v", err)
	}
	if audit.count() != 1 {
		t.Fatalf("expected 1 audit log, got %d", audit.count())
	}
	entry := audit.last()
	if entry.Action != "create" {
		t.Fatalf("action = %q, want create", entry.Action)
	}
	if entry.ResourceType != "hand" {
		t.Fatalf("resource_type = %q, want hand", entry.ResourceType)
	}
	if entry.ResourceID != "h-new" {
		t.Fatalf("resource_id = %q, want h-new", entry.ResourceID)
	}
}

func TestHandsHandlerDeleteAuditLog(t *testing.T) {
	svc := &fakeHandsService{}
	audit := &fakeAuditLogger{}
	h := NewHandsHandler(svc, WithAuditLogger(audit))

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/bots/bot-1/hands/h-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id", "hand_id")
	c.SetParamValues("bot-1", "h-1")

	if err := h.DeleteHand(c); err != nil {
		t.Fatalf("DeleteHand: %v", err)
	}
	if audit.count() != 1 {
		t.Fatalf("expected 1 audit log, got %d", audit.count())
	}
	if audit.last().Action != "delete" {
		t.Fatalf("action = %q, want delete", audit.last().Action)
	}
}

func TestHandsHandlerExecuteAuditLog(t *testing.T) {
	svc := &fakeHandsService{
		execRes: &HandExecutionResultDTO{
			Status:      "completed",
			ResultText:  "done",
			StartedAt:   time.Now(),
			CompletedAt: time.Now(),
		},
	}
	audit := &fakeAuditLogger{}
	h := NewHandsHandler(svc, WithAuditLogger(audit))

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bots/bot-1/hands/h-1/execute", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id", "hand_id")
	c.SetParamValues("bot-1", "h-1")

	if err := h.ExecuteHand(c); err != nil {
		t.Fatalf("ExecuteHand: %v", err)
	}
	if audit.count() != 1 {
		t.Fatalf("expected 1 audit log, got %d", audit.count())
	}
	if audit.last().Action != "execute" {
		t.Fatalf("action = %q, want execute", audit.last().Action)
	}
}

// --- CostTrackingHandler audit tests ---

func TestCostTrackingHandlerCreateBudgetAuditLog(t *testing.T) {
	svc := &fakeCostTrackingService{
		created: &BudgetDTO{ID: "b-new", ScopeType: "bot", ScopeID: "bot-1", Period: "monthly", LimitAmount: 200},
	}
	audit := &fakeAuditLogger{}
	h := NewCostTrackingHandler(svc, WithCostTrackingAudit(audit))

	e := echo.New()
	reqBody := `{"scope_type":"bot","scope_id":"bot-1","period":"monthly","limit_amount":200,"alert_threshold":0.8,"action_on_exceed":"warn"}`
	req := httptest.NewRequest(http.MethodPost, "/budgets", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.CreateBudget(c); err != nil {
		t.Fatalf("CreateBudget: %v", err)
	}
	if audit.count() != 1 {
		t.Fatalf("expected 1 audit log, got %d", audit.count())
	}
	entry := audit.last()
	if entry.Action != "create" || entry.ResourceType != "budget" {
		t.Fatalf("got action=%q resource_type=%q", entry.Action, entry.ResourceType)
	}
}

func TestCostTrackingHandlerDeleteBudgetAuditLog(t *testing.T) {
	svc := &fakeCostTrackingService{}
	audit := &fakeAuditLogger{}
	h := NewCostTrackingHandler(svc, WithCostTrackingAudit(audit))

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/budgets/b-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("budget_id")
	c.SetParamValues("b-1")

	if err := h.DeleteBudget(c); err != nil {
		t.Fatalf("DeleteBudget: %v", err)
	}
	if audit.count() != 1 {
		t.Fatalf("expected 1 audit log, got %d", audit.count())
	}
	if audit.last().Action != "delete" || audit.last().ResourceType != "budget" {
		t.Fatalf("got action=%q resource_type=%q", audit.last().Action, audit.last().ResourceType)
	}
}

// --- ModelRoutingHandler audit tests ---

func TestModelRoutingHandlerCreateRouteAuditLog(t *testing.T) {
	svc := &fakeModelRoutingService{
		created: &ModelRouteDTO{ID: "r-new", Name: "claude-medium", ModelID: "claude-3"},
	}
	audit := &fakeAuditLogger{}
	h := NewModelRoutingHandler(svc, WithModelRoutingAudit(audit))

	e := echo.New()
	reqBody := `{"name":"claude-medium","model_id":"claude-3","complexity_tier":"medium","priority":10}`
	req := httptest.NewRequest(http.MethodPost, "/bots/bot-1/model-routes", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")

	if err := h.CreateRoute(c); err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	if audit.count() != 1 {
		t.Fatalf("expected 1 audit log, got %d", audit.count())
	}
	entry := audit.last()
	if entry.Action != "create" || entry.ResourceType != "model_route" {
		t.Fatalf("got action=%q resource_type=%q", entry.Action, entry.ResourceType)
	}
}

func TestModelRoutingHandlerDeleteRouteAuditLog(t *testing.T) {
	svc := &fakeModelRoutingService{}
	audit := &fakeAuditLogger{}
	h := NewModelRoutingHandler(svc, WithModelRoutingAudit(audit))

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/bots/bot-1/model-routes/r-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id", "route_id")
	c.SetParamValues("bot-1", "r-1")

	if err := h.DeleteRoute(c); err != nil {
		t.Fatalf("DeleteRoute: %v", err)
	}
	if audit.count() != 1 {
		t.Fatalf("expected 1 audit log, got %d", audit.count())
	}
	if audit.last().Action != "delete" || audit.last().ResourceType != "model_route" {
		t.Fatalf("got action=%q resource_type=%q", audit.last().Action, audit.last().ResourceType)
	}
}

// --- DepartmentHandler audit tests ---

func TestDepartmentHandlerCreateAuditLog(t *testing.T) {
	svc := &fakeDepartmentService{}
	audit := &fakeAuditLogger{}
	h := NewDepartmentHandler(svc, WithDepartmentAudit(audit))

	e := echo.New()
	reqBody := `{"name":"Engineering","description":"Eng team"}`
	req := httptest.NewRequest(http.MethodPost, "/bots/bot-1/departments", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")

	if err := h.CreateDepartment(c); err != nil {
		t.Fatalf("CreateDepartment: %v", err)
	}
	if audit.count() != 1 {
		t.Fatalf("expected 1 audit log, got %d", audit.count())
	}
	entry := audit.last()
	if entry.Action != "create" || entry.ResourceType != "department" {
		t.Fatalf("got action=%q resource_type=%q", entry.Action, entry.ResourceType)
	}
}

// --- No audit logger still works ---

func TestHandsHandlerNoAuditLoggerStillWorks(t *testing.T) {
	svc := &fakeHandsService{
		created: &HandDTO{ID: "h-1", Name: "test", Type: "hand"},
	}
	h := NewHandsHandler(svc) // no audit logger

	e := echo.New()
	body := `{"markdown":"---\nname: test\n---\ntest"}`
	req := httptest.NewRequest(http.MethodPost, "/bots/bot-1/hands", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")

	if err := h.CreateHand(c); err != nil {
		t.Fatalf("CreateHand without audit: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}
}
