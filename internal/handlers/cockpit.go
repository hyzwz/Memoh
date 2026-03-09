package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

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

// CockpitReportGeneratorInterface abstracts daily report generation.
type CockpitReportGeneratorInterface interface {
	GenerateDailyReport(ctx context.Context, botID, ownerUserID string, reportDate time.Time) error
}

// CockpitServiceInterface abstracts cockpit data retrieval.
type CockpitServiceInterface interface {
	GetSummary(ctx context.Context, botID string, query CockpitSummaryQuery) (*CockpitSummaryDTO, error)
}

// CockpitHandler handles cockpit dashboard API endpoints.
type CockpitHandler struct {
	svc        CockpitServiceInterface
	reportGen  CockpitReportGeneratorInterface
	middleware []echo.MiddlewareFunc
}

// NewCockpitHandler creates a new cockpit handler.
func NewCockpitHandler(svc CockpitServiceInterface, reportGen CockpitReportGeneratorInterface, mw ...echo.MiddlewareFunc) *CockpitHandler {
	return &CockpitHandler{svc: svc, reportGen: reportGen, middleware: mw}
}

// Register registers cockpit routes.
func (h *CockpitHandler) Register(e *echo.Echo) {
	g := e.Group("/bots/:bot_id/cockpit", h.middleware...)
	g.GET("/summary", h.GetSummary)
	g.POST("/generate-report", h.GenerateReport)
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

// GenerateReport godoc
// @Summary Generate daily cockpit report for a bot
// @Tags cockpit
// @Param bot_id path string true "Bot ID"
// @Param date query string false "Report date (YYYY-MM-DD)" default(yesterday)
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Router /bots/{bot_id}/cockpit/generate-report [post]
func (h *CockpitHandler) GenerateReport(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}

	reportDate := time.Now().UTC().AddDate(0, 0, -1) // default: yesterday
	if v := c.QueryParam("date"); v != "" {
		parsed, err := time.Parse("2006-01-02", v)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid date format, use YYYY-MM-DD")
		}
		reportDate = parsed
	}

	if err := h.reportGen.GenerateDailyReport(c.Request().Context(), botID, "", reportDate); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate report: "+err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status": "generated",
		"date":   reportDate.Format("2006-01-02"),
	})
}
