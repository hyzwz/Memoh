package ctripcs

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/memohai/memoh/internal/channel"
)

func TestSendUsesReplyTargetToFocusConversationAndSubmit(t *testing.T) {
	t.Parallel()

	gateway := &testOutboundGateway{
		exists:         true,
		evaluateResult: []byte(`{"success":true,"focused":true,"submitted":true}`),
	}
	adapter := &Adapter{browserGateway: gateway}

	err := adapter.Send(context.Background(), channel.ChannelConfig{
		Credentials: map[string]any{
			"browserContextId": "ctx-123",
			"entryUrl":         "https://m.ctrip.com/",
			"inboxPageUrl":     "https://m.ctrip.com/customer-service/inbox",
			"accountLabel":     "Ctrip Support A",
		},
	}, channel.OutboundMessage{
		Message: channel.Message{
			Text: "Hello from Ctrip",
			Reply: &channel.ReplyRef{
				Target: "session:ctrip-session-123",
			},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(gateway.calls) != 3 {
		t.Fatalf("unexpected call count: %d (%v)", len(gateway.calls), gateway.calls)
	}
	if got := gateway.calls[0]; got != "exists:ctx-123" {
		t.Fatalf("unexpected first call: %q", got)
	}
	if got := gateway.calls[1]; got != "navigate:https://m.ctrip.com/customer-service/inbox" {
		t.Fatalf("unexpected second call: %q", got)
	}
	if got := gateway.calls[2]; !strings.HasPrefix(got, "evaluate:") {
		t.Fatalf("unexpected third call: %q", got)
	}
	if !strings.Contains(gateway.evaluateScripts[0], "ctrip-session-123") {
		t.Fatalf("expected evaluate script to reference reply target session id, got %q", gateway.evaluateScripts[0])
	}
	if !strings.Contains(gateway.evaluateScripts[0], "Hello from Ctrip") {
		t.Fatalf("expected evaluate script to include outbound text, got %q", gateway.evaluateScripts[0])
	}
}

func TestSendRejectsEmptyText(t *testing.T) {
	t.Parallel()

	gateway := &testOutboundGateway{}
	adapter := &Adapter{browserGateway: gateway}

	err := adapter.Send(context.Background(), channel.ChannelConfig{
		Credentials: map[string]any{
			"browserContextId": "ctx-123",
			"entryUrl":         "https://m.ctrip.com/",
			"accountLabel":     "Ctrip Support A",
		},
	}, channel.OutboundMessage{
		Message: channel.Message{
			Reply: &channel.ReplyRef{
				Target: "session:ctrip-session-123",
			},
		},
	})
	if err == nil {
		t.Fatal("expected empty text error")
	}
	if len(gateway.calls) != 0 {
		t.Fatalf("expected no gateway calls, got %v", gateway.calls)
	}
}

func TestOpenStreamBuffersChunksAndSendsFinalMessageOnClose(t *testing.T) {
	t.Parallel()

	gateway := &testOutboundGateway{
		exists:         true,
		evaluateResult: []byte(`{"success":true,"focused":true,"submitted":true}`),
	}
	adapter := &Adapter{browserGateway: gateway}

	stream, err := adapter.OpenStream(context.Background(), channel.ChannelConfig{
		Credentials: map[string]any{
			"browserContextId": "ctx-123",
			"entryUrl":         "https://m.ctrip.com/",
			"inboxPageUrl":     "https://m.ctrip.com/customer-service/inbox",
			"accountLabel":     "Ctrip Support A",
		},
	}, "session:ctrip-session-123", channel.StreamOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := stream.Push(context.Background(), channel.StreamEvent{Type: channel.StreamEventDelta, Delta: "Hello "}); err != nil {
		t.Fatalf("expected no error on delta push, got %v", err)
	}
	if err := stream.Push(context.Background(), channel.StreamEvent{Type: channel.StreamEventDelta, Delta: "world"}); err != nil {
		t.Fatalf("expected no error on delta push, got %v", err)
	}
	if err := stream.Push(context.Background(), channel.StreamEvent{
		Type: channel.StreamEventFinal,
		Final: &channel.StreamFinalizePayload{
			Message: channel.Message{Text: "Hello world"},
		},
	}); err != nil {
		t.Fatalf("expected no error on final push, got %v", err)
	}
	if len(gateway.calls) != 0 {
		t.Fatalf("expected no send before close, got %v", gateway.calls)
	}

	if err := stream.Close(context.Background()); err != nil {
		t.Fatalf("expected close to succeed, got %v", err)
	}

	if len(gateway.calls) != 3 {
		t.Fatalf("unexpected call count after close: %d (%v)", len(gateway.calls), gateway.calls)
	}
	if got := gateway.evaluateScripts[0]; !strings.Contains(got, "Hello world") {
		t.Fatalf("expected final collapsed text in evaluate script, got %q", got)
	}
	if strings.Contains(gateway.evaluateScripts[0], "Hello worldHello world") {
		t.Fatalf("expected collapsed stream text, got %q", gateway.evaluateScripts[0])
	}
}

func TestOpenStreamPropagatesSendFailureOnClose(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("composer did not clear")
	gateway := &testOutboundGateway{
		exists:        true,
		evaluateError: wantErr,
	}
	adapter := &Adapter{browserGateway: gateway}

	stream, err := adapter.OpenStream(context.Background(), channel.ChannelConfig{
		Credentials: map[string]any{
			"browserContextId": "ctx-123",
			"entryUrl":         "https://m.ctrip.com/",
			"inboxPageUrl":     "https://m.ctrip.com/customer-service/inbox",
			"accountLabel":     "Ctrip Support A",
		},
	}, "session:ctrip-session-123", channel.StreamOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := stream.Push(context.Background(), channel.StreamEvent{Type: channel.StreamEventDelta, Delta: "partial reply"}); err != nil {
		t.Fatalf("expected no error on delta push, got %v", err)
	}

	err = stream.Close(context.Background())
	if err == nil {
		t.Fatal("expected close error")
	}
	if !errors.Is(err, wantErr) && !strings.Contains(err.Error(), wantErr.Error()) {
		t.Fatalf("expected close error to include underlying send error, got %v", err)
	}
}

type testOutboundGateway struct {
	exists         bool
	existsError    error
	navigateError  error
	evaluateResult []byte
	evaluateError  error

	calls           []string
	evaluateScripts []string
	navigatedURLs   []string
}

func (g *testOutboundGateway) Exists(ctx context.Context, contextID string) (bool, error) {
	g.calls = append(g.calls, "exists:"+contextID)
	if g.existsError != nil {
		return false, g.existsError
	}
	return g.exists, nil
}

func (g *testOutboundGateway) Navigate(ctx context.Context, contextID string, url string) error {
	g.calls = append(g.calls, "navigate:"+url)
	g.navigatedURLs = append(g.navigatedURLs, url)
	return g.navigateError
}

func (g *testOutboundGateway) Evaluate(ctx context.Context, contextID string, script string) ([]byte, error) {
	g.calls = append(g.calls, "evaluate:"+contextID)
	g.evaluateScripts = append(g.evaluateScripts, script)
	if g.evaluateError != nil {
		return nil, g.evaluateError
	}
	if g.evaluateResult == nil {
		return []byte(`{"success":true}`), nil
	}
	return append([]byte(nil), g.evaluateResult...), nil
}
