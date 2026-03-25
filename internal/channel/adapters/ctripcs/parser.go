package ctripcs

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/memohai/memoh/internal/channel"
)

var ErrLoginExpired = errors.New("ctrip_cs login expired")

const sourceTransportDOMPoll = "dom_poll"

type InboxSnapshot struct {
	PageURL          string            `json:"page_url"`
	AccountLabel     string            `json:"account_label"`
	ConversationID   string            `json:"conversation_id"`
	ConversationType string            `json:"conversation_type,omitempty"`
	LoginState       string            `json:"login_state,omitempty"`
	LoginExpired     bool              `json:"login_expired,omitempty"`
	Messages         []SnapshotMessage `json:"messages,omitempty"`
}

type SnapshotMessage struct {
	MessageID  string `json:"message_id"`
	AuthorID   string `json:"author_id"`
	AuthorName string `json:"author_name,omitempty"`
	AuthorRole string `json:"author_role,omitempty"`
	Text       string `json:"text,omitempty"`
	Timestamp  string `json:"timestamp,omitempty"`
}

func ParseInboxSnapshot(raw []byte, cfg Config) (InboxSnapshot, []channel.InboundMessage, error) {
	var snapshot InboxSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return InboxSnapshot{}, nil, fmt.Errorf("decode ctrip snapshot: %w", err)
	}

	snapshot.PageURL = strings.TrimSpace(snapshot.PageURL)
	snapshot.AccountLabel = normalizeSnapshotValue(snapshot.AccountLabel, cfg.AccountLabel)
	snapshot.ConversationID = strings.TrimSpace(snapshot.ConversationID)
	snapshot.ConversationType = normalizeConversationType(snapshot.ConversationType)
	snapshot.LoginState = strings.ToLower(strings.TrimSpace(snapshot.LoginState))

	if snapshot.LoginExpired || snapshot.LoginState == "expired" || snapshot.LoginState == "login_expired" {
		return snapshot, nil, ErrLoginExpired
	}
	if snapshot.ConversationID == "" {
		return InboxSnapshot{}, nil, errors.New("ctrip_cs conversation_id is required")
	}

	replyTarget := BuildReplyTarget(snapshot)
	messages := make([]channel.InboundMessage, 0, len(snapshot.Messages))
	for _, msg := range snapshot.Messages {
		if !isCustomerAuthoredMessage(msg) {
			continue
		}

		rawMessageID := strings.TrimSpace(msg.MessageID)
		messageID := rawMessageID
		if messageID == "" {
			messageID = stableMessageID(snapshot.ConversationID, msg)
		}
		senderID := strings.TrimSpace(msg.AuthorID)
		if senderID == "" {
			senderID = messageID
		}

		receivedAt, _ := parseSnapshotTime(msg.Timestamp)
		metadata := map[string]any{
			"page_url":            snapshot.PageURL,
			"account_label":       snapshot.AccountLabel,
			"raw_conversation_id": snapshot.ConversationID,
			"source_transport":    sourceTransportDOMPoll,
			"message_timestamp":   strings.TrimSpace(msg.Timestamp),
			"message_author_role": strings.TrimSpace(msg.AuthorRole),
			"message_author_name": strings.TrimSpace(msg.AuthorName),
			"conversation_type":   snapshot.ConversationType,
		}
		if rawMessageID != "" {
			metadata["raw_message_id"] = rawMessageID
		}

		messages = append(messages, channel.InboundMessage{
			Channel: Type,
			Message: channel.Message{
				ID:   messageID,
				Text: normalizeSnapshotText(msg.Text),
				Reply: &channel.ReplyRef{
					Target:    replyTarget,
					MessageID: messageID,
				},
				Metadata: metadata,
			},
			ReplyTarget: replyTarget,
			Sender: channel.Identity{
				SubjectID:   senderID,
				DisplayName: strings.TrimSpace(msg.AuthorName),
				Attributes: map[string]string{
					"author_role": strings.TrimSpace(msg.AuthorRole),
				},
			},
			Conversation: channel.Conversation{
				ID:   snapshot.ConversationID,
				Type: snapshot.ConversationType,
				Metadata: map[string]any{
					"page_url":            snapshot.PageURL,
					"account_label":       snapshot.AccountLabel,
					"raw_conversation_id": snapshot.ConversationID,
				},
			},
			ReceivedAt: receivedAt,
			Source:     sourceTransportDOMPoll,
			Metadata: map[string]any{
				"page_url":            snapshot.PageURL,
				"account_label":       snapshot.AccountLabel,
				"raw_conversation_id": snapshot.ConversationID,
				"source_transport":    sourceTransportDOMPoll,
			},
		})
		if rawMessageID != "" {
			messages[len(messages)-1].Metadata["raw_message_id"] = rawMessageID
			messages[len(messages)-1].Message.Metadata["raw_message_id"] = rawMessageID
		}
	}

	return snapshot, messages, nil
}

func BuildReplyTarget(snapshot InboxSnapshot) string {
	handle := strings.TrimSpace(snapshot.ConversationID)
	if handle == "" {
		handle = strings.TrimSpace(snapshot.PageURL)
	}
	if handle == "" {
		return ""
	}
	return "session:" + handle
}

func isCustomerAuthoredMessage(msg SnapshotMessage) bool {
	role := strings.ToLower(strings.TrimSpace(msg.AuthorRole))
	switch role {
	case "customer", "user", "visitor", "client", "guest":
		return true
	default:
		return false
	}
}

func normalizeConversationType(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "direct"
	}
	return value
}

func normalizeSnapshotValue(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

func normalizeSnapshotText(raw string) string {
	return strings.TrimSpace(raw)
}

func parseSnapshotTime(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, errors.New("empty timestamp")
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func stableMessageID(conversationID string, msg SnapshotMessage) string {
	parts := []string{
		strings.TrimSpace(conversationID),
		strings.TrimSpace(msg.AuthorID),
		strings.TrimSpace(msg.Timestamp),
		strings.TrimSpace(msg.Text),
	}
	return strings.Join(parts, ":")
}
