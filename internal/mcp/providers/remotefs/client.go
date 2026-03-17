package remotefs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	fileAPIPort    = 9900
)

// Client communicates with the desktop File API server via Tailscale.
type Client struct {
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient creates a new File API client.
func NewClient(log *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: defaultTimeout},
		logger:     log.With(slog.String("component", "remotefs_client")),
	}
}

// ListFilesRequest is the request for listing files.
type ListFilesRequest struct {
	Path      string `json:"path"`
	Recursive bool   `json:"recursive"`
	Pattern   string `json:"pattern,omitempty"`
}

// ReadFileRequest is the request for reading a file.
type ReadFileRequest struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// SearchFilesRequest is the request for searching files.
type SearchFilesRequest struct {
	Query      string `json:"query"`
	Path       string `json:"path,omitempty"`
	Glob       string `json:"glob,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
}

// ListFiles calls the desktop File API to list directory contents.
func (c *Client) ListFiles(ctx context.Context, device DeviceInfo, req ListFilesRequest) (map[string]any, time.Duration, error) {
	return c.doRequest(ctx, device, "/api/v1/files/list", req)
}

// ReadFile calls the desktop File API to read file content.
func (c *Client) ReadFile(ctx context.Context, device DeviceInfo, req ReadFileRequest) (map[string]any, time.Duration, error) {
	return c.doRequest(ctx, device, "/api/v1/files/read", req)
}

// SearchFiles calls the desktop File API to search file contents.
func (c *Client) SearchFiles(ctx context.Context, device DeviceInfo, req SearchFilesRequest) (map[string]any, time.Duration, error) {
	return c.doRequest(ctx, device, "/api/v1/files/search", req)
}

func (c *Client) doRequest(ctx context.Context, device DeviceInfo, path string, payload any) (map[string]any, time.Duration, error) {
	start := time.Now()

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("http://%s%s", net.JoinHostPort(device.TsIP, strconv.Itoa(fileAPIPort)), path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+device.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, time.Since(start), fmt.Errorf("device %s (%s) unreachable: %w", device.Hostname, device.TsIP, err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, duration, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		if json.Unmarshal(respBody, &errResp) == nil {
			if msg, ok := errResp["error"]; ok {
				return nil, duration, fmt.Errorf("%s", msg)
			}
		}
		return nil, duration, fmt.Errorf("device returned status %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, duration, fmt.Errorf("decode response: %w", err)
	}

	return result, duration, nil
}
