package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

type fakeCockpitService struct {
	summary CockpitSummaryDTO
	err     error
}

func (f *fakeCockpitService) GetSummary(_ context.Context, _ string, _ CockpitSummaryQuery) (*CockpitSummaryDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &f.summary, nil
}

func TestCockpitHandlerSummary(t *testing.T) {
	svc := &fakeCockpitService{
		summary: CockpitSummaryDTO{
			TotalReports:    5,
			TotalSavedHours: 40.0,
			LaborCostCNY:    8000,
		},
	}
	h := NewCockpitHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bots/bot-1/cockpit/summary?days=7", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")

	if err := h.GetSummary(c); err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "8000") {
		t.Fatalf("body missing labor cost: %s", body)
	}
}

func TestCockpitHandlerMissingBotID(t *testing.T) {
	svc := &fakeCockpitService{}
	h := NewCockpitHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bots//cockpit/summary", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("")

	err := h.GetSummary(c)
	if err == nil {
		t.Fatal("expected error for missing bot_id")
	}
}
