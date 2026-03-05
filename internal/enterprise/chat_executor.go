package enterprise

import (
	"context"
	"strings"

	"github.com/memohai/memoh/internal/conversation"
	"github.com/memohai/memoh/internal/conversation/flow"
)

// ResolverChatExecutor adapts the conversation Resolver to the ChatExecutor interface.
type ResolverChatExecutor struct {
	resolver *flow.Resolver
}

func NewResolverChatExecutor(resolver *flow.Resolver) *ResolverChatExecutor {
	return &ResolverChatExecutor{resolver: resolver}
}

func (e *ResolverChatExecutor) ExecuteChat(ctx context.Context, botID, userID, query, token string) (string, error) {
	resp, err := e.resolver.Chat(ctx, conversation.ChatRequest{
		BotID:       botID,
		ChatID:      botID,
		UserID:      userID,
		Token:       token,
		Query:       query,
		DisplayName: "Hand Executor",
	})
	if err != nil {
		return "", err
	}

	// Extract the assistant's reply text from the response messages.
	var parts []string
	for _, msg := range resp.Messages {
		if msg.Role == "assistant" {
			if text := msg.TextContent(); text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "\n"), nil
}
