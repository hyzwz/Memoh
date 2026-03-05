package wecom

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/memohai/memoh/internal/channel"
)

func TestWeComAdapterSend(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cgi-bin/gettoken" {
			json.NewEncoder(w).Encode(map[string]any{
				"errcode":      0,
				"errmsg":       "ok",
				"access_token": "tok",
				"expires_in":   7200,
			})
			return
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		json.NewEncoder(w).Encode(map[string]any{"errcode": 0, "errmsg": "ok"})
	}))
	defer server.Close()

	a := NewWeComAdapter(nil)

	cfg := channel.ChannelConfig{
		Credentials: map[string]any{
			"corpId":         "ww123",
			"secret":         "sec",
			"token":          "tok",
			"encodingAESKey": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG",
		},
		ChannelType: Type,
	}

	msg := channel.OutboundMessage{
		Target: "userid:zhangsan",
		Message: channel.Message{
			Text: "Hello from Memoh!",
		},
	}

	// Override base URL for testing.
	a.testBaseURL = server.URL

	err := a.Send(context.Background(), cfg, msg)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if receivedBody["touser"] != "zhangsan" {
		t.Fatalf("touser = %v", receivedBody["touser"])
	}
	text := receivedBody["text"].(map[string]any)
	if text["content"] != "Hello from Memoh!" {
		t.Fatalf("content = %v", text["content"])
	}
}

func TestWeComAdapterSendInvalidConfig(t *testing.T) {
	a := NewWeComAdapter(nil)
	cfg := channel.ChannelConfig{
		Credentials: map[string]any{},
	}
	msg := channel.OutboundMessage{
		Target:  "userid:test",
		Message: channel.Message{Text: "hi"},
	}
	err := a.Send(context.Background(), cfg, msg)
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestWeComAdapterSendInvalidTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"errcode":      0,
			"errmsg":       "ok",
			"access_token": "tok",
			"expires_in":   7200,
		})
	}))
	defer server.Close()

	a := NewWeComAdapter(nil)
	a.testBaseURL = server.URL

	cfg := channel.ChannelConfig{
		Credentials: map[string]any{
			"corpId":         "ww123",
			"secret":         "sec",
			"token":          "tok",
			"encodingAESKey": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG",
		},
	}
	msg := channel.OutboundMessage{
		Target:  "",
		Message: channel.Message{Text: "hi"},
	}
	err := a.Send(context.Background(), cfg, msg)
	if err == nil {
		t.Fatal("expected error for empty target")
	}
}
