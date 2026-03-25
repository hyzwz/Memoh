package ctripcs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/memohai/memoh/internal/channel"
)

func TestConnectPollsAndDispatchesNewMessages(t *testing.T) {
	t.Parallel()

	runner := &fakeBrowserActionRunner{
		exists: true,
		evaluateResponses: []fakeEvaluateResponse{
			{raw: mustReadTestdata(t, "ctrip_inbox_snapshot.json")},
		},
	}
	adapter := &Adapter{browserGateway: runner}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	got := make(chan string, 8)
	conn, err := adapter.Connect(ctx, testChannelConfig(), func(_ context.Context, _ channel.ChannelConfig, msg channel.InboundMessage) error {
		got <- msg.Message.ID
		return nil
	})
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() { _ = conn.Stop(context.Background()) }()

	ids := collectIDs(t, got, 100*time.Millisecond, 2*time.Second)
	if len(ids) != 2 {
		t.Fatalf("unexpected message count: got %d ids %v", len(ids), ids)
	}
	if ids[0] != "msg-customer-1" || ids[1] != "msg-customer-2" {
		t.Fatalf("unexpected message ids: %v", ids)
	}
	if got := runner.navigateCalls(); got != 1 {
		t.Fatalf("unexpected navigate call count: %d", got)
	}
	if got := runner.evaluateCalls(); got < 1 {
		t.Fatalf("unexpected evaluate call count: %d", got)
	}
}

func TestConnectDeduplicatesMessageIDsAcrossPollingRounds(t *testing.T) {
	t.Parallel()

	firstRaw := []byte(`{
		"page_url": "https://m.ctrip.com/customer-service/inbox",
		"account_label": "Ctrip Support A",
		"conversation_id": "ctrip-session-456",
		"conversation_type": "direct",
		"login_state": "ok",
		"messages": [{
			"message_id": "msg-customer-9",
			"author_id": "customer-9001",
			"author_name": "Bob",
			"author_role": "customer",
			"text": "Need to change my flight",
			"timestamp": "2026-03-25T11:00:00Z"
		}]
	}`)
	secondRaw := []byte(`{
		"page_url": "https://m.ctrip.com/customer-service/inbox",
		"account_label": "Ctrip Support A",
		"conversation_id": "ctrip-session-456",
		"conversation_type": "direct",
		"login_state": "ok",
		"messages": [
			{
				"message_id": "msg-customer-9",
				"author_id": "customer-9001",
				"author_name": "Bob",
				"author_role": "customer",
				"text": "Need to change my flight",
				"timestamp": "2026-03-25T11:00:00Z"
			},
			{
				"message_id": "msg-customer-10",
				"author_id": "customer-9001",
				"author_name": "Bob",
				"author_role": "customer",
				"text": "I also need to update my seat",
				"timestamp": "2026-03-25T11:00:05Z"
			}
		]
	}`)

	runner := &fakeBrowserActionRunner{
		exists: true,
		evaluateResponses: []fakeEvaluateResponse{
			{raw: firstRaw},
			{raw: secondRaw},
		},
	}
	adapter := &Adapter{browserGateway: runner}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	got := make(chan string, 8)
	conn, err := adapter.Connect(ctx, testChannelConfig(), func(_ context.Context, _ channel.ChannelConfig, msg channel.InboundMessage) error {
		got <- msg.Message.ID
		return nil
	})
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() { _ = conn.Stop(context.Background()) }()

	ids := collectIDs(t, got, 100*time.Millisecond, 2*time.Second)
	if len(ids) != 2 {
		t.Fatalf("unexpected message count: got %d ids %v", len(ids), ids)
	}
	if ids[0] != "msg-customer-9" || ids[1] != "msg-customer-10" {
		t.Fatalf("unexpected message ids: %v", ids)
	}
}

func TestConnectStopsOnMissingBrowserContext(t *testing.T) {
	t.Parallel()

	runner := &fakeBrowserActionRunner{
		exists: false,
	}
	adapter := &Adapter{browserGateway: runner}

	conn, err := adapter.Connect(context.Background(), testChannelConfig(), func(context.Context, channel.ChannelConfig, channel.InboundMessage) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if conn != nil {
		t.Fatalf("expected nil connection, got %#v", conn)
	}
	if got := runner.navigateCalls(); got != 0 {
		t.Fatalf("unexpected navigate call count: %d", got)
	}
	if got := runner.evaluateCalls(); got != 0 {
		t.Fatalf("unexpected evaluate call count: %d", got)
	}
}

func TestConnectReturnsLoginExpiredAsConnectionError(t *testing.T) {
	t.Parallel()

	runner := &fakeBrowserActionRunner{
		exists: true,
		evaluateResponses: []fakeEvaluateResponse{
			{raw: mustReadTestdata(t, "ctrip_login_expired_snapshot.json")},
		},
	}
	adapter := &Adapter{browserGateway: runner}

	conn, err := adapter.Connect(context.Background(), testChannelConfig(), func(context.Context, channel.ChannelConfig, channel.InboundMessage) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrLoginExpired) {
		t.Fatalf("expected ErrLoginExpired, got %v", err)
	}
	if conn != nil {
		t.Fatalf("expected nil connection, got %#v", conn)
	}
}

type fakeEvaluateResponse struct {
	raw []byte
	err error
}

type fakeBrowserActionRunner struct {
	mu sync.Mutex

	exists bool
	err    error

	navigateURLs      []string
	evaluateScripts   []string
	evaluateResponses []fakeEvaluateResponse
}

func (f *fakeBrowserActionRunner) Exists(context.Context, string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.exists, f.err
}

func (f *fakeBrowserActionRunner) Navigate(_ context.Context, _ string, url string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.navigateURLs = append(f.navigateURLs, url)
	return f.err
}

func (f *fakeBrowserActionRunner) Evaluate(ctx context.Context, _ string, script string) ([]byte, error) {
	f.mu.Lock()
	f.evaluateScripts = append(f.evaluateScripts, script)
	idx := len(f.evaluateScripts) - 1
	var resp fakeEvaluateResponse
	if idx < len(f.evaluateResponses) {
		resp = f.evaluateResponses[idx]
	}
	f.mu.Unlock()

	if resp.raw != nil || resp.err != nil {
		return resp.raw, resp.err
	}

	<-ctx.Done()
	return nil, ctx.Err()
}

func (f *fakeBrowserActionRunner) navigateCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.navigateURLs)
}

func (f *fakeBrowserActionRunner) evaluateCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.evaluateScripts)
}

func collectIDs(t *testing.T, ch <-chan string, quiet time.Duration, timeout time.Duration) []string {
	t.Helper()

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	quietTimer := time.NewTimer(quiet)
	defer quietTimer.Stop()

	ids := make([]string, 0, 4)
	for {
		select {
		case id := <-ch:
			ids = append(ids, id)
			if !quietTimer.Stop() {
				select {
				case <-quietTimer.C:
				default:
				}
			}
			quietTimer.Reset(quiet)
		case <-quietTimer.C:
			return ids
		case <-deadline.C:
			t.Fatalf("timed out waiting for message ids, got %v", ids)
		}
	}
}

func testChannelConfig() channel.ChannelConfig {
	return channel.ChannelConfig{
		ID:          "cfg-ctripcs-1",
		BotID:       "bot-1",
		ChannelType: Type,
		Credentials: map[string]any{
			"browserContextId": "ctx-ctripcs-1",
			"entryUrl":         "https://m.ctrip.com/",
			"accountLabel":     "Ctrip Support A",
			"pollIntervalMs":   10,
		},
	}
}
