package tailscale

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// HeartbeatConfig configures the heartbeat sender.
type HeartbeatConfig struct {
	ServerURL string
	DeviceID  string
	Token     string
	TsIP      string
	Interval  time.Duration
}

// Heartbeat sends periodic heartbeats to the Memoh server.
type Heartbeat struct {
	config HeartbeatConfig
	client *http.Client
	logger *slog.Logger
	cancel context.CancelFunc
}

// NewHeartbeat creates a heartbeat sender.
func NewHeartbeat(cfg HeartbeatConfig, log *slog.Logger) *Heartbeat {
	if cfg.Interval == 0 {
		cfg.Interval = 30 * time.Second
	}
	return &Heartbeat{
		config: cfg,
		client: &http.Client{Timeout: 10 * time.Second},
		logger: log.With(slog.String("component", "heartbeat")),
	}
}

// Start begins sending heartbeats in the background.
func (h *Heartbeat) Start(ctx context.Context) {
	ctx, h.cancel = context.WithCancel(ctx)
	go h.loop(ctx)
}

// Stop cancels the heartbeat loop.
func (h *Heartbeat) Stop() {
	if h.cancel != nil {
		h.cancel()
	}
}

func (h *Heartbeat) loop(ctx context.Context) {
	ticker := time.NewTicker(h.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.send(ctx)
		}
	}
}

func (h *Heartbeat) send(ctx context.Context) {
	payload := map[string]any{
		"device_id": h.config.DeviceID,
		"ts_ip":     h.config.TsIP,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.config.ServerURL+"/api/v1/devices/heartbeat", bytes.NewReader(body))
	if err != nil {
		h.logger.Error("failed to create heartbeat request", slog.String("error", err.Error()))
		return
	}
	req.Header.Set("Authorization", "Bearer "+h.config.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.Warn("heartbeat failed", slog.String("error", err.Error()))
		return
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		h.logger.Warn("heartbeat returned non-200", slog.Int("status", resp.StatusCode))
	}
}
