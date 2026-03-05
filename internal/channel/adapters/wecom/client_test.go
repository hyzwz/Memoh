package wecom

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cgi-bin/gettoken" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		corpID := r.URL.Query().Get("corpid")
		secret := r.URL.Query().Get("corpsecret")
		if corpID == "" || secret == "" {
			t.Fatal("missing corpid or corpsecret")
		}
		json.NewEncoder(w).Encode(map[string]any{
			"errcode":      0,
			"errmsg":       "ok",
			"access_token": "test-token-123",
			"expires_in":   7200,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "ww123", "secret123")
	token, err := client.GetAccessToken(context.Background())
	if err != nil {
		t.Fatalf("GetAccessToken: %v", err)
	}
	if token != "test-token-123" {
		t.Fatalf("token = %q", token)
	}
}

func TestGetAccessTokenError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"errcode": 40013,
			"errmsg":  "invalid corpid",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad", "bad")
	_, err := client.GetAccessToken(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSendTextMessage(t *testing.T) {
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
		if r.URL.Path != "/cgi-bin/message/send" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		json.NewEncoder(w).Encode(map[string]any{
			"errcode": 0,
			"errmsg":  "ok",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "ww123", "secret")
	err := client.SendTextMessage(context.Background(), "zhangsan", "Hello!")
	if err != nil {
		t.Fatalf("SendTextMessage: %v", err)
	}
	if receivedBody["touser"] != "zhangsan" {
		t.Fatalf("touser = %v", receivedBody["touser"])
	}
	if receivedBody["msgtype"] != "text" {
		t.Fatalf("msgtype = %v", receivedBody["msgtype"])
	}
	text := receivedBody["text"].(map[string]any)
	if text["content"] != "Hello!" {
		t.Fatalf("content = %v", text["content"])
	}
}
