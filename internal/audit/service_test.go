package audit

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeWriter collects entries written by the service for verification.
type fakeWriter struct {
	mu      sync.Mutex
	entries []Entry
}

func (f *fakeWriter) WriteEntries(entries []Entry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, entries...)
	return nil
}

func (f *fakeWriter) getEntries() []Entry {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]Entry, len(f.entries))
	copy(result, f.entries)
	return result
}

// --- Test: Log is non-blocking ---

func TestLogIsNonBlocking(t *testing.T) {
	writer := &fakeWriter{}
	svc := NewService(writer, Config{BufferSize: 10, FlushInterval: time.Hour})
	svc.Start()
	defer svc.Stop()

	// Log should return immediately without blocking.
	done := make(chan struct{})
	go func() {
		svc.Log(Entry{
			UserID: "user-1",
			Action: ActionLogin,
		})
		close(done)
	}()

	select {
	case <-done:
		// OK - returned quickly
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Log() blocked for over 100ms, should be non-blocking")
	}
}

// --- Test: Entries are flushed to writer ---

func TestEntriesFlushedToWriter(t *testing.T) {
	writer := &fakeWriter{}
	svc := NewService(writer, Config{BufferSize: 100, FlushInterval: 50 * time.Millisecond})
	svc.Start()

	svc.Log(Entry{UserID: "user-1", Action: ActionLogin, ResourceType: "session"})
	svc.Log(Entry{UserID: "user-2", Action: ActionBotCreate, ResourceType: "bot", ResourceID: "bot-1"})

	// Wait for flush interval.
	time.Sleep(150 * time.Millisecond)

	entries := writer.getEntries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries flushed, got %d", len(entries))
	}

	if entries[0].UserID != "user-1" || entries[0].Action != ActionLogin {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}
	if entries[1].UserID != "user-2" || entries[1].Action != ActionBotCreate {
		t.Fatalf("unexpected second entry: %+v", entries[1])
	}

	svc.Stop()
}

// --- Test: Stop flushes remaining entries ---

func TestStopFlushesRemaining(t *testing.T) {
	writer := &fakeWriter{}
	svc := NewService(writer, Config{BufferSize: 100, FlushInterval: time.Hour}) // Long interval
	svc.Start()

	svc.Log(Entry{UserID: "user-1", Action: ActionChat})
	svc.Log(Entry{UserID: "user-2", Action: ActionToolCall})
	svc.Log(Entry{UserID: "user-3", Action: ActionSkillInvoke})

	// Stop should flush all remaining entries before returning.
	svc.Stop()

	entries := writer.getEntries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries flushed on stop, got %d", len(entries))
	}
}

// --- Test: Buffer full triggers early flush ---

func TestBufferFullTriggersFlush(t *testing.T) {
	writer := &fakeWriter{}
	svc := NewService(writer, Config{BufferSize: 3, FlushInterval: time.Hour})
	svc.Start()
	defer svc.Stop()

	// Fill the buffer to capacity.
	svc.Log(Entry{Action: "a"})
	svc.Log(Entry{Action: "b"})
	svc.Log(Entry{Action: "c"})

	// Give the flush goroutine time to process.
	time.Sleep(50 * time.Millisecond)

	entries := writer.getEntries()
	if len(entries) < 3 {
		t.Fatalf("expected at least 3 entries flushed when buffer full, got %d", len(entries))
	}
}

// --- Test: Scrub sensitive data ---

func TestScrubDetail(t *testing.T) {
	tests := []struct {
		name   string
		detail map[string]any
		key    string
		want   string
	}{
		{
			name:   "api_key is masked",
			detail: map[string]any{"api_key": "sk-1234567890abcdef1234567890abcdef"},
			key:    "api_key",
			want:   "sk-12345***************************",
		},
		{
			name:   "password is redacted",
			detail: map[string]any{"password": "supersecret123"},
			key:    "password",
			want:   "[REDACTED]",
		},
		{
			name:   "secret is redacted",
			detail: map[string]any{"secret": "mysecretvalue"},
			key:    "secret",
			want:   "[REDACTED]",
		},
		{
			name:   "normal field is unchanged",
			detail: map[string]any{"username": "alice"},
			key:    "username",
			want:   "alice",
		},
		{
			name:   "long content is truncated",
			detail: map[string]any{"content": string(make([]byte, 2000))},
			key:    "content",
			want:   "", // will check length only
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scrubbed := ScrubDetail(tt.detail)
			val, ok := scrubbed[tt.key]
			if !ok {
				t.Fatalf("key %q missing from scrubbed detail", tt.key)
			}
			strVal, ok := val.(string)
			if !ok {
				t.Fatalf("expected string value for key %q, got %T", tt.key, val)
			}

			if tt.name == "long content is truncated" {
				if len(strVal) > 1024+len("[truncated]") {
					t.Fatalf("expected content truncated to ~1024 chars, got %d", len(strVal))
				}
				return
			}

			if strVal != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, strVal)
			}
		})
	}
}

// --- Test: Scrub nested sensitive data ---

func TestScrubDetailNested(t *testing.T) {
	detail := map[string]any{
		"config": map[string]any{
			"api_key":  "sk-nestedkey1234567890abcdef12345",
			"password": "nestedsecret",
			"name":     "safe",
		},
		"items": []any{
			map[string]any{"token": "abc123", "value": "ok"},
		},
		"name": "top-level",
	}

	scrubbed := ScrubDetail(detail)

	// Nested map: api_key should be masked
	nested, ok := scrubbed["config"].(map[string]any)
	if !ok {
		t.Fatal("expected config to be map")
	}
	if nested["api_key"] != "sk-neste*************************" {
		t.Fatalf("nested api_key = %q", nested["api_key"])
	}
	if nested["password"] != "[REDACTED]" {
		t.Fatalf("nested password = %q", nested["password"])
	}
	if nested["name"] != "safe" {
		t.Fatalf("nested name = %q", nested["name"])
	}

	// Nested array of maps: token should be redacted
	items, ok := scrubbed["items"].([]any)
	if !ok {
		t.Fatal("expected items to be []any")
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatal("expected item to be map")
	}
	if item["token"] != "[REDACTED]" {
		t.Fatalf("nested array token = %q", item["token"])
	}
	if item["value"] != "ok" {
		t.Fatalf("nested array value = %q", item["value"])
	}

	// Top-level normal field unchanged
	if scrubbed["name"] != "top-level" {
		t.Fatalf("top-level name = %q", scrubbed["name"])
	}
}

// --- Test: Dropped and Errors counters ---

func TestDroppedCounter(t *testing.T) {
	writer := &fakeWriter{}
	svc := NewService(writer, Config{BufferSize: 1, FlushInterval: time.Hour})
	// Don't start — buffer will fill and stay full.

	svc.Log(Entry{Action: "a"}) // fills buffer
	svc.Log(Entry{Action: "b"}) // dropped
	svc.Log(Entry{Action: "c"}) // dropped

	if svc.Dropped() < 1 {
		t.Fatalf("expected drops >= 1, got %d", svc.Dropped())
	}
}

func TestFlushErrorCounter(t *testing.T) {
	errWriter := &failWriter{}
	svc := NewService(errWriter, Config{BufferSize: 100, FlushInterval: 20 * time.Millisecond})
	svc.Start()

	svc.Log(Entry{Action: "a"})
	time.Sleep(80 * time.Millisecond)

	svc.Stop()
	if svc.Errors() < 1 {
		t.Fatalf("expected errors >= 1, got %d", svc.Errors())
	}
}

type failWriter struct{}

func (f *failWriter) WriteEntries(_ []Entry) error {
	return errFlush
}

var errFlush = errors.New("write failed")
