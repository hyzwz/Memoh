package hands

import "strings"

// MatchMessageTrigger checks if a message matches any message_keyword trigger in the hand.
func MatchMessageTrigger(h Hand, message string) bool {
	msgLower := strings.ToLower(message)
	for _, tr := range h.Triggers {
		if tr.Event == "message_keyword" && tr.Match != "" {
			if strings.Contains(msgLower, strings.ToLower(tr.Match)) {
				return true
			}
		}
	}
	return false
}

// MatchWebhookTrigger checks if a webhook path matches any webhook trigger in the hand.
func MatchWebhookTrigger(h Hand, path string) bool {
	for _, tr := range h.Triggers {
		if tr.Event == "webhook" && tr.Path == path {
			return true
		}
	}
	return false
}
