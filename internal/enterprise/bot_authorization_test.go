package enterprise

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	authz "github.com/memohai/memoh/internal/rbac"
)

type fakeBotAuthorizer struct {
	err      error
	userID   string
	botID    string
	resource authz.Resource
	action   authz.Action
}

func (f *fakeBotAuthorizer) Authorize(_ echo.Context, userID, botID string, resource authz.Resource, action authz.Action) error {
	f.userID = userID
	f.botID = botID
	f.resource = resource
	f.action = action
	return f.err
}

func TestRequireBotPermissionHandsExecute(t *testing.T) {
	t.Parallel()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bots/bot-1/hands/hand-1/execute", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id", "hand_id")
	c.SetParamValues("bot-1", "hand-1")
	c.Set("user", &jwt.Token{
		Valid:  true,
		Claims: jwt.MapClaims{"user_id": "user-1"},
	})

	fake := &fakeBotAuthorizer{}
	mw := RequireBotPermission(fake, authz.ResourceTools, ResolveHandsAction)

	err := mw(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c)
	if err != nil {
		t.Fatalf("middleware returned error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
	if fake.userID != "user-1" || fake.botID != "bot-1" {
		t.Fatalf("unexpected identity: user=%q bot=%q", fake.userID, fake.botID)
	}
	if fake.resource != authz.ResourceTools || fake.action != authz.ActionExecute {
		t.Fatalf("unexpected permission check: resource=%q action=%q", fake.resource, fake.action)
	}
}

func TestRequireBotPermissionDenied(t *testing.T) {
	t.Parallel()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bots/bot-1/cockpit/summary", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")
	c.Set("user", &jwt.Token{
		Valid:  true,
		Claims: jwt.MapClaims{"user_id": "user-1"},
	})

	fake := &fakeBotAuthorizer{err: authz.ErrAccessDenied}
	mw := RequireBotPermission(fake, authz.ResourceSettings, ResolveCockpitAction)

	err := mw(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})(c)
	if err == nil {
		t.Fatal("expected permission error")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.Code != http.StatusForbidden {
		t.Fatalf("status = %d", httpErr.Code)
	}
}
