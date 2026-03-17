package main

import (
	"context"
	"fmt"
)

// App struct holds the application state
type App struct {
	ctx context.Context
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
func (a *App) LoginWithPassword(serverURL, username, password, hostname, platform string) (map[string]any, error) {
	// TODO: Implement POST /api/v1/auth/desktop/login
	return nil, fmt.Errorf("not implemented")
}

// GetStatus returns the current connection status
func (a *App) GetStatus() map[string]any {
	return map[string]any{
		"connected": false,
		"tsIP":      "",
		"hostname":  "",
		"platform":  "",
	}
}
