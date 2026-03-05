package routing

import (
	"strings"
)

// ComplexityTier represents the complexity level of a user request.
type ComplexityTier string

const (
	TierSimple  ComplexityTier = "simple"
	TierMedium  ComplexityTier = "medium"
	TierComplex ComplexityTier = "complex"
)

var complexKeywords = []string{
	"analyze", "compare", "implement", "refactor",
	"debug", "architecture", "design", "evaluate",
	"code", "algorithm", "optimize", "performance",
	"summarize", "review", "explain", "research",
	"investigate", "diagnose", "migrate", "integrate",
}

var simpleKeywords = map[string]bool{
	"hi": true, "hello": true, "hey": true,
	"thanks": true, "thank you": true,
	"ok": true, "yes": true, "no": true,
	"help": true, "bye": true,
}

// ClassifyComplexity returns the complexity tier for a message
// based on heuristic rules (length, keywords, conversation depth).
func ClassifyComplexity(message string, historyLen int) ComplexityTier {
	score := 0
	msgLower := strings.ToLower(strings.TrimSpace(message))

	// Factor 1: Message length
	msgLen := len(message)
	switch {
	case msgLen > 2000:
		score += 5
	case msgLen > 500:
		score += 2
	case msgLen > 200:
		score += 1
	}

	// Factor 2: Simple keyword match (exact short message)
	if simpleKeywords[msgLower] {
		score -= 3
	}

	// Factor 3: Complex keyword signals
	complexHits := 0
	for _, kw := range complexKeywords {
		if strings.Contains(msgLower, kw) {
			score += 2
			complexHits++
		}
	}
	// Multiple complex keywords signal higher complexity.
	if complexHits >= 2 {
		score += 1
	}

	// Factor 4: Conversation depth
	if historyLen > 20 {
		score += 2
	} else if historyLen > 10 {
		score += 1
	}

	switch {
	case score <= 0:
		return TierSimple
	case score <= 4:
		return TierMedium
	default:
		return TierComplex
	}
}
