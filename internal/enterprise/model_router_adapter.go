package enterprise

import (
	"context"
	"log/slog"

	"github.com/memohai/memoh/internal/conversation/flow"
	"github.com/memohai/memoh/internal/db"
	"github.com/memohai/memoh/internal/db/sqlc"
)

// ModelRouterAdapter adapts sqlc queries to the flow.ModelRouter interface.
type ModelRouterAdapter struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

func NewModelRouterAdapter(log *slog.Logger, queries *sqlc.Queries) *ModelRouterAdapter {
	return &ModelRouterAdapter{queries: queries, logger: log}
}

func (a *ModelRouterAdapter) ListEnabledRoutes(ctx context.Context, botID string) ([]flow.ModelRouteEntry, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return nil, err
	}
	rows, err := a.queries.ListModelRoutes(ctx, pgBotID)
	if err != nil {
		return nil, err
	}
	var entries []flow.ModelRouteEntry
	for _, r := range rows {
		if !r.IsEnabled {
			continue
		}
		entries = append(entries, flow.ModelRouteEntry{
			ModelID:        uuidToString(r.ModelID),
			Priority:       int(r.Priority),
			ComplexityTier: r.ComplexityTier,
			IsEnabled:      r.IsEnabled,
		})
	}
	return entries, nil
}
