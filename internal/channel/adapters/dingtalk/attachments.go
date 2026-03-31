package dingtalk

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/memohai/memoh/internal/channel"
	"github.com/memohai/memoh/internal/media"
)

type assetOpener interface {
	Open(ctx context.Context, botID, contentHash string) (io.ReadCloser, media.Asset, error)
}

type attachmentUpload struct {
	Data     []byte
	FileName string
	Mime     string
	Type     channel.AttachmentType
	MediaID  string
}

type attachmentSource struct {
	Data     []byte
	FileName string
	Mime     string
}

func (a *Adapter) resolveAttachment(ctx context.Context, cfg Config, channelCfg channel.ChannelConfig, attachment channel.Attachment) (channel.AttachmentPayload, error) {
	src, err := a.readAttachmentSource(ctx, cfg, channelCfg, attachment)
	if err != nil {
		return channel.AttachmentPayload{}, err
	}
	if len(src.Data) == 0 {
		return channel.AttachmentPayload{}, errors.New("dingtalk attachment resolved empty bytes")
	}
	return channel.AttachmentPayload{
		Reader: io.NopCloser(bytes.NewReader(src.Data)),
		Mime:   src.Mime,
		Name:   src.FileName,
		Size:   int64(len(src.Data)),
	}, nil
}

func (a *Adapter) prepareOutboundAttachment(ctx context.Context, cfg Config, channelCfg channel.ChannelConfig, attachment channel.Attachment) (attachmentUpload, error) {
	src, err := a.readAttachmentSource(ctx, cfg, channelCfg, attachment)
	if err != nil {
		return attachmentUpload{}, err
	}
	if len(src.Data) == 0 {
		return attachmentUpload{}, errors.New("dingtalk attachment resolved empty bytes")
	}
	return attachmentUpload{
		Data:     src.Data,
		FileName: deriveAttachmentName(attachment, src.FileName, src.Mime),
		Mime:     src.Mime,
		Type:     normalizeOutboundAttachmentType(attachment.Type),
	}, nil
}

func (a *Adapter) readAttachmentSource(ctx context.Context, cfg Config, channelCfg channel.ChannelConfig, attachment channel.Attachment) (attachmentSource, error) {
	if remoteURL := strings.TrimSpace(attachment.URL); isHTTPURL(remoteURL) {
		return a.readRemoteAttachment(ctx, attachment, remoteURL)
	}
	if rawBase64 := extractRawBase64(attachment); rawBase64 != "" {
		data, err := decodeBase64Data(rawBase64)
		if err != nil {
			return attachmentSource{}, err
		}
		return attachmentSource{
			Data:     data,
			FileName: deriveAttachmentName(attachment, "", attachment.Mime),
			Mime:     strings.TrimSpace(attachment.Mime),
		}, nil
	}
	if contentHash := strings.TrimSpace(attachment.ContentHash); contentHash != "" {
		if a == nil || a.assets == nil {
			return attachmentSource{}, errors.New("dingtalk attachment content_hash requires asset opener")
		}
		botID := strings.TrimSpace(channelCfg.BotID)
		if attachment.Metadata != nil {
			if override, ok := attachment.Metadata["bot_id"].(string); ok && strings.TrimSpace(override) != "" {
				botID = strings.TrimSpace(override)
			}
		}
		if botID == "" {
			return attachmentSource{}, errors.New("dingtalk attachment content_hash requires bot_id context")
		}
		reader, asset, err := a.assets.Open(ctx, botID, contentHash)
		if err != nil {
			return attachmentSource{}, err
		}
		defer func() { _ = reader.Close() }()
		data, err := media.ReadAllWithLimit(reader, media.MaxAssetBytes)
		if err != nil {
			return attachmentSource{}, err
		}
		if len(data) == 0 {
			return attachmentSource{}, errors.New("dingtalk attachment resolved empty bytes")
		}
		fileName := deriveAttachmentName(attachment, "", asset.Mime)
		if fileName == "" {
			fileName = deriveFileNameFromMime(asset.Mime, attachment.Type)
		}
		mimeType := strings.TrimSpace(asset.Mime)
		if mimeType == "" {
			mimeType = strings.TrimSpace(attachment.Mime)
		}
		return attachmentSource{Data: data, FileName: fileName, Mime: mimeType}, nil
	}

	downloadCode, mediaID := attachmentDownloadReferences(attachment)
	if downloadCode == "" && mediaID == "" {
		return attachmentSource{}, errors.New("dingtalk attachment requires URL, base64, content_hash, or download code")
	}

	clientFactory := newClient
	if a != nil && a.newClient != nil {
		clientFactory = a.newClient
	}
	dingtalkClient := clientFactory(cfg)
	data, response, err := dingtalkClient.downloadAttachment(ctx, cfg, downloadCode, mediaID)
	if err != nil {
		return attachmentSource{}, err
	}

	fileName := deriveAttachmentName(attachment, response.FileName, response.MimeType)
	if fileName == "" {
		fileName = deriveFileNameFromMime(response.MimeType, attachment.Type)
	}
	mimeType := strings.TrimSpace(attachment.Mime)
	if mimeType == "" {
		mimeType = strings.TrimSpace(response.MimeType)
	}
	return attachmentSource{Data: data, FileName: fileName, Mime: mimeType}, nil
}

func (a *Adapter) readRemoteAttachment(ctx context.Context, attachment channel.Attachment, remoteURL string) (attachmentSource, error) {
	u, err := url.Parse(remoteURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return attachmentSource{}, fmt.Errorf("invalid attachment url: %s", remoteURL)
	}
	clientFactory := newClient
	if a != nil && a.newClient != nil {
		clientFactory = a.newClient
	}
	httpClient := clientFactory(Config{}).httpClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL, nil)
	if err != nil {
		return attachmentSource{}, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return attachmentSource{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return attachmentSource{}, fmt.Errorf("attachment fetch failed: status=%d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, media.MaxAssetBytes+1))
	if err != nil {
		return attachmentSource{}, err
	}
	if len(data) == 0 {
		return attachmentSource{}, errors.New("dingtalk attachment download returned empty bytes")
	}
	if int64(len(data)) > media.MaxAssetBytes {
		return attachmentSource{}, fmt.Errorf("%w: max %d bytes", media.ErrAssetTooLarge, media.MaxAssetBytes)
	}
	mimeType := strings.TrimSpace(attachment.Mime)
	if mimeType == "" {
		mimeType = strings.TrimSpace(resp.Header.Get("Content-Type"))
		if idx := strings.Index(mimeType, ";"); idx >= 0 {
			mimeType = strings.TrimSpace(mimeType[:idx])
		}
	}
	fileName := deriveAttachmentName(attachment, filepath.Base(u.Path), mimeType)
	return attachmentSource{Data: data, FileName: fileName, Mime: mimeType}, nil
}

func extractRawBase64(att channel.Attachment) string {
	if candidate := strings.TrimSpace(att.Base64); candidate != "" {
		if strings.HasPrefix(strings.ToLower(candidate), "data:") {
			if idx := strings.Index(candidate, ","); idx >= 0 && idx < len(candidate)-1 {
				return candidate[idx+1:]
			}
			return ""
		}
		return candidate
	}

	candidate := strings.TrimSpace(att.URL)
	if strings.HasPrefix(strings.ToLower(candidate), "data:") {
		if idx := strings.Index(candidate, ","); idx >= 0 && idx < len(candidate)-1 {
			return candidate[idx+1:]
		}
	}
	return ""
}

func decodeBase64Data(raw string) ([]byte, error) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return nil, errors.New("dingtalk attachment base64 is empty")
	}
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.RawURLEncoding} {
		if data, err := enc.DecodeString(candidate); err == nil {
			return data, nil
		}
	}
	return nil, errors.New("dingtalk attachment base64 is invalid")
}

func attachmentDownloadReferences(att channel.Attachment) (string, string) {
	downloadCode := strings.TrimSpace(att.PlatformKey)
	mediaID := ""
	if att.Metadata != nil {
		if value, ok := att.Metadata["download_code"].(string); ok && strings.TrimSpace(value) != "" {
			downloadCode = strings.TrimSpace(value)
		}
		if value, ok := att.Metadata["media_id"].(string); ok && strings.TrimSpace(value) != "" {
			mediaID = strings.TrimSpace(value)
		}
	}
	if mediaID == "" {
		mediaID = strings.TrimSpace(att.PlatformKey)
	}
	return downloadCode, mediaID
}

func deriveAttachmentName(att channel.Attachment, fallbackName, fallbackMime string) string {
	if name := strings.TrimSpace(att.Name); name != "" {
		return name
	}
	if name := strings.TrimSpace(fallbackName); name != "" {
		return name
	}
	if rawURL := strings.TrimSpace(att.URL); rawURL != "" && !strings.HasPrefix(strings.ToLower(rawURL), "data:") {
		if base := filepath.Base(rawURL); base != "." && base != "/" && base != "" {
			return base
		}
	}
	return deriveFileNameFromMime(firstNonEmpty(att.Mime, fallbackMime), att.Type)
}

func deriveFileNameFromMime(mimeType string, attType channel.AttachmentType) string {
	ext := mimeExtension(mimeType)
	base := "attachment"
	switch attType {
	case channel.AttachmentImage, channel.AttachmentGIF:
		base = "image"
	case channel.AttachmentVideo:
		base = "video"
	case channel.AttachmentVoice, channel.AttachmentAudio:
		base = "audio"
	case channel.AttachmentFile:
		base = "file"
	}
	return base + ext
}

func mimeExtension(mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/wav", "audio/x-wav":
		return ".wav"
	case "audio/amr":
		return ".amr"
	case "application/pdf":
		return ".pdf"
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func isHTTPURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(u.Scheme)) {
	case "http", "https":
		return u.Host != ""
	default:
		return false
	}
}

func normalizeOutboundAttachmentType(attType channel.AttachmentType) channel.AttachmentType {
	switch attType {
	case channel.AttachmentImage, channel.AttachmentGIF:
		return channel.AttachmentImage
	default:
		return channel.AttachmentFile
	}
}

func fileTypeFromNameAndMime(name, mimeType string) string {
	lowerName := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.HasSuffix(lowerName, ".xlsx"):
		return "xlsx"
	case strings.HasSuffix(lowerName, ".xls"):
		return "xls"
	case strings.HasSuffix(lowerName, ".pdf"):
		return "pdf"
	case strings.HasSuffix(lowerName, ".zip"):
		return "zip"
	case strings.HasSuffix(lowerName, ".rar"):
		return "rar"
	case strings.HasSuffix(lowerName, ".docx"):
		return "docx"
	case strings.HasSuffix(lowerName, ".doc"):
		return "doc"
	case strings.HasSuffix(lowerName, ".png"):
		return "png"
	case strings.HasSuffix(lowerName, ".jpg"), strings.HasSuffix(lowerName, ".jpeg"):
		return "jpg"
	case strings.HasSuffix(lowerName, ".gif"):
		return "gif"
	case strings.HasSuffix(lowerName, ".mp3"):
		return "mp3"
	case strings.HasSuffix(lowerName, ".wav"):
		return "wav"
	case strings.HasSuffix(lowerName, ".amr"):
		return "amr"
	}
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "application/pdf":
		return "pdf"
	case "image/png":
		return "png"
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/gif":
		return "gif"
	case "audio/mpeg", "audio/mp3":
		return "mp3"
	case "audio/wav", "audio/x-wav":
		return "wav"
	case "audio/amr":
		return "amr"
	}
	return "bin"
}
