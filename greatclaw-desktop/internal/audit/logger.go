package audit

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Entry represents a single audit log entry.
type Entry struct {
	Timestamp time.Time      `json:"timestamp"`
	Action    string         `json:"action"`
	Path      string         `json:"path"`
	Result    string         `json:"result"`
	Error     string         `json:"error,omitempty"`
	Duration  time.Duration  `json:"duration_ms"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// Logger writes audit entries to a local JSON-lines file.
type Logger struct {
	file   *os.File
	mu     sync.Mutex
	logger *slog.Logger
}

// NewLogger creates a file-based audit logger.
func NewLogger(path string, log *slog.Logger) (*Logger, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	return &Logger{
		file:   f,
		logger: log.With(slog.String("component", "audit")),
	}, nil
}

// Log writes an audit entry.
func (l *Logger) Log(entry Entry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		l.logger.Error("failed to marshal audit entry", slog.String("error", err.Error()))
		return
	}
	data = append(data, '\n')
	if _, err := l.file.Write(data); err != nil {
		l.logger.Error("failed to write audit entry", slog.String("error", err.Error()))
	}
}

// Close flushes and closes the audit log.
func (l *Logger) Close() error {
	return l.file.Close()
}
