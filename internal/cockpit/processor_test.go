package cockpit

import (
	"testing"
)

func TestProcessDailyReport(t *testing.T) {
	tests := []struct {
		name       string
		jsonStr    string
		wantErr    bool
		wantTasks  int
		wantSaved  float64
		wantAIMins int
		wantMult   float64
		wantInnov  int
	}{
		{
			name: "valid report with 3 tasks",
			jsonStr: `{
				"date": "2026-03-04",
				"summary": "Helped with 3 tasks today, saving approximately 8 hours.",
				"tasks": [
					{
						"category": "document",
						"description": "Generated quarterly report",
						"estimated_manual_hours": 4.0,
						"actual_ai_minutes": 30,
						"status": "completed",
						"output_type": "document",
						"quality_score": 4.5,
						"innovation_tag": false
					},
					{
						"category": "analysis",
						"description": "Market data analysis",
						"estimated_manual_hours": 3.0,
						"actual_ai_minutes": 25,
						"status": "completed",
						"output_type": "analysis",
						"quality_score": 4.0,
						"innovation_tag": true
					},
					{
						"category": "communication",
						"description": "Drafted email responses",
						"estimated_manual_hours": 1.0,
						"actual_ai_minutes": 10,
						"status": "completed",
						"output_type": "communication",
						"quality_score": 3.5,
						"innovation_tag": false
					}
				],
				"pending_tasks": ["Follow up on Q1 results"]
			}`,
			wantTasks:  3,
			wantSaved:  8.0,
			wantAIMins: 65,
			wantMult:   7.38, // 8.0 * 60 / 65 ≈ 7.38
			wantInnov:  1,
		},
		{
			name: "empty tasks",
			jsonStr: `{
				"date": "2026-03-04",
				"summary": "No work yesterday",
				"tasks": [],
				"pending_tasks": []
			}`,
			wantTasks:  0,
			wantSaved:  0,
			wantAIMins: 0,
			wantMult:   0,
			wantInnov:  0,
		},
		{
			name:    "invalid JSON",
			jsonStr: "not json at all",
			wantErr: true,
		},
		{
			name:    "empty string",
			jsonStr: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := ProcessDailyReport(tt.jsonStr)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ProcessDailyReport: %v", err)
			}

			if len(report.Tasks) != tt.wantTasks {
				t.Fatalf("tasks = %d, want %d", len(report.Tasks), tt.wantTasks)
			}
			if report.TotalEstimatedSavedHours != tt.wantSaved {
				t.Fatalf("saved_hours = %.2f, want %.2f", report.TotalEstimatedSavedHours, tt.wantSaved)
			}
			if report.TotalAITimeMinutes != tt.wantAIMins {
				t.Fatalf("ai_minutes = %d, want %d", report.TotalAITimeMinutes, tt.wantAIMins)
			}
			if tt.wantMult > 0 {
				diff := report.EfficiencyMultiplier - tt.wantMult
				if diff < -0.1 || diff > 0.1 {
					t.Fatalf("multiplier = %.2f, want ~%.2f", report.EfficiencyMultiplier, tt.wantMult)
				}
			}
			if report.InnovationCount != tt.wantInnov {
				t.Fatalf("innovation_count = %d, want %d", report.InnovationCount, tt.wantInnov)
			}
		})
	}
}

func TestCalculateSummary(t *testing.T) {
	config := EfficiencyConfig{
		LaborCostHourlyCNY: 200,
	}

	reports := []DailyReport{
		{
			TotalEstimatedSavedHours: 8.0,
			TotalAITimeMinutes:       65,
			EfficiencyMultiplier:     7.38,
			InnovationCount:          1,
		},
		{
			TotalEstimatedSavedHours: 5.0,
			TotalAITimeMinutes:       40,
			EfficiencyMultiplier:     7.5,
			InnovationCount:          0,
		},
		{
			TotalEstimatedSavedHours: 3.0,
			TotalAITimeMinutes:       30,
			EfficiencyMultiplier:     6.0,
			InnovationCount:          2,
		},
	}

	summary := CalculateSummary(reports, config)

	if summary.TotalReports != 3 {
		t.Fatalf("total_reports = %d, want 3", summary.TotalReports)
	}
	if summary.TotalSavedHours != 16.0 {
		t.Fatalf("total_saved_hours = %.2f, want 16.0", summary.TotalSavedHours)
	}
	if summary.TotalAITimeMinutes != 135 {
		t.Fatalf("total_ai_minutes = %d, want 135", summary.TotalAITimeMinutes)
	}
	if summary.TotalInnovations != 3 {
		t.Fatalf("total_innovations = %d, want 3", summary.TotalInnovations)
	}
	// Weighted multiplier: (16.0 * 60) / 135 ≈ 7.11
	expectedAvg := 16.0 * 60 / 135.0
	diff := summary.AverageEfficiencyMultiplier - expectedAvg
	if diff < -0.1 || diff > 0.1 {
		t.Fatalf("avg_multiplier = %.2f, want ~%.2f", summary.AverageEfficiencyMultiplier, expectedAvg)
	}
	// Labor cost: 16 hours * 200 CNY/hr = 3200
	if summary.EquivalentLaborCostCNY != 3200 {
		t.Fatalf("labor_cost = %.2f, want 3200", summary.EquivalentLaborCostCNY)
	}
}

func TestProcessDailyReportNegativeValues(t *testing.T) {
	jsonStr := `{
		"date": "2026-03-04",
		"summary": "Test",
		"tasks": [
			{
				"category": "document",
				"description": "Bad data",
				"estimated_manual_hours": -5.0,
				"actual_ai_minutes": -10,
				"status": "completed",
				"quality_score": 4.0,
				"innovation_tag": false
			}
		]
	}`
	report, err := ProcessDailyReport(jsonStr)
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalEstimatedSavedHours != 0 {
		t.Fatalf("negative hours should be clamped to 0, got %.2f", report.TotalEstimatedSavedHours)
	}
	if report.TotalAITimeMinutes != 0 {
		t.Fatalf("negative minutes should be clamped to 0, got %d", report.TotalAITimeMinutes)
	}
}

func TestProcessDailyReportDateParsing(t *testing.T) {
	jsonStr := `{
		"date": "2026-03-04",
		"summary": "Test",
		"tasks": []
	}`
	report, err := ProcessDailyReport(jsonStr)
	if err != nil {
		t.Fatal(err)
	}
	if report.ReportDate.IsZero() {
		t.Fatal("expected ReportDate to be set")
	}
	if report.ReportDate.Day() != 4 || report.ReportDate.Month() != 3 {
		t.Fatalf("ReportDate = %v, expected 2026-03-04", report.ReportDate)
	}
}

func TestCalculateSummaryEmpty(t *testing.T) {
	config := EfficiencyConfig{LaborCostHourlyCNY: 200}
	summary := CalculateSummary(nil, config)

	if summary.TotalReports != 0 {
		t.Fatalf("expected 0 reports, got %d", summary.TotalReports)
	}
	if summary.TotalSavedHours != 0 {
		t.Fatalf("expected 0 saved hours, got %.2f", summary.TotalSavedHours)
	}
	if summary.EquivalentLaborCostCNY != 0 {
		t.Fatalf("expected 0 labor cost, got %.2f", summary.EquivalentLaborCostCNY)
	}
}
