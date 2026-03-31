package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/memohai/memoh/internal/channel"
)

const defaultAPIBaseURL = "https://api.dingtalk.com"

type client struct {
	baseURL       string
	httpClient    *http.Client
	replyTextFunc func(context.Context, string, string) error
}

func newClient(cfg Config) *client {
	baseURL := cfg.APIBaseURL
	if baseURL == "" {
		baseURL = defaultAPIBaseURL
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	return &client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (c *client) accessToken(ctx context.Context, cfg Config) (string, error) {
	resp, err := c.postJSON(ctx, "/v1.0/oauth2/accessToken", AccessTokenRequest{
		AppKey:    cfg.AppKey,
		AppSecret: cfg.AppSecret,
	}, nil)
	if err != nil {
		return "", fmt.Errorf("dingtalk access token: %w", err)
	}

	var body AccessTokenResponse
	if err := json.Unmarshal(resp, &body); err != nil {
		return "", fmt.Errorf("dingtalk access token: parse response: %w", err)
	}
	if strings.TrimSpace(body.AccessToken) == "" {
		return "", fmt.Errorf("dingtalk access token: empty access token")
	}
	return strings.TrimSpace(body.AccessToken), nil
}

func (c *client) selfIdentity(ctx context.Context, accessToken string) (SelfIdentityResponse, error) {
	headers := http.Header{}
	headers.Set("x-acs-dingtalk-access-token", accessToken)

	resp, err := c.postJSON(ctx, "/v1.0/robot/me", SelfIdentityRequest{
		AccessToken: accessToken,
	}, headers)
	if err != nil {
		return SelfIdentityResponse{}, fmt.Errorf("dingtalk discover self: %w", err)
	}

	var body SelfIdentityResponse
	if err := json.Unmarshal(resp, &body); err != nil {
		return SelfIdentityResponse{}, fmt.Errorf("dingtalk discover self: parse response: %w", err)
	}
	return body, nil
}

func (c *client) postJSON(ctx context.Context, path string, payload any, headers http.Header) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return c.doJSON(req, headers)
}

func (c *client) postAbsoluteJSON(ctx context.Context, targetURL string, payload any, headers http.Header) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return c.doJSON(req, headers)
}

func (c *client) doJSON(req *http.Request, headers http.Header) ([]byte, error) {
	req.Header.Set("Content-Type", "application/json")
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return respBody, nil
}

func (c *client) replyText(ctx context.Context, sessionWebhook, text string) error {
	if c != nil && c.replyTextFunc != nil {
		return c.replyTextFunc(ctx, sessionWebhook, text)
	}
	if c == nil {
		return fmt.Errorf("dingtalk reply text: client is not configured")
	}
	if strings.TrimSpace(sessionWebhook) == "" {
		return fmt.Errorf("dingtalk reply text: session webhook is required")
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("dingtalk reply text: text is required")
	}
	_, err := c.postAbsoluteJSON(ctx, strings.TrimSpace(sessionWebhook), TextReplyPayload{
		MsgType: "text",
		Text: InboundTextBlock{
			Content: strings.TrimSpace(text),
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("dingtalk reply text: %w", err)
	}
	return nil
}

func (c *client) sendConversationText(ctx context.Context, cfg Config, conversationID, text string) error {
	return c.sendProactiveText(ctx, cfg, Target{Family: targetFamilyConversation, Value: conversationID}, text)
}

func (c *client) sendUserText(ctx context.Context, cfg Config, target Target, text string) error {
	if target.Family != targetFamilyUser {
		return fmt.Errorf("dingtalk send user text: unsupported target family %q", target.Family)
	}
	return c.sendProactiveText(ctx, cfg, target, text)
}

func (c *client) sendProactiveText(ctx context.Context, cfg Config, target Target, text string) error {
	if c == nil {
		return fmt.Errorf("dingtalk proactive send: client is not configured")
	}
	trimmedText := strings.TrimSpace(text)
	if trimmedText == "" {
		return fmt.Errorf("dingtalk proactive send: text is required")
	}
	robotCode := strings.TrimSpace(cfg.RobotCode)
	if robotCode == "" {
		return fmt.Errorf("dingtalk proactive send: robot code is required")
	}
	accessToken, err := c.accessToken(ctx, cfg)
	if err != nil {
		return err
	}
	headers := http.Header{}
	headers.Set("x-acs-dingtalk-access-token", accessToken)

	msgParam, err := json.Marshal(map[string]string{"content": trimmedText})
	if err != nil {
		return fmt.Errorf("dingtalk proactive send: encode msgParam: %w", err)
	}

	payload := ProactiveSendPayload{
		MsgKey:    "sampleText",
		MsgParam:  string(msgParam),
		RobotCode: robotCode,
	}

	switch target.Family {
	case targetFamilyConversation:
		if strings.TrimSpace(target.Value) == "" {
			return fmt.Errorf("dingtalk proactive send: conversation id is required")
		}
		payload.OpenConversationID = strings.TrimSpace(target.Value)
		_, err = c.postJSON(ctx, "/v1.0/robot/groupMessages/send", payload, headers)
	case targetFamilyUser:
		if strings.TrimSpace(target.Value) == "" {
			return fmt.Errorf("dingtalk proactive send: user identifier is required")
		}
		switch target.UserSubtype {
		case targetUserSubtypeStaff:
		case targetUserSubtypeUnion:
		default:
			return fmt.Errorf("dingtalk proactive send: unsupported user subtype %q", target.UserSubtype)
		}
		payload.UserIDs = []string{strings.TrimSpace(target.Value)}
		_, err = c.postJSON(ctx, "/v1.0/robot/oToMessages/batchSend", payload, headers)
	default:
		return fmt.Errorf("dingtalk proactive send: unsupported target family %q", target.Family)
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *client) uploadMedia(ctx context.Context, cfg Config, mediaType, fileName string, data []byte) (string, error) {
	if c == nil {
		return "", fmt.Errorf("dingtalk media upload: client is not configured")
	}
	robotCode := strings.TrimSpace(cfg.RobotCode)
	if robotCode == "" {
		return "", fmt.Errorf("dingtalk media upload: robot code is required")
	}
	accessToken, err := c.accessToken(ctx, cfg)
	if err != nil {
		return "", err
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("type", strings.TrimSpace(mediaType)); err != nil {
		return "", err
	}
	part, err := writer.CreateFormFile("media", strings.TrimSpace(fileName))
	if err != nil {
		return "", err
	}
	if _, err := part.Write(data); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	uploadURL := c.baseURL + "/media/upload?access_token=" + url.QueryEscape(accessToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("dingtalk media upload: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var uploadResp MediaUploadResponse
	if err := json.Unmarshal(respBody, &uploadResp); err != nil {
		return "", fmt.Errorf("dingtalk media upload: parse response: %w", err)
	}
	if strings.TrimSpace(uploadResp.MediaID) == "" {
		return "", fmt.Errorf("dingtalk media upload: empty media id")
	}
	return strings.TrimSpace(uploadResp.MediaID), nil
}

func (c *client) downloadAttachment(ctx context.Context, cfg Config, downloadCode, mediaID string) ([]byte, AttachmentDownloadResponse, error) {
	if c == nil {
		return nil, AttachmentDownloadResponse{}, fmt.Errorf("dingtalk media download: client is not configured")
	}
	robotCode := strings.TrimSpace(cfg.RobotCode)
	if robotCode == "" {
		return nil, AttachmentDownloadResponse{}, fmt.Errorf("dingtalk media download: robot code is required")
	}
	request := AttachmentDownloadRequest{
		RobotCode:    robotCode,
		DownloadCode: strings.TrimSpace(downloadCode),
		MediaID:      strings.TrimSpace(mediaID),
	}
	accessToken, err := c.accessToken(ctx, cfg)
	if err != nil {
		return nil, AttachmentDownloadResponse{}, err
	}
	headers := http.Header{}
	headers.Set("x-acs-dingtalk-access-token", accessToken)
	respBody, err := c.postJSON(ctx, "/v1.0/robot/messageFiles/download", request, headers)
	if err != nil {
		return nil, AttachmentDownloadResponse{}, fmt.Errorf("dingtalk media download: %w", err)
	}

	var body AttachmentDownloadResponse
	if err := json.Unmarshal(respBody, &body); err != nil {
		return nil, AttachmentDownloadResponse{}, fmt.Errorf("dingtalk media download: parse response: %w", err)
	}
	downloadURL := strings.TrimSpace(body.DownloadURL)
	if downloadURL == "" {
		downloadURL = strings.TrimSpace(body.URL)
	}
	if downloadURL == "" {
		return nil, AttachmentDownloadResponse{}, fmt.Errorf("dingtalk media download: empty download url")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, AttachmentDownloadResponse{}, err
	}
	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, AttachmentDownloadResponse{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, AttachmentDownloadResponse{}, err
	}
	if len(data) == 0 {
		return nil, AttachmentDownloadResponse{}, fmt.Errorf("dingtalk media download: empty attachment bytes")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, AttachmentDownloadResponse{}, fmt.Errorf("dingtalk media download: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, body, nil
}

func (c *client) sendWebhookAttachment(ctx context.Context, sessionWebhook string, upload attachmentUpload) error {
	if c == nil {
		return fmt.Errorf("dingtalk webhook attachment send: client is not configured")
	}
	target := strings.TrimSpace(sessionWebhook)
	if target == "" {
		return fmt.Errorf("dingtalk webhook attachment send: session webhook is required")
	}

	payload, err := buildWebhookMediaPayload(upload)
	if err != nil {
		return err
	}
	_, err = c.postAbsoluteJSON(ctx, target, payload, nil)
	if err != nil {
		return fmt.Errorf("dingtalk webhook attachment send: %w", err)
	}
	return nil
}

func (c *client) sendProactiveAttachment(ctx context.Context, cfg Config, target Target, upload attachmentUpload) error {
	if c == nil {
		return fmt.Errorf("dingtalk proactive attachment send: client is not configured")
	}
	robotCode := strings.TrimSpace(cfg.RobotCode)
	if robotCode == "" {
		return fmt.Errorf("dingtalk proactive attachment send: robot code is required")
	}
	accessToken, err := c.accessToken(ctx, cfg)
	if err != nil {
		return err
	}
	headers := http.Header{}
	headers.Set("x-acs-dingtalk-access-token", accessToken)

	templateKey, param, err := buildProactiveAttachmentTemplate(upload)
	if err != nil {
		return err
	}
	payload := ProactiveSendPayload{
		MsgKey:    templateKey,
		MsgParam:  param,
		RobotCode: robotCode,
		UserIDs:   nil,
	}
	switch target.Family {
	case targetFamilyConversation:
		payload.OpenConversationID = strings.TrimSpace(target.Value)
	case targetFamilyUser:
		payload.UserIDs = []string{strings.TrimSpace(target.Value)}
	default:
		return fmt.Errorf("dingtalk proactive attachment send: unsupported target family %q", target.Family)
	}

	path := "/v1.0/robot/groupMessages/send"
	if target.Family == targetFamilyUser {
		path = "/v1.0/robot/oToMessages/batchSend"
	}
	_, err = c.postJSON(ctx, path, payload, headers)
	if err != nil {
		return fmt.Errorf("dingtalk proactive attachment send: %w", err)
	}
	return nil
}

func buildWebhookMediaPayload(upload attachmentUpload) (WebhookMediaPayload, error) {
	switch normalizeOutboundAttachmentType(upload.Type) {
	case channel.AttachmentImage:
		return WebhookMediaPayload{
			MsgType: "image",
			Image: &WebhookImageBlock{
				MediaID: strings.TrimSpace(upload.MediaID),
			},
		}, nil
	default:
		fileType := fileTypeFromNameAndMime(upload.FileName, upload.Mime)
		return WebhookMediaPayload{
			MsgType: "file",
			File: &WebhookFileBlock{
				MediaID:  strings.TrimSpace(upload.MediaID),
				FileName: strings.TrimSpace(upload.FileName),
				FileType: fileType,
			},
		}, nil
	}
}

func buildProactiveAttachmentTemplate(upload attachmentUpload) (string, string, error) {
	mediaID := strings.TrimSpace(upload.MediaID)
	if mediaID == "" {
		return "", "", fmt.Errorf("dingtalk attachment media id is required")
	}
	switch normalizeOutboundAttachmentType(upload.Type) {
	case channel.AttachmentImage:
		param, err := json.Marshal(AttachmentUploadParam{
			PhotoURL: mediaID,
		})
		if err != nil {
			return "", "", err
		}
		return "sampleImageMsg", string(param), nil
	default:
		fileType := fileTypeFromNameAndMime(upload.FileName, upload.Mime)
		param, err := json.Marshal(AttachmentUploadParam{
			MediaID:  mediaID,
			FileName: strings.TrimSpace(upload.FileName),
			FileType: fileType,
		})
		if err != nil {
			return "", "", err
		}
		return "sampleFile", string(param), nil
	}
}

func (c *client) createAICard(ctx context.Context, cfg Config, target Target, replyTarget string, sourceMessageID, title, content string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("dingtalk ai card create: client is not configured")
	}
	robotCode := strings.TrimSpace(cfg.RobotCode)
	if robotCode == "" {
		return "", fmt.Errorf("dingtalk ai card create: robot code is required")
	}
	accessToken, err := c.accessToken(ctx, cfg)
	if err != nil {
		return "", err
	}

	payload := AICardCreateRequest{
		RobotCode:       robotCode,
		Title:           strings.TrimSpace(title),
		Content:         strings.TrimSpace(content),
		SourceMessageID: strings.TrimSpace(sourceMessageID),
	}
	switch target.Family {
	case targetFamilyWebhook:
		payload.SessionWebhook = strings.TrimSpace(replyTarget)
	case targetFamilyConversation:
		payload.OpenConversationID = strings.TrimSpace(target.Value)
	case targetFamilyUser:
		return "", fmt.Errorf("dingtalk ai card create: user targets are not supported")
	}

	headers := http.Header{}
	headers.Set("x-acs-dingtalk-access-token", accessToken)
	resp, err := c.postJSON(ctx, "/v1.0/robot/aiCards/create", payload, headers)
	if err != nil {
		return "", fmt.Errorf("dingtalk ai card create: %w", err)
	}
	var body AICardCreateResponse
	if err := json.Unmarshal(resp, &body); err != nil {
		return "", fmt.Errorf("dingtalk ai card create: parse response: %w", err)
	}
	if strings.TrimSpace(body.CardID) == "" {
		return "", fmt.Errorf("dingtalk ai card create: empty card id")
	}
	return strings.TrimSpace(body.CardID), nil
}

func (c *client) updateAICard(ctx context.Context, cfg Config, cardID, sourceMessageID, content string, final bool) error {
	if c == nil {
		return fmt.Errorf("dingtalk ai card update: client is not configured")
	}
	robotCode := strings.TrimSpace(cfg.RobotCode)
	if robotCode == "" {
		return fmt.Errorf("dingtalk ai card update: robot code is required")
	}
	cardID = strings.TrimSpace(cardID)
	if cardID == "" {
		return fmt.Errorf("dingtalk ai card update: card id is required")
	}
	accessToken, err := c.accessToken(ctx, cfg)
	if err != nil {
		return err
	}
	headers := http.Header{}
	headers.Set("x-acs-dingtalk-access-token", accessToken)
	_, err = c.postJSON(ctx, "/v1.0/robot/aiCards/update", AICardUpdateRequest{
		RobotCode:       robotCode,
		CardID:          cardID,
		Content:         strings.TrimSpace(content),
		Final:           final,
		SourceMessageID: strings.TrimSpace(sourceMessageID),
	}, headers)
	if err != nil {
		return fmt.Errorf("dingtalk ai card update: %w", err)
	}
	return nil
}

func discoverExternalIdentity(self SelfIdentityResponse) string {
	switch {
	case strings.TrimSpace(self.UserID) != "":
		return "chatbot_user_id:" + strings.TrimSpace(self.UserID)
	case strings.TrimSpace(self.StaffID) != "":
		return "user:staff:" + strings.TrimSpace(self.StaffID)
	case strings.TrimSpace(self.UnionID) != "":
		return "user:union:" + strings.TrimSpace(self.UnionID)
	case strings.TrimSpace(self.RobotCode) != "":
		return "robot_code:" + strings.TrimSpace(self.RobotCode)
	default:
		return ""
	}
}
