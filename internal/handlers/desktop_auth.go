package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/memohai/memoh/internal/accounts"
	"github.com/memohai/memoh/internal/auth"
	"github.com/memohai/memoh/internal/bots"
)

// DesktopAuthHandler handles authentication for desktop clients.
type DesktopAuthHandler struct {
	accountService *accounts.Service
	botService     *bots.Service
	pool           *pgxpool.Pool
	jwtSecret      string
	expiresIn      time.Duration
	logger         *slog.Logger
}

// DesktopLoginRequest is the payload for desktop client authentication.
type DesktopLoginRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"` //nolint:gosec // intentional: JSON request field
	Hostname  string `json:"hostname"`
	Platform  string `json:"platform"`
	TsIP      string `json:"ts_ip,omitempty"`
	TsNodeKey string `json:"ts_node_key,omitempty"`
}

// DesktopLoginResponse is the response for successful desktop authentication.
type DesktopLoginResponse struct {
	AccessToken string `json:"access_token"` //nolint:gosec // intentional: JWT
	TokenType   string `json:"token_type"`
	ExpiresAt   string `json:"expires_at"`
	APIToken    string `json:"api_token"` //nolint:gosec // intentional: one-time display
	DeviceID    string `json:"device_id"`
	UserID      string `json:"user_id"`
	BotID       string `json:"bot_id,omitempty"`
	BotName     string `json:"bot_name,omitempty"`
}

// NewDesktopAuthHandler creates a new desktop auth handler.
func NewDesktopAuthHandler(
	log *slog.Logger,
	accountService *accounts.Service,
	botService *bots.Service,
	pool *pgxpool.Pool,
	jwtSecret string,
	expiresIn time.Duration,
) *DesktopAuthHandler {
	return &DesktopAuthHandler{
		accountService: accountService,
		botService:     botService,
		pool:           pool,
		jwtSecret:      jwtSecret,
		expiresIn:      expiresIn,
		logger:         log.With(slog.String("handler", "desktop_auth")),
	}
}

// Register registers the desktop auth routes.
func (h *DesktopAuthHandler) Register(e *echo.Echo) {
	e.POST("/api/v1/auth/desktop/login", h.Login)
}

// Login godoc
// @Summary Desktop client login
// @Description Authenticate a desktop client with username/password and register the device
// @Tags desktop-auth
// @Param payload body DesktopLoginRequest true "Desktop login request"
// @Success 200 {object} DesktopLoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/auth/desktop/login [post]
func (h *DesktopAuthHandler) Login(c echo.Context) error {
	if h.accountService == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "account service not configured")
	}

	var req DesktopLoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Hostname = strings.TrimSpace(req.Hostname)
	req.Platform = strings.TrimSpace(req.Platform)

	if req.Username == "" || strings.TrimSpace(req.Password) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "username and password are required")
	}
	if req.Hostname == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "hostname is required")
	}

	// Validate platform
	switch req.Platform {
	case "darwin", "windows", "linux":
		// valid
	case "":
		req.Platform = "unknown"
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "platform must be darwin, windows, or linux")
	}

	ctx := c.Request().Context()

	// Authenticate user
	account, err := h.accountService.Login(ctx, req.Username, req.Password)
	if err != nil {
		if errors.Is(err, accounts.ErrInvalidCredentials) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
		}
		if errors.Is(err, accounts.ErrInactiveAccount) {
			return echo.NewHTTPError(http.StatusUnauthorized, "user is inactive")
		}
		h.logger.Error("login failed", slog.String("error", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "authentication failed")
	}

	// Generate API token for file access
	apiToken, err := generateAPIToken()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	// Hash the API token for storage
	tokenHash, err := bcrypt.GenerateFromPassword([]byte(apiToken), bcrypt.DefaultCost)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to hash token")
	}

	// Upsert the remote device using raw SQL (sqlc types not yet generated)
	tokenExpiry := time.Now().Add(24 * time.Hour)
	var deviceID string
	err = h.pool.QueryRow(ctx, `
		INSERT INTO remote_devices (
			user_id, hostname, ts_ip, ts_node_key, platform, client_version,
			status, api_token_hash, token_issued_at, token_expires_at, paired_at, last_seen_at
		) VALUES ($1, $2, $3, $4, $5, '', 'online', $6, now(), $7, now(), now())
		ON CONFLICT (user_id, hostname) DO UPDATE SET
			ts_ip = EXCLUDED.ts_ip,
			ts_node_key = EXCLUDED.ts_node_key,
			platform = EXCLUDED.platform,
			status = 'online',
			api_token_hash = EXCLUDED.api_token_hash,
			token_issued_at = now(),
			token_expires_at = EXCLUDED.token_expires_at,
			last_seen_at = now()
		RETURNING id`,
		account.ID, req.Hostname, nilIfEmpty(req.TsIP), nilIfEmpty(req.TsNodeKey),
		req.Platform, string(tokenHash), tokenExpiry,
	).Scan(&deviceID)
	if err != nil {
		h.logger.Error("failed to upsert device",
			slog.String("error", err.Error()),
			slog.String("user_id", account.ID),
			slog.String("hostname", req.Hostname),
		)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to register device")
	}

	// Generate JWT for API access
	token, expiresAt, err := auth.GenerateToken(account.ID, h.jwtSecret, h.expiresIn)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate JWT")
	}

	// Find user's first bot (if any)
	var botID, botName string
	if h.botService != nil {
		botList, err := h.botService.ListByOwner(ctx, account.ID)
		if err == nil && len(botList) > 0 {
			botID = botList[0].ID
			botName = botList[0].DisplayName
		}
	}

	h.logger.Info("desktop login successful",
		slog.String("user_id", account.ID),
		slog.String("device_id", deviceID),
		slog.String("hostname", req.Hostname),
		slog.String("platform", req.Platform),
	)

	return c.JSON(http.StatusOK, DesktopLoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt.Format(time.RFC3339),
		APIToken:    apiToken, // plaintext, shown once
		DeviceID:    deviceID,
		UserID:      account.ID,
		BotID:       botID,
		BotName:     botName,
	})
}

// generateAPIToken creates a cryptographically random API token.
func generateAPIToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
