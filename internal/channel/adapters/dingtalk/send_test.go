package dingtalk

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/memohai/memoh/internal/channel"
)

func TestSendWebhookReplyText(t *testing.T) {
	t.Parallel()

	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL: "https://dingtalk.test",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					if r.URL.String() != "https://example.com/session" {
						t.Fatalf("unexpected webhook url: %s", r.URL.String())
					}
					var body TextReplyPayload
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatalf("decode webhook body: %v", err)
					}
					if body.MsgType != "text" {
						t.Fatalf("unexpected msgtype: %#v", body.MsgType)
					}
					if body.Text.Content != "hello webhook" {
						t.Fatalf("unexpected text content: %#v", body.Text.Content)
					}
					return jsonResponse(t, map[string]any{"success": true}), nil
				}),
			},
		}
	}

	err := adapter.Send(context.Background(), channel.ChannelConfig{
		ID:    "cfg-1",
		BotID: "bot-1",
		Credentials: map[string]any{
			"appKey":    "key",
			"appSecret": "secret",
			"robotCode": "robot-6",
		},
	}, channel.OutboundMessage{
		Target: "webhook:https://example.com/session",
		Message: channel.Message{
			Text: " hello webhook ",
		},
	})
	if err != nil {
		t.Fatalf("send webhook reply: %v", err)
	}
}

func TestSendConversationText(t *testing.T) {
	t.Parallel()

	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL: "https://dingtalk.test",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					switch r.URL.Path {
					case "/v1.0/oauth2/accessToken":
						return jsonResponse(t, AccessTokenResponse{AccessToken: "token-1"}), nil
					case "/v1.0/robot/groupMessages/send":
						if got := r.Header.Get("x-acs-dingtalk-access-token"); got != "token-1" {
							t.Fatalf("unexpected access token header: %q", got)
						}
						var body ProactiveSendPayload
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Fatalf("decode proactive conversation body: %v", err)
						}
						if body.RobotCode != "robot-1" {
							t.Fatalf("unexpected robotCode: %#v", body.RobotCode)
						}
						if body.OpenConversationID != "cid-1" {
							t.Fatalf("unexpected openConversationId: %#v", body.OpenConversationID)
						}
						if body.MsgKey != "sampleText" {
							t.Fatalf("unexpected msgKey: %#v", body.MsgKey)
						}
						var param InboundTextBlock
						if err := json.Unmarshal([]byte(body.MsgParam), &param); err != nil {
							t.Fatalf("decode msgParam: %v", err)
						}
						if param.Content != "hello conversation" {
							t.Fatalf("unexpected msgParam content: %#v", param.Content)
						}
						return jsonResponse(t, map[string]any{"processQueryKey": "pqk-1"}), nil
					default:
						t.Fatalf("unexpected request path: %s", r.URL.Path)
						return nil, nil
					}
				}),
			},
		}
	}

	err := adapter.Send(context.Background(), channel.ChannelConfig{
		ID:    "cfg-2",
		BotID: "bot-1",
		Credentials: map[string]any{
			"appKey":    "key",
			"appSecret": "secret",
			"robotCode": "robot-1",
		},
	}, channel.OutboundMessage{
		Target: "conversation:cid-1",
		Message: channel.Message{
			Text: "hello conversation",
		},
	})
	if err != nil {
		t.Fatalf("send conversation text: %v", err)
	}
}

func TestSendUserText(t *testing.T) {
	t.Parallel()

	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL: "https://dingtalk.test",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					switch r.URL.Path {
					case "/v1.0/oauth2/accessToken":
						return jsonResponse(t, AccessTokenResponse{AccessToken: "token-2"}), nil
					case "/v1.0/robot/oToMessages/batchSend":
						if got := r.Header.Get("x-acs-dingtalk-access-token"); got != "token-2" {
							t.Fatalf("unexpected access token header: %q", got)
						}
						var body ProactiveSendPayload
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Fatalf("decode proactive user body: %v", err)
						}
						if body.RobotCode != "robot-2" {
							t.Fatalf("unexpected robotCode: %#v", body.RobotCode)
						}
						if len(body.UserIDs) != 1 || body.UserIDs[0] != "union-1" {
							t.Fatalf("unexpected userIds: %#v", body.UserIDs)
						}
						if body.MsgKey != "sampleText" {
							t.Fatalf("unexpected msgKey: %#v", body.MsgKey)
						}
						var param InboundTextBlock
						if err := json.Unmarshal([]byte(body.MsgParam), &param); err != nil {
							t.Fatalf("decode msgParam: %v", err)
						}
						if param.Content != "hello user" {
							t.Fatalf("unexpected msgParam content: %#v", param.Content)
						}
						return jsonResponse(t, map[string]any{"taskId": "task-1"}), nil
					default:
						t.Fatalf("unexpected request path: %s", r.URL.Path)
						return nil, nil
					}
				}),
			},
		}
	}

	err := adapter.Send(context.Background(), channel.ChannelConfig{
		ID:    "cfg-3",
		BotID: "bot-1",
		Credentials: map[string]any{
			"appKey":    "key",
			"appSecret": "secret",
			"robotCode": "robot-2",
		},
	}, channel.OutboundMessage{
		Target: "user:union:union-1",
		Message: channel.Message{
			Text: "hello user",
		},
	})
	if err != nil {
		t.Fatalf("send user text: %v", err)
	}
}

func TestSendStaffUserText(t *testing.T) {
	t.Parallel()

	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL: "https://dingtalk.test",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					switch r.URL.Path {
					case "/v1.0/oauth2/accessToken":
						return jsonResponse(t, AccessTokenResponse{AccessToken: "token-4"}), nil
					case "/v1.0/robot/oToMessages/batchSend":
						var body ProactiveSendPayload
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Fatalf("decode proactive staff user body: %v", err)
						}
						if len(body.UserIDs) != 1 || body.UserIDs[0] != "staff-1" {
							t.Fatalf("unexpected userIds: %#v", body.UserIDs)
						}
						return jsonResponse(t, map[string]any{"taskId": "task-2"}), nil
					default:
						t.Fatalf("unexpected request path: %s", r.URL.Path)
						return nil, nil
					}
				}),
			},
		}
	}

	err := adapter.Send(context.Background(), channel.ChannelConfig{
		ID:    "cfg-3b",
		BotID: "bot-1",
		Credentials: map[string]any{
			"appKey":    "key",
			"appSecret": "secret",
			"robotCode": "robot-2",
		},
	}, channel.OutboundMessage{
		Target: "user:staff:staff-1",
		Message: channel.Message{
			Text: "hello staff user",
		},
	})
	if err != nil {
		t.Fatalf("send staff user text: %v", err)
	}
}

func TestSendFallsBackToSelfIdentityRobotCode(t *testing.T) {
	t.Parallel()

	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL: "https://dingtalk.test",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					switch r.URL.Path {
					case "/v1.0/oauth2/accessToken":
						return jsonResponse(t, AccessTokenResponse{AccessToken: "token-3"}), nil
					case "/v1.0/robot/groupMessages/send":
						var body ProactiveSendPayload
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Fatalf("decode proactive conversation body: %v", err)
						}
						if body.RobotCode != "robot-from-self" {
							t.Fatalf("unexpected robotCode fallback: %#v", body.RobotCode)
						}
						return jsonResponse(t, map[string]any{"processQueryKey": "pqk-2"}), nil
					default:
						t.Fatalf("unexpected request path: %s", r.URL.Path)
						return nil, nil
					}
				}),
			},
		}
	}

	err := adapter.Send(context.Background(), channel.ChannelConfig{
		ID:    "cfg-4",
		BotID: "bot-1",
		Credentials: map[string]any{
			"appKey":    "key",
			"appSecret": "secret",
		},
		SelfIdentity: map[string]any{
			"robotCode": "robot-from-self",
		},
	}, channel.OutboundMessage{
		Target: "conversation:cid-2",
		Message: channel.Message{
			Text: "hello fallback",
		},
	})
	if err != nil {
		t.Fatalf("send with self identity robotCode fallback: %v", err)
	}
}

func TestSendRejectsEmptyText(t *testing.T) {
	t.Parallel()

	adapter := NewDingTalkAdapter(nil)
	err := adapter.Send(context.Background(), channel.ChannelConfig{
		ID: "cfg-5",
		Credentials: map[string]any{
			"appKey":    "key",
			"appSecret": "secret",
			"robotCode": "robot-6",
		},
	}, channel.OutboundMessage{
		Target:  "webhook:https://example.com/session",
		Message: channel.Message{},
	})
	if err == nil || !strings.Contains(err.Error(), "message text is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendSplitsTextAndAttachments(t *testing.T) {
	t.Parallel()

	var calls []string
	adapter := NewDingTalkAdapter(nil)
	adapter.SetAssetOpener(&testAssetOpener{
		data:  []byte("image-bytes"),
		asset: testAsset{mime: "image/png"},
	})
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL: "https://dingtalk.test",
			httpClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					switch r.URL.Path {
					case "/v1.0/oauth2/accessToken":
						return jsonResponse(t, AccessTokenResponse{AccessToken: "token-attachments"}), nil
					case "/media/upload":
						calls = append(calls, "upload")
						return jsonResponse(t, MediaUploadResponse{MediaID: "media-attachments", Type: "image"}), nil
					case "/session":
						calls = append(calls, "attachment")
						var body WebhookMediaPayload
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Fatalf("decode webhook body: %v", err)
						}
						if body.MsgType != "image" {
							t.Fatalf("unexpected msgtype: %q", body.MsgType)
						}
						return jsonResponse(t, map[string]any{"success": true}), nil
					default:
						t.Fatalf("unexpected request url: %s", r.URL.String())
						return nil, nil
					}
				}),
			},
			replyTextFunc: func(_ context.Context, sessionWebhook, text string) error {
				if sessionWebhook != "https://example.com/session" {
					t.Fatalf("unexpected webhook: %s", sessionWebhook)
				}
				if strings.TrimSpace(text) != "hello" {
					t.Fatalf("unexpected text: %q", text)
				}
				calls = append(calls, "text")
				return nil
			},
		}
	}

	err := adapter.Send(context.Background(), channel.ChannelConfig{
		ID:    "cfg-6",
		BotID: "bot-6",
		Credentials: map[string]any{
			"appKey":    "key",
			"appSecret": "secret",
			"robotCode": "robot-6",
		},
	}, channel.OutboundMessage{
		Target: "webhook:https://example.com/session",
		Message: channel.Message{
			Text: "hello",
			Attachments: []channel.Attachment{{
				Type:        channel.AttachmentImage,
				ContentHash: "hash-image",
				Name:        "picture.png",
			}},
		},
	})
	if err != nil {
		t.Fatalf("send with attachments: %v", err)
	}
	if len(calls) != 3 || calls[0] != "text" || calls[1] != "upload" || calls[2] != "attachment" {
		t.Fatalf("unexpected call order: %#v", calls)
	}
}

func TestSendRejectsMalformedTarget(t *testing.T) {
	t.Parallel()

	adapter := NewDingTalkAdapter(nil)
	err := adapter.Send(context.Background(), channel.ChannelConfig{
		ID: "cfg-7",
		Credentials: map[string]any{
			"appKey":    "key",
			"appSecret": "secret",
		},
	}, channel.OutboundMessage{
		Target: "invalid-target",
		Message: channel.Message{
			Text: "hello",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "family prefix") {
		t.Fatalf("unexpected error: %v", err)
	}
}
