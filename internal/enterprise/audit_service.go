package enterprise

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/memohai/memoh/internal/db"
	"github.com/memohai/memoh/internal/db/sqlc"
	"github.com/memohai/memoh/internal/handlers"
)

// AuditQueryService implements handlers.AuditQueryServiceInterface.
type AuditQueryService struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

func NewAuditQueryService(log *slog.Logger, queries *sqlc.Queries) *AuditQueryService {
	return &AuditQueryService{
		queries: queries,
		logger:  log.With(slog.String("service", "audit")),
	}
}

func (s *AuditQueryService) ListAuditLogs(ctx context.Context, query handlers.AuditLogQuery) ([]handlers.AuditLogDTO, int64, error) {
	pgBotID, err := db.ParseUUID(query.BotID)
	if err != nil {
		return nil, 0, err
	}

	params := sqlc.ListAuditLogsParams{
		BotID:      pgBotID,
		PageSize:   int32(query.Limit),
		PageOffset: int32(query.Offset),
	}
	if query.UserID != "" {
		pgUserID, err := db.ParseUUID(query.UserID)
		if err != nil {
			return nil, 0, err
		}
		params.UserID = pgUserID
	}
	if query.Action != "" {
		params.Action = pgtype.Text{String: query.Action, Valid: true}
	}

	rows, err := s.queries.ListAuditLogs(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.queries.CountAuditLogs(ctx, sqlc.CountAuditLogsParams{
		BotID:        pgBotID,
		UserID:       params.UserID,
		Action:       params.Action,
		ResourceType: pgtype.Text{},
	})
	if err != nil {
		return nil, 0, err
	}

	out := make([]handlers.AuditLogDTO, len(rows))
	for i, r := range rows {
		out[i] = auditLogToDTO(r)
	}
	return out, count, nil
}

func auditLogToDTO(a sqlc.AuditLog) handlers.AuditLogDTO {
	var detail map[string]any
	if a.Detail != nil {
		_ = json.Unmarshal(a.Detail, &detail)
	}

	return handlers.AuditLogDTO{
		ID:        uuidToString(a.ID),
		UserID:    uuidToString(a.UserID),
		BotID:     uuidToString(a.BotID),
		Action:    a.Action,
		Detail:    detail,
		Timestamp: db.TimeFromPg(a.CreatedAt),
	}
}
