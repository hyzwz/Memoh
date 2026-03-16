package enterprise

import (
	"context"
	"log/slog"

	"github.com/memohai/memoh/internal/db"
	"github.com/memohai/memoh/internal/db/sqlc"
)

// ModelRouteEntry represents a single model routing rule (mirrors flow.ModelRouteEntry).
type ModelRouteEntry struct {
	ModelID        string
	Priority       int
	ComplexityTier string
	IsEnabled      bool
}

// ModelRouterAdapter adapts sqlc queries to list enabled model routes for a bot.
type ModelRouterAdapter struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

func NewModelRouterAdapter(log *slog.Logger, queries *sqlc.Queries) *ModelRouterAdapter {
	return &ModelRouterAdapter{queries: queries, logger: log}
}

func (a *ModelRouterAdapter) ListEnabledRoutes(ctx context.Context, botID string) ([]ModelRouteEntry, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return nil, err
	}
	rows, err := a.queries.ListModelRoutes(ctx, pgBotID)
	if err != nil {
		return nil, err
	}
	var entries []ModelRouteEntry
	for _, r := range rows {
		if !r.IsEnabled {
			continue
		}
		entries = append(entries, ModelRouteEntry{
			ModelID:        uuidToString(r.ModelID),
			Priority:       int(r.Priority),
			ComplexityTier: r.ComplexityTier,
			IsEnabled:      r.IsEnabled,
		})
	}
	return entries, nil
}
