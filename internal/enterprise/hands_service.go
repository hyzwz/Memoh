package enterprise

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/memohai/memoh/internal/auth"
	"github.com/memohai/memoh/internal/db"
	"github.com/memohai/memoh/internal/db/sqlc"
	"github.com/memohai/memoh/internal/handlers"
)

// ChatExecutor abstracts sending a chat query to the agent gateway.
// This decouples hand execution from the concrete Resolver implementation.
type ChatExecutor interface {
	ExecuteChat(ctx context.Context, botID, userID, query, token string) (string, error)
}

// HandsService implements handlers.HandsServiceInterface.
type HandsService struct {
	queries     *sqlc.Queries
	logger      *slog.Logger
	chatExec    ChatExecutor
	jwtSecret   string
}

func NewHandsService(log *slog.Logger, queries *sqlc.Queries, chatExec ChatExecutor, jwtSecret string) *HandsService {
	return &HandsService{
		queries:   queries,
		logger:    log.With(slog.String("service", "hands")),
		chatExec:  chatExec,
		jwtSecret: jwtSecret,
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

	// Fetch and verify the hand belongs to this bot.
	hand, err := s.queries.GetHandByID(ctx, pgHandID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, handlers.ErrHandNotFound
		}
		return nil, err
	}
	if hand.BotID != pgBotID {
		return nil, handlers.ErrHandNotFound
	}
	if !hand.IsEnabled {
		return nil, fmt.Errorf("hand is disabled")
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

	// Resolve the bot owner for JWT token generation.
	bot, err := s.queries.GetBotByID(ctx, pgBotID)
	if err != nil {
		s.updateExecLog(ctx, execLog.ID, "failed", "", "failed to get bot: "+err.Error())
		return nil, fmt.Errorf("failed to get bot: %w", err)
	}
	ownerUserID := uuidToString(bot.OwnerUserID)

	// Generate internal token for the agent gateway call.
	token, _, err := auth.GenerateToken(ownerUserID, s.jwtSecret, handExecTokenTTL)
	if err != nil {
		s.updateExecLog(ctx, execLog.ID, "failed", "", "failed to generate token: "+err.Error())
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Execute the hand content via the agent gateway.
	resultText, execErr := s.chatExec.ExecuteChat(ctx, botID, ownerUserID, hand.Content, token)

	completedAt := time.Now()
	status := "completed"
	errMsg := ""
	if execErr != nil {
		status = "failed"
		errMsg = execErr.Error()
		resultText = ""
	}

	s.updateExecLog(ctx, execLog.ID, status, resultText, errMsg)

	return &handlers.HandExecutionResultDTO{
		Status:       status,
		ResultText:   resultText,
		ErrorMessage: errMsg,
		StartedAt:    now,
		CompletedAt:  completedAt,
	}, execErr
}

const handExecTokenTTL = 10 * time.Minute

func (s *HandsService) updateExecLog(ctx context.Context, id pgtype.UUID, status, resultText, errMsg string) {
	if _, err := s.queries.UpdateHandExecutionLog(ctx, sqlc.UpdateHandExecutionLogParams{
		ID:           id,
		Status:       status,
		ResultText:   pgtype.Text{String: resultText, Valid: resultText != ""},
		ErrorMessage: pgtype.Text{String: errMsg, Valid: errMsg != ""},
		CompletedAt:  pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}); err != nil {
		s.logger.Error("failed to update hand execution log", slog.String("error", err.Error()))
	}
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
