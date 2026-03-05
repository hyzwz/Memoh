package enterprise

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/memohai/memoh/internal/db"
	"github.com/memohai/memoh/internal/db/sqlc"
)

// CockpitReportGenerator generates daily reports from bot activity data.
type CockpitReportGenerator struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

func NewCockpitReportGenerator(log *slog.Logger, queries *sqlc.Queries) *CockpitReportGenerator {
	return &CockpitReportGenerator{
		queries: queries,
		logger:  log.With(slog.String("service", "cockpit-report")),
	}
}

// GenerateDailyReport creates or updates a daily report for a bot on the given date.
// It gathers message stats, token usage, and hand execution data,
// then computes efficiency metrics and stores the report.
func (g *CockpitReportGenerator) GenerateDailyReport(ctx context.Context, botID, ownerUserID string, reportDate time.Time) error {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return fmt.Errorf("invalid bot id: %w", err)
	}
	pgUserID, err := db.ParseUUID(ownerUserID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	dayStart := time.Date(reportDate.Year(), reportDate.Month(), reportDate.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.AddDate(0, 0, 1)

	tsStart := pgtype.Timestamptz{Time: dayStart, Valid: true}
	tsEnd := pgtype.Timestamptz{Time: dayEnd, Valid: true}

	// Gather daily stats.
	msgStats, err := g.queries.CockpitDailyMessageStats(ctx, sqlc.CockpitDailyMessageStatsParams{
		BotID:    pgBotID,
		DayStart: tsStart,
		DayEnd:   tsEnd,
	})
	if err != nil {
		return fmt.Errorf("message stats: %w", err)
	}

	handStats, err := g.queries.CockpitDailyHandExecutionStats(ctx, sqlc.CockpitDailyHandExecutionStatsParams{
		BotID:    pgBotID,
		DayStart: tsStart,
		DayEnd:   tsEnd,
	})
	if err != nil {
		return fmt.Errorf("hand execution stats: %w", err)
	}

	tokenUsage, err := g.queries.GetTotalTokenUsage(ctx, sqlc.GetTotalTokenUsageParams{
		BotID:    pgBotID,
		FromTime: tsStart,
		ToTime:   tsEnd,
	})
	if err != nil {
		return fmt.Errorf("token usage: %w", err)
	}

	// Compute metrics.
	savedHours := estimateSavedHours(msgStats.ConversationCount, handStats.CompletedExecutions)
	aiMinutes := estimateAITimeMinutes(tokenUsage.InputTokens, tokenUsage.OutputTokens)
	efficiency := computeEfficiencyMultiplier(savedHours, int32(aiMinutes))

	// Build summary and task lists.
	summary := fmt.Sprintf("Daily report: %d messages across %d conversations, %d hand executions (%d completed, %d failed). Estimated %.1f hours saved.",
		msgStats.MessageCount, msgStats.ConversationCount,
		handStats.TotalExecutions, handStats.CompletedExecutions, handStats.FailedExecutions,
		savedHours)

	tasks, _ := json.Marshal(map[string]any{
		"messages":       msgStats.MessageCount,
		"conversations":  msgStats.ConversationCount,
		"hand_completed": handStats.CompletedExecutions,
		"input_tokens":   tokenUsage.InputTokens,
		"output_tokens":  tokenUsage.OutputTokens,
	})
	pendingTasks, _ := json.Marshal(map[string]any{
		"hand_failed": handStats.FailedExecutions,
	})

	_, err = g.queries.UpsertCockpitDailyReport(ctx, sqlc.UpsertCockpitDailyReportParams{
		BotID:                    pgBotID,
		UserID:                   pgUserID,
		ReportDate:               pgtype.Date{Time: dayStart, Valid: true},
		Summary:                  summary,
		Tasks:                    tasks,
		PendingTasks:             pendingTasks,
		TotalEstimatedSavedHours: float64ToNumeric(savedHours),
		TotalAiTimeMinutes:       int32(aiMinutes),
		EfficiencyMultiplier:     float64ToNumeric(efficiency),
		InnovationCount:          int32(handStats.CompletedExecutions),
	})
	if err != nil {
		return fmt.Errorf("upsert report: %w", err)
	}

	g.logger.Info("daily report generated",
		slog.String("bot_id", botID),
		slog.String("date", dayStart.Format("2006-01-02")),
		slog.Float64("saved_hours", savedHours),
		slog.Int("ai_minutes", aiMinutes),
	)
	return nil
}

// estimateSavedHours estimates human hours saved by AI activity.
// Each conversation saves ~12 min (0.2 hr) of human effort.
// Each completed hand execution saves ~30 min (0.5 hr).
func estimateSavedHours(conversations, handExecutions int64) float64 {
	return float64(conversations)*0.2 + float64(handExecutions)*0.5
}

// estimateAITimeMinutes estimates AI interaction time from token counts.
// Rough heuristic: ~50 tokens/sec processing, plus overhead.
func estimateAITimeMinutes(inputTokens, outputTokens int64) int {
	totalTokens := inputTokens + outputTokens
	if totalTokens == 0 {
		return 0
	}
	// Estimate: ~50 tokens/sec effective throughput (including network, thinking)
	seconds := float64(totalTokens) / 50.0
	minutes := int(seconds / 60.0)
	if minutes < 1 && totalTokens > 0 {
		minutes = 1
	}
	return minutes
}

// computeEfficiencyMultiplier computes saved_hours / ai_hours.
// Returns 1.0 if AI time is zero (to avoid division by zero).
func computeEfficiencyMultiplier(savedHours float64, aiMinutes int32) float64 {
	if aiMinutes == 0 {
		if savedHours > 0 {
			return 1.0
		}
		return 0
	}
	aiHours := float64(aiMinutes) / 60.0
	return savedHours / aiHours
}
