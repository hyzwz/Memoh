package wecombot

import (
	"context"
	"strings"
	"testing"

	"github.com/memohai/memoh/internal/channel"
)

type fakeReplyCall struct {
	sourceReqID string
	streamID    string
	content     string
	finish      bool
}

type fakeStreamClient struct {
	connected bool
	sends     []struct {
		chatID  string
		content string
	}
	replies []fakeReplyCall
}

func (c *fakeStreamClient) Connected() bool {
	return c.connected
}

func (c *fakeStreamClient) SendMarkdown(_ context.Context, chatID, content string) error {
	c.sends = append(c.sends, struct {
		chatID  string
		content string
	}{chatID: chatID, content: content})
	return nil
}

func (c *fakeStreamClient) ReplyStream(_ context.Context, sourceReqID, streamID, content string, finish bool) error {
	c.replies = append(c.replies, fakeReplyCall{
		sourceReqID: sourceReqID,
		streamID:    streamID,
		content:     content,
		finish:      finish,
	})
	return nil
}

func TestOutboundStreamRepliesViaWebSocket(t *testing.T) {
	t.Parallel()

	client := &fakeStreamClient{connected: true}
	stream := newOutboundStream(client, targetChatID, "chat-1", "req-1", true)

	ctx := context.Background()
	if err := stream.Push(ctx, channel.StreamEvent{Type: channel.StreamEventStatus, Status: channel.StreamStatusStarted}); err != nil {
		t.Fatalf("push start: %v", err)
	}
	if err := stream.Push(ctx, channel.StreamEvent{Type: channel.StreamEventDelta, Delta: "Hello "}); err != nil {
		t.Fatalf("push delta 1: %v", err)
	}
	if err := stream.Push(ctx, channel.StreamEvent{Type: channel.StreamEventDelta, Delta: "world"}); err != nil {
		t.Fatalf("push delta 2: %v", err)
	}
	if err := stream.Push(ctx, channel.StreamEvent{
		Type:  channel.StreamEventFinal,
		Final: &channel.StreamFinalizePayload{},
	}); err != nil {
		t.Fatalf("push final: %v", err)
	}

	if len(client.replies) != 4 {
		t.Fatalf("unexpected reply count: %d", len(client.replies))
	}
	if client.replies[0].content != thinkingPlaceholder || client.replies[0].finish {
		t.Fatalf("unexpected thinking reply: %#v", client.replies[0])
	}
	if client.replies[1].content != "Hello " || client.replies[1].finish {
		t.Fatalf("unexpected first delta reply: %#v", client.replies[1])
	}
	if client.replies[2].content != "Hello world" || client.replies[2].finish {
		t.Fatalf("unexpected second delta reply: %#v", client.replies[2])
	}
	if client.replies[3].content != "Hello world" || !client.replies[3].finish {
		t.Fatalf("unexpected final reply: %#v", client.replies[3])
	}
}

func TestOutboundStreamFallsBackToSendMarkdown(t *testing.T) {
	t.Parallel()

	client := &fakeStreamClient{connected: true}
	stream := newOutboundStream(client, targetChatID, "chat-1", "", false)

	ctx := context.Background()
	if err := stream.Push(ctx, channel.StreamEvent{Type: channel.StreamEventDelta, Delta: "Hello"}); err != nil {
		t.Fatalf("push delta: %v", err)
	}
	if err := stream.Push(ctx, channel.StreamEvent{
		Type: channel.StreamEventFinal,
		Final: &channel.StreamFinalizePayload{Message: channel.Message{
			Text: "ignored because buffered text wins",
		}},
	}); err != nil {
		t.Fatalf("push final: %v", err)
	}

	if len(client.sends) != 1 {
		t.Fatalf("unexpected send count: %d", len(client.sends))
	}
	if client.sends[0].chatID != "chat-1" || client.sends[0].content != "Hello" {
		t.Fatalf("unexpected proactive send: %#v", client.sends[0])
	}
}

func TestOutboundStreamRequiresChatIDForProactiveSend(t *testing.T) {
	t.Parallel()

	client := &fakeStreamClient{connected: true}
	stream := newOutboundStream(client, targetUserID, "user-1", "", false)

	err := stream.Push(context.Background(), channel.StreamEvent{
		Type: channel.StreamEventFinal,
		Final: &channel.StreamFinalizePayload{Message: channel.Message{
			Text: "hello",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "requires chatid target") {
		t.Fatalf("unexpected error: %v", err)
	}
}
