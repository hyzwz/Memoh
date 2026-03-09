package enterprise

import (
	"context"
	"log/slog"

	"github.com/memohai/memoh/internal/channel/inbound"
	"github.com/memohai/memoh/internal/db"
	"github.com/memohai/memoh/internal/db/sqlc"
)

// BudgetCheckerAdapter adapts CostTrackingService to the inbound.BudgetChecker interface.
type BudgetCheckerAdapter struct {
	svc     *CostTrackingService
	queries *sqlc.Queries
	logger  *slog.Logger
}

func NewBudgetCheckerAdapter(log *slog.Logger, svc *CostTrackingService, queries *sqlc.Queries) *BudgetCheckerAdapter {
	return &BudgetCheckerAdapter{svc: svc, queries: queries, logger: log}
}

func (b *BudgetCheckerAdapter) CheckChatBudget(ctx context.Context, botID, userID string) (*inbound.BudgetCheckResult, error) {
	// Check all applicable scopes. Strictest wins (block > downgrade > warn > allow).
	scopes := []struct {
		scopeType string
		scopeID   string
	}{
		{"system", "system"},
		{"bot", botID},
	}
	if userID != "" {
		scopes = append(scopes, struct {
			scopeType string
			scopeID   string
		}{"user", userID})
	}
	if b != nil && b.queries != nil && userID != "" {
		if pgUserID, err := db.ParseUUID(userID); err == nil {
			departments, err := b.queries.ListUserDepartments(ctx, pgUserID)
			if err != nil {
				return nil, err
			}
			for _, item := range departments {
				scopes = append(scopes, struct {
					scopeType string
					scopeID   string
				}{
					scopeType: "department",
					scopeID:   item.ID.String(),
				})
			}
		}
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
			if budgetActionSeverity(result.Action) > budgetActionSeverity(merged.Action) {
				merged.Action = result.Action
			}
		}
	}
	return merged, nil
}

func budgetActionSeverity(action string) int {
	switch action {
	case "block":
		return 3
	case "downgrade":
		return 2
	case "warn":
		return 1
	default:
		return 0
	}
}
