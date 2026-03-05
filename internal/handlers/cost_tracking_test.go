package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

// --- Fake service ---

type fakeCostTrackingService struct {
	budgets []BudgetDTO
	created *BudgetDTO
	check   *BudgetCheckDTO
	err     error
}

func (f *fakeCostTrackingService) ListBudgets(_ context.Context, scopeType string) ([]BudgetDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.budgets, nil
}

func (f *fakeCostTrackingService) CreateBudget(_ context.Context, req CreateBudgetRequest) (*BudgetDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.created, nil
}

func (f *fakeCostTrackingService) DeleteBudget(_ context.Context, id string) error {
	return f.err
}

func (f *fakeCostTrackingService) CheckBudget(_ context.Context, scopeType, scopeID string) (*BudgetCheckDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.check, nil
}

// --- Tests ---

func TestCostTrackingHandlerListBudgets(t *testing.T) {
	svc := &fakeCostTrackingService{
		budgets: []BudgetDTO{
			{ID: "b-1", ScopeType: "bot", ScopeID: "bot-1", Period: "monthly", LimitAmount: 100},
		},
	}
	h := NewCostTrackingHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/budgets?scope_type=bot", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.ListBudgets(c); err != nil {
		t.Fatalf("ListBudgets: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "bot-1") {
		t.Fatalf("body: %s", rec.Body.String())
	}
}

func TestCostTrackingHandlerCreateBudget(t *testing.T) {
	svc := &fakeCostTrackingService{
		created: &BudgetDTO{ID: "b-new", ScopeType: "bot", ScopeID: "bot-1", Period: "monthly", LimitAmount: 200},
	}
	h := NewCostTrackingHandler(svc)

	e := echo.New()
	reqBody := `{"scope_type":"bot","scope_id":"bot-1","period":"monthly","limit_amount":200,"alert_threshold":0.8,"action_on_exceed":"warn"}`
	req := httptest.NewRequest(http.MethodPost, "/budgets", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.CreateBudget(c); err != nil {
		t.Fatalf("CreateBudget: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestCostTrackingHandlerCreateBudgetMissingFields(t *testing.T) {
	svc := &fakeCostTrackingService{}
	h := NewCostTrackingHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/budgets", strings.NewReader(`{"scope_type":"bot"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateBudget(c)
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
}

func TestCostTrackingHandlerDeleteBudget(t *testing.T) {
	svc := &fakeCostTrackingService{}
	h := NewCostTrackingHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/budgets/b-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("budget_id")
	c.SetParamValues("b-1")

	if err := h.DeleteBudget(c); err != nil {
		t.Fatalf("DeleteBudget: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestCostTrackingHandlerCheckBudget(t *testing.T) {
	svc := &fakeCostTrackingService{
		check: &BudgetCheckDTO{
			Allowed:    true,
			Spent:      45.50,
			Limit:      100,
			Percentage: 45.5,
		},
	}
	h := NewCostTrackingHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/budgets/check?scope_type=bot&scope_id=bot-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.CheckBudget(c); err != nil {
		t.Fatalf("CheckBudget: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "45.5") {
		t.Fatalf("body: %s", rec.Body.String())
	}
}

func TestCostTrackingHandlerCheckBudgetMissingParams(t *testing.T) {
	svc := &fakeCostTrackingService{}
	h := NewCostTrackingHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/budgets/check", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CheckBudget(c)
	if err == nil {
		t.Fatal("expected error for missing params")
	}
}
