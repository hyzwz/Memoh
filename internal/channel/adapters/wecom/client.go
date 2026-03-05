package wecom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const defaultBaseURL = "https://qyapi.weixin.qq.com"

// Client provides HTTP access to the WeCom API.
type Client struct {
	baseURL    string
	corpID     string
	corpSecret string
	agentID    int
	httpClient *http.Client
}

// NewClient creates a WeCom API client.
func NewClient(baseURL, corpID, corpSecret string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL:    baseURL,
		corpID:     corpID,
		corpSecret: corpSecret,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// SetAgentID sets the agent ID for outbound messages.
func (c *Client) SetAgentID(id int) {
	c.agentID = id
}

type tokenResponse struct {
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// GetAccessToken retrieves an access token from WeCom.
func (c *Client) GetAccessToken(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/cgi-bin/gettoken?corpid=%s&corpsecret=%s", c.baseURL, c.corpID, c.corpSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("wecom gettoken: %s (errcode: %d)", result.ErrMsg, result.ErrCode)
	}
	return result.AccessToken, nil
}

type sendResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// SendTextMessage sends a text message to a WeCom user.
func (c *Client) SendTextMessage(ctx context.Context, toUser, content string) error {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	body := map[string]any{
		"touser":  toUser,
		"msgtype": "text",
		"agentid": c.agentID,
		"text": map[string]string{
			"content": content,
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/cgi-bin/message/send?access_token=%s", c.baseURL, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result sendResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("wecom send: %s (errcode: %d)", result.ErrMsg, result.ErrCode)
	}
	return nil
}
