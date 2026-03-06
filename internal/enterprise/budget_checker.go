package enterprise

import (
	"context"
	"log/slog"

	"github.com/memohai/memoh/internal/channel/inbound"
)

// BudgetCheckerAdapter adapts CostTrackingService to the inbound.BudgetChecker interface.
type BudgetCheckerAdapter struct {
	svc    *CostTrackingService
	logger *slog.Logger
}

func NewBudgetCheckerAdapter(log *slog.Logger, svc *CostTrackingService) *BudgetCheckerAdapter {
	return &BudgetCheckerAdapter{svc: svc, logger: log}
}

func (b *BudgetCheckerAdapter) CheckChatBudget(ctx context.Context, botID, userID string) (*inbound.BudgetCheckResult, error) {
	// Check bot-level budget first
	result, err := b.svc.CheckBudget(ctx, "bot", botID)
	if err != nil {
		return nil, err
	}

	return &inbound.BudgetCheckResult{
		Allowed: result.Allowed,
		Alert:   result.Alert,
		Action:  result.Action,
	}, nil
}
