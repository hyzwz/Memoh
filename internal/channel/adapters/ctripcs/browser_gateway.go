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
