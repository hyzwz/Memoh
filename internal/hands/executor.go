package hands

import (
	"context"
	"errors"
	"time"

	"github.com/memohai/memoh/internal/cost"
)

var (
	ErrBudgetExceeded  = errors.New("budget exceeded")
	ErrApprovalRequired = errors.New("approval required")
)

// AgentRequest is the input to the agent gateway for hand execution.
type AgentRequest struct {
	BotID        string
	HandName     string
	Content      string   // System prompt from hand
	AllowedTools []string
	MaxTokens    int
	TriggerType  string
}

// AgentResponse is the output from the agent gateway.
type AgentResponse struct {
	ResultText string
	Usage      map[string]any
}

// Agent abstracts the agent gateway call.
type Agent interface {
	ExecuteHand(ctx context.Context, req AgentRequest) (*AgentResponse, error)
}

// BudgetChecker checks if a hand execution is within budget.
type BudgetChecker interface {
	CheckHandBudget(ctx context.Context, botID string) cost.BudgetCheckResult
}

// ExecutionLog records a hand execution for persistence.
type ExecutionLog struct {
	HandName     string
	BotID        string
	TriggerType  string
	TriggerDetail map[string]any
	Status       string
	ResultText   string
	ErrorMessage string
	Usage        map[string]any
	StartedAt    time.Time
	CompletedAt  *time.Time
}

// ExecutionLogger persists execution logs.
type ExecutionLogger interface {
	LogExecution(ctx context.Context, log *ExecutionLog) error
}

// Executor orchestrates hand execution with pre-checks.
type Executor struct {
	agent   Agent
	budget  BudgetChecker
	logger  ExecutionLogger
}

// NewExecutor creates a new hand executor.
func NewExecutor(agent Agent, budget BudgetChecker, logger ExecutionLogger) *Executor {
	return &Executor{
		agent:  agent,
		budget: budget,
		logger: logger,
	}
}

// Execute runs a hand with pre-checks (budget, approval) and logs the result.
func (e *Executor) Execute(ctx context.Context, botID string, hand Hand, trigger TriggerContext) (*ExecutionResult, error) {
	startedAt := time.Now()

	// Pre-check: budget
	budgetResult := e.budget.CheckHandBudget(ctx, botID)
	if !budgetResult.Allowed {
		now := time.Now()
		e.logExecution(ctx, &ExecutionLog{
			HandName:    hand.Name,
			BotID:       botID,
			TriggerType: trigger.Type,
			TriggerDetail: trigger.Detail,
			Status:      "cancelled",
			ErrorMessage: "budget exceeded",
			StartedAt:   startedAt,
			CompletedAt: &now,
		})
		return nil, ErrBudgetExceeded
	}

	// Pre-check: approval
	if hand.Permissions.RequireApproval {
		now := time.Now()
		e.logExecution(ctx, &ExecutionLog{
			HandName:    hand.Name,
			BotID:       botID,
			TriggerType: trigger.Type,
			TriggerDetail: trigger.Detail,
			Status:      "approval_pending",
			StartedAt:   startedAt,
			CompletedAt: &now,
		})
		return nil, ErrApprovalRequired
	}

	// Build agent request.
	req := AgentRequest{
		BotID:        botID,
		HandName:     hand.Name,
		Content:      hand.Content,
		AllowedTools: hand.AllowedTools,
		MaxTokens:    hand.Permissions.MaxTokensPerRun,
		TriggerType:  trigger.Type,
	}

	// Execute via agent.
	resp, err := e.agent.ExecuteHand(ctx, req)
	completedAt := time.Now()

	if err != nil || resp == nil {
		errMsg := "unknown error: nil response"
		if err != nil {
			errMsg = err.Error()
		}
		// Agent failure — log as failed, but don't return error to caller.
		result := &ExecutionResult{
			Status:       "failed",
			ErrorMessage: errMsg,
			StartedAt:    startedAt,
			CompletedAt:  completedAt,
		}
		e.logExecution(ctx, &ExecutionLog{
			HandName:     hand.Name,
			BotID:        botID,
			TriggerType:  trigger.Type,
			TriggerDetail: trigger.Detail,
			Status:       "failed",
			ErrorMessage: errMsg,
			StartedAt:    startedAt,
			CompletedAt:  &completedAt,
		})
		return result, nil
	}

	// Success.
	result := &ExecutionResult{
		Status:      "completed",
		ResultText:  resp.ResultText,
		Usage:       resp.Usage,
		StartedAt:   startedAt,
		CompletedAt: completedAt,
	}
	e.logExecution(ctx, &ExecutionLog{
		HandName:     hand.Name,
		BotID:        botID,
		TriggerType:  trigger.Type,
		TriggerDetail: trigger.Detail,
		Status:       "completed",
		ResultText:   resp.ResultText,
		Usage:        resp.Usage,
		StartedAt:    startedAt,
		CompletedAt:  &completedAt,
	})

	return result, nil
}

func (e *Executor) logExecution(ctx context.Context, log *ExecutionLog) {
	// Best-effort logging — don't fail the execution.
	_ = e.logger.LogExecution(ctx, log)
}
