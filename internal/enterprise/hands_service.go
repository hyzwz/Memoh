package enterprise

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/memohai/memoh/internal/db"
	"github.com/memohai/memoh/internal/db/sqlc"
	"github.com/memohai/memoh/internal/handlers"
)

// HandsService implements handlers.HandsServiceInterface.
type HandsService struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

func NewHandsService(log *slog.Logger, queries *sqlc.Queries) *HandsService {
	return &HandsService{
		queries: queries,
		logger:  log.With(slog.String("service", "hands")),
	}
}

func (s *HandsService) List(ctx context.Context, botID string) ([]handlers.HandDTO, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListHands(ctx, pgBotID)
	if err != nil {
		return nil, err
	}
	out := make([]handlers.HandDTO, len(rows))
	for i, r := range rows {
		out[i] = handToDTO(r)
	}
	return out, nil
}

func (s *HandsService) Get(ctx context.Context, botID, id string) (*handlers.HandDTO, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return nil, err
	}
	pgID, err := db.ParseUUID(id)
	if err != nil {
		return nil, err
	}
	hand, err := s.queries.GetHandByID(ctx, pgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, handlers.ErrHandNotFound
		}
		return nil, err
	}
	if hand.BotID != pgBotID {
		return nil, handlers.ErrHandNotFound
	}
	dto := handToDTO(hand)
	return &dto, nil
}

func (s *HandsService) CreateFromMarkdown(ctx context.Context, botID, markdown string) (*handlers.HandDTO, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return nil, err
	}

	name, desc, content := parseHandMarkdown(markdown)

	hand, err := s.queries.CreateHand(ctx, sqlc.CreateHandParams{
		BotID:       pgBotID,
		Name:        name,
		Description: desc,
		Content:     content,
		Type:        "hand",
		IsEnabled:   true,
	})
	if err != nil {
		return nil, err
	}
	dto := handToDTO(hand)
	return &dto, nil
}

func (s *HandsService) Delete(ctx context.Context, botID, id string) error {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return err
	}
	pgID, err := db.ParseUUID(id)
	if err != nil {
		return err
	}
	hand, err := s.queries.GetHandByID(ctx, pgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return handlers.ErrHandNotFound
		}
		return err
	}
	if hand.BotID != pgBotID {
		return handlers.ErrHandNotFound
	}
	return s.queries.DeleteHand(ctx, pgID)
}

func (s *HandsService) Execute(ctx context.Context, botID, handID, triggerType string) (*handlers.HandExecutionResultDTO, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return nil, err
	}
	pgHandID, err := db.ParseUUID(handID)
	if err != nil {
		return nil, err
	}

	if triggerType == "" {
		triggerType = "manual"
	}

	now := time.Now()
	execLog, err := s.queries.CreateHandExecutionLog(ctx, sqlc.CreateHandExecutionLogParams{
		HandID:      pgHandID,
		BotID:       pgBotID,
		TriggerType: triggerType,
		Status:      "running",
	})
	if err != nil {
		return nil, err
	}

	// TODO: Actual hand execution logic would be wired here.
	completedAt := time.Now()
	if _, err := s.queries.UpdateHandExecutionLog(ctx, sqlc.UpdateHandExecutionLogParams{
		ID:          execLog.ID,
		Status:      "completed",
		ResultText:  pgtype.Text{String: "Execution completed", Valid: true},
		CompletedAt: pgtype.Timestamptz{Time: completedAt, Valid: true},
	}); err != nil {
		s.logger.Error("failed to update hand execution log", slog.String("error", err.Error()))
	}

	return &handlers.HandExecutionResultDTO{
		Status:      "completed",
		ResultText:  "Execution completed",
		StartedAt:   now,
		CompletedAt: completedAt,
	}, nil
}

func handToDTO(h sqlc.Hand) handlers.HandDTO {
	var triggers []any
	if h.Triggers != nil {
		_ = json.Unmarshal(h.Triggers, &triggers)
	}
	var metadata map[string]any
	if h.Metadata != nil {
		_ = json.Unmarshal(h.Metadata, &metadata)
	}
	var schedule map[string]any
	if h.SchedulePattern.Valid {
		schedule = map[string]any{
			"pattern":  h.SchedulePattern.String,
			"timezone": db.TextToString(h.ScheduleTimezone),
		}
	}

	return handlers.HandDTO{
		ID:          uuidToString(h.ID),
		Name:        h.Name,
		Description: h.Description,
		Type:        h.Type,
		Content:     h.Content,
		IsEnabled:   h.IsEnabled,
		Schedule:    schedule,
		Triggers:    triggers,
		Metadata:    metadata,
	}
}

func parseHandMarkdown(md string) (name, description, content string) {
	md = strings.TrimSpace(md)
	if !strings.HasPrefix(md, "---") {
		return "unnamed", "", md
	}

	parts := strings.SplitN(md[3:], "---", 2)
	if len(parts) < 2 {
		return "unnamed", "", md
	}

	frontmatter := parts[0]
	content = strings.TrimSpace(parts[1])

	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		}
		if strings.HasPrefix(line, "description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}
	if name == "" {
		name = "unnamed"
	}
	return name, description, content
}
