package remotefs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	mcpgw "github.com/memohai/memoh/internal/mcp"
)

// ErrNoOnlineDevices indicates no online devices were found for the user.
var ErrNoOnlineDevices = errors.New("no online devices found — is your desktop client running?")

// ErrMultipleDevices indicates multiple online devices were found.
var ErrMultipleDevices = errors.New("multiple online devices found")

// BotOwnerResolver looks up the owner user ID for a bot.
type BotOwnerResolver interface {
	GetBotOwnerID(ctx context.Context, botID string) (string, error)
}

// DeviceStore queries remote devices from the database.
type DeviceStore interface {
	GetOnlineDevices(ctx context.Context, userID string) ([]StoredDevice, error)
}

// StoredDevice holds device data from the database.
type StoredDevice struct {
	ID       string
	UserID   string
	Hostname string
	TsIP     string
	Token    string
}

// DBDeviceResolver resolves devices by looking up the bot owner's online devices.
type DBDeviceResolver struct {
	botResolver BotOwnerResolver
	deviceStore DeviceStore
	logger      *slog.Logger
}

// NewDBDeviceResolver creates a new database-backed device resolver.
func NewDBDeviceResolver(botResolver BotOwnerResolver, deviceStore DeviceStore, log *slog.Logger) *DBDeviceResolver {
	return &DBDeviceResolver{
		botResolver: botResolver,
		deviceStore: deviceStore,
		logger:      log.With(slog.String("component", "device_resolver")),
	}
}

// ResolveDevice finds the appropriate device for a bot's file operations.
// It uses the session user ID when available (multi-member bots), falling back to the bot owner.
func (r *DBDeviceResolver) ResolveDevice(ctx context.Context, botID string, session mcpgw.ToolSessionContext) (DeviceInfo, error) {
	// Prefer session user (the actual requester) over bot owner for multi-member bots
	userID := session.UserID
	if userID == "" {
		ownerID, err := r.botResolver.GetBotOwnerID(ctx, botID)
		if err != nil {
			return DeviceInfo{}, fmt.Errorf("resolve bot owner: %w", err)
		}
		userID = ownerID
	}

	// Get online devices for the user
	devices, err := r.deviceStore.GetOnlineDevices(ctx, userID)
	if err != nil {
		return DeviceInfo{}, fmt.Errorf("query devices: %w", err)
	}

	if len(devices) == 0 {
		return DeviceInfo{}, ErrNoOnlineDevices
	}

	// If request comes from a desktop client, prefer that device
	if session.CurrentPlatform == "desktop" && session.ChannelIdentityID != "" {
		for _, d := range devices {
			if d.ID == session.ChannelIdentityID {
				return toDeviceInfo(d), nil
			}
		}
	}

	// Single device: use it directly
	if len(devices) == 1 {
		return toDeviceInfo(devices[0]), nil
	}

	// Multiple devices: return error so the agent can ask the user
	names := make([]string, len(devices))
	for i, d := range devices {
		names[i] = fmt.Sprintf("%s (%s)", d.Hostname, d.TsIP)
	}
	return DeviceInfo{}, fmt.Errorf("%w: %v — ask the user which device to use", ErrMultipleDevices, names)
}

func toDeviceInfo(d StoredDevice) DeviceInfo {
	return DeviceInfo{
		DeviceID: d.ID,
		UserID:   d.UserID,
		TsIP:     d.TsIP,
		Token:    d.Token,
		Hostname: d.Hostname,
	}
}
