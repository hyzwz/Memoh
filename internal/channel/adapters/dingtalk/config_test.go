package dingtalk

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNormalizeConfigRequiresCredentials(t *testing.T) {
	t.Parallel()

	if _, err := normalizeConfig(map[string]any{
		"appSecret": "secret",
	}); err == nil {
		t.Fatal("expected appKey validation error")
	}

	if _, err := normalizeConfig(map[string]any{
		"appKey": "key",
	}); err == nil {
		t.Fatal("expected appSecret validation error")
	}
}

func TestNormalizeConfigNormalizesAliasesAndDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  map[string]any
	}{
		{
			name: "underscore aliases",
			raw: map[string]any{
				"app_key":    "  app-key  ",
				"app_secret": "  app-secret  ",
			},
		},
		{
			name: "client aliases",
			raw: map[string]any{
				"clientId":     "  client-id  ",
				"clientSecret": "  client-secret  ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeConfig(tt.raw)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got["appKey"] == "" {
				t.Fatalf("expected normalized appKey, got %#v", got["appKey"])
			}
			if got["appSecret"] == "" {
				t.Fatalf("expected normalized appSecret, got %#v", got["appSecret"])
			}
			if got["groupSessionScope"] != "group" {
				t.Fatalf("unexpected default groupSessionScope: %#v", got["groupSessionScope"])
			}
			if got["requireMentionInGroup"] != true {
				t.Fatalf("unexpected default requireMentionInGroup: %#v", got["requireMentionInGroup"])
			}
			if got["enableAICard"] != false {
				t.Fatalf("unexpected default enableAICard: %#v", got["enableAICard"])
			}
			commands, ok := got["sessionResetCommands"].([]string)
			if !ok {
				t.Fatalf("expected normalized sessionResetCommands slice, got %#v", got["sessionResetCommands"])
			}
			if len(commands) != 2 || commands[0] != "/new" || commands[1] != "新会话" {
				t.Fatalf("unexpected default sessionResetCommands: %#v", commands)
			}
		})
	}
}

func TestNormalizeConfigNormalizesResetCommands(t *testing.T) {
	t.Parallel()

	got, err := normalizeConfig(map[string]any{
		"appKey":                "key",
		"appSecret":             "secret",
		"enableSessionCommands": true,
		"sessionResetCommands":  " /new , 新会话 , /new ,   ",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	commands, ok := got["sessionResetCommands"].([]string)
	if !ok {
		t.Fatalf("expected normalized sessionResetCommands slice, got %#v", got["sessionResetCommands"])
	}
	if len(commands) != 2 || commands[0] != "/new" || commands[1] != "新会话" {
		t.Fatalf("unexpected normalized sessionResetCommands: %#v", commands)
	}
}

func TestNormalizeConfigRejectsFractionalCardUpdateThrottleMs(t *testing.T) {
	t.Parallel()

	_, err := normalizeConfig(map[string]any{
		"appKey":               "key",
		"appSecret":            "secret",
		"cardUpdateThrottleMs": 250.5,
	})
	if err == nil {
		t.Fatal("expected fractional cardUpdateThrottleMs validation error")
	}
}

func TestNormalizeUserConfigAcceptsSupportedFields(t *testing.T) {
	t.Parallel()

	got, err := normalizeUserConfig(map[string]any{
		"conversationId": "  cid-1  ",
		"staffId":        "  staff-1  ",
		"unionId":        "  union-1  ",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got["conversationId"] != "cid-1" {
		t.Fatalf("unexpected conversationId: %#v", got["conversationId"])
	}
	if got["staffId"] != "staff-1" {
		t.Fatalf("unexpected staffId: %#v", got["staffId"])
	}
	if got["unionId"] != "union-1" {
		t.Fatalf("unexpected unionId: %#v", got["unionId"])
	}
}

func TestNormalizeUserConfigRequiresAtLeastOneIdentifier(t *testing.T) {
	t.Parallel()

	if _, err := normalizeUserConfig(map[string]any{}); err == nil {
		t.Fatal("expected user config validation error")
	}
}

func TestDiscoverSelf(t *testing.T) {
	t.Parallel()

	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL: "https://dingtalk.test",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					switch r.URL.Path {
					case "/v1.0/oauth2/accessToken":
						if r.Method != http.MethodPost {
							t.Fatalf("unexpected method: %s", r.Method)
						}

						var req AccessTokenRequest
						if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
							t.Fatalf("decode access token request: %v", err)
						}
						if req.AppKey != "key" || req.AppSecret != "secret" {
							t.Fatalf("unexpected token request: %#v", req)
						}

						return jsonResponse(t, AccessTokenResponse{
							AccessToken: "token-123",
							ExpiresIn:   7200,
						}), nil
					case "/v1.0/robot/me":
						if got := r.Header.Get("x-acs-dingtalk-access-token"); got != "token-123" {
							t.Fatalf("unexpected access token header: %q", got)
						}
						return jsonResponse(t, SelfIdentityResponse{
							RobotCode: "robot-code-1",
							AppID:     "app-id-1",
							UserID:    "chatbot-user-1",
							StaffID:   "staff-1",
							UnionID:   "union-1",
						}), nil
					default:
						t.Fatalf("unexpected request path: %s", r.URL.Path)
						return nil, nil
					}
				}),
			},
		}
	}

	identity, externalID, err := adapter.DiscoverSelf(context.Background(), map[string]any{
		"appKey":     "key",
		"appSecret":  "secret",
		"robotCode":  "ignored-by-discovery",
		"apiBaseURL": "ignored-by-injected-client",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if identity["chatbotUserId"] != "chatbot-user-1" {
		t.Fatalf("unexpected chatbotUserId: %#v", identity["chatbotUserId"])
	}
	if identity["staffId"] != "staff-1" {
		t.Fatalf("unexpected staffId: %#v", identity["staffId"])
	}
	if identity["unionId"] != "union-1" {
		t.Fatalf("unexpected unionId: %#v", identity["unionId"])
	}
	if identity["robotCode"] != "robot-code-1" {
		t.Fatalf("unexpected robotCode: %#v", identity["robotCode"])
	}
	if identity["appId"] != "app-id-1" {
		t.Fatalf("unexpected appId: %#v", identity["appId"])
	}
	if externalID != "chatbot_user_id:chatbot-user-1" {
		t.Fatalf("unexpected external identity: %q", externalID)
	}
}

func TestDiscoverSelfDoesNotFallBackToConfiguredRobotCode(t *testing.T) {
	t.Parallel()

	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL: "https://dingtalk.test",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					switch r.URL.Path {
					case "/v1.0/oauth2/accessToken":
						return jsonResponse(t, AccessTokenResponse{
							AccessToken: "token-123",
							ExpiresIn:   7200,
						}), nil
					case "/v1.0/robot/me":
						return jsonResponse(t, SelfIdentityResponse{
							AppID: "app-id-1",
						}), nil
					default:
						t.Fatalf("unexpected request path: %s", r.URL.Path)
						return nil, nil
					}
				}),
			},
		}
	}

	identity, externalID, err := adapter.DiscoverSelf(context.Background(), map[string]any{
		"appKey":    "key",
		"appSecret": "secret",
		"robotCode": "configured-only",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := identity["robotCode"]; ok {
		t.Fatalf("unexpected robotCode fallback in discovered identity: %#v", identity)
	}
	if externalID != "" {
		t.Fatalf("unexpected external identity fallback: %q", externalID)
	}
}

func TestDiscoverSelfUsesTypedStaffExternalIdentity(t *testing.T) {
	t.Parallel()

	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL: "https://dingtalk.test",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					switch r.URL.Path {
					case "/v1.0/oauth2/accessToken":
						return jsonResponse(t, AccessTokenResponse{
							AccessToken: "token-123",
							ExpiresIn:   7200,
						}), nil
					case "/v1.0/robot/me":
						return jsonResponse(t, SelfIdentityResponse{
							StaffID: "staff-42",
						}), nil
					default:
						t.Fatalf("unexpected request path: %s", r.URL.Path)
						return nil, nil
					}
				}),
			},
		}
	}

	_, externalID, err := adapter.DiscoverSelf(context.Background(), map[string]any{
		"appKey":    "key",
		"appSecret": "secret",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if externalID != "user:staff:staff-42" {
		t.Fatalf("unexpected staff external identity: %q", externalID)
	}
}

func TestDiscoverSelfUsesTypedUnionExternalIdentity(t *testing.T) {
	t.Parallel()

	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL: "https://dingtalk.test",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					switch r.URL.Path {
					case "/v1.0/oauth2/accessToken":
						return jsonResponse(t, AccessTokenResponse{
							AccessToken: "token-123",
							ExpiresIn:   7200,
						}), nil
					case "/v1.0/robot/me":
						return jsonResponse(t, SelfIdentityResponse{
							UnionID: "union-42",
						}), nil
					default:
						t.Fatalf("unexpected request path: %s", r.URL.Path)
						return nil, nil
					}
				}),
			},
		}
	}

	_, externalID, err := adapter.DiscoverSelf(context.Background(), map[string]any{
		"appKey":    "key",
		"appSecret": "secret",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if externalID != "user:union:union-42" {
		t.Fatalf("unexpected union external identity: %q", externalID)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func jsonResponse(t *testing.T, payload any) *http.Response {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(body))),
	}
}
