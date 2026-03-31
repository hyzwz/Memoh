package dingtalk

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/memohai/memoh/internal/channel"
	routepkg "github.com/memohai/memoh/internal/channel/route"
)

func TestEventToInboundMessagePrivateText(t *testing.T) {
	t.Parallel()

	result, ok := eventToInboundMessage(Config{}, "bot-1", InboundMessageEnvelope{
		ConversationID:            "cid-private",
		ConversationType:          "1",
		SenderID:                  "chatbot-user-1",
		SenderStaffID:             "staff-1",
		SenderNick:                "Alice",
		SessionWebhook:            "https://example.com/session",
		SessionWebhookExpiredTime: 1711272000000,
		RobotCode:                 "robot-1",
		ChatbotUserID:             "chatbot-1",
		CardSupported:             boolPtr(true),
		MsgID:                     "msg-1",
		MsgType:                   "text",
		Text: InboundTextBlock{
			Content: "hello dingtalk",
		},
		CreateAt: "2026-03-23T12:00:00Z",
	})
	if !ok {
		t.Fatal("expected inbound message")
	}
	if result.Command != nil {
		t.Fatalf("unexpected session command: %#v", result.Command)
	}
	msg := result.Message
	if msg.Channel != Type {
		t.Fatalf("unexpected channel: %s", msg.Channel)
	}
	if msg.BotID != "bot-1" {
		t.Fatalf("unexpected bot id: %s", msg.BotID)
	}
	if msg.Message.ID != "msg-1" {
		t.Fatalf("unexpected message id: %s", msg.Message.ID)
	}
	if msg.Message.Text != "hello dingtalk" {
		t.Fatalf("unexpected text: %q", msg.Message.Text)
	}
	if msg.ReplyTarget != "webhook:https://example.com/session" {
		t.Fatalf("unexpected reply target: %s", msg.ReplyTarget)
	}
	if msg.RouteKey != "dingtalk:bot-1:cid-private" {
		t.Fatalf("unexpected route key: %s", msg.RouteKey)
	}
	if msg.Conversation.ID != "cid-private" {
		t.Fatalf("unexpected conversation id: %s", msg.Conversation.ID)
	}
	if msg.Conversation.Type != "private" {
		t.Fatalf("unexpected conversation type: %s", msg.Conversation.Type)
	}
	if msg.Sender.SubjectID != "staff-1" {
		t.Fatalf("unexpected sender subject: %s", msg.Sender.SubjectID)
	}
	if msg.Sender.DisplayName != "Alice" {
		t.Fatalf("unexpected sender display name: %s", msg.Sender.DisplayName)
	}
	if got := msg.Sender.Attribute("sender_id"); got != "chatbot-user-1" {
		t.Fatalf("unexpected sender_id attribute: %s", got)
	}
	if got, _ := msg.Metadata["session_webhook"].(string); got != "https://example.com/session" {
		t.Fatalf("unexpected session_webhook metadata: %#v", msg.Metadata["session_webhook"])
	}
	if got, _ := msg.Metadata["chatbot_user_id"].(string); got != "chatbot-1" {
		t.Fatalf("unexpected chatbot_user_id metadata: %#v", msg.Metadata["chatbot_user_id"])
	}
	if got, ok := msg.Metadata["card_supported"].(bool); !ok || !got {
		t.Fatalf("unexpected card_supported metadata: %#v", msg.Metadata["card_supported"])
	}
	if mentioned, _ := msg.Metadata["is_mentioned"].(bool); mentioned {
		t.Fatal("private message should not require mention")
	}
	expectedAt := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	if !msg.ReceivedAt.Equal(expectedAt) {
		t.Fatalf("unexpected received at: %s", msg.ReceivedAt)
	}
}

func TestEventToInboundMessageGroupMentionGating(t *testing.T) {
	t.Parallel()

	base := InboundMessageEnvelope{
		ConversationID:   "cid-group",
		ConversationType: "2",
		SenderID:         "chatbot-user-2",
		SenderUnionID:    "union-2",
		SenderNick:       "Bob",
		SessionWebhook:   "https://example.com/group-session",
		MsgID:            "msg-group",
		MsgType:          "text",
		Text: InboundTextBlock{
			Content: "@bot hi",
		},
		CreateAt: "2026-03-23T13:00:00Z",
	}

	result, ok := eventToInboundMessage(Config{
		RequireMentionInGroup: true,
		GroupSessionScope:     defaultGroupSessionScope,
	}, "bot-2", func() InboundMessageEnvelope {
		msg := base
		msg.IsInAtList = true
		return msg
	}())
	if !ok {
		t.Fatal("expected mentioned group message")
	}
	if result.Message.RouteKey != "dingtalk:bot-2:cid-group" {
		t.Fatalf("unexpected group route key: %s", result.Message.RouteKey)
	}
	if mentioned, _ := result.Message.Metadata["is_mentioned"].(bool); !mentioned {
		t.Fatal("expected mentioned group metadata")
	}

	if _, ok := eventToInboundMessage(Config{
		RequireMentionInGroup: true,
		GroupSessionScope:     defaultGroupSessionScope,
	}, "bot-2", base); ok {
		t.Fatal("expected non-mentioned group message to be ignored")
	}
}

func TestRouteKeyGeneration(t *testing.T) {
	t.Parallel()

	event := InboundMessageEnvelope{
		ConversationID:   "cid-group",
		ConversationType: "2",
		SenderStaffID:    "staff-3",
		MsgID:            "msg-route",
		MsgType:          "text",
		Text: InboundTextBlock{
			Content: "route me",
		},
		IsInAtList: true,
	}

	groupResult, ok := eventToInboundMessage(Config{
		RequireMentionInGroup: false,
		GroupSessionScope:     defaultGroupSessionScope,
	}, "bot-3", event)
	if !ok {
		t.Fatal("expected group-scoped inbound message")
	}
	if groupResult.Message.RouteKey != "dingtalk:bot-3:cid-group" {
		t.Fatalf("unexpected group route key: %s", groupResult.Message.RouteKey)
	}

	groupSenderResult, ok := eventToInboundMessage(Config{
		RequireMentionInGroup: false,
		GroupSessionScope:     groupSessionScopeGroupSender,
	}, "bot-3", event)
	if !ok {
		t.Fatal("expected group_sender-scoped inbound message")
	}
	if groupSenderResult.Message.RouteKey != "dingtalk:bot-3:cid-group:staff-3" {
		t.Fatalf("unexpected group_sender route key: %s", groupSenderResult.Message.RouteKey)
	}
	if groupSenderResult.Message.Conversation.ThreadID != "staff-3" {
		t.Fatalf("unexpected group_sender thread id: %s", groupSenderResult.Message.Conversation.ThreadID)
	}
}

func TestSessionCommandDetectionAndResetSignal(t *testing.T) {
	t.Parallel()

	cfg := Config{
		GroupSessionScope:     groupSessionScopeGroupSender,
		RequireMentionInGroup: true,
		EnableSessionCommands: true,
		SessionResetCommands:  []string{"/new", "新会话"},
	}

	tests := []struct {
		name string
		text string
	}{
		{name: "slash command", text: "/new"},
		{name: "chinese command", text: " 新会话 "},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, ok := eventToInboundMessage(cfg, "bot-4", InboundMessageEnvelope{
				ConversationID:   "cid-command",
				ConversationType: "2",
				SenderUnionID:    "union-4",
				SenderNick:       "Carol",
				SessionWebhook:   "https://example.com/command",
				MsgID:            "msg-command",
				MsgType:          "text",
				IsInAtList:       true,
				Text: InboundTextBlock{
					Content: tc.text,
				},
			})
			if !ok {
				t.Fatal("expected command message")
			}
			if result.Command == nil {
				t.Fatal("expected session command signal")
			}
			if result.Command.Name != sessionCommandReset {
				t.Fatalf("unexpected command name: %s", result.Command.Name)
			}
			if result.Command.Normalized != strings.TrimSpace(tc.text) {
				t.Fatalf("unexpected normalized command: %q", result.Command.Normalized)
			}
			if !result.Command.ResetCurrentRoute {
				t.Fatal("expected command to mark current route for reset")
			}
			if result.Command.RouteKey != "dingtalk:bot-4:cid-command:union-4" {
				t.Fatalf("unexpected reset route key: %s", result.Command.RouteKey)
			}
			if result.Message.Message.Text != strings.TrimSpace(tc.text) {
				t.Fatalf("command message text should be preserved for downstream consumers, got %q", result.Message.Message.Text)
			}
			if result.Message.Metadata["session_command"] != sessionCommandReset {
				t.Fatalf("unexpected session_command metadata: %#v", result.Message.Metadata["session_command"])
			}
		})
	}
}

func TestStableSenderSubjectIDSelectionOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event InboundMessageEnvelope
		want  string
	}{
		{
			name: "prefers staff id",
			event: InboundMessageEnvelope{
				SenderStaffID: "staff-5",
				SenderUnionID: "union-5",
				SenderID:      "sender-5",
				SenderNick:    "display-5",
			},
			want: "staff-5",
		},
		{
			name: "falls back to union id",
			event: InboundMessageEnvelope{
				SenderUnionID: "union-6",
				SenderID:      "sender-6",
				SenderNick:    "display-6",
			},
			want: "union-6",
		},
		{
			name: "falls back to sender id",
			event: InboundMessageEnvelope{
				SenderID:   "sender-7",
				SenderNick: "display-7",
			},
			want: "sender-7",
		},
		{
			name: "does not use display name as stable subject",
			event: InboundMessageEnvelope{
				SenderNick: "display-8",
			},
			want: "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := stableSenderSubjectID(tc.event); got != tc.want {
				t.Fatalf("unexpected sender subject id: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestEventToInboundMessageRejectsMissingStableSenderIdentifier(t *testing.T) {
	t.Parallel()

	if _, ok := eventToInboundMessage(Config{}, "bot-5", InboundMessageEnvelope{
		ConversationID:   "cid-missing-sender",
		ConversationType: "1",
		SenderNick:       "display-only",
		SessionWebhook:   "https://example.com/session",
		MsgID:            "msg-missing-sender",
		MsgType:          "text",
		Text: InboundTextBlock{
			Content: "hello",
		},
	}); ok {
		t.Fatal("expected inbound event with no stable sender identifier to be rejected")
	}
}

func TestEventToInboundMessageAttachmentReferences(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		event           InboundMessageEnvelope
		wantType        string
		wantPlatformKey string
		wantURL         string
		wantName        string
		wantDurationMS  int64
	}{
		{
			name: "image",
			event: InboundMessageEnvelope{
				ConversationID:   "cid-attachment",
				ConversationType: "1",
				SenderStaffID:    "staff-image",
				SessionWebhook:   "https://example.com/image-session",
				MsgID:            "msg-image",
				MsgType:          "image",
				Image: &InboundMediaBlock{
					DownloadCode: "download-image",
					URL:          "https://example.com/image.png",
					FileName:     "image.png",
					MimeType:     "image/png",
					Size:         1234,
				},
			},
			wantType:        "image",
			wantPlatformKey: "download-image",
			wantURL:         "https://example.com/image.png",
			wantName:        "image.png",
		},
		{
			name: "file",
			event: InboundMessageEnvelope{
				ConversationID:   "cid-attachment",
				ConversationType: "1",
				SenderStaffID:    "staff-file",
				SessionWebhook:   "https://example.com/file-session",
				MsgID:            "msg-file",
				MsgType:          "file",
				File: &InboundMediaBlock{
					MediaID:  "media-file",
					FileName: "notes.txt",
					MimeType: "text/plain",
					Size:     4096,
				},
			},
			wantType:        "file",
			wantPlatformKey: "media-file",
			wantName:        "notes.txt",
		},
		{
			name: "audio",
			event: InboundMessageEnvelope{
				ConversationID:   "cid-attachment",
				ConversationType: "1",
				SenderStaffID:    "staff-audio",
				SessionWebhook:   "https://example.com/audio-session",
				MsgID:            "msg-audio",
				MsgType:          "audio",
				Audio: &InboundAudioBlock{
					InboundMediaBlock: InboundMediaBlock{
						DownloadCode: "download-audio",
						FileName:     "voice.mp3",
						MimeType:     "audio/mpeg",
					},
					DurationMs: 2300,
				},
			},
			wantType:        "audio",
			wantPlatformKey: "download-audio",
			wantName:        "voice.mp3",
			wantDurationMS:  2300,
		},
		{
			name: "voice",
			event: InboundMessageEnvelope{
				ConversationID:   "cid-attachment",
				ConversationType: "1",
				SenderStaffID:    "staff-voice",
				SessionWebhook:   "https://example.com/voice-session",
				MsgID:            "msg-voice",
				MsgType:          "voice",
				Audio: &InboundAudioBlock{
					InboundMediaBlock: InboundMediaBlock{
						MediaID:  "media-voice",
						FileName: "voice.amr",
						MimeType: "audio/amr",
					},
					DurationMs: 1800,
				},
			},
			wantType:        "voice",
			wantPlatformKey: "media-voice",
			wantName:        "voice.amr",
			wantDurationMS:  1800,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, ok := eventToInboundMessage(Config{}, "bot-attachments", tc.event)
			if !ok {
				t.Fatal("expected inbound attachment message")
			}
			if result.Command != nil {
				t.Fatalf("unexpected session command: %#v", result.Command)
			}
			if result.Message.Message.Text != "" {
				t.Fatalf("unexpected text for attachment-only event: %q", result.Message.Message.Text)
			}
			if len(result.Message.Message.Attachments) != 1 {
				t.Fatalf("unexpected attachments: %d", len(result.Message.Message.Attachments))
			}
			att := result.Message.Message.Attachments[0]
			if string(att.Type) != tc.wantType {
				t.Fatalf("unexpected attachment type: %s", att.Type)
			}
			if att.PlatformKey != tc.wantPlatformKey {
				t.Fatalf("unexpected attachment platform key: %q", att.PlatformKey)
			}
			if att.URL != tc.wantURL {
				t.Fatalf("unexpected attachment url: %q", att.URL)
			}
			if att.Name != tc.wantName {
				t.Fatalf("unexpected attachment name: %q", att.Name)
			}
			if att.DurationMs != tc.wantDurationMS {
				t.Fatalf("unexpected attachment duration: %d", att.DurationMs)
			}
			if att.SourcePlatform != string(Type) {
				t.Fatalf("unexpected source platform: %q", att.SourcePlatform)
			}
		})
	}
}

func TestConnectStartsLongLivedReceiverAndDispatchesSupportedEvents(t *testing.T) {
	t.Parallel()

	fakeStream := &fakeDingTalkStreamClient{
		started: make(chan struct{}),
		done:    make(chan struct{}),
	}
	replyLog := &fakeReplyLog{}
	adapter := NewDingTalkAdapter(nil)
	adapter.newStreamClient = func(_ Config, handler streamEventHandler) (streamClient, error) {
		fakeStream.handler = handler
		return fakeStream, nil
	}
	adapter.newClient = func(_ Config) *client {
		return &client{replyTextFunc: replyLog.replyText}
	}

	received := make(chan channel.InboundMessage, 1)
	conn, err := adapter.Connect(context.Background(), channel.ChannelConfig{
		ID:          "cfg-1",
		BotID:       "bot-1",
		ChannelType: Type,
		Credentials: map[string]any{
			"appKey":    "app-key",
			"appSecret": "app-secret",
		},
	}, func(_ context.Context, _ channel.ChannelConfig, msg channel.InboundMessage) error {
		received <- msg
		return nil
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	select {
	case <-fakeStream.started:
	case <-time.After(time.Second):
		t.Fatal("receiver did not start")
	}

	ack := fakeStream.emit(t, StreamCallbackEnvelope{
		Headers: StreamCallbackHeaders{
			Topic: streamTopicChatbotMessage,
		},
		Data: mustJSON(t, InboundMessageEnvelope{
			ConversationID:   "cid-private",
			ConversationType: "1",
			SenderStaffID:    "staff-1",
			SenderNick:       "Alice",
			SessionWebhook:   "https://example.com/session",
			MsgID:            "msg-1",
			MsgType:          "text",
			Text: InboundTextBlock{
				Content: "hello stream",
			},
			CreateAt: "2026-03-24T12:00:00Z",
		}),
	})
	if !ack.Success {
		t.Fatalf("expected success ACK, got %#v", ack)
	}

	select {
	case msg := <-received:
		if msg.Message.Text != "hello stream" {
			t.Fatalf("unexpected text: %q", msg.Message.Text)
		}
	case <-time.After(time.Second):
		t.Fatal("handler did not receive supported inbound event")
	}

	if len(replyLog.messages) != 0 {
		t.Fatalf("unexpected reply messages: %#v", replyLog.messages)
	}
	if err := conn.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestConnectACKsUnsupportedEventsWithoutDispatch(t *testing.T) {
	t.Parallel()

	fakeStream := &fakeDingTalkStreamClient{
		started: make(chan struct{}),
		done:    make(chan struct{}),
	}
	adapter := NewDingTalkAdapter(nil)
	adapter.newStreamClient = func(_ Config, handler streamEventHandler) (streamClient, error) {
		fakeStream.handler = handler
		return fakeStream, nil
	}

	handlerCalled := make(chan struct{}, 1)
	conn, err := adapter.Connect(context.Background(), channel.ChannelConfig{
		ID:          "cfg-unsupported",
		BotID:       "bot-unsupported",
		ChannelType: Type,
		Credentials: map[string]any{
			"appKey":    "app-key",
			"appSecret": "app-secret",
		},
	}, func(context.Context, channel.ChannelConfig, channel.InboundMessage) error {
		handlerCalled <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() {
		_ = conn.Stop(context.Background())
	}()

	select {
	case <-fakeStream.started:
	case <-time.After(time.Second):
		t.Fatal("receiver did not start")
	}

	unsupportedAck := fakeStream.emit(t, StreamCallbackEnvelope{
		Headers: StreamCallbackHeaders{
			Topic: "unsupported/topic",
		},
		Data: mustJSON(t, map[string]any{"ignored": true}),
	})
	if !unsupportedAck.Success {
		t.Fatalf("expected unsupported event to be ACKed, got %#v", unsupportedAck)
	}

	messageTypeAck := fakeStream.emit(t, StreamCallbackEnvelope{
		Headers: StreamCallbackHeaders{
			Topic: streamTopicChatbotMessage,
		},
		Data: mustJSON(t, InboundMessageEnvelope{
			ConversationID:   "cid-unsupported",
			ConversationType: "1",
			SenderStaffID:    "staff-unsupported",
			SessionWebhook:   "https://example.com/session",
			MsgID:            "msg-unsupported",
			MsgType:          "markdown",
		}),
	})
	if !messageTypeAck.Success {
		t.Fatalf("expected unsupported message event to be ACKed, got %#v", messageTypeAck)
	}

	select {
	case <-handlerCalled:
		t.Fatal("unsupported events should not reach the inbound handler")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestConnectReturnsErrorACKOnDecodeFailure(t *testing.T) {
	t.Parallel()

	fakeStream := &fakeDingTalkStreamClient{
		started: make(chan struct{}),
		done:    make(chan struct{}),
	}
	adapter := NewDingTalkAdapter(nil)
	adapter.newStreamClient = func(_ Config, handler streamEventHandler) (streamClient, error) {
		fakeStream.handler = handler
		return fakeStream, nil
	}

	conn, err := adapter.Connect(context.Background(), channel.ChannelConfig{
		ID:          "cfg-decode-failure",
		BotID:       "bot-decode-failure",
		ChannelType: Type,
		Credentials: map[string]any{
			"appKey":    "app-key",
			"appSecret": "app-secret",
		},
	}, func(context.Context, channel.ChannelConfig, channel.InboundMessage) error {
		return nil
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() {
		_ = conn.Stop(context.Background())
	}()

	select {
	case <-fakeStream.started:
	case <-time.After(time.Second):
		t.Fatal("receiver did not start")
	}

	ack := fakeStream.emit(t, StreamCallbackEnvelope{
		Headers: StreamCallbackHeaders{
			Topic: streamTopicChatbotMessage,
		},
		Data: json.RawMessage(`{`),
	})
	if ack.Success {
		t.Fatalf("expected decode failure to return error ACK, got %#v", ack)
	}
	if !strings.Contains(ack.Message, "unexpected end of JSON input") {
		t.Fatalf("unexpected decode failure ACK message: %q", ack.Message)
	}
}

func TestConnectReturnsErrorACKWhenHandlerFails(t *testing.T) {
	t.Parallel()

	fakeStream := &fakeDingTalkStreamClient{
		started: make(chan struct{}),
		done:    make(chan struct{}),
	}
	adapter := NewDingTalkAdapter(nil)
	adapter.newStreamClient = func(_ Config, handler streamEventHandler) (streamClient, error) {
		fakeStream.handler = handler
		return fakeStream, nil
	}

	conn, err := adapter.Connect(context.Background(), channel.ChannelConfig{
		ID:          "cfg-handler-failure",
		BotID:       "bot-handler-failure",
		ChannelType: Type,
		Credentials: map[string]any{
			"appKey":    "app-key",
			"appSecret": "app-secret",
		},
	}, func(context.Context, channel.ChannelConfig, channel.InboundMessage) error {
		return errors.New("handler exploded")
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() {
		_ = conn.Stop(context.Background())
	}()

	select {
	case <-fakeStream.started:
	case <-time.After(time.Second):
		t.Fatal("receiver did not start")
	}

	ack := fakeStream.emit(t, StreamCallbackEnvelope{
		Headers: StreamCallbackHeaders{
			Topic: streamTopicChatbotMessage,
		},
		Data: mustJSON(t, InboundMessageEnvelope{
			ConversationID:   "cid-handler-failure",
			ConversationType: "1",
			SenderStaffID:    "staff-handler-failure",
			SessionWebhook:   "https://example.com/session",
			MsgID:            "msg-handler-failure",
			MsgType:          "text",
			Text: InboundTextBlock{
				Content: "fail me",
			},
		}),
	})
	if ack.Success {
		t.Fatalf("expected handler failure to return error ACK, got %#v", ack)
	}
	if !strings.Contains(ack.Message, "handler exploded") {
		t.Fatalf("unexpected handler failure ACK message: %q", ack.Message)
	}
}

func TestConnectReturnsErrorACKWhenResetFails(t *testing.T) {
	t.Parallel()

	fakeStream := &fakeDingTalkStreamClient{
		started: make(chan struct{}),
		done:    make(chan struct{}),
	}
	replyLog := &fakeReplyLog{}
	routes := &fakeDingTalkRouteService{
		route:     routepkg.Route{ID: "route-reset-failure"},
		deleteErr: errors.New("delete failed"),
	}
	adapter := NewDingTalkAdapter(nil)
	adapter.newStreamClient = func(_ Config, handler streamEventHandler) (streamClient, error) {
		fakeStream.handler = handler
		return fakeStream, nil
	}
	adapter.newClient = func(_ Config) *client {
		return &client{replyTextFunc: replyLog.replyText}
	}
	adapter.SetSessionCommandHandler(NewRouteResetHandler(routes))

	conn, err := adapter.Connect(context.Background(), channel.ChannelConfig{
		ID:          "cfg-reset-failure",
		BotID:       "bot-reset-failure",
		ChannelType: Type,
		Credentials: map[string]any{
			"appKey":                "app-key",
			"appSecret":             "app-secret",
			"enableSessionCommands": true,
			"groupSessionScope":     groupSessionScopeGroupSender,
		},
	}, func(context.Context, channel.ChannelConfig, channel.InboundMessage) error {
		return nil
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() {
		_ = conn.Stop(context.Background())
	}()

	select {
	case <-fakeStream.started:
	case <-time.After(time.Second):
		t.Fatal("receiver did not start")
	}

	ack := fakeStream.emit(t, StreamCallbackEnvelope{
		Headers: StreamCallbackHeaders{
			Topic: streamTopicChatbotMessage,
		},
		Data: mustJSON(t, InboundMessageEnvelope{
			ConversationID:   "cid-reset-failure",
			ConversationType: "2",
			SenderUnionID:    "union-reset-failure",
			SessionWebhook:   "https://example.com/reset-session",
			MsgID:            "msg-reset-failure",
			MsgType:          "text",
			IsInAtList:       true,
			Text: InboundTextBlock{
				Content: "/new",
			},
		}),
	})
	if ack.Success {
		t.Fatalf("expected reset failure to return error ACK, got %#v", ack)
	}
	if !strings.Contains(ack.Message, "delete failed") {
		t.Fatalf("unexpected reset failure ACK message: %q", ack.Message)
	}
	if len(replyLog.messages) != 0 {
		t.Fatalf("reset failure should not send confirmation reply, got %#v", replyLog.messages)
	}
}

func TestConnectInterceptsResetCommands(t *testing.T) {
	t.Parallel()

	fakeStream := &fakeDingTalkStreamClient{
		started: make(chan struct{}),
		done:    make(chan struct{}),
	}
	replyLog := &fakeReplyLog{}
	routes := &fakeDingTalkRouteService{
		route: routepkg.Route{ID: "route-1"},
	}
	adapter := NewDingTalkAdapter(nil)
	adapter.newStreamClient = func(_ Config, handler streamEventHandler) (streamClient, error) {
		fakeStream.handler = handler
		return fakeStream, nil
	}
	adapter.newClient = func(_ Config) *client {
		return &client{replyTextFunc: replyLog.replyText}
	}
	adapter.SetSessionCommandHandler(NewRouteResetHandler(routes))

	handlerCalled := make(chan struct{}, 1)
	conn, err := adapter.Connect(context.Background(), channel.ChannelConfig{
		ID:          "cfg-command",
		BotID:       "bot-command",
		ChannelType: Type,
		Credentials: map[string]any{
			"appKey":                "app-key",
			"appSecret":             "app-secret",
			"enableSessionCommands": true,
			"groupSessionScope":     groupSessionScopeGroupSender,
		},
	}, func(context.Context, channel.ChannelConfig, channel.InboundMessage) error {
		handlerCalled <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() {
		_ = conn.Stop(context.Background())
	}()

	select {
	case <-fakeStream.started:
	case <-time.After(time.Second):
		t.Fatal("receiver did not start")
	}

	ack := fakeStream.emit(t, StreamCallbackEnvelope{
		Headers: StreamCallbackHeaders{
			Topic: streamTopicChatbotMessage,
		},
		Data: mustJSON(t, InboundMessageEnvelope{
			ConversationID:   "cid-command",
			ConversationType: "2",
			SenderUnionID:    "union-1",
			SessionWebhook:   "https://example.com/command-session",
			MsgID:            "msg-command",
			MsgType:          "text",
			IsInAtList:       true,
			Text: InboundTextBlock{
				Content: "/new",
			},
		}),
	})
	if !ack.Success {
		t.Fatalf("expected command event to be ACKed, got %#v", ack)
	}

	select {
	case <-handlerCalled:
		t.Fatal("session reset commands should not reach the inbound handler")
	case <-time.After(100 * time.Millisecond):
	}

	if len(routes.findCalls) != 1 {
		t.Fatalf("expected one route lookup, got %d", len(routes.findCalls))
	}
	if routes.findCalls[0].threadID != "union-1" {
		t.Fatalf("unexpected reset thread id: %s", routes.findCalls[0].threadID)
	}
	if len(routes.deleteCalls) != 1 || routes.deleteCalls[0] != "route-1" {
		t.Fatalf("unexpected route deletions: %#v", routes.deleteCalls)
	}
	if len(replyLog.messages) != 1 || replyLog.messages[0] != "Started a new session." {
		t.Fatalf("unexpected reply log: %#v", replyLog.messages)
	}
	if len(replyLog.targets) != 1 || replyLog.targets[0] != "https://example.com/command-session" {
		t.Fatalf("unexpected reply targets: %#v", replyLog.targets)
	}
}

func TestRouteResetHandlerIgnoresMissingRoute(t *testing.T) {
	t.Parallel()

	routes := &fakeDingTalkRouteService{findErr: pgx.ErrNoRows}
	handler := NewRouteResetHandler(routes)

	reply, err := handler.HandleReset(context.Background(), channel.ChannelConfig{
		BotID: "bot-reset",
	}, channel.InboundMessage{
		Channel: Type,
		Conversation: channel.Conversation{
			ID:       "cid-reset",
			ThreadID: "staff-reset",
		},
	}, SessionCommand{Name: sessionCommandReset})
	if err != nil {
		t.Fatalf("handle reset: %v", err)
	}
	if reply != defaultSessionResetConfirmation {
		t.Fatalf("unexpected reply: %q", reply)
	}
	if len(routes.deleteCalls) != 0 {
		t.Fatalf("unexpected route deletions: %#v", routes.deleteCalls)
	}
	if len(routes.findCalls) != 1 || routes.findCalls[0].threadID != "staff-reset" {
		t.Fatalf("unexpected route lookup: %#v", routes.findCalls)
	}
}

func TestConnectionStopClosesReceiverGracefully(t *testing.T) {
	t.Parallel()

	fakeStream := &fakeDingTalkStreamClient{
		started: make(chan struct{}),
		done:    make(chan struct{}),
	}
	adapter := NewDingTalkAdapter(nil)
	adapter.newStreamClient = func(_ Config, handler streamEventHandler) (streamClient, error) {
		fakeStream.handler = handler
		return fakeStream, nil
	}

	conn, err := adapter.Connect(context.Background(), channel.ChannelConfig{
		ID:          "cfg-stop",
		BotID:       "bot-stop",
		ChannelType: Type,
		Credentials: map[string]any{
			"appKey":    "app-key",
			"appSecret": "app-secret",
		},
	}, func(context.Context, channel.ChannelConfig, channel.InboundMessage) error {
		return nil
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	select {
	case <-fakeStream.started:
	case <-time.After(time.Second):
		t.Fatal("receiver did not start")
	}

	if err := conn.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if conn.Running() {
		t.Fatal("connection should not be running after stop")
	}
	select {
	case <-fakeStream.done:
	case <-time.After(time.Second):
		t.Fatal("receiver did not stop")
	}
}

func TestNewSDKStreamClientDisablesAutoReconnect(t *testing.T) {
	t.Parallel()

	client, err := newSDKStreamClient(Config{
		AppKey:    "app-key",
		AppSecret: "app-secret",
	}, func(context.Context, StreamCallbackEnvelope) streamAck {
		return streamAck{Success: true}
	})
	if err != nil {
		t.Fatalf("new sdk stream client: %v", err)
	}

	sdkClient, ok := client.(*sdkStreamClient)
	if !ok {
		t.Fatalf("unexpected stream client type: %T", client)
	}
	if sdkClient.client.AutoReconnect {
		t.Fatal("sdk stream client should disable auto reconnect so Stop fully stops the receiver")
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return data
}

type fakeDingTalkStreamClient struct {
	handler streamEventHandler
	started chan struct{}
	done    chan struct{}
	once    sync.Once
}

func (f *fakeDingTalkStreamClient) Start(ctx context.Context) error {
	close(f.started)
	<-ctx.Done()
	return nil
}

func (f *fakeDingTalkStreamClient) Close() error {
	f.once.Do(func() {
		close(f.done)
	})
	return nil
}

func (f *fakeDingTalkStreamClient) Started() <-chan struct{} {
	return f.started
}

func (f *fakeDingTalkStreamClient) emit(t *testing.T, envelope StreamCallbackEnvelope) streamAck {
	t.Helper()

	if f.handler == nil {
		t.Fatal("stream handler not configured")
	}
	return f.handler(context.Background(), envelope)
}

type fakeReplyLog struct {
	mu       sync.Mutex
	targets  []string
	messages []string
	err      error
}

func (f *fakeReplyLog) replyText(_ context.Context, sessionWebhook, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.targets = append(f.targets, sessionWebhook)
	f.messages = append(f.messages, text)
	return f.err
}

type fakeDingTalkRouteFindCall struct {
	botID          string
	platform       string
	conversationID string
	threadID       string
}

type fakeDingTalkRouteService struct {
	mu          sync.Mutex
	findCalls   []fakeDingTalkRouteFindCall
	deleteCalls []string
	route       routepkg.Route
	findErr     error
	deleteErr   error
}

func (f *fakeDingTalkRouteService) Find(_ context.Context, botID, platform, conversationID, threadID string) (routepkg.Route, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.findCalls = append(f.findCalls, fakeDingTalkRouteFindCall{
		botID:          botID,
		platform:       platform,
		conversationID: conversationID,
		threadID:       threadID,
	})
	if f.findErr != nil {
		return routepkg.Route{}, f.findErr
	}
	return f.route, nil
}

func (f *fakeDingTalkRouteService) Delete(_ context.Context, routeID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.deleteCalls = append(f.deleteCalls, routeID)
	return f.deleteErr
}
