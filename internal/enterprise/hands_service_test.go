package enterprise

import (
	"context"
	"testing"
)

// fakeChatExecutor implements ChatExecutor for tests.
type fakeChatExecutor struct {
	resultText string
	err        error
	gotBotID   string
	gotQuery   string
}

func (f *fakeChatExecutor) ExecuteChat(ctx context.Context, botID, userID, query, token string) (string, error) {
	f.gotBotID = botID
	f.gotQuery = query
	if f.err != nil {
		return "", f.err
	}
	return f.resultText, nil
}

func TestChatExecutorInterface(t *testing.T) {
	// Verify that the ChatExecutor interface is properly defined and usable.
	var exec ChatExecutor = &fakeChatExecutor{resultText: "done"}
	result, err := exec.ExecuteChat(context.Background(), "bot-1", "user-1", "do something", "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "done" {
		t.Fatalf("result = %q, want done", result)
	}
}

func TestParseHandMarkdown(t *testing.T) {
	md := "---\nname: daily-report\ndescription: A daily summary\n---\nGenerate the report."
	name, desc, content := parseHandMarkdown(md)
	if name != "daily-report" {
		t.Fatalf("name = %q, want daily-report", name)
	}
	if desc != "A daily summary" {
		t.Fatalf("desc = %q, want A daily summary", desc)
	}
	if content != "Generate the report." {
		t.Fatalf("content = %q", content)
	}
}

func TestParseHandMarkdownNoFrontmatter(t *testing.T) {
	md := "Just some plain content."
	name, desc, content := parseHandMarkdown(md)
	if name != "unnamed" {
		t.Fatalf("name = %q, want unnamed", name)
	}
	if desc != "" {
		t.Fatalf("desc = %q, want empty", desc)
	}
	if content != "Just some plain content." {
		t.Fatalf("content = %q", content)
	}
}

func TestParseHandMarkdownEmptyName(t *testing.T) {
	md := "---\ndescription: test\n---\nContent here."
	name, _, _ := parseHandMarkdown(md)
	if name != "unnamed" {
		t.Fatalf("name = %q, want unnamed", name)
	}
}
