package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	"github.com/memohai/memoh/internal/bots"
	"github.com/memohai/memoh/internal/conversation"
)

type fakeSessionsService struct {
	items       []conversation.ConversationListItem
	created     *conversation.Conversation
	deletedID   string
	lastBotID   string
	lastUserID  string
	lastCreate  *conversation.CreateRequest
	err         error
}

func (f *fakeSessionsService) ListByBotAndChannelIdentity(_ context.Context, botID, channelIdentityID string) ([]conversation.ConversationListItem, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.lastBotID = botID
	f.lastUserID = channelIdentityID
	return f.items, nil
}

func (f *fakeSessionsService) Create(_ context.Context, botID, channelIdentityID string, req conversation.CreateRequest) (conversation.Conversation, error) {
	if f.err != nil {
		return conversation.Conversation{}, f.err
	}
	f.lastBotID = botID
	f.lastUserID = channelIdentityID
	f.lastCreate = &req
	if f.created != nil {
		return *f.created, nil
	}
	return conversation.Conversation{}, nil
}

func (f *fakeSessionsService) Delete(_ context.Context, conversationID string) error {
	if f.err != nil {
		return f.err
	}
	f.deletedID = conversationID
	return nil
}

func TestSessionsHandlerList(t *testing.T) {
	now := time.Now().UTC()
	svc := &fakeSessionsService{
		items: []conversation.ConversationListItem{
			{
				ID:              "chat-1",
				BotID:           "bot-1",
				Kind:            conversation.KindDirect,
				Title:           "Primary chat",
				CreatedBy:       "user_1",
				Metadata:        map[string]any{"type": "chat", "channel_type": "web"},
				CreatedAt:       now,
				UpdatedAt:       now,
				AccessMode:      conversation.AccessModeParticipant,
				ParticipantRole: conversation.RoleOwner,
			},
		},
	}
	h := &SessionsHandler{
		service: svc,
		authorize: func(context.Context, string, string) (bots.Bot, error) {
			return bots.Bot{ID: "bot-1"}, nil
		},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bots/bot-1/sessions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")
	c.Set("user", &jwt.Token{
		Valid:  true,
		Claims: jwt.MapClaims{"sub": "user_1", "user_id": "user_1"},
	})

	if err := h.ListSessions(c); err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "chat-1") {
		t.Fatalf("body missing session id: %s", body)
	}
	if !strings.Contains(body, "\"items\"") {
		t.Fatalf("body missing items wrapper: %s", body)
	}
}

func TestSessionsHandlerCreate(t *testing.T) {
	now := time.Now().UTC()
	svc := &fakeSessionsService{
		created: &conversation.Conversation{
			ID:        "chat-new",
			BotID:     "bot-1",
			Kind:      conversation.KindDirect,
			Title:     "New chat",
			CreatedBy: "user_1",
			Metadata:  map[string]any{"type": "chat"},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	h := &SessionsHandler{
		service: svc,
		authorize: func(context.Context, string, string) (bots.Bot, error) {
			return bots.Bot{ID: "bot-1"}, nil
		},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bots/bot-1/sessions", strings.NewReader(`{"title":"New chat","metadata":{"type":"chat"}}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id")
	c.SetParamValues("bot-1")
	c.Set("user", &jwt.Token{
		Valid:  true,
		Claims: jwt.MapClaims{"sub": "user_1", "user_id": "user_1"},
	})

	if err := h.CreateSession(c); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}
	if svc.lastCreate == nil || svc.lastCreate.Title != "New chat" {
		t.Fatalf("create request not forwarded: %#v", svc.lastCreate)
	}
	if !strings.Contains(rec.Body.String(), "chat-new") {
		t.Fatalf("body: %s", rec.Body.String())
	}
}

func TestSessionsHandlerDelete(t *testing.T) {
	svc := &fakeSessionsService{}
	h := &SessionsHandler{
		service: svc,
		authorize: func(context.Context, string, string) (bots.Bot, error) {
			return bots.Bot{ID: "bot-1"}, nil
		},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/bots/bot-1/sessions/chat-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("bot_id", "id")
	c.SetParamValues("bot-1", "chat-1")
	c.Set("user", &jwt.Token{
		Valid:  true,
		Claims: jwt.MapClaims{"sub": "user_1", "user_id": "user_1"},
	})

	if err := h.DeleteSession(c); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
	if svc.deletedID != "chat-1" {
		t.Fatalf("deleted id = %q", svc.deletedID)
	}
}
