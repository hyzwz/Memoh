package ctripcs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseInboxSnapshotBuildsInboundMessages(t *testing.T) {
	t.Parallel()

	raw := mustReadTestdata(t, "ctrip_inbox_snapshot.json")

	snapshot, messages, err := ParseInboxSnapshot(raw, Config{AccountLabel: "Ctrip Support A"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if snapshot.ConversationID != "ctrip-session-123" {
		t.Fatalf("unexpected conversation id: %q", snapshot.ConversationID)
	}
	if snapshot.PageURL != "https://m.ctrip.com/customer-service/inbox" {
		t.Fatalf("unexpected page url: %q", snapshot.PageURL)
	}
	if len(messages) != 2 {
		t.Fatalf("unexpected inbound message count: %d", len(messages))
	}

	first := messages[0]
	if first.Channel != Type {
		t.Fatalf("unexpected channel: %s", first.Channel)
	}
	if first.BotID != "" {
		t.Fatalf("unexpected bot id: %q", first.BotID)
	}
	if first.Message.ID != "msg-customer-1" {
		t.Fatalf("unexpected message id: %q", first.Message.ID)
	}
	if first.Conversation.ID != "ctrip-session-123" {
		t.Fatalf("unexpected conversation id: %q", first.Conversation.ID)
	}
	if first.Conversation.Type != "direct" {
		t.Fatalf("unexpected conversation type: %q", first.Conversation.Type)
	}
	if first.ReplyTarget != "session:ctrip-session-123" {
		t.Fatalf("unexpected reply target: %q", first.ReplyTarget)
	}
	if first.Sender.SubjectID != "customer-1001" {
		t.Fatalf("unexpected sender subject: %q", first.Sender.SubjectID)
	}
	if first.Sender.DisplayName != "Alice" {
		t.Fatalf("unexpected sender display name: %q", first.Sender.DisplayName)
	}
	if first.Message.Text != "I need help with my booking" {
		t.Fatalf("unexpected text: %q", first.Message.Text)
	}
	if got := first.Metadata["page_url"]; got != "https://m.ctrip.com/customer-service/inbox" {
		t.Fatalf("unexpected page_url metadata: %#v", got)
	}
	if got := first.Metadata["account_label"]; got != "Ctrip Support A" {
		t.Fatalf("unexpected account_label metadata: %#v", got)
	}
	if got := first.Metadata["raw_conversation_id"]; got != "ctrip-session-123" {
		t.Fatalf("unexpected raw_conversation_id metadata: %#v", got)
	}
	if got := first.Metadata["raw_message_id"]; got != "msg-customer-1" {
		t.Fatalf("unexpected raw_message_id metadata: %#v", got)
	}
	if got := first.Metadata["source_transport"]; got != "dom_poll" {
		t.Fatalf("unexpected source_transport metadata: %#v", got)
	}
}

func TestParseInboxSnapshotSkipsAgentAuthoredMessages(t *testing.T) {
	t.Parallel()

	raw := mustReadTestdata(t, "ctrip_inbox_snapshot.json")

	_, messages, err := ParseInboxSnapshot(raw, Config{AccountLabel: "Ctrip Support A"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	for _, msg := range messages {
		if msg.Sender.SubjectID == "agent-2001" {
			t.Fatalf("unexpected agent-authored message: %#v", msg)
		}
	}
}

func TestParseInboxSnapshotDetectsLoginExpired(t *testing.T) {
	t.Parallel()

	raw := mustReadTestdata(t, "ctrip_login_expired_snapshot.json")

	_, _, err := ParseInboxSnapshot(raw, Config{AccountLabel: "Ctrip Support A"})
	if !errors.Is(err, ErrLoginExpired) {
		t.Fatalf("expected ErrLoginExpired, got %v", err)
	}
}

func TestBuildReplyTargetUsesStableConversationHandle(t *testing.T) {
	t.Parallel()

	firstRaw := mustReadTestdata(t, "ctrip_reply_target_snapshot.json")
	secondRaw := mustReadTestdata(t, "ctrip_reply_target_snapshot.json")

	firstSnapshot, _, err := ParseInboxSnapshot(firstRaw, Config{AccountLabel: "Ctrip Support A"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	secondSnapshot, _, err := ParseInboxSnapshot(secondRaw, Config{AccountLabel: "Ctrip Support A"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	firstReply := BuildReplyTarget(firstSnapshot)
	secondReply := BuildReplyTarget(secondSnapshot)
	if firstReply != secondReply {
		t.Fatalf("expected stable reply target, got %q and %q", firstReply, secondReply)
	}
	if firstReply != "session:ctrip-session-456" {
		t.Fatalf("unexpected reply target: %q", firstReply)
	}
}

func mustReadTestdata(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("testdata", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
