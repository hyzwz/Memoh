package wecombot

import (
	"encoding/json"
	"testing"

	"github.com/memohai/memoh/internal/channel"
)

func TestBuildInboundMessageGroupMixed(t *testing.T) {
	t.Parallel()

	body, err := json.Marshal(inboundBody{
		MsgID:    "msg-1",
		AIBotID:  "bot-ext-1",
		ChatID:   "chat-group-1",
		ChatType: "group",
		From:     inboundFrom{UserID: "user-1"},
		MsgType:  "mixed",
		Mixed: &mixedBody{Items: []mixedItem{
			{MsgType: "text", Text: &textBody{Content: "@bot hello world"}},
			{MsgType: "image", Image: &mediaBody{URL: "https://example.com/a.png", AESKey: "aes-key"}},
			{MsgType: "file", File: &mediaBody{URL: "https://example.com/doc.md", AESKey: "aes-key-file"}},
		}},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	msg, ok, err := buildInboundMessage(channel.ChannelConfig{BotID: "bot-1"}, wsFrame{
		Cmd:     wsCmdCallback,
		Headers: wsHeaders{ReqID: "req-123"},
		Body:    body,
	})
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}
	if !ok {
		t.Fatal("expected inbound message")
	}
	if msg.ReplyTarget != "chatid:chat-group-1" {
		t.Fatalf("unexpected reply target: %s", msg.ReplyTarget)
	}
	if msg.Conversation.ID != "chat-group-1" || msg.Conversation.Type != "group" {
		t.Fatalf("unexpected conversation: %#v", msg.Conversation)
	}
	if msg.Message.Text != "hello world" {
		t.Fatalf("unexpected text: %q", msg.Message.Text)
	}
	if got := metadataString(msg.Metadata, "source_req_id"); got != "req-123" {
		t.Fatalf("unexpected source_req_id: %q", got)
	}
	if got := msg.Sender.Attribute("chatid"); got != "chat-group-1" {
		t.Fatalf("unexpected sender chatid: %q", got)
	}
	if len(msg.Message.Attachments) != 2 {
		t.Fatalf("unexpected attachment count: %d", len(msg.Message.Attachments))
	}
	if got := attachmentMetadataString(msg.Message.Attachments[0].Metadata, "aes_key"); got != "aes-key" {
		t.Fatalf("unexpected aes key: %q", got)
	}
	if msg.Message.Attachments[1].Type != channel.AttachmentFile {
		t.Fatalf("unexpected second attachment type: %q", msg.Message.Attachments[1].Type)
	}
	if got := attachmentMetadataString(msg.Message.Attachments[1].Metadata, "aes_key"); got != "aes-key-file" {
		t.Fatalf("unexpected file aes key: %q", got)
	}
	if mentioned, _ := msg.Metadata["is_mentioned"].(bool); !mentioned {
		t.Fatal("expected group mention marker")
	}
}

func TestBuildInboundMessageDirectPrefersChatIDReplyTarget(t *testing.T) {
	t.Parallel()

	body, err := json.Marshal(inboundBody{
		MsgID:    "msg-2",
		AIBotID:  "bot-ext-1",
		ChatID:   "chat-direct-1",
		ChatType: "single",
		From:     inboundFrom{UserID: "user-2"},
		MsgType:  "text",
		Text:     &textBody{Content: "hello"},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	msg, ok, err := buildInboundMessage(channel.ChannelConfig{BotID: "bot-1"}, wsFrame{
		Cmd:  wsCmdCallback,
		Body: body,
	})
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}
	if !ok {
		t.Fatal("expected inbound message")
	}
	if msg.ReplyTarget != "chatid:chat-direct-1" {
		t.Fatalf("unexpected reply target: %s", msg.ReplyTarget)
	}
	if msg.Conversation.ID != "chat-direct-1" || msg.Conversation.Type != "direct" {
		t.Fatalf("unexpected conversation: %#v", msg.Conversation)
	}
	if got := msg.Sender.Attribute("chat_type"); got != "direct" {
		t.Fatalf("unexpected chat_type: %q", got)
	}
}

func TestBuildInboundMessageAcceptsEmptyCmdWhenBodyHasMsgType(t *testing.T) {
	t.Parallel()

	body, err := json.Marshal(inboundBody{
		MsgID:    "msg-3",
		AIBotID:  "bot-ext-1",
		ChatType: "single",
		From:     inboundFrom{UserID: "user-3"},
		MsgType:  "text",
		Text:     &textBody{Content: "11111111"},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	msg, ok, err := buildInboundMessage(channel.ChannelConfig{BotID: "bot-1"}, wsFrame{
		Headers: wsHeaders{ReqID: "req-345"},
		Body:    body,
	})
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}
	if !ok {
		t.Fatal("expected inbound message")
	}
	if msg.Message.Text != "11111111" {
		t.Fatalf("unexpected text: %q", msg.Message.Text)
	}
	if msg.Sender.SubjectID != "user-3" {
		t.Fatalf("unexpected sender subject: %q", msg.Sender.SubjectID)
	}
	if got := metadataString(msg.Metadata, "source_req_id"); got != "req-345" {
		t.Fatalf("unexpected source_req_id: %q", got)
	}
}

func TestBuildInboundMessageIgnoresEventMsgTypeWithoutCmd(t *testing.T) {
	t.Parallel()

	body := []byte(`{"msgid":"evt-1","msgtype":"event","event":{"eventtype":"enter_chat"}}`)
	_, ok, err := buildInboundMessage(channel.ChannelConfig{BotID: "bot-1"}, wsFrame{Body: body})
	if err != nil {
		t.Fatalf("build inbound: %v", err)
	}
	if ok {
		t.Fatal("expected event body to be ignored")
	}
}
