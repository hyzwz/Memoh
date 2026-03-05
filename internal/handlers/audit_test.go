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

type fakeAuditQueryService struct {
	logs  []AuditLogDTO
	total int64
	err   error
}

func (f *fakeAuditQueryService) ListAuditLogs(_ context.Context, _ AuditLogQuery) ([]AuditLogDTO, int64, error) {
	return f.logs, f.total, f.err
}

func TestAuditHandlerList(t *testing.T) {
	svc := &fakeAuditQueryService{
		logs: []AuditLogDTO{
			{ID: "log-1", Action: "chat.send", UserID: "u1", Timestamp: time.Now()},
		},
		total: 1,
	}
	h := NewAuditHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bots/bot-1/audit-logs?limit=10", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")

	if err := h.ListAuditLogs(c); err != nil {
		t.Fatalf("ListAuditLogs: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "chat.send") {
		t.Fatalf("body missing action: %s", body)
	}
}

func TestAuditHandlerMissingBotID(t *testing.T) {
	svc := &fakeAuditQueryService{}
	h := NewAuditHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bots//audit-logs", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("")

	err := h.ListAuditLogs(c)
	if err == nil {
		t.Fatal("expected error for missing bot_id")
	}
}
