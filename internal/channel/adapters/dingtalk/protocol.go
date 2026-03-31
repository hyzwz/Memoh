package dingtalk

import "encoding/json"

// AccessTokenRequest models the DingTalk app credential exchange inputs without
// committing to a transport shape before the real implementation is added.
type AccessTokenRequest struct {
	AppKey    string `json:"appKey,omitempty"`
	AppSecret string `json:"appSecret,omitempty"`
}

// AccessTokenResponse models the token response fields used by later tasks.
type AccessTokenResponse struct {
	AccessToken string `json:"access_token,omitempty"`
	ExpiresIn   int64  `json:"expires_in,omitempty"`
}

// SelfIdentityRequest carries the access token for identity lookup.
type SelfIdentityRequest struct {
	AccessToken string `json:"accessToken,omitempty"`
}

// SelfIdentityResponse captures the stable identifiers returned by DingTalk.
type SelfIdentityResponse struct {
	RobotCode string `json:"robotCode,omitempty"`
	AppID     string `json:"appId,omitempty"`
	UserID    string `json:"userId,omitempty"`
	StaffID   string `json:"staffId,omitempty"`
	UnionID   string `json:"unionId,omitempty"`
}

// StreamCallbackEnvelope mirrors the DingTalk stream callback envelope shape.
// The data field is intentionally raw because DingTalk examples may encode it
// as JSON text or as a nested object depending on the event family.
type StreamCallbackEnvelope struct {
	Headers StreamCallbackHeaders `json:"headers,omitempty"`
	Data    json.RawMessage       `json:"data,omitempty"`
}

type StreamCallbackHeaders struct {
	Topic       string `json:"topic,omitempty"`
	MessageID   string `json:"messageId,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	Time        string `json:"time,omitempty"`
	EventType   string `json:"eventType,omitempty"`
	EventID     string `json:"eventId,omitempty"`
}

// InboundMessageEnvelope mirrors the bot message event shape used by DingTalk.
type InboundMessageEnvelope struct {
	ConversationID            string             `json:"conversationId,omitempty"`
	ConversationType          string             `json:"conversationType,omitempty"`
	SenderID                  string             `json:"senderId,omitempty"`
	SenderStaffID             string             `json:"senderStaffId,omitempty"`
	SenderUnionID             string             `json:"senderUnionId,omitempty"`
	SenderNick                string             `json:"senderNick,omitempty"`
	SessionWebhook            string             `json:"sessionWebhook,omitempty"`
	SessionWebhookExpiredTime int64              `json:"sessionWebhookExpiredTime,omitempty"`
	RobotCode                 string             `json:"robotCode,omitempty"`
	ChatbotUserID             string             `json:"chatbotUserId,omitempty"`
	CardSupported             *bool              `json:"cardSupported,omitempty"`
	MsgID                     string             `json:"msgId,omitempty"`
	MsgType                   string             `json:"msgtype,omitempty"`
	IsInAtList                bool               `json:"isInAtList,omitempty"`
	AtUsers                   []InboundAtUser    `json:"atUsers,omitempty"`
	Text                      InboundTextBlock   `json:"text,omitempty"`
	Image                     *InboundMediaBlock `json:"image,omitempty"`
	File                      *InboundMediaBlock `json:"file,omitempty"`
	Audio                     *InboundAudioBlock `json:"audio,omitempty"`
	CreateAt                  string             `json:"createAt,omitempty"`
}

type InboundTextBlock struct {
	Content string `json:"content,omitempty"`
}

type InboundAtUser struct {
	DingtalkID string `json:"dingtalkId,omitempty"`
	StaffID    string `json:"staffId,omitempty"`
	UnionID    string `json:"unionId,omitempty"`
}

type InboundMediaBlock struct {
	DownloadCode string `json:"downloadCode,omitempty"`
	MediaID      string `json:"mediaId,omitempty"`
	URL          string `json:"url,omitempty"`
	FileName     string `json:"fileName,omitempty"`
	MimeType     string `json:"mimeType,omitempty"`
	Size         int64  `json:"size,omitempty"`
}

type InboundAudioBlock struct {
	InboundMediaBlock
	DurationMs int64 `json:"durationMs,omitempty"`
}

// TextReplyPayload models the in-session webhook message payload body.
type TextReplyPayload struct {
	MsgType string           `json:"msgtype,omitempty"`
	Text    InboundTextBlock `json:"text,omitempty"`
}

// ProactiveSendPayload models the outbound proactive send request shape.
type ProactiveSendPayload struct {
	MsgKey             string   `json:"msgKey,omitempty"`
	MsgParam           string   `json:"msgParam,omitempty"`
	OpenConversationID string   `json:"openConversationId,omitempty"`
	RobotCode          string   `json:"robotCode,omitempty"`
	UserIDs            []string `json:"userIds,omitempty"`
}

// AICardCreateRequest models a minimal AI card creation request.
type AICardCreateRequest struct {
	RobotCode          string `json:"robotCode,omitempty"`
	OpenConversationID string `json:"openConversationId,omitempty"`
	SessionWebhook     string `json:"sessionWebhook,omitempty"`
	Title              string `json:"title,omitempty"`
	Content            string `json:"content,omitempty"`
	SourceMessageID    string `json:"sourceMessageId,omitempty"`
}

// AICardCreateResponse models the minimal AI card creation response.
type AICardCreateResponse struct {
	CardID string `json:"cardId,omitempty"`
}

// AICardUpdateRequest models a minimal AI card update request.
type AICardUpdateRequest struct {
	RobotCode       string `json:"robotCode,omitempty"`
	CardID          string `json:"cardId,omitempty"`
	Content         string `json:"content,omitempty"`
	Final           bool   `json:"final,omitempty"`
	SourceMessageID string `json:"sourceMessageId,omitempty"`
}

// AttachmentDownloadRequest models the robot file download request.
type AttachmentDownloadRequest struct {
	RobotCode    string `json:"robotCode,omitempty"`
	DownloadCode string `json:"downloadCode,omitempty"`
	MediaID      string `json:"mediaId,omitempty"`
}

// AttachmentDownloadResponse models the file download response.
type AttachmentDownloadResponse struct {
	DownloadURL string `json:"downloadUrl,omitempty"`
	URL         string `json:"url,omitempty"`
	FileName    string `json:"fileName,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	MediaID     string `json:"mediaId,omitempty"`
}

// MediaUploadResponse models the media upload response.
type MediaUploadResponse struct {
	MediaID   string `json:"media_id,omitempty"`
	CreatedAt int64  `json:"created_at,omitempty"`
	Type      string `json:"type,omitempty"`
}

// WebhookMediaPayload models attachment payloads sent through session webhooks.
type WebhookMediaPayload struct {
	MsgType string             `json:"msgtype,omitempty"`
	Image   *WebhookImageBlock `json:"image,omitempty"`
	File    *WebhookFileBlock  `json:"file,omitempty"`
	Audio   *WebhookAudioBlock `json:"audio,omitempty"`
}

type WebhookImageBlock struct {
	MediaID string `json:"mediaId,omitempty"`
}

type WebhookFileBlock struct {
	MediaID  string `json:"mediaId,omitempty"`
	FileName string `json:"fileName,omitempty"`
	FileType string `json:"fileType,omitempty"`
}

type WebhookAudioBlock struct {
	MediaID  string `json:"mediaId,omitempty"`
	Duration string `json:"duration,omitempty"`
}

// AttachmentUploadParam models the msgParam body for proactive media templates.
type AttachmentUploadParam struct {
	PhotoURL string `json:"photoURL,omitempty"`
	MediaID  string `json:"mediaId,omitempty"`
	FileName string `json:"fileName,omitempty"`
	FileType string `json:"fileType,omitempty"`
	Duration string `json:"duration,omitempty"`
}
