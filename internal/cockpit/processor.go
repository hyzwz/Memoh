package cockpit

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"
)

// ProcessDailyReport parses the JSON output from a daily-report hand execution
// and computes aggregate metrics.
func ProcessDailyReport(jsonStr string) (*DailyReport, error) {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return nil, errors.New("empty report JSON")
	}

	var input DailyReportInput
	if err := json.Unmarshal([]byte(jsonStr), &input); err != nil {
		return nil, err
	}

	report := &DailyReport{
		Summary:      input.Summary,
		Tasks:        input.Tasks,
		PendingTasks: input.PendingTasks,
	}

	// Parse report date.
	if input.Date != "" {
		if t, err := time.Parse("2006-01-02", input.Date); err == nil {
			report.ReportDate = t
		}
	}

	var totalSaved float64
	var totalAIMins int
	var innovations int

	for _, task := range input.Tasks {
		hours := task.EstimatedManualHours
		mins := task.ActualAIMinutes
		if hours < 0 {
			hours = 0
		}
		if mins < 0 {
			mins = 0
		}
		totalSaved += hours
		totalAIMins += mins
		if task.InnovationTag {
			innovations++
		}
	}

	report.TotalEstimatedSavedHours = totalSaved
	report.TotalAITimeMinutes = totalAIMins
	report.InnovationCount = innovations

	// Efficiency multiplier = (saved_hours * 60) / ai_minutes
	// Represents how many minutes of human work each minute of AI replaces.
	if totalAIMins > 0 {
		report.EfficiencyMultiplier = roundTo2(totalSaved * 60 / float64(totalAIMins))
	}

	return report, nil
}

// CalculateSummary aggregates multiple daily reports into a summary.
func CalculateSummary(reports []DailyReport, config EfficiencyConfig) Summary {
	if len(reports) == 0 {
		return Summary{}
	}

	var s Summary
	s.TotalReports = len(reports)

	for _, r := range reports {
		s.TotalSavedHours += r.TotalEstimatedSavedHours
		s.TotalAITimeMinutes += r.TotalAITimeMinutes
		s.TotalInnovations += r.InnovationCount
	}

	// Weighted efficiency: (total_saved_hours * 60) / total_ai_minutes
	if s.TotalAITimeMinutes > 0 {
		s.AverageEfficiencyMultiplier = roundTo2(s.TotalSavedHours * 60 / float64(s.TotalAITimeMinutes))
	}
	s.EquivalentLaborCostCNY = roundTo2(s.TotalSavedHours * config.LaborCostHourlyCNY)

	return s
}

func roundTo2(f float64) float64 {
	return math.Round(f*100) / 100
}
