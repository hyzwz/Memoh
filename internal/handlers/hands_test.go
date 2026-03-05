package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

// --- Fake service ---

type fakeHandsService struct {
	hands   []HandDTO
	created *HandDTO
	execRes *HandExecutionResultDTO
	err     error
}

func (f *fakeHandsService) List(_ context.Context, botID string) ([]HandDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.hands, nil
}

func (f *fakeHandsService) Get(_ context.Context, botID, id string) (*HandDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, h := range f.hands {
		if h.ID == id {
			return &h, nil
		}
	}
	return nil, ErrHandNotFound
}

func (f *fakeHandsService) CreateFromMarkdown(_ context.Context, botID, markdown string) (*HandDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.created, nil
}

func (f *fakeHandsService) Delete(_ context.Context, botID, id string) error {
	return f.err
}

func (f *fakeHandsService) Execute(_ context.Context, botID, handID, triggerType string) (*HandExecutionResultDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.execRes, nil
}

// --- Tests ---

func TestHandsHandlerList(t *testing.T) {
	svc := &fakeHandsService{
		hands: []HandDTO{
			{ID: "h-1", Name: "daily-report", Type: "hand", IsEnabled: true},
			{ID: "h-2", Name: "weekly-digest", Type: "hand", IsEnabled: true},
		},
	}
	h := NewHandsHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bots/bot-1/hands", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")

	if err := h.ListHands(c); err != nil {
		t.Fatalf("ListHands: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "daily-report") {
		t.Fatalf("body missing daily-report: %s", body)
	}
	if !strings.Contains(body, "weekly-digest") {
		t.Fatalf("body missing weekly-digest: %s", body)
	}
}

func TestHandsHandlerGet(t *testing.T) {
	svc := &fakeHandsService{
		hands: []HandDTO{
			{ID: "h-1", Name: "daily-report", Type: "hand", Description: "Generates daily report"},
		},
	}
	h := NewHandsHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bots/bot-1/hands/h-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id", "hand_id")
	c.SetParamValues("bot-1", "h-1")

	if err := h.GetHand(c); err != nil {
		t.Fatalf("GetHand: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "daily-report") {
		t.Fatalf("body: %s", rec.Body.String())
	}
}

func TestHandsHandlerGetNotFound(t *testing.T) {
	svc := &fakeHandsService{hands: []HandDTO{}}
	h := NewHandsHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bots/bot-1/hands/missing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id", "hand_id")
	c.SetParamValues("bot-1", "missing")

	err := h.GetHand(c)
	if err == nil {
		t.Fatal("expected error for not found")
	}
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %v", err)
	}
}

func TestHandsHandlerCreate(t *testing.T) {
	svc := &fakeHandsService{
		created: &HandDTO{ID: "h-new", Name: "new-hand", Type: "hand"},
	}
	h := NewHandsHandler(svc)

	markdown := `---
name: new-hand
type: hand
---
Generate something useful.`

	e := echo.New()
	body := `{"markdown":"` + strings.ReplaceAll(markdown, "\n", "\\n") + `"}`
	req := httptest.NewRequest(http.MethodPost, "/bots/bot-1/hands", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")

	if err := h.CreateHand(c); err != nil {
		t.Fatalf("CreateHand: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "new-hand") {
		t.Fatalf("body: %s", rec.Body.String())
	}
}

func TestHandsHandlerCreateMissingMarkdown(t *testing.T) {
	svc := &fakeHandsService{}
	h := NewHandsHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bots/bot-1/hands", strings.NewReader(`{"markdown":""}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")

	err := h.CreateHand(c)
	if err == nil {
		t.Fatal("expected error for empty markdown")
	}
}

func TestHandsHandlerDelete(t *testing.T) {
	svc := &fakeHandsService{}
	h := NewHandsHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/bots/bot-1/hands/h-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id", "hand_id")
	c.SetParamValues("bot-1", "h-1")

	if err := h.DeleteHand(c); err != nil {
		t.Fatalf("DeleteHand: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandsHandlerExecute(t *testing.T) {
	svc := &fakeHandsService{
		execRes: &HandExecutionResultDTO{
			Status:     "completed",
			ResultText: "Report generated",
			StartedAt:  time.Now(),
			CompletedAt: time.Now(),
		},
	}
	h := NewHandsHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bots/bot-1/hands/h-1/execute", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id", "hand_id")
	c.SetParamValues("bot-1", "h-1")

	if err := h.ExecuteHand(c); err != nil {
		t.Fatalf("ExecuteHand: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "completed") {
		t.Fatalf("body: %s", rec.Body.String())
	}
}

func TestHandsHandlerMissingBotID(t *testing.T) {
	svc := &fakeHandsService{}
	h := NewHandsHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bots//hands", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("")

	err := h.ListHands(c)
	if err == nil {
		t.Fatal("expected error for missing bot_id")
	}
}
