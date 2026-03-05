package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

// CockpitSummaryDTO is the API representation of cockpit summary data.
type CockpitSummaryDTO struct {
	TotalReports              int     `json:"total_reports"`
	TotalSavedHours           float64 `json:"total_saved_hours"`
	TotalAITimeMinutes        int     `json:"total_ai_time_minutes"`
	AverageEfficiencyMultiple float64 `json:"average_efficiency_multiple"`
	TotalInnovations          int     `json:"total_innovations"`
	LaborCostCNY              float64 `json:"labor_cost_cny"`
}

// CockpitSummaryQuery holds query parameters for cockpit data.
type CockpitSummaryQuery struct {
	Days int
}

// CockpitServiceInterface abstracts cockpit data retrieval.
type CockpitServiceInterface interface {
	GetSummary(ctx context.Context, botID string, query CockpitSummaryQuery) (*CockpitSummaryDTO, error)
}

// CockpitHandler handles cockpit dashboard API endpoints.
type CockpitHandler struct {
	svc CockpitServiceInterface
}

// NewCockpitHandler creates a new cockpit handler.
func NewCockpitHandler(svc CockpitServiceInterface) *CockpitHandler {
	return &CockpitHandler{svc: svc}
}

// Register registers cockpit routes.
func (h *CockpitHandler) Register(e *echo.Echo) {
	e.GET("/bots/:bot_id/cockpit/summary", h.GetSummary)
}

// GetSummary godoc
// @Summary Get cockpit efficiency summary
// @Tags cockpit
// @Param bot_id path string true "Bot ID"
// @Param days query int false "Number of days" default(7)
// @Success 200 {object} CockpitSummaryDTO
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/cockpit/summary [get]
func (h *CockpitHandler) GetSummary(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	days := 7
	if v := c.QueryParam("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	summary, err := h.svc.GetSummary(c.Request().Context(), botID, CockpitSummaryQuery{Days: days})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get cockpit summary")
	}

	return c.JSON(http.StatusOK, summary)
}
