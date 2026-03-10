package mcp

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/memohai/memoh/internal/config"
	dbsqlc "github.com/memohai/memoh/internal/db/sqlc"
)

// IdleReaper periodically stops containers that have been idle beyond the
// configured timeout. It runs as a background goroutine.
type IdleReaper struct {
	manager *Manager
	queries *dbsqlc.Queries
	cfg     config.MCPConfig
	logger  *slog.Logger
}

// NewIdleReaper creates a new idle reaper. If IdleTimeoutMinutes is 0, Run() returns immediately.
func NewIdleReaper(logger *slog.Logger, manager *Manager, queries *dbsqlc.Queries, cfg config.MCPConfig) *IdleReaper {
	return &IdleReaper{
		manager: manager,
		queries: queries,
		cfg:     cfg,
		logger:  logger.With(slog.String("component", "idle-reaper")),
	}
}

// Run starts the idle reaper loop. It blocks until ctx is cancelled.
func (r *IdleReaper) Run(ctx context.Context) {
	if r.cfg.IdleTimeoutMinutes <= 0 {
		r.logger.Info("idle reaper disabled (idle_timeout_minutes = 0)")
		return
	}

	interval := 1 * time.Minute
	r.logger.Info("idle reaper started",
		slog.Int("idle_timeout_minutes", r.cfg.IdleTimeoutMinutes),
		slog.Duration("check_interval", interval))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("idle reaper stopped")
			return
		case <-ticker.C:
			r.reap(ctx)
		}
	}
}

func (r *IdleReaper) reap(ctx context.Context) {
	threshold := pgtype.Timestamptz{
		Time:  time.Now().Add(-time.Duration(r.cfg.IdleTimeoutMinutes) * time.Minute),
		Valid: true,
	}

	rows, err := r.queries.ListIdleRunningContainers(ctx, threshold)
	if err != nil {
		r.logger.Error("failed to list idle containers", slog.Any("error", err))
		return
	}
	if len(rows) == 0 {
		return
	}

	r.logger.Info("found idle containers", slog.Int("count", len(rows)))
	for _, row := range rows {
		botID := uuid.UUID(row.BotID.Bytes).String()
		if err := r.manager.StopIdle(ctx, botID); err != nil {
			r.logger.Error("failed to stop idle container",
				slog.String("bot_id", botID), slog.Any("error", err))
			continue
		}
		r.logger.Info("stopped idle container", slog.String("bot_id", botID))
	}
}
