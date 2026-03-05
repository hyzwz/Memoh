package enterprise

import (
	"context"
	"errors"
	"log/slog"
	"math/big"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

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
	budget, err := s.queries.GetBudget(ctx, sqlc.GetBudgetParams{
		ScopeType: scopeType,
		ScopeID:   scopeID,
	})
	if err != nil {
		// No budget configured for this scope — allow by default.
		if errors.Is(err, pgx.ErrNoRows) {
			return &handlers.BudgetCheckDTO{
				Allowed: true,
				Spent:   0,
				Limit:   0,
			}, nil
		}
		return nil, err
	}

	limit := numericToFloat64(budget.LimitAmount)
	threshold := numericToFloat64(budget.AlertThreshold)

	// TODO: compute actual spending from token_usage table
	spent := 0.0
	percentage := 0.0
	if limit > 0 {
		percentage = (spent / limit) * 100
	}

	return &handlers.BudgetCheckDTO{
		Allowed:    spent < limit,
		Alert:      percentage >= threshold*100,
		Spent:      spent,
		Limit:      limit,
		Percentage: percentage,
		Action:     budget.ActionOnExceed,
	}, nil
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
