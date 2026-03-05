package wecom

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/memohai/memoh/internal/channel"
)

var errConfigNotFound = errors.New("wecom config not found")

type wecomConfigStore interface {
	ListConfigsByType(ctx context.Context, channelType channel.ChannelType) ([]channel.ChannelConfig, error)
}

type wecomInboundManager interface {
	HandleInbound(ctx context.Context, cfg channel.ChannelConfig, msg channel.InboundMessage) error
}

// WeComWebhookHandler receives WeCom callback requests and routes them into
// the channel manager's inbound pipeline.
type WeComWebhookHandler struct {
	logger  *slog.Logger
	store   wecomConfigStore
	manager wecomInboundManager
}

// NewWeComWebhookHandler creates a WeComWebhookHandler with the given dependencies.
func NewWeComWebhookHandler(log *slog.Logger, store wecomConfigStore, manager wecomInboundManager) *WeComWebhookHandler {
	if log == nil {
		log = slog.Default()
	}
	return &WeComWebhookHandler{
		logger:  log.With(slog.String("handler", "wecom_webhook")),
		store:   store,
		manager: manager,
	}
}

// NewWeComWebhookServerHandler is a DI-friendly constructor for fx/dig.
func NewWeComWebhookServerHandler(log *slog.Logger, store *channel.Store, manager *channel.Manager) *WeComWebhookHandler {
	return NewWeComWebhookHandler(log, store, manager)
}

// Register registers webhook callback routes.
func (h *WeComWebhookHandler) Register(e *echo.Echo) {
	e.GET("/channels/wecom/webhook/:config_id", h.HandleVerify)
	e.POST("/channels/wecom/webhook/:config_id", h.HandleMessage)
}

func (h *WeComWebhookHandler) findConfig(ctx context.Context, configID string) (channel.ChannelConfig, error) {
	configs, err := h.store.ListConfigsByType(ctx, Type)
	if err != nil {
		return channel.ChannelConfig{}, err
	}
	for _, c := range configs {
		if c.ID == configID {
			return c, nil
		}
	}
	return channel.ChannelConfig{}, fmt.Errorf("%w: %s", errConfigNotFound, configID)
}

// HandleVerify handles WeCom URL verification (GET).
func (h *WeComWebhookHandler) HandleVerify(c echo.Context) error {
	configID := strings.TrimSpace(c.Param("config_id"))
	if configID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "config_id required")
	}

	cfg, err := h.findConfig(c.Request().Context(), configID)
	if err != nil {
		if errors.Is(err, errConfigNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "config lookup failed")
	}

	wecomCfg, err := parseConfig(cfg.Credentials)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "invalid wecom config")
	}

	aesKey, err := DecodeAESKey(wecomCfg.EncodingAESKey)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "invalid encoding key")
	}

	q := c.QueryParams()
	msgSignature := q.Get("msg_signature")
	timestamp := q.Get("timestamp")
	nonce := q.Get("nonce")
	echoStr := q.Get("echostr")

	sig := CalcSignature(wecomCfg.Token, timestamp, nonce, echoStr)
	if !constantTimeCompare(sig, msgSignature) {
		return echo.NewHTTPError(http.StatusForbidden, "invalid signature")
	}

	decrypted, receivedCorpID, err := Decrypt(aesKey, echoStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "decrypt failed")
	}
	if receivedCorpID != wecomCfg.CorpID {
		return echo.NewHTTPError(http.StatusForbidden, "corpID mismatch")
	}

	return c.String(http.StatusOK, decrypted)
}

// HandleMessage handles WeCom inbound messages (POST).
func (h *WeComWebhookHandler) HandleMessage(c echo.Context) error {
	configID := strings.TrimSpace(c.Param("config_id"))
	if configID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "config_id required")
	}

	cfg, err := h.findConfig(c.Request().Context(), configID)
	if err != nil {
		if errors.Is(err, errConfigNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "config lookup failed")
	}

	wecomCfg, err := parseConfig(cfg.Credentials)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "invalid wecom config")
	}

	aesKey, err := DecodeAESKey(wecomCfg.EncodingAESKey)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "invalid encoding key")
	}

	q := c.QueryParams()
	msgSignature := q.Get("msg_signature")
	timestamp := q.Get("timestamp")
	nonce := q.Get("nonce")

	body, err := io.ReadAll(io.LimitReader(c.Request().Body, maxBodySize))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "read body failed")
	}

	var envelope struct {
		XMLName    xml.Name `xml:"xml"`
		ToUserName string   `xml:"ToUserName"`
		Encrypt    string   `xml:"Encrypt"`
		AgentID    string   `xml:"AgentID"`
	}
	if err := xml.Unmarshal(body, &envelope); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid XML")
	}

	sig := CalcSignature(wecomCfg.Token, timestamp, nonce, envelope.Encrypt)
	if !constantTimeCompare(sig, msgSignature) {
		return echo.NewHTTPError(http.StatusForbidden, "invalid signature")
	}

	decrypted, receivedCorpID, err := Decrypt(aesKey, envelope.Encrypt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "decrypt failed")
	}
	if receivedCorpID != wecomCfg.CorpID {
		return echo.NewHTTPError(http.StatusForbidden, "corpID mismatch")
	}

	var msg WeComMessage
	if err := xml.Unmarshal([]byte(decrypted), &msg); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "parse message failed")
	}

	// Only process text messages for now.
	if msg.MsgType != "text" {
		return c.String(http.StatusOK, "success")
	}

	// Use "userid:<from>" as conversation ID to isolate per-user context.
	conversationID := "userid:" + msg.FromUserName

	inbound := channel.InboundMessage{
		Channel: Type,
		Message: channel.Message{
			ID:   msg.MsgID,
			Text: msg.Content,
		},
		BotID:       cfg.BotID,
		ReplyTarget: conversationID,
		Sender: channel.Identity{
			SubjectID:   msg.FromUserName,
			DisplayName: msg.FromUserName,
		},
		Conversation: channel.Conversation{
			ID:   conversationID,
			Type: "direct",
		},
		ReceivedAt: time.Unix(msg.CreateTime, 0),
		Source:     "wecom",
	}

	if err := h.manager.HandleInbound(c.Request().Context(), cfg, inbound); err != nil {
		h.logger.Error("wecom inbound failed", slog.String("error", err.Error()))
	}

	return c.String(http.StatusOK, "success")
}
