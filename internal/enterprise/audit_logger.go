package enterprise

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/memohai/memoh/internal/db"
	"github.com/memohai/memoh/internal/db/sqlc"
)

// AuditLogger provides a simple API for writing audit log entries.
type AuditLogger struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

func NewAuditLogger(log *slog.Logger, queries *sqlc.Queries) *AuditLogger {
	return &AuditLogger{
		queries: queries,
		logger:  log.With(slog.String("component", "audit-logger")),
	}
}

// Log writes an audit log entry. Errors are logged but not returned
// to avoid disrupting the primary operation.
func (a *AuditLogger) Log(ctx context.Context, userID, botID, action, resourceType, resourceID, ipAddress, userAgent string, detail any) {
	pgUserID, _ := db.ParseUUID(userID)
	pgBotID, _ := db.ParseUUID(botID)

	var detailJSON []byte
	if detail != nil {
		detailJSON, _ = json.Marshal(detail)
	}
	if detailJSON == nil {
		detailJSON = []byte("{}")
	}

	_, err := a.queries.InsertAuditLog(ctx, sqlc.InsertAuditLogParams{
		UserID:       pgUserID,
		BotID:        pgBotID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   pgtype.Text{String: resourceID, Valid: resourceID != ""},
		Detail:       detailJSON,
		IpAddress:    pgtype.Text{String: ipAddress, Valid: ipAddress != ""},
		UserAgent:    pgtype.Text{String: userAgent, Valid: userAgent != ""},
	})
	if err != nil {
		a.logger.Error("failed to write audit log",
			slog.String("action", action),
			slog.String("resource_type", resourceType),
			slog.String("error", err.Error()),
		)
	}
}
