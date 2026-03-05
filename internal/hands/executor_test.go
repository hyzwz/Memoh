package hands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/memohai/memoh/internal/cost"
)

type fakeAgent struct {
	resultText string
	err        error
	called     bool
}

func (f *fakeAgent) ExecuteHand(_ context.Context, _ AgentRequest) (*AgentResponse, error) {
	f.called = true
	if f.err != nil {
		return nil, f.err
	}
	return &AgentResponse{
		ResultText: f.resultText,
		Usage:      map[string]any{"input_tokens": 100, "output_tokens": 50},
	}, nil
}

type fakeBudgetChecker struct {
	result cost.BudgetCheckResult
}

func (f *fakeBudgetChecker) CheckHandBudget(_ context.Context, _ string) cost.BudgetCheckResult {
	return f.result
}

type fakeExecutionLogger struct {
	logs []*ExecutionLog
}

func (f *fakeExecutionLogger) LogExecution(_ context.Context, log *ExecutionLog) error {
	f.logs = append(f.logs, log)
	return nil
}

func TestExecutorSuccess(t *testing.T) {
	agent := &fakeAgent{resultText: "Daily report generated successfully."}
	budget := &fakeBudgetChecker{result: cost.BudgetCheckResult{Allowed: true}}
	logger := &fakeExecutionLogger{}

	executor := NewExecutor(agent, budget, logger)

	hand := Hand{
		Name:    "daily-report",
		Type:    "hand",
		Content: "Generate a daily report.",
		AllowedTools: []string{"web_fetch", "send"},
		Permissions: PermissionsConfig{MaxTokensPerRun: 50000},
	}
	trigger := TriggerContext{Type: TriggerSchedule}

	result, err := executor.Execute(context.Background(), "bot-1", hand, trigger)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("status = %q", result.Status)
	}
	if result.ResultText != "Daily report generated successfully." {
		t.Fatalf("result = %q", result.ResultText)
	}
	if !agent.called {
		t.Fatal("agent was not called")
	}
	if len(logger.logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logger.logs))
	}
	if logger.logs[0].Status != "completed" {
		t.Fatalf("log status = %q", logger.logs[0].Status)
	}
}

func TestExecutorBudgetBlocked(t *testing.T) {
	agent := &fakeAgent{resultText: "should not be called"}
	budget := &fakeBudgetChecker{result: cost.BudgetCheckResult{Allowed: false, Action: cost.ActionBlock}}
	logger := &fakeExecutionLogger{}

	executor := NewExecutor(agent, budget, logger)

	hand := Hand{Name: "expensive-hand", Type: "hand", Content: "Do something costly"}
	trigger := TriggerContext{Type: TriggerManual}

	_, err := executor.Execute(context.Background(), "bot-1", hand, trigger)
	if err == nil {
		t.Fatal("expected error for budget blocked")
	}
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("expected ErrBudgetExceeded, got %v", err)
	}
	if agent.called {
		t.Fatal("agent should not be called when budget blocked")
	}
	if len(logger.logs) != 1 || logger.logs[0].Status != "cancelled" {
		t.Fatal("expected cancelled log entry")
	}
}

func TestExecutorApprovalRequired(t *testing.T) {
	agent := &fakeAgent{resultText: "result"}
	budget := &fakeBudgetChecker{result: cost.BudgetCheckResult{Allowed: true}}
	logger := &fakeExecutionLogger{}

	executor := NewExecutor(agent, budget, logger)

	hand := Hand{
		Name:    "sensitive-hand",
		Type:    "hand",
		Content: "Sensitive operation",
		Permissions: PermissionsConfig{RequireApproval: true},
	}
	trigger := TriggerContext{Type: TriggerSchedule}

	_, err := executor.Execute(context.Background(), "bot-1", hand, trigger)
	if err == nil {
		t.Fatal("expected error for approval required")
	}
	if !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("expected ErrApprovalRequired, got %v", err)
	}
	if agent.called {
		t.Fatal("agent should not be called when approval pending")
	}
	if len(logger.logs) != 1 || logger.logs[0].Status != "approval_pending" {
		t.Fatal("expected approval_pending log entry")
	}
}

func TestExecutorAgentFailure(t *testing.T) {
	agent := &fakeAgent{err: errors.New("model timeout")}
	budget := &fakeBudgetChecker{result: cost.BudgetCheckResult{Allowed: true}}
	logger := &fakeExecutionLogger{}

	executor := NewExecutor(agent, budget, logger)

	hand := Hand{Name: "failing-hand", Type: "hand", Content: "Will fail"}
	trigger := TriggerContext{Type: TriggerManual}

	result, err := executor.Execute(context.Background(), "bot-1", hand, trigger)
	if err != nil {
		t.Fatalf("Execute should not return error for agent failure, got %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("status = %q, want failed", result.Status)
	}
	if result.ErrorMessage != "model timeout" {
		t.Fatalf("error_message = %q", result.ErrorMessage)
	}
	if len(logger.logs) != 1 || logger.logs[0].Status != "failed" {
		t.Fatal("expected failed log entry")
	}
}

func TestExecutorNilResponse(t *testing.T) {
	agent := &fakeAgent{resultText: ""} // will return nil resp
	agent.err = nil
	budget := &fakeBudgetChecker{result: cost.BudgetCheckResult{Allowed: true}}
	logger := &fakeExecutionLogger{}

	executor := NewExecutor(agent, budget, logger)

	// Override to return nil, nil
	nilAgent := &nilRespAgent{}
	executor.agent = nilAgent

	hand := Hand{Name: "nil-hand", Type: "hand", Content: "Test"}
	trigger := TriggerContext{Type: TriggerManual}

	result, err := executor.Execute(context.Background(), "bot-1", hand, trigger)
	if err != nil {
		t.Fatalf("Execute should not return error, got %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("status = %q, want failed", result.Status)
	}
	if result.ErrorMessage != "unknown error: nil response" {
		t.Fatalf("error_message = %q", result.ErrorMessage)
	}
}

type nilRespAgent struct{}

func (n *nilRespAgent) ExecuteHand(_ context.Context, _ AgentRequest) (*AgentResponse, error) {
	return nil, nil
}

func TestExecutorCompletedAt(t *testing.T) {
	agent := &fakeAgent{resultText: "done"}
	budget := &fakeBudgetChecker{result: cost.BudgetCheckResult{Allowed: true}}
	logger := &fakeExecutionLogger{}

	executor := NewExecutor(agent, budget, logger)

	hand := Hand{Name: "timed-hand", Type: "hand", Content: "Quick task"}
	trigger := TriggerContext{Type: TriggerManual}

	before := time.Now()
	result, err := executor.Execute(context.Background(), "bot-1", hand, trigger)
	if err != nil {
		t.Fatal(err)
	}

	if result.StartedAt.Before(before) {
		t.Fatal("started_at should be >= test start time")
	}
	if result.CompletedAt.Before(result.StartedAt) {
		t.Fatal("completed_at should be >= started_at")
	}
}
