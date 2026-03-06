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
	// Check all applicable scopes: bot, then user. Strictest wins (block > warn > allow).
	scopes := []struct {
		scopeType string
		scopeID   string
	}{
		{"bot", botID},
	}
	if userID != "" {
		scopes = append(scopes, struct {
			scopeType string
			scopeID   string
		}{"user", userID})
	}

	merged := &inbound.BudgetCheckResult{Allowed: true}
	for _, scope := range scopes {
		result, err := b.svc.CheckBudget(ctx, scope.scopeType, scope.scopeID)
		if err != nil {
			return nil, err
		}
		if !result.Allowed {
			// Block is the strictest — immediately deny
			return &inbound.BudgetCheckResult{
				Allowed: false,
				Alert:   true,
				Action:  result.Action,
			}, nil
		}
		if result.Alert {
			merged.Alert = true
			merged.Action = result.Action
		}
	}
	return merged, nil
}
