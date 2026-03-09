package wecombot

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/memohai/memoh/internal/channel"
)

const (
	wsCmdSubscribe     = "aibot_subscribe"
	wsCmdHeartbeat     = "ping"
	wsCmdReply         = "aibot_respond_msg"
	wsCmdSendMessage   = "aibot_send_msg"
	wsCmdCallback      = "aibot_msg_callback"
	wsCmdEventCallback = "aibot_event_callback"

	defaultHeartbeatInterval = 30 * time.Second
	replyAckTimeout          = 5 * time.Second
	heartbeatAckTimeout      = 10 * time.Second
	connectTimeout           = 10 * time.Second
	readTimeout              = 65 * time.Second
	writeTimeout             = 15 * time.Second
	reconnectMaxDelay        = 30 * time.Second

	thinkingPlaceholder = "<think></think>"
)

type wsFrame struct {
	Cmd     string          `json:"cmd,omitempty"`
	Headers wsHeaders       `json:"headers"`
	Body    json.RawMessage `json:"body,omitempty"`
	ErrCode int             `json:"errcode,omitempty"`
	ErrMsg  string          `json:"errmsg,omitempty"`
	Extra   map[string]any  `json:"-"`
}

type wsHeaders struct {
	ReqID string `json:"req_id"`
}

type outboundFrame struct {
	Cmd     string         `json:"cmd,omitempty"`
	Headers wsHeaders      `json:"headers"`
	Body    map[string]any `json:"body,omitempty"`
}

type inboundBody struct {
	MsgID       string      `json:"msgid"`
	AIBotID     string      `json:"aibotid"`
	ChatID      string      `json:"chatid,omitempty"`
	ChatType    string      `json:"chattype"`
	From        inboundFrom `json:"from"`
	CreateTime  int64       `json:"create_time,omitempty"`
	ResponseURL string      `json:"response_url,omitempty"`
	MsgType     string      `json:"msgtype"`
	Text        *textBody   `json:"text,omitempty"`
	Image       *mediaBody  `json:"image,omitempty"`
	Voice       *voiceBody  `json:"voice,omitempty"`
	Mixed       *mixedBody  `json:"mixed,omitempty"`
	File        *mediaBody  `json:"file,omitempty"`
	Quote       *quoteBody  `json:"quote,omitempty"`
}

type inboundEnvelope struct {
	MsgType string `json:"msgtype"`
}

type inboundFrom struct {
	UserID string `json:"userid"`
}

type textBody struct {
	Content string `json:"content"`
}

type voiceBody struct {
	Content string `json:"content"`
}

type mediaBody struct {
	URL    string `json:"url"`
	AESKey string `json:"aeskey,omitempty"`
}

type mixedBody struct {
	Items []mixedItem `json:"msg_item"`
}

type mixedItem struct {
	MsgType string     `json:"msgtype"`
	Text    *textBody  `json:"text,omitempty"`
	Image   *mediaBody `json:"image,omitempty"`
	File    *mediaBody `json:"file,omitempty"`
}

type quoteBody struct {
	MsgType string     `json:"msgtype"`
	Text    *textBody  `json:"text,omitempty"`
	Image   *mediaBody `json:"image,omitempty"`
	Mixed   *mixedBody `json:"mixed,omitempty"`
	Voice   *voiceBody `json:"voice,omitempty"`
	File    *mediaBody `json:"file,omitempty"`
}

type targetType string

const (
	targetUserID targetType = "userid"
	targetChatID targetType = "chatid"
)

func generateReqID(prefix string) string {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%d_%x", prefix, time.Now().UnixMilli(), buf[:])
}

func buildStreamReplyBody(streamID, content string, finish bool) map[string]any {
	body := map[string]any{
		"msgtype": "stream",
		"stream": map[string]any{
			"id":      streamID,
			"finish":  finish,
			"content": content,
		},
	}
	return body
}

func buildSendMarkdownBody(chatID, content string) map[string]any {
	return map[string]any{
		"chatid":  chatID,
		"msgtype": "markdown",
		"markdown": map[string]any{
			"content": content,
		},
	}
}

func buildInboundMessage(cfg channel.ChannelConfig, frame wsFrame) (channel.InboundMessage, bool, error) {
	envelope, ok, err := parseInboundEnvelope(frame)
	if err != nil {
		return channel.InboundMessage{}, false, err
	}
	if !ok {
		return channel.InboundMessage{}, false, nil
	}
	if frame.Cmd == wsCmdEventCallback || strings.EqualFold(strings.TrimSpace(envelope.MsgType), "event") {
		return channel.InboundMessage{}, false, nil
	}
	var body inboundBody
	if err := json.Unmarshal(frame.Body, &body); err != nil {
		return channel.InboundMessage{}, false, fmt.Errorf("decode wecom ai bot frame: %w", err)
	}
	text, attachments, mentioned := parseInboundContent(body)
	if strings.TrimSpace(text) == "" && len(attachments) == 0 {
		return channel.InboundMessage{}, false, nil
	}
	userID := strings.TrimSpace(body.From.UserID)
	chatID := strings.TrimSpace(body.ChatID)
	conversationType := "group"
	if strings.EqualFold(strings.TrimSpace(body.ChatType), "single") {
		conversationType = "direct"
	}
	conversationID := chatID
	if conversationID == "" {
		conversationID = userID
	}
	replyTarget := "userid:" + userID
	if chatID != "" {
		replyTarget = "chatid:" + chatID
	}
	attrs := map[string]string{
		"userid":    userID,
		"user_id":   userID,
		"chat_type": conversationType,
	}
	if chatID != "" {
		attrs["chatid"] = chatID
	}
	msg := channel.InboundMessage{
		Channel: Type,
		Message: channel.Message{
			ID:          strings.TrimSpace(body.MsgID),
			Format:      channel.MessageFormatPlain,
			Text:        strings.TrimSpace(text),
			Attachments: attachments,
		},
		BotID:       cfg.BotID,
		ReplyTarget: replyTarget,
		Sender: channel.Identity{
			SubjectID:   userID,
			DisplayName: userID,
			Attributes:  attrs,
		},
		Conversation: channel.Conversation{
			ID:   conversationID,
			Type: conversationType,
			Metadata: map[string]any{
				"aibotid": strings.TrimSpace(body.AIBotID),
				"chatid":  chatID,
			},
		},
		ReceivedAt: receivedAt(body.CreateTime),
		Source:     string(Type),
		Metadata: map[string]any{
			"source_req_id": strings.TrimSpace(frame.Headers.ReqID),
			"aibotid":       strings.TrimSpace(body.AIBotID),
			"response_url":  strings.TrimSpace(body.ResponseURL),
		},
	}
	if conversationType == "group" && mentioned {
		msg.Metadata["is_mentioned"] = true
	}
	return msg, true, nil
}

func parseInboundEnvelope(frame wsFrame) (inboundEnvelope, bool, error) {
	if len(frame.Body) == 0 {
		return inboundEnvelope{}, false, nil
	}
	var envelope inboundEnvelope
	if err := json.Unmarshal(frame.Body, &envelope); err != nil {
		return inboundEnvelope{}, false, fmt.Errorf("decode wecom ai bot envelope: %w", err)
	}
	if strings.TrimSpace(envelope.MsgType) == "" {
		return inboundEnvelope{}, false, nil
	}
	return envelope, true, nil
}

func receivedAt(ts int64) time.Time {
	if ts <= 0 {
		return time.Now().UTC()
	}
	return time.Unix(ts, 0).UTC()
}

func parseInboundContent(body inboundBody) (string, []channel.Attachment, bool) {
	var (
		textParts   []string
		attachments []channel.Attachment
		isMentioned bool
	)
	appendMedia := func(kind channel.AttachmentType, media *mediaBody) {
		if media == nil || strings.TrimSpace(media.URL) == "" {
			return
		}
		attachments = append(attachments, newEncryptedAttachment(kind, media.URL, media.AESKey))
	}
	switch strings.TrimSpace(body.MsgType) {
	case "mixed":
		if body.Mixed != nil {
			for _, item := range body.Mixed.Items {
				switch strings.TrimSpace(item.MsgType) {
				case "text":
					if item.Text != nil && strings.TrimSpace(item.Text.Content) != "" {
						textParts = append(textParts, strings.TrimSpace(item.Text.Content))
					}
				case "image":
					appendMedia(channel.AttachmentImage, item.Image)
				case "file":
					appendMedia(channel.AttachmentFile, item.File)
				}
			}
		}
	case "voice":
		if body.Voice != nil && strings.TrimSpace(body.Voice.Content) != "" {
			textParts = append(textParts, strings.TrimSpace(body.Voice.Content))
		}
	case "image":
		appendMedia(channel.AttachmentImage, body.Image)
	case "file":
		appendMedia(channel.AttachmentFile, body.File)
	default:
		if body.Text != nil && strings.TrimSpace(body.Text.Content) != "" {
			textParts = append(textParts, strings.TrimSpace(body.Text.Content))
		}
	}
	if body.Quote != nil {
		switch strings.TrimSpace(body.Quote.MsgType) {
		case "text":
			if body.Quote.Text != nil && strings.TrimSpace(body.Quote.Text.Content) != "" && len(textParts) == 0 {
				textParts = append(textParts, strings.TrimSpace(body.Quote.Text.Content))
			}
		case "voice":
			if body.Quote.Voice != nil && strings.TrimSpace(body.Quote.Voice.Content) != "" && len(textParts) == 0 {
				textParts = append(textParts, strings.TrimSpace(body.Quote.Voice.Content))
			}
		case "image":
			appendMedia(channel.AttachmentImage, body.Quote.Image)
		case "file":
			appendMedia(channel.AttachmentFile, body.Quote.File)
		case "mixed":
			if body.Quote.Mixed != nil {
				for _, item := range body.Quote.Mixed.Items {
					switch strings.TrimSpace(item.MsgType) {
					case "image":
						appendMedia(channel.AttachmentImage, item.Image)
					case "file":
						appendMedia(channel.AttachmentFile, item.File)
					}
				}
			}
		}
	}
	text := strings.Join(textParts, "\n")
	if strings.EqualFold(strings.TrimSpace(body.ChatType), "group") {
		text, isMentioned = stripGroupMentions(text)
	}
	return strings.TrimSpace(text), attachments, isMentioned
}

func stripGroupMentions(text string) (string, bool) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return strings.TrimSpace(text), false
	}
	filtered := make([]string, 0, len(fields))
	mentioned := false
	for _, field := range fields {
		if strings.HasPrefix(field, "@") {
			mentioned = true
			continue
		}
		filtered = append(filtered, field)
	}
	if len(filtered) == 0 {
		return "", mentioned
	}
	return strings.TrimSpace(strings.Join(filtered, " ")), mentioned
}

func newEncryptedAttachment(kind channel.AttachmentType, rawURL, aesKey string) channel.Attachment {
	metadata := map[string]any{}
	if strings.TrimSpace(aesKey) != "" {
		metadata["aes_key"] = strings.TrimSpace(aesKey)
	}
	return channel.Attachment{
		Type:           kind,
		PlatformKey:    strings.TrimSpace(rawURL),
		SourcePlatform: Type.String(),
		Metadata:       metadata,
	}
}
