package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

type fakeDepartmentService struct {
	departments []DepartmentDTO
	err         error
}

func (f *fakeDepartmentService) ListDepartments(_ context.Context, _ string) ([]DepartmentDTO, error) {
	return f.departments, f.err
}

func (f *fakeDepartmentService) CreateDepartment(_ context.Context, _ string, _ CreateDepartmentRequest) (*DepartmentDTO, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &DepartmentDTO{ID: "dept-1", Name: "Engineering"}, nil
}

func TestDepartmentHandlerList(t *testing.T) {
	svc := &fakeDepartmentService{
		departments: []DepartmentDTO{
			{ID: "dept-1", Name: "Engineering"},
			{ID: "dept-2", Name: "Design"},
		},
	}
	h := NewDepartmentHandler(svc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bots/bot-1/departments", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")

	if err := h.ListDepartments(c); err != nil {
		t.Fatalf("ListDepartments: %v", err)
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
	req := httptest.NewRequest(http.MethodPost, "/bots/bot-1/departments", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")

	if err := h.CreateDepartment(c); err != nil {
		t.Fatalf("CreateDepartment: %v", err)
	}
	if rec.Code != http.StatusCreated {
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

	err := h.ListDepartments(c)
	if err == nil {
		t.Fatal("expected error for missing bot_id")
	}
}
