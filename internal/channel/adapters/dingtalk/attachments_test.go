package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/memohai/memoh/internal/channel"
	"github.com/memohai/memoh/internal/media"
)

func TestResolveAttachmentDownloadsFromDingTalk(t *testing.T) {
	t.Parallel()

	var downloadReq AttachmentDownloadRequest
	var downloadCalls int
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/v1.0/oauth2/accessToken":
			return jsonHTTPResponse(map[string]any{"access_token": "token-1"}), nil
		case "/v1.0/robot/messageFiles/download":
			downloadCalls++
			if err := json.NewDecoder(r.Body).Decode(&downloadReq); err != nil {
				t.Fatalf("decode download request: %v", err)
			}
			if downloadReq.DownloadCode != "download-code-1" {
				t.Fatalf("unexpected downloadCode: %#v", downloadReq.DownloadCode)
			}
			return jsonHTTPResponse(AttachmentDownloadResponse{
				DownloadURL: "http://dingtalk.test/downloaded",
				FileName:    "voice.mp3",
				MimeType:    "audio/mpeg",
			}), nil
		case "/downloaded":
			return responseWithBody(200, "voice-bytes", "audio/mpeg"), nil
		default:
			return responseWithBody(404, "not found", "text/plain"), nil
		}
	})}

	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL:    "http://dingtalk.test",
			httpClient: httpClient,
		}
	}

	payload, err := adapter.ResolveAttachment(context.Background(), channel.ChannelConfig{
		ID:    "cfg-1",
		BotID: "bot-1",
		Credentials: map[string]any{
			"appKey":    "key",
			"appSecret": "secret",
			"robotCode": "robot-1",
		},
	}, channel.Attachment{
		Type:        channel.AttachmentAudio,
		PlatformKey: "download-code-1",
		Name:        "voice.mp3",
	})
	if err != nil {
		t.Fatalf("resolve attachment: %v", err)
	}
	if downloadCalls != 1 {
		t.Fatalf("unexpected download calls: %d", downloadCalls)
	}
	data, err := io.ReadAll(payload.Reader)
	if err != nil {
		t.Fatalf("read payload: %v", err)
	}
	if string(data) != "voice-bytes" {
		t.Fatalf("unexpected payload bytes: %q", data)
	}
	if payload.Name != "voice.mp3" {
		t.Fatalf("unexpected payload name: %q", payload.Name)
	}
	if payload.Mime != "audio/mpeg" {
		t.Fatalf("unexpected payload mime: %q", payload.Mime)
	}
}

func TestResolveAttachmentRejectsMissingReference(t *testing.T) {
	t.Parallel()

	adapter := NewDingTalkAdapter(nil)
	_, err := adapter.ResolveAttachment(context.Background(), channel.ChannelConfig{
		ID:    "cfg-1",
		BotID: "bot-1",
		Credentials: map[string]any{
			"appKey":    "key",
			"appSecret": "secret",
			"robotCode": "robot-1",
		},
	}, channel.Attachment{})
	if err == nil || !strings.Contains(err.Error(), "requires URL, base64, content_hash, or download code") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendOutboundImageAttachmentUploadsThenSends(t *testing.T) {
	t.Parallel()

	var calls []string
	var uploadType string
	var uploadData []byte
	var uploadFileName string
	var uploadMediaID string
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/v1.0/oauth2/accessToken":
			return jsonHTTPResponse(map[string]any{"access_token": "token-2"}), nil
		case "/media/upload":
			calls = append(calls, "upload")
			if got := r.URL.Query().Get("access_token"); got != "token-2" {
				t.Fatalf("unexpected access token: %q", got)
			}
			mt, err := r.MultipartReader()
			if err != nil {
				t.Fatalf("multipart reader: %v", err)
			}
			for {
				part, err := mt.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("next part: %v", err)
				}
				switch part.FormName() {
				case "type":
					b, _ := io.ReadAll(part)
					uploadType = string(b)
				case "media":
					uploadFileName = part.FileName()
					b, _ := io.ReadAll(part)
					uploadData = b
				}
			}
			uploadMediaID = "media-image-1"
			return jsonHTTPResponse(MediaUploadResponse{MediaID: uploadMediaID, Type: uploadType}), nil
		case "/v1.0/robot/groupMessages/send":
			calls = append(calls, "send")
			var body ProactiveSendPayload
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode send body: %v", err)
			}
			if body.MsgKey != "sampleImageMsg" {
				t.Fatalf("unexpected msgKey: %q", body.MsgKey)
			}
			var param AttachmentUploadParam
			if err := json.Unmarshal([]byte(body.MsgParam), &param); err != nil {
				t.Fatalf("decode msgParam: %v", err)
			}
			if param.PhotoURL != uploadMediaID {
				t.Fatalf("unexpected photoURL: %q", param.PhotoURL)
			}
			if body.OpenConversationID != "cid-1" {
				t.Fatalf("unexpected openConversationId: %q", body.OpenConversationID)
			}
			return jsonHTTPResponse(map[string]any{"processQueryKey": "pqk-1"}), nil
		default:
			return responseWithBody(404, "not found", "text/plain"), nil
		}
	})}

	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL:    "http://dingtalk.test",
			httpClient: httpClient,
		}
	}
	adapter.SetAssetOpener(&testAssetOpener{
		data:  []byte("image-bytes"),
		asset: testAsset{mime: "image/png"},
	})

	err := adapter.Send(context.Background(), channel.ChannelConfig{
		ID:    "cfg-2",
		BotID: "bot-2",
		Credentials: map[string]any{
			"appKey":    "key",
			"appSecret": "secret",
			"robotCode": "robot-2",
		},
	}, channel.OutboundMessage{
		Target: "conversation:cid-1",
		Message: channel.Message{
			Attachments: []channel.Attachment{{
				Type:        channel.AttachmentImage,
				ContentHash: "hash-image",
				Name:        "picture.png",
			}},
		},
	})
	if err != nil {
		t.Fatalf("send image attachment: %v", err)
	}
	if len(calls) != 2 || calls[0] != "upload" || calls[1] != "send" {
		t.Fatalf("unexpected call order: %#v", calls)
	}
	if uploadType != "image" {
		t.Fatalf("unexpected upload type: %q", uploadType)
	}
	if !bytes.Equal(uploadData, []byte("image-bytes")) {
		t.Fatalf("unexpected upload data: %q", uploadData)
	}
	if uploadFileName != "picture.png" {
		t.Fatalf("unexpected upload file name: %q", uploadFileName)
	}
}

func TestSendOutboundFileAttachmentUploadsThenSendsAfterText(t *testing.T) {
	t.Parallel()

	var calls []string
	var sendBodies []ProactiveSendPayload
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/v1.0/oauth2/accessToken":
			return jsonHTTPResponse(map[string]any{"access_token": "token-3"}), nil
		case "/media/upload":
			calls = append(calls, "upload")
			return jsonHTTPResponse(MediaUploadResponse{MediaID: "media-file-1", Type: "file"}), nil
		case "/v1.0/robot/groupMessages/send":
			calls = append(calls, "send")
			var body ProactiveSendPayload
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode send body: %v", err)
			}
			sendBodies = append(sendBodies, body)
			return jsonHTTPResponse(map[string]any{"processQueryKey": "pqk-2"}), nil
		default:
			return responseWithBody(404, "not found", "text/plain"), nil
		}
	})}

	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL:    "http://dingtalk.test",
			httpClient: httpClient,
		}
	}
	adapter.SetAssetOpener(&testAssetOpener{
		data:  []byte("file-bytes"),
		asset: testAsset{mime: "application/pdf"},
	})

	err := adapter.Send(context.Background(), channel.ChannelConfig{
		ID:    "cfg-3",
		BotID: "bot-3",
		Credentials: map[string]any{
			"appKey":    "key",
			"appSecret": "secret",
			"robotCode": "robot-3",
		},
	}, channel.OutboundMessage{
		Target: "conversation:cid-2",
		Message: channel.Message{
			Text: "hello file",
			Attachments: []channel.Attachment{{
				Type:        channel.AttachmentFile,
				ContentHash: "hash-file",
				Name:        "report.pdf",
			}},
		},
	})
	if err != nil {
		t.Fatalf("send file attachment: %v", err)
	}
	if len(calls) != 3 || calls[0] != "send" || calls[1] != "upload" || calls[2] != "send" {
		t.Fatalf("unexpected call order: %#v", calls)
	}
	if len(sendBodies) != 2 {
		t.Fatalf("unexpected send body count: %d", len(sendBodies))
	}
	if sendBodies[0].MsgKey != "sampleText" {
		t.Fatalf("unexpected first msgKey: %q", sendBodies[0].MsgKey)
	}
	if sendBodies[1].MsgKey != "sampleFile" {
		t.Fatalf("unexpected second msgKey: %q", sendBodies[1].MsgKey)
	}
	var param AttachmentUploadParam
	if err := json.Unmarshal([]byte(sendBodies[1].MsgParam), &param); err != nil {
		t.Fatalf("decode file msgParam: %v", err)
	}
	if param.MediaID != "media-file-1" || param.FileName != "report.pdf" || param.FileType != "pdf" {
		t.Fatalf("unexpected file param: %#v", param)
	}
}

func TestSendOutboundAudioFallsBackToFile(t *testing.T) {
	t.Parallel()

	var sendBody ProactiveSendPayload
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/v1.0/oauth2/accessToken":
			return jsonHTTPResponse(map[string]any{"access_token": "token-4"}), nil
		case "/media/upload":
			return jsonHTTPResponse(MediaUploadResponse{MediaID: "media-audio-1", Type: "file"}), nil
		case "/v1.0/robot/oToMessages/batchSend":
			if err := json.NewDecoder(r.Body).Decode(&sendBody); err != nil {
				t.Fatalf("decode send body: %v", err)
			}
			return jsonHTTPResponse(map[string]any{"taskId": "task-1"}), nil
		default:
			return responseWithBody(404, "not found", "text/plain"), nil
		}
	})}

	adapter := NewDingTalkAdapter(nil)
	adapter.newClient = func(cfg Config) *client {
		return &client{
			baseURL:    "http://dingtalk.test",
			httpClient: httpClient,
		}
	}
	adapter.SetAssetOpener(&testAssetOpener{
		data:  []byte("audio-bytes"),
		asset: testAsset{mime: "audio/mpeg"},
	})

	err := adapter.Send(context.Background(), channel.ChannelConfig{
		ID:    "cfg-4",
		BotID: "bot-4",
		Credentials: map[string]any{
			"appKey":    "key",
			"appSecret": "secret",
			"robotCode": "robot-4",
		},
	}, channel.OutboundMessage{
		Target: "user:union:union-4",
		Message: channel.Message{
			Attachments: []channel.Attachment{{
				Type:        channel.AttachmentAudio,
				ContentHash: "hash-audio",
				Name:        "voice.mp3",
			}},
		},
	})
	if err != nil {
		t.Fatalf("send audio attachment: %v", err)
	}
	if sendBody.MsgKey != "sampleFile" {
		t.Fatalf("unexpected msgKey: %q", sendBody.MsgKey)
	}
	var param AttachmentUploadParam
	if err := json.Unmarshal([]byte(sendBody.MsgParam), &param); err != nil {
		t.Fatalf("decode audio msgParam: %v", err)
	}
	if param.MediaID != "media-audio-1" || param.FileName != "voice.mp3" || param.FileType != "mp3" {
		t.Fatalf("unexpected audio param: %#v", param)
	}
}

func TestSendRejectsMissingAttachmentBytes(t *testing.T) {
	t.Parallel()

	adapter := NewDingTalkAdapter(nil)
	adapter.SetAssetOpener(&testAssetOpener{
		data:  nil,
		asset: testAsset{mime: "application/pdf"},
	})
	err := adapter.Send(context.Background(), channel.ChannelConfig{
		ID:    "cfg-5",
		BotID: "bot-5",
		Credentials: map[string]any{
			"appKey":    "key",
			"appSecret": "secret",
			"robotCode": "robot-5",
		},
	}, channel.OutboundMessage{
		Target: "conversation:cid-5",
		Message: channel.Message{
			Attachments: []channel.Attachment{{
				Type:        channel.AttachmentFile,
				ContentHash: "hash-missing",
			}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "resolved empty bytes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveAttachmentRejectsEmptyResolvedBytes(t *testing.T) {
	t.Parallel()

	adapter := NewDingTalkAdapter(nil)
	adapter.SetAssetOpener(&testAssetOpener{
		data:  []byte{},
		asset: testAsset{mime: "application/pdf"},
	})
	_, err := adapter.ResolveAttachment(context.Background(), channel.ChannelConfig{
		ID:    "cfg-5",
		BotID: "bot-5",
		Credentials: map[string]any{
			"appKey":    "key",
			"appSecret": "secret",
			"robotCode": "robot-5",
		},
	}, channel.Attachment{
		Type:        channel.AttachmentFile,
		ContentHash: "hash-empty",
	})
	if err == nil || !strings.Contains(err.Error(), "resolved empty bytes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type testAsset struct {
	mime string
}

type testAssetOpener struct {
	data  []byte
	asset testAsset
}

func (o *testAssetOpener) Open(_ context.Context, _, _ string) (io.ReadCloser, media.Asset, error) {
	return io.NopCloser(bytes.NewReader(o.data)), media.Asset{Mime: o.asset.mime}, nil
}

func jsonHTTPResponse(v any) *http.Response {
	body, _ := json.Marshal(v)
	return responseWithBody(200, string(body), "application/json")
}

func responseWithBody(status int, body, contentType string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{contentType}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
