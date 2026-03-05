package cost

// TokenUsage represents token counts from an LLM call.
type TokenUsage struct {
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	ReasoningTokens  int64
}

// ModelPricing holds per-million-token pricing for a model.
type ModelPricing struct {
	InputPricePerMTok      float64
	OutputPricePerMTok     float64
	CacheReadPricePerMTok  float64
	CacheWritePricePerMTok float64
	ReasoningPricePerMTok  float64
}

// CostBreakdown is the result of a cost calculation.
type CostBreakdown struct {
	InputCost      float64
	OutputCost     float64
	CacheReadCost  float64
	CacheWriteCost float64
	ReasoningCost  float64
	TotalCost      float64
}

// Calculate computes cost for a given usage and pricing.
// Negative token counts are clamped to zero.
func Calculate(pricing ModelPricing, usage TokenUsage) CostBreakdown {
	cb := CostBreakdown{
		InputCost:      tokenCost(usage.InputTokens, pricing.InputPricePerMTok),
		OutputCost:     tokenCost(usage.OutputTokens, pricing.OutputPricePerMTok),
		CacheReadCost:  tokenCost(usage.CacheReadTokens, pricing.CacheReadPricePerMTok),
		CacheWriteCost: tokenCost(usage.CacheWriteTokens, pricing.CacheWritePricePerMTok),
		ReasoningCost:  tokenCost(usage.ReasoningTokens, pricing.ReasoningPricePerMTok),
	}
	cb.TotalCost = cb.InputCost + cb.OutputCost + cb.CacheReadCost + cb.CacheWriteCost + cb.ReasoningCost
	return cb
}

func tokenCost(tokens int64, pricePerMTok float64) float64 {
	if tokens <= 0 || pricePerMTok <= 0 {
		return 0
	}
	return float64(tokens) / 1_000_000 * pricePerMTok
}

// Budget represents a spending limit.
type Budget struct {
	LimitAmount    float64
	AlertThreshold float64 // 0.0-1.0, e.g. 0.80 = alert at 80%
	ActionOnExceed string
}

// Budget exceed actions.
const (
	ActionWarn      = "warn"
	ActionBlock     = "block"
	ActionDowngrade = "downgrade"
)

// BudgetCheckResult is the outcome of a budget check.
type BudgetCheckResult struct {
	Allowed    bool
	Alert      bool
	Spent      float64
	Limit      float64
	Percentage float64
	Action     string
}

// CheckBudget evaluates spending against a budget.
// Negative spent values are treated as zero.
func CheckBudget(budget Budget, spent float64) BudgetCheckResult {
	if spent < 0 {
		spent = 0
	}
	// Zero limit means no budget restriction.
	if budget.LimitAmount <= 0 {
		return BudgetCheckResult{
			Allowed: true,
			Alert:   false,
			Spent:   spent,
			Limit:   0,
		}
	}

	pct := spent / budget.LimitAmount
	overThreshold := pct >= budget.AlertThreshold
	overLimit := spent >= budget.LimitAmount

	result := BudgetCheckResult{
		Allowed:    true,
		Alert:      overThreshold,
		Spent:      spent,
		Limit:      budget.LimitAmount,
		Percentage: pct * 100,
		Action:     budget.ActionOnExceed,
	}

	if overLimit && budget.ActionOnExceed == ActionBlock {
		result.Allowed = false
	}

	return result
}
