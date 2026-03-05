package cockpit

import "time"

// DailyReportInput is the parsed JSON from a daily-report hand execution.
type DailyReportInput struct {
	Date         string        `json:"date"`
	Summary      string        `json:"summary"`
	Tasks        []TaskInput   `json:"tasks"`
	PendingTasks []string      `json:"pending_tasks"`
}

// TaskInput is a single task from the daily report JSON.
type TaskInput struct {
	Category             string  `json:"category"`
	Description          string  `json:"description"`
	EstimatedManualHours float64 `json:"estimated_manual_hours"`
	ActualAIMinutes      int     `json:"actual_ai_minutes"`
	Status               string  `json:"status"`
	OutputType           string  `json:"output_type"`
	QualityScore         float64 `json:"quality_score"`
	InnovationTag        bool    `json:"innovation_tag"`
}

// Summary is the aggregated efficiency metrics for a period.
type Summary struct {
	PeriodStart                time.Time
	PeriodEnd                  time.Time
	TotalReports               int
	TotalSavedHours            float64
	TotalAITimeMinutes         int
	AverageEfficiencyMultiplier float64
	TotalInnovations           int
	EquivalentLaborCostCNY     float64
}

// EfficiencyConfig holds cockpit calculation settings.
type EfficiencyConfig struct {
	LaborCostHourlyCNY float64  `json:"labor_cost_hourly_cny"`
	Categories         []string `json:"categories"`
}

// DailyReport is a processed report stored in the database.
type DailyReport struct {
	BotID                    string
	UserID                   string
	ReportDate               time.Time
	Summary                  string
	Tasks                    []TaskInput
	PendingTasks             []string
	TotalEstimatedSavedHours float64
	TotalAITimeMinutes       int
	EfficiencyMultiplier     float64
	InnovationCount          int
}
