package ctripcs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type browserActionRunner interface {
	Exists(ctx context.Context, contextID string) (bool, error)
	Navigate(ctx context.Context, contextID string, url string) error
	Evaluate(ctx context.Context, contextID string, script string) ([]byte, error)
}

type browserGatewayClient struct {
	baseURL string
	client  *http.Client
}

type browserGatewayActionRequest struct {
	Action         string   `json:"action"`
	URL            string   `json:"url,omitempty"`
	Selector       string   `json:"selector,omitempty"`
	Text           string   `json:"text,omitempty"`
	Script         string   `json:"script,omitempty"`
	Key            string   `json:"key,omitempty"`
	Value          string   `json:"value,omitempty"`
	TargetSelector string   `json:"target_selector,omitempty"`
	Files          []string `json:"files,omitempty"`
	FullPage       bool     `json:"full_page,omitempty"`
	TabIndex       int      `json:"tab_index,omitempty"`
	Direction      string   `json:"direction,omitempty"`
	Amount         int      `json:"amount,omitempty"`
	Timeout        int      `json:"timeout,omitempty"`
}

type browserGatewayActionResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Result json.RawMessage `json:"result"`
	} `json:"data"`
	Error string `json:"error,omitempty"`
}

type browserGatewayExistsResponse struct {
	Exists bool `json:"exists"`
}

func (c *browserGatewayClient) Exists(ctx context.Context, contextID string) (bool, error) {
	if c == nil {
		return false, fmt.Errorf("browser gateway client is not configured")
	}
	contextID = strings.TrimSpace(contextID)
	if contextID == "" {
		return false, fmt.Errorf("browser context id is required")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	if baseURL == "" {
		return false, fmt.Errorf("browser gateway base url is required")
	}

	endpoint := baseURL + "/context/" + url.PathEscape(contextID) + "/exists"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("create browser gateway exists request: %w", err)
	}

	httpClient := c.client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("browser gateway exists failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload browserGatewayExistsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return false, fmt.Errorf("decode browser gateway exists response: %w", err)
	}
	return payload.Exists, nil
}

func (c *browserGatewayClient) Navigate(ctx context.Context, contextID string, url string) error {
	_, err := c.doAction(ctx, contextID, browserGatewayActionRequest{
		Action: "navigate",
		URL:    url,
	})
	return err
}

func (c *browserGatewayClient) Evaluate(ctx context.Context, contextID string, script string) ([]byte, error) {
	raw, err := c.doAction(ctx, contextID, browserGatewayActionRequest{
		Action: "evaluate",
		Script: script,
	})
	if err != nil {
		return nil, err
	}

	var response browserGatewayActionResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode browser gateway evaluate response: %w", err)
	}
	if !response.Success {
		if trimmed := strings.TrimSpace(response.Error); trimmed != "" {
			return nil, fmt.Errorf("browser gateway evaluate failed: %s", trimmed)
		}
		return nil, fmt.Errorf("browser gateway evaluate failed")
	}
	if len(response.Data.Result) == 0 {
		return []byte("null"), nil
	}
	return append([]byte(nil), response.Data.Result...), nil
}

func (c *browserGatewayClient) doAction(ctx context.Context, contextID string, req browserGatewayActionRequest) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("browser gateway client is not configured")
	}
	contextID = strings.TrimSpace(contextID)
	if contextID == "" {
		return nil, fmt.Errorf("browser context id is required")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("browser gateway base url is required")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode browser gateway action request: %w", err)
	}

	endpoint := baseURL + "/context/" + url.PathEscape(contextID) + "/action"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create browser gateway request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := c.client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("browser gateway action failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return respBody, nil
}
