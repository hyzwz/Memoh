package remotefs

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBAuditLogger writes file access audit logs to the database.
type DBAuditLogger struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewDBAuditLogger creates a database-backed audit logger.
func NewDBAuditLogger(pool *pgxpool.Pool, log *slog.Logger) *DBAuditLogger {
	return &DBAuditLogger{
		pool:   pool,
		logger: log.With(slog.String("component", "remotefs_audit")),
	}
}

const insertAuditSQL = `INSERT INTO remotefs_audit_log (
	device_id, bot_id, user_id, action, file_path,
	file_size, result, error_msg, metadata, duration_ms
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

// LogAccess records a file access operation in the database.
// This is best-effort — errors are logged but do not propagate.
func (l *DBAuditLogger) LogAccess(ctx context.Context, entry AuditEntry) {
	if l.pool == nil {
		return
	}

	// Fire and forget — audit logging should not block tool execution
	go func() {
		_, err := l.pool.Exec(ctx, insertAuditSQL,
			entry.DeviceID,
			entry.BotID,
			entry.UserID,
			entry.Action,
			entry.FilePath,
			nilIfZero(entry.FileSize),
			entry.Result,
			nilIfEmpty(entry.ErrorMsg),
			"{}",
			nilIfZeroInt(entry.DurationMs),
		)
		if err != nil {
			l.logger.Error("failed to write audit log",
				slog.String("error", err.Error()),
				slog.String("action", entry.Action),
				slog.String("path", entry.FilePath),
			)
		}
	}()
}

func nilIfZero(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func nilIfZeroInt(v int) any {
	if v == 0 {
		return nil
	}
	return v
}

func nilIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}
