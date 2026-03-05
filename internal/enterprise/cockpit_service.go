package enterprise

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/memohai/memoh/internal/db"
	"github.com/memohai/memoh/internal/db/sqlc"
	"github.com/memohai/memoh/internal/handlers"
)

// CockpitService implements handlers.CockpitServiceInterface.
type CockpitService struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

func NewCockpitService(log *slog.Logger, queries *sqlc.Queries) *CockpitService {
	return &CockpitService{
		queries: queries,
		logger:  log.With(slog.String("service", "cockpit")),
	}
}

func (s *CockpitService) GetSummary(ctx context.Context, botID string, query handlers.CockpitSummaryQuery) (*handlers.CockpitSummaryDTO, error) {
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		return nil, err
	}

	days := query.Days
	if days <= 0 {
		days = 7
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	agg, err := s.queries.CockpitAggregateByDateRange(ctx, sqlc.CockpitAggregateByDateRangeParams{
		BotID:     pgBotID,
		StartDate: pgtype.Date{Time: startDate, Valid: true},
		EndDate:   pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	savedHours := numericToFloat64(agg.TotalSavedHours)
	laborCostCNY := savedHours * 150.0 // ~150 CNY/hr average for tech staff

	return &handlers.CockpitSummaryDTO{
		TotalReports:              int(agg.TotalReports),
		TotalSavedHours:           savedHours,
		TotalAITimeMinutes:        int(agg.TotalAiMinutes),
		AverageEfficiencyMultiple: numericToFloat64(agg.AvgEfficiency),
		TotalInnovations:          int(agg.TotalInnovations),
		LaborCostCNY:              laborCostCNY,
	}, nil
}
