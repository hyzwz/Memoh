package wecombot

import (
	"testing"

	"github.com/memohai/memoh/internal/channel"
)

func TestNormalizeConfig(t *testing.T) {
	t.Parallel()

	got, err := normalizeConfig(map[string]any{
		"bot_id":               "bot-123",
		"secret":               "sec-123",
		"send_thinking_prompt": "false",
	})
	if err != nil {
		t.Fatalf("normalize config: %v", err)
	}
	if got["botId"] != "bot-123" {
		t.Fatalf("unexpected botId: %#v", got["botId"])
	}
	if got["websocketUrl"] != defaultWebSocketURL {
		t.Fatalf("unexpected websocketUrl: %#v", got["websocketUrl"])
	}
	if got["sendThinkingPrompt"] != false {
		t.Fatalf("unexpected sendThinkingPrompt: %#v", got["sendThinkingPrompt"])
	}
}

func TestNormalizeUserConfigAllowsChatIDOnly(t *testing.T) {
	t.Parallel()

	got, err := normalizeUserConfig(map[string]any{
		"chat_id": "chat-123",
	})
	if err != nil {
		t.Fatalf("normalize user config: %v", err)
	}
	if got["chatid"] != "chat-123" {
		t.Fatalf("unexpected chatid: %#v", got["chatid"])
	}
}

func TestResolveTargetPrefersChatID(t *testing.T) {
	t.Parallel()

	target, err := resolveTarget(map[string]any{
		"userid": "zhangsan",
		"chatid": "chat-direct-1",
	})
	if err != nil {
		t.Fatalf("resolve target: %v", err)
	}
	if target != "chatid:chat-direct-1" {
		t.Fatalf("unexpected target: %s", target)
	}
}

func TestNormalizeTarget(t *testing.T) {
	t.Parallel()

	if got := normalizeTarget("wecom:user:alice"); got != "userid:alice" {
		t.Fatalf("unexpected user target: %s", got)
	}
	if got := normalizeTarget("chat:ww-chat-1"); got != "chatid:ww-chat-1" {
		t.Fatalf("unexpected chat target: %s", got)
	}
	if got := normalizeTarget("bob"); got != "userid:bob" {
		t.Fatalf("unexpected fallback target: %s", got)
	}
}

func TestBuildUserConfigKeepsDirectChatID(t *testing.T) {
	t.Parallel()

	got := buildUserConfig(channel.Identity{
		SubjectID: "user-1",
		Attributes: map[string]string{
			"userid":    "user-1",
			"chatid":    "chat-direct-1",
			"chat_type": "direct",
		},
	})
	if got["userid"] != "user-1" {
		t.Fatalf("unexpected userid: %#v", got["userid"])
	}
	if got["chatid"] != "chat-direct-1" {
		t.Fatalf("unexpected chatid: %#v", got["chatid"])
	}
}

func TestBuildUserConfigDropsGroupChatID(t *testing.T) {
	t.Parallel()

	got := buildUserConfig(channel.Identity{
		SubjectID: "user-1",
		Attributes: map[string]string{
			"userid":    "user-1",
			"chatid":    "chat-group-1",
			"chat_type": "group",
		},
	})
	if got["userid"] != "user-1" {
		t.Fatalf("unexpected userid: %#v", got["userid"])
	}
	if _, ok := got["chatid"]; ok {
		t.Fatalf("group chatid should not be persisted: %#v", got["chatid"])
	}
}

func TestMatchBindingMatchesChatID(t *testing.T) {
	t.Parallel()

	if !matchBinding(map[string]any{"chatid": "chat-direct-1"}, channel.BindingCriteria{
		Attributes: map[string]string{"chatid": "chat-direct-1"},
	}) {
		t.Fatal("expected chatid binding match")
	}
}
