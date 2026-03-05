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

type fakeModelRoutingService struct {
	routes  []ModelRouteDTO
	created *ModelRouteDTO
	err     error
}

func (f *fakeModelRoutingService) ListRoutes(_ context.Context, botID string) ([]ModelRouteDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.routes, nil
}

func (f *fakeModelRoutingService) CreateRoute(_ context.Context, botID string, req CreateModelRouteRequest) (*ModelRouteDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.created, nil
}

func (f *fakeModelRoutingService) DeleteRoute(_ context.Context, botID, id string) error {
	return f.err
}

// --- Tests ---

func TestModelRoutingHandlerList(t *testing.T) {
	svc := &fakeModelRoutingService{
		routes: []ModelRouteDTO{
			{ID: "r-1", Name: "gpt4-complex", ModelID: "gpt-4", ComplexityTier: "complex", IsEnabled: true},
			{ID: "r-2", Name: "gpt3-simple", ModelID: "gpt-3.5", ComplexityTier: "simple", IsEnabled: true},
		},
	}
	h := NewModelRoutingHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bots/bot-1/model-routes", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")

	if err := h.ListRoutes(c); err != nil {
		t.Fatalf("ListRoutes: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "gpt4-complex") {
		t.Fatalf("body missing route: %s", body)
	}
}

func TestModelRoutingHandlerCreate(t *testing.T) {
	svc := &fakeModelRoutingService{
		created: &ModelRouteDTO{ID: "r-new", Name: "claude-medium", ModelID: "claude-3", ComplexityTier: "medium"},
	}
	h := NewModelRoutingHandler(svc)

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
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "claude-medium") {
		t.Fatalf("body: %s", rec.Body.String())
	}
}

func TestModelRoutingHandlerCreateMissingName(t *testing.T) {
	svc := &fakeModelRoutingService{}
	h := NewModelRoutingHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bots/bot-1/model-routes", strings.NewReader(`{"model_id":"x"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")

	err := h.CreateRoute(c)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestModelRoutingHandlerDelete(t *testing.T) {
	svc := &fakeModelRoutingService{}
	h := NewModelRoutingHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/bots/bot-1/model-routes/r-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id", "route_id")
	c.SetParamValues("bot-1", "r-1")

	if err := h.DeleteRoute(c); err != nil {
		t.Fatalf("DeleteRoute: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestModelRoutingHandlerMissingBotID(t *testing.T) {
	svc := &fakeModelRoutingService{}
	h := NewModelRoutingHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bots//model-routes", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("")

	err := h.ListRoutes(c)
	if err == nil {
		t.Fatal("expected error for missing bot_id")
	}
}
