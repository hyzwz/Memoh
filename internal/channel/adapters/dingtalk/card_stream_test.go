package dingtalk

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/memohai/memoh/internal/channel"
)

func TestOpenStreamCreatesCardSession(t *testing.T) {
	t.Parallel()

	var createReq AICardCreateRequest
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/v1.0/oauth2/accessToken":
			return jsonResponse(t, map[string]any{"access_token": "token-1"}), nil
		case "/v1.0/robot/aiCards/create":
			if err := json.NewDecoder(r.Body).Decode(&createReq); err != nil {
				t.Fatalf("decode create req: %v", err)
			}
			return jsonResponse(t, AICardCreateResponse{CardID: "card-1"}), nil
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
			return nil, nil
		}
	})

	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL:    "https://dingtalk.test",
			httpClient: &http.Client{Transport: transport},
		}
	}

	stream, err := adapter.OpenStream(context.Background(), channel.ChannelConfig{
		ID:    "cfg-1",
		BotID: "bot-1",
		Credentials: map[string]any{
			"appKey":               "key",
			"appSecret":            "secret",
			"robotCode":            "robot-1",
			"enableAICard":         true,
			"cardUpdateThrottleMs": 100,
		},
	}, "webhook:https://example.com/session", channel.StreamOptions{})
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	s, ok := stream.(*cardStream)
	if !ok {
		t.Fatalf("unexpected stream type %T", stream)
	}
	if s.cardIDValue() != "card-1" {
		t.Fatalf("expected bootstrap card id, got %q", s.cardIDValue())
	}
	if createReq.Title != defaultCardTitle {
		t.Fatalf("unexpected title: %#v", createReq.Title)
	}
	if createReq.SessionWebhook != "https://example.com/session" {
		t.Fatalf("unexpected sessionWebhook: %#v", createReq.SessionWebhook)
	}
}

func TestCardStreamBuffersAndThrottlesDeltas(t *testing.T) {
	t.Parallel()

	var createCalls int
	var updateBodies []AICardUpdateRequest
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/v1.0/oauth2/accessToken":
			return jsonResponse(t, map[string]any{"access_token": "token-2"}), nil
		case "/v1.0/robot/aiCards/create":
			createCalls++
			return jsonResponse(t, AICardCreateResponse{CardID: "card-2"}), nil
		case "/v1.0/robot/aiCards/update":
			var body AICardUpdateRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update req: %v", err)
			}
			updateBodies = append(updateBodies, body)
			return jsonResponse(t, map[string]any{"ok": true}), nil
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
			return nil, nil
		}
	})

	now := time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC)
	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL:    "https://dingtalk.test",
			httpClient: &http.Client{Transport: transport},
		}
	}

	stream, err := adapter.OpenStream(context.Background(), channel.ChannelConfig{
		ID:    "cfg-2",
		BotID: "bot-2",
		Credentials: map[string]any{
			"appKey":               "key",
			"appSecret":            "secret",
			"robotCode":            "robot-2",
			"enableAICard":         true,
			"cardUpdateThrottleMs": 500,
		},
	}, "conversation:cid-1", channel.StreamOptions{})
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	s := stream.(*cardStream)
	s.now = func() time.Time { return now }
	s.mu.Lock()
	s.lastUpdate = now
	s.mu.Unlock()

	if err := s.Push(context.Background(), channel.StreamEvent{Type: channel.StreamEventDelta, Delta: "hello"}); err != nil {
		t.Fatalf("push delta: %v", err)
	}
	if len(updateBodies) != 0 {
		t.Fatalf("expected throttled delta to stay buffered, got %d updates", len(updateBodies))
	}

	now = now.Add(600 * time.Millisecond)
	if err := s.Push(context.Background(), channel.StreamEvent{Type: channel.StreamEventDelta, Delta: " world"}); err != nil {
		t.Fatalf("push delta 2: %v", err)
	}
	if len(updateBodies) != 1 {
		t.Fatalf("expected one update after throttle, got %d", len(updateBodies))
	}
	if updateBodies[0].Content != "hello world" {
		t.Fatalf("unexpected buffered content: %#v", updateBodies[0].Content)
	}
	if createCalls != 1 {
		t.Fatalf("unexpected create calls: %d", createCalls)
	}
}

func TestCardStreamFinalFlushesContent(t *testing.T) {
	t.Parallel()

	var finalUpdate AICardUpdateRequest
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/v1.0/oauth2/accessToken":
			return jsonResponse(t, map[string]any{"access_token": "token-3"}), nil
		case "/v1.0/robot/aiCards/create":
			return jsonResponse(t, AICardCreateResponse{CardID: "card-3"}), nil
		case "/v1.0/robot/aiCards/update":
			if err := json.NewDecoder(r.Body).Decode(&finalUpdate); err != nil {
				t.Fatalf("decode final update: %v", err)
			}
			return jsonResponse(t, map[string]any{"ok": true}), nil
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
			return nil, nil
		}
	})

	now := time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC)
	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL:    "https://dingtalk.test",
			httpClient: &http.Client{Transport: transport},
		}
	}

	stream, err := adapter.OpenStream(context.Background(), channel.ChannelConfig{
		ID:    "cfg-3",
		BotID: "bot-3",
		Credentials: map[string]any{
			"appKey":               "key",
			"appSecret":            "secret",
			"robotCode":            "robot-3",
			"enableAICard":         true,
			"cardUpdateThrottleMs": 500,
		},
	}, "conversation:cid-2", channel.StreamOptions{})
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	s := stream.(*cardStream)
	s.now = func() time.Time { return now }

	if err := s.Push(context.Background(), channel.StreamEvent{Type: channel.StreamEventDelta, Delta: "partial"}); err != nil {
		t.Fatalf("push delta: %v", err)
	}
	if err := s.Push(context.Background(), channel.StreamEvent{
		Type: channel.StreamEventFinal,
		Final: &channel.StreamFinalizePayload{
			Message: channel.Message{Text: "final content"},
		},
	}); err != nil {
		t.Fatalf("push final: %v", err)
	}
	if !finalUpdate.Final {
		t.Fatal("expected final update flag")
	}
	if finalUpdate.Content != "final content" {
		t.Fatalf("unexpected final content: %#v", finalUpdate.Content)
	}
}

func TestCardStreamErrorFallsBackToPlainText(t *testing.T) {
	t.Parallel()

	var sentText string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/v1.0/oauth2/accessToken":
			return jsonResponse(t, map[string]any{"access_token": "token-4"}), nil
		case "/v1.0/robot/aiCards/create":
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     make(http.Header),
				Body:       http.NoBody,
			}, nil
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
			return nil, nil
		}
	})

	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL:    "https://dingtalk.test",
			httpClient: &http.Client{Transport: transport},
			replyTextFunc: func(_ context.Context, _ string, text string) error {
				sentText = text
				return nil
			},
		}
	}

	stream, err := adapter.OpenStream(context.Background(), channel.ChannelConfig{
		ID:    "cfg-4",
		BotID: "bot-4",
		Credentials: map[string]any{
			"appKey":               "key",
			"appSecret":            "secret",
			"robotCode":            "robot-4",
			"enableAICard":         true,
			"cardUpdateThrottleMs": 500,
		},
	}, "webhook:https://example.com/session", channel.StreamOptions{})
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	s := stream.(*cardStream)

	if sentText == "" {
		t.Fatal("expected fallback send during failed bootstrap")
	}
	if !strings.Contains(sentText, defaultCardTitle) {
		t.Fatalf("unexpected fallback text: %q", sentText)
	}
	if err := s.Push(context.Background(), channel.StreamEvent{Type: channel.StreamEventError, Error: "stream failed"}); err != nil {
		t.Fatalf("push error: %v", err)
	}
}

func TestCardStreamUpdateFailureFallsBackToPlainText(t *testing.T) {
	t.Parallel()

	var sentText string
	var updateCalls int
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/v1.0/oauth2/accessToken":
			return jsonResponse(t, map[string]any{"access_token": "token-5"}), nil
		case "/v1.0/robot/aiCards/create":
			return jsonResponse(t, AICardCreateResponse{CardID: "card-5"}), nil
		case "/v1.0/robot/aiCards/update":
			updateCalls++
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     make(http.Header),
				Body:       http.NoBody,
			}, nil
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
			return nil, nil
		}
	})

	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL:    "https://dingtalk.test",
			httpClient: &http.Client{Transport: transport},
			replyTextFunc: func(_ context.Context, _ string, text string) error {
				sentText = text
				return nil
			},
		}
	}

	stream, err := adapter.OpenStream(context.Background(), channel.ChannelConfig{
		ID:    "cfg-4b",
		BotID: "bot-4",
		Credentials: map[string]any{
			"appKey":               "key",
			"appSecret":            "secret",
			"robotCode":            "robot-4",
			"enableAICard":         true,
			"cardUpdateThrottleMs": 0,
		},
	}, "webhook:https://example.com/session", channel.StreamOptions{})
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	s := stream.(*cardStream)
	s.now = func() time.Time { return time.Date(2026, 3, 24, 10, 1, 0, 0, time.UTC) }
	s.mu.Lock()
	s.lastUpdate = time.Time{}
	s.mu.Unlock()

	if err := s.Push(context.Background(), channel.StreamEvent{Type: channel.StreamEventDelta, Delta: "hello degraded"}); err != nil {
		t.Fatalf("push delta: %v", err)
	}
	if updateCalls != 1 {
		t.Fatalf("expected one update attempt, got %d", updateCalls)
	}
	if sentText != "hello degraded" {
		t.Fatalf("unexpected fallback text: %q", sentText)
	}
}

func TestCardStreamUpdateFailureFallsBackToConversationSend(t *testing.T) {
	t.Parallel()

	var proactiveBody ProactiveSendPayload
	var updateCalls int
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/v1.0/oauth2/accessToken":
			return jsonResponse(t, map[string]any{"access_token": "token-6"}), nil
		case "/v1.0/robot/aiCards/create":
			return jsonResponse(t, AICardCreateResponse{CardID: "card-6"}), nil
		case "/v1.0/robot/aiCards/update":
			updateCalls++
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     make(http.Header),
				Body:       http.NoBody,
			}, nil
		case "/v1.0/robot/groupMessages/send":
			if err := json.NewDecoder(r.Body).Decode(&proactiveBody); err != nil {
				t.Fatalf("decode proactive conversation send: %v", err)
			}
			return jsonResponse(t, map[string]any{"processQueryKey": "pqk-6"}), nil
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
			return nil, nil
		}
	})

	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL:    "https://dingtalk.test",
			httpClient: &http.Client{Transport: transport},
		}
	}

	stream, err := adapter.OpenStream(context.Background(), channel.ChannelConfig{
		ID:    "cfg-4c",
		BotID: "bot-4",
		Credentials: map[string]any{
			"appKey":               "key",
			"appSecret":            "secret",
			"robotCode":            "robot-4",
			"enableAICard":         true,
			"cardUpdateThrottleMs": 0,
		},
	}, "conversation:cid-6", channel.StreamOptions{})
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	s := stream.(*cardStream)
	s.now = func() time.Time { return time.Date(2026, 3, 24, 10, 2, 0, 0, time.UTC) }
	s.mu.Lock()
	s.lastUpdate = time.Time{}
	s.mu.Unlock()

	if err := s.Push(context.Background(), channel.StreamEvent{Type: channel.StreamEventDelta, Delta: "hello proactive"}); err != nil {
		t.Fatalf("push delta: %v", err)
	}
	if updateCalls != 1 {
		t.Fatalf("expected one update attempt, got %d", updateCalls)
	}
	if proactiveBody.OpenConversationID != "cid-6" {
		t.Fatalf("unexpected openConversationId: %#v", proactiveBody.OpenConversationID)
	}
	var msgParam InboundTextBlock
	if err := json.Unmarshal([]byte(proactiveBody.MsgParam), &msgParam); err != nil {
		t.Fatalf("decode proactive msgParam: %v", err)
	}
	if msgParam.Content != "hello proactive" {
		t.Fatalf("unexpected proactive content: %#v", msgParam.Content)
	}
}

func TestOpenStreamRejectsMalformedTarget(t *testing.T) {
	t.Parallel()

	adapter := NewDingTalkAdapter(nil)
	_, err := adapter.OpenStream(context.Background(), channel.ChannelConfig{
		ID: "cfg-5",
		Credentials: map[string]any{
			"appKey":    "key",
			"appSecret": "secret",
		},
	}, "not-a-target", channel.StreamOptions{})
	if err == nil {
		t.Fatal("expected malformed target error")
	}
}

func TestOpenStreamRejectsUserTargets(t *testing.T) {
	t.Parallel()

	adapter := NewDingTalkAdapter(nil)
	_, err := adapter.OpenStream(context.Background(), channel.ChannelConfig{
		ID: "cfg-6",
		Credentials: map[string]any{
			"appKey":       "key",
			"appSecret":    "secret",
			"enableAICard": true,
			"robotCode":    "robot-6",
		},
	}, "user:staff:staff-1", channel.StreamOptions{})
	if err == nil || !strings.Contains(err.Error(), "does not support user targets") {
		t.Fatalf("unexpected error: %v", err)
	}
}
