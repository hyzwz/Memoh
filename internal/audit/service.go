package audit

import (
	"log/slog"
	"sync/atomic"
	"time"
)

// Writer persists audit entries to storage.
type Writer interface {
	WriteEntries(entries []Entry) error
}

// Config configures the audit service.
type Config struct {
	BufferSize    int
	FlushInterval time.Duration
	Logger        *slog.Logger
}

// Service provides non-blocking audit logging with buffered writes.
type Service struct {
	writer   Writer
	config   Config
	logger   *slog.Logger
	buffer   chan Entry
	done     chan struct{}
	stopped  chan struct{}
	dropped  atomic.Int64
	errCount atomic.Int64
}

// NewService creates a new audit service.
func NewService(writer Writer, config Config) *Service {
	if config.BufferSize <= 0 {
		config.BufferSize = 1000
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = time.Second
	}
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		writer:  writer,
		config:  config,
		logger:  logger,
		buffer:  make(chan Entry, config.BufferSize),
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

// Start begins the background flush goroutine.
func (s *Service) Start() {
	go s.flushLoop()
}

// Log adds an entry to the buffer. Non-blocking; drops entry if buffer is full.
func (s *Service) Log(entry Entry) {
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	select {
	case s.buffer <- entry:
	default:
		cnt := s.dropped.Add(1)
		if cnt == 1 || cnt%100 == 0 {
			s.logger.Warn("audit entry dropped: buffer full",
				"total_dropped", cnt,
				"action", entry.Action,
			)
		}
	}
}

// Dropped returns the number of entries dropped due to buffer full.
func (s *Service) Dropped() int64 {
	return s.dropped.Load()
}

// Errors returns the number of flush write errors.
func (s *Service) Errors() int64 {
	return s.errCount.Load()
}

// Stop signals the flush loop to exit and flushes remaining entries.
func (s *Service) Stop() {
	close(s.done)
	<-s.stopped
}

func (s *Service) flushLoop() {
	defer close(s.stopped)
	ticker := time.NewTicker(s.config.FlushInterval)
	defer ticker.Stop()

	var batch []Entry

	for {
		select {
		case entry := <-s.buffer:
			batch = append(batch, entry)
			if len(batch) >= s.config.BufferSize {
				s.flush(batch)
				batch = nil
			}
		case <-ticker.C:
			if len(batch) > 0 {
				s.flush(batch)
				batch = nil
			}
		case <-s.done:
			// Drain remaining buffer entries.
			for {
				select {
				case entry := <-s.buffer:
					batch = append(batch, entry)
				default:
					if len(batch) > 0 {
						s.flush(batch)
					}
					return
				}
			}
		}
	}
}

func (s *Service) flush(batch []Entry) {
	if len(batch) == 0 {
		return
	}
	if err := s.writer.WriteEntries(batch); err != nil {
		cnt := s.errCount.Add(1)
		s.logger.Error("audit flush failed",
			"error", err,
			"batch_size", len(batch),
			"total_errors", cnt,
		)
	}
}
