package enterprise

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/memohai/memoh/internal/db"
	"github.com/memohai/memoh/internal/db/sqlc"
	"github.com/memohai/memoh/internal/handlers"
)

// CostTrackingService implements handlers.CostTrackingServiceInterface.
type CostTrackingService struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

func NewCostTrackingService(log *slog.Logger, queries *sqlc.Queries) *CostTrackingService {
	return &CostTrackingService{
		queries: queries,
		logger:  log.With(slog.String("service", "cost-tracking")),
	}
}

func (s *CostTrackingService) ListBudgets(ctx context.Context, scopeType string) ([]handlers.BudgetDTO, error) {
	scope := pgtype.Text{}
	if scopeType != "" {
		scope = pgtype.Text{String: scopeType, Valid: true}
	}
	rows, err := s.queries.ListBudgets(ctx, scope)
	if err != nil {
		return nil, err
	}
	out := make([]handlers.BudgetDTO, len(rows))
	for i, r := range rows {
		out[i] = budgetToDTO(r)
	}
	return out, nil
}

func (s *CostTrackingService) CreateBudget(ctx context.Context, req handlers.CreateBudgetRequest) (*handlers.BudgetDTO, error) {
	budget, err := s.queries.CreateBudget(ctx, sqlc.CreateBudgetParams{
		ScopeType:      req.ScopeType,
		ScopeID:        req.ScopeID,
		Period:         req.Period,
		LimitAmount:    float64ToNumeric(req.LimitAmount),
		AlertThreshold: float64ToNumeric(req.AlertThreshold),
		ActionOnExceed: req.ActionOnExceed,
		IsEnabled:      true,
	})
	if err != nil {
		return nil, err
	}
	dto := budgetToDTO(budget)
	return &dto, nil
}

func (s *CostTrackingService) DeleteBudget(ctx context.Context, id string) error {
	pgID, err := parseUUID(id)
	if err != nil {
		return err
	}
	return s.queries.DeleteBudget(ctx, pgID)
}

func (s *CostTrackingService) CheckBudget(ctx context.Context, scopeType, scopeID string) (*handlers.BudgetCheckDTO, error) {
	budgets, err := s.queries.ListBudgetsByScope(ctx, sqlc.ListBudgetsByScopeParams{
		ScopeType: scopeType,
		ScopeID:   scopeID,
	})
	if err != nil {
		return nil, err
	}
	if len(budgets) == 0 {
		return &handlers.BudgetCheckDTO{
			Allowed: true,
			Spent:   0,
			Limit:   0,
		}, nil
	}

	// Check ALL budgets and enforce the strictest one.
	// Only action_on_exceed="block" actually denies; "warn"/"downgrade" set Alert.
	// NOTE: Currently only scopeType="bot" computes real spending.
	// Other scope types (user, department) will need their own spending queries.
	result := &handlers.BudgetCheckDTO{
		Allowed: true,
	}

	for _, budget := range budgets {
		limit := numericToFloat64(budget.LimitAmount)
		threshold := numericToFloat64(budget.AlertThreshold)

		spent := 0.0
		if scopeType == "bot" {
			var spendErr error
			spent, spendErr = s.computeBotSpending(ctx, scopeID, budget.Period)
			if spendErr != nil {
				// Fail closed: if we can't determine spending, deny access.
				return &handlers.BudgetCheckDTO{
					Allowed: false,
					Alert:   true,
					Action:  "block",
				}, nil
			}
		}

		percentage := 0.0
		if limit > 0 {
			percentage = (spent / limit) * 100
		}

		exceeded := limit > 0 && spent >= limit

		if exceeded {
			switch budget.ActionOnExceed {
			case "block":
				// Block action: deny access. Always overrides warn/downgrade.
				result.Allowed = false
				result.Spent = spent
				result.Limit = limit
				result.Percentage = percentage
				result.Action = "block"
			case "warn", "downgrade":
				// Non-blocking: set alert but don't deny.
				result.Alert = true
				// Update reported values if this is the tightest exceeded budget.
				if result.Allowed && (result.Limit == 0 || percentage > result.Percentage) {
					result.Spent = spent
					result.Limit = limit
					result.Percentage = percentage
					result.Action = budget.ActionOnExceed
				}
			}
		}

		if threshold > 0 && percentage >= threshold*100 {
			result.Alert = true
		}

		// Track the tightest non-exceeded budget for reporting.
		if !exceeded && result.Allowed && (result.Limit == 0 || (limit > 0 && percentage > result.Percentage)) {
			result.Spent = spent
			result.Limit = limit
			result.Percentage = percentage
			result.Action = budget.ActionOnExceed
		}
	}

	return result, nil
}

// computeBotSpending calculates total spending for a bot in the current period
// using token usage data from both messages and heartbeat logs.
// Returns an error if spending cannot be determined (caller should fail closed).
func (s *CostTrackingService) computeBotSpending(ctx context.Context, botID, period string) (float64, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return 0, fmt.Errorf("invalid bot id: %w", err)
	}

	now := time.Now()
	start, end := periodToTimeRange(period, now)

	timeParams := pgtype.Timestamptz{Time: start, Valid: true}
	timeEnd := pgtype.Timestamptz{Time: end, Valid: true}

	// Fetch message token usage.
	msgUsage, err := s.queries.GetTotalTokenUsage(ctx, sqlc.GetTotalTokenUsageParams{
		BotID:    pgBotID,
		FromTime: timeParams,
		ToTime:   timeEnd,
	})
	if err != nil {
		s.logger.Error("failed to get message token usage", slog.String("error", err.Error()))
		return 0, fmt.Errorf("message token usage: %w", err)
	}

	// Fetch heartbeat token usage.
	hbUsage, err := s.queries.GetTotalHeartbeatTokenUsage(ctx, sqlc.GetTotalHeartbeatTokenUsageParams{
		BotID:    pgBotID,
		FromTime: timeParams,
		ToTime:   timeEnd,
	})
	if err != nil {
		s.logger.Error("failed to get heartbeat token usage", slog.String("error", err.Error()))
		return 0, fmt.Errorf("heartbeat token usage: %w", err)
	}

	// Default pricing: $3/Mtok input, $15/Mtok output (Claude Sonnet-class).
	const defaultInputPrice = 3.0
	const defaultOutputPrice = 15.0
	const defaultCacheReadPrice = 0.3
	const defaultCacheWritePrice = 3.75
	const defaultReasoningPrice = 15.0

	// Sum message + heartbeat token counts.
	totalInput := msgUsage.InputTokens + hbUsage.InputTokens
	totalOutput := msgUsage.OutputTokens + hbUsage.OutputTokens
	totalCacheRead := msgUsage.CacheReadTokens + hbUsage.CacheReadTokens
	totalCacheWrite := msgUsage.CacheWriteTokens + hbUsage.CacheWriteTokens
	totalReasoning := msgUsage.ReasoningTokens + hbUsage.ReasoningTokens

	cost := computeTokenCost(
		totalInput, totalOutput,
		totalCacheRead, totalCacheWrite, totalReasoning,
		defaultInputPrice, defaultOutputPrice,
		defaultCacheReadPrice, defaultCacheWritePrice, defaultReasoningPrice,
	)
	return cost, nil
}

func budgetToDTO(b sqlc.Budget) handlers.BudgetDTO {
	return handlers.BudgetDTO{
		ID:             uuidToString(b.ID),
		ScopeType:      b.ScopeType,
		ScopeID:        b.ScopeID,
		Period:         b.Period,
		LimitAmount:    numericToFloat64(b.LimitAmount),
		AlertThreshold: numericToFloat64(b.AlertThreshold),
		ActionOnExceed: b.ActionOnExceed,
		IsEnabled:      b.IsEnabled,
	}
}

func float64ToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	// Convert via big.Float → big.Int with 2 decimal places.
	bf := new(big.Float).SetFloat64(f)
	// Multiply by 100 using big.Float for precision, then convert to Int.
	scaled := new(big.Float).Mul(bf, new(big.Float).SetInt64(100))
	intVal, _ := scaled.Int(nil)
	n.Int = intVal
	n.Exp = -2
	n.Valid = true
	return n
}

func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

// periodToTimeRange returns the start (inclusive) and end (exclusive) of the
// current period window containing the given time.
func periodToTimeRange(period string, now time.Time) (time.Time, time.Time) {
	now = now.UTC()
	switch period {
	case "daily":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 0, 1)
	case "weekly":
		weekday := now.Weekday()
		if weekday == time.Sunday {
			weekday = 7
		}
		start := time.Date(now.Year(), now.Month(), now.Day()-int(weekday-time.Monday), 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 0, 7)
	case "monthly":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0)
	default:
		// Fall back to monthly.
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0)
	}
}

// computeTokenCost calculates the dollar cost from token counts and per-Mtok prices.
func computeTokenCost(inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens, reasoningTokens int64, inputPrice, outputPrice, cacheReadPrice, cacheWritePrice, reasoningPrice float64) float64 {
	const mtok = 1_000_000.0
	return float64(inputTokens)/mtok*inputPrice +
		float64(outputTokens)/mtok*outputPrice +
		float64(cacheReadTokens)/mtok*cacheReadPrice +
		float64(cacheWriteTokens)/mtok*cacheWritePrice +
		float64(reasoningTokens)/mtok*reasoningPrice
}
