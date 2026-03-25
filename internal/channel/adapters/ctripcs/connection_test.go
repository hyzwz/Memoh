package ctripcs

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/memohai/memoh/internal/channel"
	"github.com/memohai/memoh/internal/config"
)

func TestAdapterDescriptorExposesExpectedCapabilities(t *testing.T) {
	t.Parallel()

	desc := NewAdapter(nil, config.BrowserGatewayConfig{}).Descriptor()
	if desc.Type != Type {
		t.Fatalf("unexpected type: %q", desc.Type)
	}
	if desc.DisplayName != "Ctrip Customer Service" {
		t.Fatalf("unexpected display name: %q", desc.DisplayName)
	}
	if !desc.Capabilities.Text {
		t.Fatal("expected text capability")
	}
	if !desc.Capabilities.Reply {
		t.Fatal("expected reply capability")
	}
	if !desc.Capabilities.Streaming {
		t.Fatal("expected streaming capability")
	}
}

func TestAdapterImplementsReceiverAndSender(t *testing.T) {
	t.Parallel()

	adapter := NewAdapter(nil, config.BrowserGatewayConfig{})

	var _ channel.Adapter = adapter
	var _ channel.Receiver = adapter
	var _ channel.Sender = adapter
	var _ channel.StreamSender = adapter

	if _, ok := any(adapter).(channel.Adapter); !ok {
		t.Fatal("adapter does not implement channel.Adapter")
	}
	if _, ok := any(adapter).(channel.Receiver); !ok {
		t.Fatal("adapter does not implement channel.Receiver")
	}
	if _, ok := any(adapter).(channel.Sender); !ok {
		t.Fatal("adapter does not implement channel.Sender")
	}
	if _, ok := any(adapter).(channel.StreamSender); !ok {
		t.Fatal("adapter does not implement channel.StreamSender")
	}
}

func TestConnectPollsAndDispatchesNewMessages(t *testing.T) {
	t.Parallel()

	runner := &fakeBrowserActionRunner{
		exists: true,
		evaluateResponses: []fakeEvaluateResponse{
			{raw: mustReadTestdata(t, "ctrip_inbox_snapshot.json")},
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

func TestConnectReturnsPromptlyWhenStartupEvaluateFails(t *testing.T) {
	t.Parallel()

	runner := &fakeBrowserActionRunner{
		exists: true,
		evaluateResponses: []fakeEvaluateResponse{
			{err: errors.New("transient evaluate failure")},
		},
	}
	adapter := &Adapter{browserGateway: runner}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan struct {
		conn channel.Connection
		err  error
	}, 1)
	start := time.Now()
	go func() {
		conn, err := adapter.Connect(ctx, testChannelConfig(), func(context.Context, channel.ChannelConfig, channel.InboundMessage) error {
			return nil
		})
		done <- struct {
			conn channel.Connection
			err  error
		}{conn: conn, err: err}
	}()

	select {
	case result := <-done:
		if result.err != nil {
			t.Fatalf("Connect() error = %v", result.err)
		}
		if result.conn == nil {
			t.Fatal("expected connection, got nil")
		}
		if elapsed := time.Since(start); elapsed > 250*time.Millisecond {
			t.Fatalf("Connect() took too long: %v", elapsed)
		}
		if err := result.conn.Stop(context.Background()); err != nil {
			t.Fatalf("Stop() error = %v", err)
		}
	case <-time.After(400 * time.Millisecond):
		t.Fatal("Connect() did not return promptly")
	}
}

func TestConnectStartupProbeDoesNotInvokeHandlerBeforeReturn(t *testing.T) {
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

	handlerCalled := make(chan struct{}, 1)
	done := make(chan struct {
		conn channel.Connection
		err  error
	}, 1)

	go func() {
		conn, err := adapter.Connect(ctx, testChannelConfig(), func(_ context.Context, _ channel.ChannelConfig, _ channel.InboundMessage) error {
			select {
			case handlerCalled <- struct{}{}:
			default:
			}
			return nil
		})
		done <- struct {
			conn channel.Connection
			err  error
		}{conn: conn, err: err}
	}()

	var result struct {
		conn channel.Connection
		err  error
	}
	select {
	case result = <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Connect() did not return")
	}
	if result.err != nil {
		t.Fatalf("Connect() error = %v", result.err)
	}
	select {
	case <-handlerCalled:
		t.Fatal("startup probe invoked handler before Connect() returned")
	default:
	}
	if result.conn == nil {
		t.Fatal("expected connection, got nil")
	}
	if err := result.conn.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
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

func TestConnectRetriesSameMessageAfterHandlerFailure(t *testing.T) {
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
		"messages": [{
			"message_id": "msg-customer-9",
			"author_id": "customer-9001",
			"author_name": "Bob",
			"author_role": "customer",
			"text": "Need to change my flight",
			"timestamp": "2026-03-25T11:00:00Z"
		}]
	}`)

	runner := &fakeBrowserActionRunner{
		exists: true,
		evaluateResponses: []fakeEvaluateResponse{
			{raw: firstRaw},
			{raw: secondRaw},
			{raw: secondRaw},
		},
	}
	adapter := &Adapter{browserGateway: runner}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	calls := make(chan string, 4)
	var attempts atomic.Int32
	handler := func(_ context.Context, _ channel.ChannelConfig, msg channel.InboundMessage) error {
		calls <- msg.Message.ID
		if attempts.Add(1) == 1 {
			return errors.New("downstream failure")
		}
		return nil
	}

	conn, err := adapter.Connect(ctx, testChannelConfig(), handler)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() { _ = conn.Stop(context.Background()) }()

	first := waitForCall(t, calls, 2*time.Second)
	second := waitForCall(t, calls, 2*time.Second)
	if first != "msg-customer-9" || second != "msg-customer-9" {
		t.Fatalf("unexpected message ids: %q, %q", first, second)
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

func TestConnectStopsWhileEvaluateIsBlocked(t *testing.T) {
	t.Parallel()

	runner := &fakeBrowserActionRunner{
		exists:      true,
		evalStarted: make(chan struct{}),
	}
	adapter := &Adapter{browserGateway: runner}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := adapter.Connect(ctx, testChannelConfig(), func(context.Context, channel.ChannelConfig, channel.InboundMessage) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	select {
	case <-runner.evalStarted:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected evaluate to start")
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
	defer stopCancel()
	if err := conn.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if conn.Running() {
		t.Fatal("expected connection to stop running")
	}
}

func TestConnectReturnsPromptlyWhenParentContextCanceledDuringStartupEvaluate(t *testing.T) {
	t.Parallel()

	runner := &fakeBrowserActionRunner{
		exists:      true,
		evalStarted: make(chan struct{}),
	}
	adapter := &Adapter{browserGateway: runner}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct {
		conn channel.Connection
		err  error
	}, 1)
	go func() {
		conn, err := adapter.Connect(ctx, testChannelConfig(), func(context.Context, channel.ChannelConfig, channel.InboundMessage) error {
			return nil
		})
		done <- struct {
			conn channel.Connection
			err  error
		}{conn: conn, err: err}
	}()

	select {
	case <-runner.evalStarted:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected evaluate to start")
	}

	cancel()

	select {
	case result := <-done:
		if result.err != nil {
			t.Fatalf("Connect() error = %v", result.err)
		}
		if result.conn == nil {
			t.Fatal("expected connection, got nil")
		}
		if err := result.conn.Stop(context.Background()); err != nil {
			t.Fatalf("Stop() error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Connect() did not return after parent cancellation")
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

	exists    bool
	existsErr error

	navigateErr error

	navigateURLs      []string
	evaluateScripts   []string
	evaluateResponses []fakeEvaluateResponse

	evalStarted     chan struct{}
	evalStartedOnce sync.Once
}

func (f *fakeBrowserActionRunner) Exists(context.Context, string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.exists, f.existsErr
}

func (f *fakeBrowserActionRunner) Navigate(_ context.Context, _ string, url string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.navigateURLs = append(f.navigateURLs, url)
	return f.navigateErr
}

func (f *fakeBrowserActionRunner) Evaluate(ctx context.Context, _ string, script string) ([]byte, error) {
	if f.evalStarted != nil {
		f.evalStartedOnce.Do(func() {
			close(f.evalStarted)
		})
	}
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

func waitForCall(t *testing.T, ch <-chan string, timeout time.Duration) string {
	t.Helper()

	select {
	case value := <-ch:
		return value
	case <-time.After(timeout):
		t.Fatal("timed out waiting for handler call")
		return ""
	}
}

func waitUntil(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
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
