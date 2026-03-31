package dingtalk

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/memohai/memoh/internal/channel"
)

type InboundEventResult struct {
	Message channel.InboundMessage
	Command *SessionCommand
}

func streamEnvelopeToInboundEvent(cfg Config, botID string, envelope StreamCallbackEnvelope) (InboundEventResult, bool, error) {
	if strings.TrimSpace(envelope.Headers.Topic) != streamTopicChatbotMessage {
		return InboundEventResult{}, false, nil
	}

	var event InboundMessageEnvelope
	if err := decodeStreamEnvelopeData(envelope.Data, &event); err != nil {
		return InboundEventResult{}, false, err
	}

	result, ok := eventToInboundMessage(cfg, botID, event)
	return result, ok, nil
}

func eventToInboundMessage(cfg Config, botID string, event InboundMessageEnvelope) (InboundEventResult, bool) {
	conversationID := strings.TrimSpace(event.ConversationID)
	conversationType := normalizeConversationType(event.ConversationType)
	senderSubjectID := stableSenderSubjectID(event)

	if conversationID == "" || conversationType == "" || senderSubjectID == "" {
		return InboundEventResult{}, false
	}
	if conversationType == "group" && cfg.RequireMentionInGroup && !event.IsInAtList {
		return InboundEventResult{}, false
	}

	text, attachments, ok := parseInboundContent(event)
	if !ok {
		return InboundEventResult{}, false
	}

	replyTarget := normalizeInboundReplyTarget(event.SessionWebhook)
	routeKey := buildRouteKey(cfg, strings.TrimSpace(botID), conversationID, conversationType, senderSubjectID)
	msg := channel.InboundMessage{
		Channel: Type,
		BotID:   strings.TrimSpace(botID),
		Message: channel.Message{
			ID:          strings.TrimSpace(event.MsgID),
			Format:      channel.MessageFormatPlain,
			Text:        text,
			Attachments: attachments,
		},
		ReplyTarget: replyTarget,
		RouteKey:    routeKey,
		Sender: channel.Identity{
			SubjectID:   senderSubjectID,
			DisplayName: strings.TrimSpace(event.SenderNick),
			Attributes: map[string]string{
				"sender_id":       strings.TrimSpace(event.SenderID),
				"sender_staff_id": strings.TrimSpace(event.SenderStaffID),
				"sender_union_id": strings.TrimSpace(event.SenderUnionID),
			},
		},
		Conversation: channel.Conversation{
			ID:       conversationID,
			Type:     conversationType,
			ThreadID: buildInboundThreadID(cfg, conversationType, senderSubjectID),
		},
		ReceivedAt: parseInboundTime(event.CreateAt),
		Source:     string(Type),
		Metadata: map[string]any{
			"session_webhook":              strings.TrimSpace(event.SessionWebhook),
			"session_webhook_expired_time": event.SessionWebhookExpiredTime,
			"conversation_type":            conversationType,
			"sender_staff_id":              strings.TrimSpace(event.SenderStaffID),
			"sender_union_id":              strings.TrimSpace(event.SenderUnionID),
			"robot_code":                   strings.TrimSpace(event.RobotCode),
			"chatbot_user_id":              strings.TrimSpace(event.ChatbotUserID),
			"card_supported":               inboundCardSupported(event.CardSupported),
			"is_mentioned":                 conversationType == "group" && event.IsInAtList,
			"at_users":                     normalizeAtUsers(event.AtUsers),
			"raw_msgtype":                  strings.TrimSpace(event.MsgType),
		},
	}

	result := InboundEventResult{Message: msg}
	if command := detectSessionCommand(cfg, text, routeKey, replyTarget); command != nil {
		result.Command = command
		result.Message.Metadata["session_command"] = command.Name
		result.Message.Metadata["session_command_normalized"] = command.Normalized
		result.Message.Metadata["reset_current_route"] = command.ResetCurrentRoute
		result.Message.Metadata["skip_llm_processing"] = true
	}

	return result, true
}

func parseInboundContent(event InboundMessageEnvelope) (string, []channel.Attachment, bool) {
	switch strings.ToLower(strings.TrimSpace(event.MsgType)) {
	case "text":
		text := strings.TrimSpace(event.Text.Content)
		if text == "" {
			return "", nil, false
		}
		return text, nil, true
	case "image":
		att, ok := inboundAttachmentFromMedia(channel.AttachmentImage, event.Image)
		if !ok {
			return "", nil, false
		}
		return "", []channel.Attachment{att}, true
	case "file":
		att, ok := inboundAttachmentFromMedia(channel.AttachmentFile, event.File)
		if !ok {
			return "", nil, false
		}
		return "", []channel.Attachment{att}, true
	case "audio":
		att, ok := inboundAttachmentFromAudio(channel.AttachmentAudio, event.Audio)
		if !ok {
			return "", nil, false
		}
		return "", []channel.Attachment{att}, true
	case "voice":
		att, ok := inboundAttachmentFromAudio(channel.AttachmentVoice, event.Audio)
		if !ok {
			return "", nil, false
		}
		return "", []channel.Attachment{att}, true
	default:
		return "", nil, false
	}
}

func decodeStreamEnvelopeData(raw json.RawMessage, target any) error {
	data := bytes.TrimSpace(raw)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return errors.New("dingtalk stream event data is empty")
	}
	if data[0] == '"' {
		var encoded string
		if err := json.Unmarshal(data, &encoded); err != nil {
			return err
		}
		data = []byte(encoded)
	}
	return json.Unmarshal(data, target)
}

func normalizeConversationType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "single", "private", "direct":
		return "private"
	case "2", "group":
		return "group"
	default:
		return ""
	}
}

func stableSenderSubjectID(event InboundMessageEnvelope) string {
	for _, candidate := range []string{
		event.SenderStaffID,
		event.SenderUnionID,
		event.SenderID,
	} {
		if value := strings.TrimSpace(candidate); value != "" {
			return value
		}
	}
	return ""
}

func buildRouteKey(cfg Config, botID, conversationID, conversationType, senderSubjectID string) string {
	if conversationType == "group" && strings.TrimSpace(cfg.GroupSessionScope) == groupSessionScopeGroupSender {
		return channel.GenerateRoutingKey(string(Type), botID, conversationID, conversationType, senderSubjectID)
	}
	return channel.GenerateRoutingKey(string(Type), botID, conversationID, "private", "")
}

func buildInboundThreadID(cfg Config, conversationType, senderSubjectID string) string {
	if conversationType != "group" {
		return ""
	}
	if strings.TrimSpace(cfg.GroupSessionScope) != groupSessionScopeGroupSender {
		return ""
	}
	return strings.TrimSpace(senderSubjectID)
}

func normalizeInboundReplyTarget(sessionWebhook string) string {
	value := strings.TrimSpace(sessionWebhook)
	if value == "" {
		return ""
	}
	return targetFamilyWebhook + ":" + value
}

func normalizeAtUsers(users []InboundAtUser) []map[string]string {
	if len(users) == 0 {
		return nil
	}
	result := make([]map[string]string, 0, len(users))
	for _, user := range users {
		entry := map[string]string{}
		if value := strings.TrimSpace(user.DingtalkID); value != "" {
			entry["dingtalkId"] = value
		}
		if value := strings.TrimSpace(user.StaffID); value != "" {
			entry["staffId"] = value
		}
		if value := strings.TrimSpace(user.UnionID); value != "" {
			entry["unionId"] = value
		}
		if len(entry) > 0 {
			result = append(result, entry)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func inboundAttachmentFromMedia(attType channel.AttachmentType, media *InboundMediaBlock) (channel.Attachment, bool) {
	if media == nil {
		return channel.Attachment{}, false
	}
	platformKey := strings.TrimSpace(media.DownloadCode)
	if platformKey == "" {
		platformKey = strings.TrimSpace(media.MediaID)
	}
	url := strings.TrimSpace(media.URL)
	if platformKey == "" && url == "" {
		return channel.Attachment{}, false
	}
	return channel.NormalizeInboundChannelAttachment(channel.Attachment{
		Type:           attType,
		URL:            url,
		PlatformKey:    platformKey,
		SourcePlatform: string(Type),
		Name:           strings.TrimSpace(media.FileName),
		Size:           media.Size,
		Mime:           strings.TrimSpace(media.MimeType),
		Metadata: map[string]any{
			"download_code": strings.TrimSpace(media.DownloadCode),
			"media_id":      strings.TrimSpace(media.MediaID),
		},
	}), true
}

func inboundAttachmentFromAudio(attType channel.AttachmentType, audio *InboundAudioBlock) (channel.Attachment, bool) {
	if audio == nil {
		return channel.Attachment{}, false
	}
	attachment, ok := inboundAttachmentFromMedia(attType, &audio.InboundMediaBlock)
	if !ok {
		return channel.Attachment{}, false
	}
	attachment.DurationMs = audio.DurationMs
	return attachment, true
}

func inboundCardSupported(value *bool) bool {
	return value != nil && *value
}

func parseInboundTime(raw string) time.Time {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts.UTC()
	}
	if unixMillis, err := strconv.ParseInt(value, 10, 64); err == nil {
		if len(value) > 10 {
			return time.UnixMilli(unixMillis).UTC()
		}
		return time.Unix(unixMillis, 0).UTC()
	}
	return time.Time{}
}
