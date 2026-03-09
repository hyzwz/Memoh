package wecombot

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/memohai/memoh/internal/channel"
	"github.com/memohai/memoh/internal/media"
)

func (*WeComBotAdapter) ResolveAttachment(ctx context.Context, _ channel.ChannelConfig, attachment channel.Attachment) (channel.AttachmentPayload, error) {
	rawURL := strings.TrimSpace(attachment.PlatformKey)
	if rawURL == "" {
		rawURL = strings.TrimSpace(attachment.URL)
	}
	if rawURL == "" {
		return channel.AttachmentPayload{}, errors.New("wecom_ai_bot attachment platform_key or url is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return channel.AttachmentPayload{}, fmt.Errorf("build download request: %w", err)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req) //nolint:gosec // URL comes from WeCom AI Bot callback payload.
	if err != nil {
		return channel.AttachmentPayload{}, fmt.Errorf("download attachment: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return channel.AttachmentPayload{}, fmt.Errorf("download attachment status: %d", resp.StatusCode)
	}
	maxBytes := media.MaxAssetBytes
	if resp.ContentLength > maxBytes {
		_, _ = io.Copy(io.Discard, resp.Body)
		return channel.AttachmentPayload{}, fmt.Errorf("%w: max %d bytes", media.ErrAssetTooLarge, maxBytes)
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return channel.AttachmentPayload{}, fmt.Errorf("read attachment: %w", err)
	}
	if int64(len(payload)) > maxBytes {
		return channel.AttachmentPayload{}, fmt.Errorf("%w: max %d bytes", media.ErrAssetTooLarge, maxBytes)
	}
	if aesKey := attachmentMetadataString(attachment.Metadata, "aes_key"); aesKey != "" {
		payload, err = decryptAttachment(payload, aesKey)
		if err != nil {
			return channel.AttachmentPayload{}, err
		}
	}
	name := strings.TrimSpace(attachment.Name)
	if name == "" {
		name = filenameFromDisposition(resp.Header.Get("Content-Disposition"))
	}
	if name == "" {
		name = path.Base(strings.TrimSpace(req.URL.Path))
	}
	mimeType := strings.TrimSpace(attachment.Mime)
	if mimeType == "" {
		mimeType = strings.TrimSpace(resp.Header.Get("Content-Type"))
		if idx := strings.Index(mimeType, ";"); idx >= 0 {
			mimeType = strings.TrimSpace(mimeType[:idx])
		}
	}
	if mimeType == "" {
		mimeType = http.DetectContentType(payload)
	}
	return channel.AttachmentPayload{
		Reader: io.NopCloser(bytes.NewReader(payload)),
		Mime:   mimeType,
		Name:   name,
		Size:   int64(len(payload)),
	}, nil
}

func attachmentMetadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func decryptAttachment(encrypted []byte, aesKey string) ([]byte, error) {
	trimmed := strings.TrimSpace(aesKey)
	// WeCom AI Bot aeskey may use standard, URL-safe, or unpadded base64.
	var key []byte
	var err error
	for _, enc := range [](*base64.Encoding){
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	} {
		key, err = enc.DecodeString(trimmed)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("decode wecom ai bot aes key: %w", err)
	}
	if len(key) < 32 {
		return nil, errors.New("invalid wecom ai bot aes key length")
	}
	block, err := aes.NewCipher(key[:32])
	if err != nil {
		return nil, fmt.Errorf("build aes cipher: %w", err)
	}
	if len(encrypted)%aes.BlockSize != 0 {
		return nil, errors.New("encrypted attachment length is not a multiple of block size")
	}
	plain := make([]byte, len(encrypted))
	cipher.NewCBCDecrypter(block, key[:aes.BlockSize]).CryptBlocks(plain, encrypted)
	unpadded, err := stripPKCS7Padding(plain)
	if err != nil {
		return nil, err
	}
	return unpadded, nil
}

func stripPKCS7Padding(raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty decrypted payload")
	}
	padLen := int(raw[len(raw)-1])
	if padLen < 1 || padLen > 32 || padLen > len(raw) {
		return nil, fmt.Errorf("invalid PKCS#7 padding value: %d", padLen)
	}
	for _, b := range raw[len(raw)-padLen:] {
		if int(b) != padLen {
			return nil, errors.New("invalid PKCS#7 padding bytes")
		}
	}
	return raw[:len(raw)-padLen], nil
}

func filenameFromDisposition(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if _, params, err := mime.ParseMediaType(value); err == nil {
		if name := strings.TrimSpace(params["filename*"]); name != "" {
			if idx := strings.Index(name, "''"); idx >= 0 {
				name = name[idx+2:]
			}
			if decoded, err := urlPathUnescape(name); err == nil {
				return decoded
			}
		}
		if name := strings.TrimSpace(params["filename"]); name != "" {
			if decoded, err := urlPathUnescape(name); err == nil {
				return decoded
			}
			return name
		}
	}
	return ""
}

func urlPathUnescape(value string) (string, error) {
	replacer := strings.NewReplacer("+", "%20")
	return url.QueryUnescape(replacer.Replace(value))
}
