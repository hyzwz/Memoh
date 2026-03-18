package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"
)

// App struct holds the application state
type App struct {
	ctx       context.Context
	serverURL string
	token     string
	apiToken  string
	deviceID  string
	userID    string
	botID     string
	botName   string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(_ context.Context) {
	// Cleanup: stop tsnet, file API server, etc.
}

// LoginWithPassword authenticates the user against the Memoh server
func (a *App) LoginWithPassword(serverURL, username, password string) (map[string]any, error) {
	hostname, _ := os.Hostname()
	platform := runtime.GOOS

	payload := map[string]string{
		"username": username,
		"password": password,
		"hostname": hostname,
		"platform": platform,
	}
	body, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 10 * time.Second}
	url := serverURL + "/api/v1/auth/desktop/login"

	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := "login failed"
		if m, ok := result["message"].(string); ok {
			msg = m
		}
		return nil, fmt.Errorf("%s", msg)
	}

	// Store session state
	a.serverURL = serverURL
	a.token, _ = result["access_token"].(string)
	a.apiToken, _ = result["api_token"].(string)
	a.deviceID, _ = result["device_id"].(string)
	a.userID, _ = result["user_id"].(string)
	a.botID, _ = result["bot_id"].(string)
	a.botName, _ = result["bot_name"].(string)

	return result, nil
}

// GetStatus returns the current connection status
func (a *App) GetStatus() map[string]any {
	return map[string]any{
		"connected": a.token != "",
		"serverURL": a.serverURL,
		"deviceID":  a.deviceID,
		"userID":    a.userID,
		"botID":     a.botID,
		"botName":   a.botName,
		"hostname":  getHostname(),
		"platform":  runtime.GOOS,
	}
}

func getHostname() string {
	h, _ := os.Hostname()
	return h
}
