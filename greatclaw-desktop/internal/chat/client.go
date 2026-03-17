package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// Client communicates with the Memoh server for chat operations.
type Client struct {
	baseURL string
	token   string
	client  *http.Client
	logger  *slog.Logger
}

// NewClient creates a new chat client.
func NewClient(baseURL, token string, log *slog.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		client:  &http.Client{},
		logger:  log.With(slog.String("component", "chat_client")),
	}
}

// SendMessage sends a message to the bot via the Memoh API.
func (c *Client) SendMessage(ctx context.Context, botID, message string) (string, error) {
	payload := map[string]any{
		"bot_id":  botID,
		"message": message,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/chat/send", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server returned %d: %s", resp.StatusCode, string(data))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if id, ok := result["message_id"].(string); ok {
		return id, nil
	}
	return "", nil
}
