package wecom

import (
	"crypto/subtle"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

// WeComMessage represents a decrypted inbound message from WeCom.
type WeComMessage struct {
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	CreateTime   int64  `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
	MsgID        string `xml:"MsgId"`
	AgentID      string `xml:"AgentID"`
	PicURL       string `xml:"PicUrl"`
	MediaID      string `xml:"MediaId"`
	Event        string `xml:"Event"`
	EventKey     string `xml:"EventKey"`
}

// MessageHandler is a callback for received messages.
type MessageHandler func(msg *WeComMessage)

// WebhookHandler handles WeCom callback requests (URL verification and message receiving).
type WebhookHandler struct {
	token          string
	encodingAESKey string
	corpID         string
	aesKey         []byte
	handler        MessageHandler
}

// NewWebhookHandler creates a webhook handler for WeCom callbacks.
// Returns error if encodingAESKey is invalid.
func NewWebhookHandler(token, encodingAESKey, corpID string) (*WebhookHandler, error) {
	aesKey, err := DecodeAESKey(encodingAESKey)
	if err != nil {
		return nil, fmt.Errorf("invalid encodingAESKey: %w", err)
	}
	return &WebhookHandler{
		token:          token,
		encodingAESKey: encodingAESKey,
		corpID:         corpID,
		aesKey:         aesKey,
	}, nil
}

// maxBodySize limits the webhook request body to 1MB to prevent DoS.
const maxBodySize = 1 << 20

// constantTimeCompare compares two hex-encoded SHA1 signatures in constant time.
func constantTimeCompare(a, b string) bool {
	aBytes, errA := hex.DecodeString(a)
	bBytes, errB := hex.DecodeString(b)
	if errA != nil || errB != nil {
		return false
	}
	return subtle.ConstantTimeCompare(aBytes, bBytes) == 1
}

// OnMessage registers a handler for incoming messages.
func (h *WebhookHandler) OnMessage(fn MessageHandler) {
	h.handler = fn
}

// ServeHTTP implements http.Handler.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleVerify(w, r)
	case http.MethodPost:
		h.handleMessage(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleVerify handles WeCom URL verification (GET request).
func (h *WebhookHandler) handleVerify(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	msgSignature := q.Get("msg_signature")
	timestamp := q.Get("timestamp")
	nonce := q.Get("nonce")
	echoStr := q.Get("echostr")

	// Verify signature (constant-time).
	sig := CalcSignature(h.token, timestamp, nonce, echoStr)
	if !constantTimeCompare(sig, msgSignature) {
		http.Error(w, "invalid signature", http.StatusForbidden)
		return
	}

	// Decrypt echostr and verify corpID.
	decrypted, receivedCorpID, err := Decrypt(h.aesKey, echoStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("decrypt failed: %v", err), http.StatusBadRequest)
		return
	}
	if receivedCorpID != h.corpID {
		http.Error(w, "corpID mismatch", http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(decrypted))
}

// handleMessage handles WeCom inbound messages (POST request).
func (h *WebhookHandler) handleMessage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	msgSignature := q.Get("msg_signature")
	timestamp := q.Get("timestamp")
	nonce := q.Get("nonce")

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	// Parse encrypted XML envelope.
	var envelope struct {
		XMLName    xml.Name `xml:"xml"`
		ToUserName string   `xml:"ToUserName"`
		Encrypt    string   `xml:"Encrypt"`
		AgentID    string   `xml:"AgentID"`
	}
	if err := xml.Unmarshal(body, &envelope); err != nil {
		http.Error(w, "invalid XML", http.StatusBadRequest)
		return
	}

	// Verify signature (constant-time).
	sig := CalcSignature(h.token, timestamp, nonce, envelope.Encrypt)
	if !constantTimeCompare(sig, msgSignature) {
		http.Error(w, "invalid signature", http.StatusForbidden)
		return
	}

	// Decrypt message and verify corpID.
	decrypted, receivedCorpID, err := Decrypt(h.aesKey, envelope.Encrypt)
	if err != nil {
		http.Error(w, fmt.Sprintf("decrypt failed: %v", err), http.StatusBadRequest)
		return
	}
	if receivedCorpID != h.corpID {
		http.Error(w, "corpID mismatch", http.StatusForbidden)
		return
	}

	// Parse the decrypted XML message.
	var msg WeComMessage
	if err := xml.Unmarshal([]byte(decrypted), &msg); err != nil {
		http.Error(w, fmt.Sprintf("parse message failed: %v", err), http.StatusBadRequest)
		return
	}

	if h.handler != nil {
		h.handler(&msg)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))
}
