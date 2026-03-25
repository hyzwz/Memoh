package ctripcs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/memohai/memoh/internal/channel"
)

const (
	ctripSeenTTL           = 30 * time.Minute
	ctripSeenTrimInterval  = 5 * time.Minute
	ctripTransientMaxDelay = 5 * time.Second
)

var ctripTransientBackoffs = []time.Duration{
	250 * time.Millisecond,
	500 * time.Millisecond,
	1 * time.Second,
	2 * time.Second,
	5 * time.Second,
}

var errMissingBrowserContext = errors.New("ctrip_cs browser context is missing")

const ctripInboxSnapshotScript = `(() => {
  const root = globalThis.__MEMOH__ || globalThis.window?.__MEMOH__ || {};
  if (root && typeof root === 'object') {
    return root.snapshot ?? root.inbox ?? root.data ?? root;
  }
  return root;
})()`

type poller struct {
	cfg        Config
	channelCfg channel.ChannelConfig
	gateway    browserActionRunner
	seen       map[string]time.Time
	handler    channel.InboundHandler
	logger     *slog.Logger

	started  bool
	lastTrim time.Time
}

func (a *Adapter) Connect(ctx context.Context, cfg channel.ChannelConfig, handler channel.InboundHandler) (channel.Connection, error) {
	if handler == nil {
		return nil, errors.New("ctrip_cs inbound handler is required")
	}
	if a == nil || a.browserGateway == nil {
		return nil, errors.New("ctrip_cs browser gateway is not configured")
	}

	parsed, err := parseConfig(cfg.Credentials)
	if err != nil {
		return nil, err
	}

	p := &poller{
		cfg:        parsed,
		channelCfg: cfg,
		gateway:    a.browserGateway,
		seen:       make(map[string]time.Time),
		handler:    handler,
		logger:     a.logger,
	}

	if err := p.ensureBrowserContext(ctx); err != nil {
		return nil, err
	}

	connCtx, cancel := context.WithCancel(ctx)
	var stopOnce sync.Once
	done := make(chan struct{})
	conn := channel.NewConnection(cfg, func(stopCtx context.Context) error {
		stopOnce.Do(cancel)
		select {
		case <-done:
			return nil
		case <-stopCtx.Done():
			return stopCtx.Err()
		}
	})

	go func() {
		defer close(done)
		if err := p.run(connCtx); err != nil {
			if !errors.Is(err, context.Canceled) {
				p.logWarn("receiver stopped", err)
			}
			if !errors.Is(err, context.Canceled) {
				go func() {
					_ = conn.Stop(context.Background())
				}()
			}
		}
	}()

	return conn, nil
}

func (p *poller) ensureBrowserContext(ctx context.Context) error {
	if p == nil {
		return errors.New("ctrip_cs poller is not configured")
	}
	if p.gateway == nil {
		return errors.New("ctrip_cs browser gateway is not configured")
	}
	contextID := strings.TrimSpace(p.cfg.BrowserContextID)
	if contextID == "" {
		return errors.New("ctrip_cs browserContextId is required")
	}
	exists, err := p.gateway.Exists(ctx, contextID)
	if err != nil {
		return fmt.Errorf("check browser context: %w", err)
	}
	if !exists {
		return errMissingBrowserContext
	}
	return nil
}

func (p *poller) run(ctx context.Context) error {
	interval := time.Duration(p.cfg.PollIntervalMS) * time.Millisecond
	if interval <= 0 {
		interval = time.Duration(defaultPollIntervalMS) * time.Millisecond
	}

	for {
		if err := p.pollOnce(ctx); err != nil {
			if isHardConnectionError(err) {
				return err
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			p.logWarn("poll failed, retrying", err)
			for attempt := 0; ; attempt++ {
				if !sleepContext(ctx, ctripBackoffDelay(attempt)) {
					return ctx.Err()
				}
				if err := p.pollOnce(ctx); err != nil {
					if isHardConnectionError(err) {
						return err
					}
					if ctx.Err() != nil {
						return ctx.Err()
					}
					p.logWarn("poll retry failed", err)
					continue
				}
				break
			}
		}
		if !sleepContext(ctx, interval) {
			return ctx.Err()
		}
	}
}

func (p *poller) pollOnce(ctx context.Context) error {
	if p == nil {
		return errors.New("ctrip_cs poller is not configured")
	}
	if p.gateway == nil {
		return errors.New("ctrip_cs browser gateway is not configured")
	}
	contextID := strings.TrimSpace(p.cfg.BrowserContextID)
	if contextID == "" {
		return errors.New("ctrip_cs browserContextId is required")
	}

	exists, err := p.gateway.Exists(ctx, contextID)
	if err != nil {
		return fmt.Errorf("check browser context: %w", err)
	}
	if !exists {
		return errMissingBrowserContext
	}

	if !p.started {
		if err := p.gateway.Navigate(ctx, contextID, p.cfg.EntryURL); err != nil {
			return fmt.Errorf("navigate entry url: %w", err)
		}
		p.started = true
	}

	raw, err := p.gateway.Evaluate(ctx, contextID, ctripInboxSnapshotScript)
	if err != nil {
		return fmt.Errorf("evaluate inbox snapshot: %w", err)
	}

	_, messages, err := ParseInboxSnapshot(raw, p.cfg)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	p.trimSeen(now)
	for _, msg := range messages {
		id := strings.TrimSpace(msg.Message.ID)
		if id == "" {
			continue
		}
		if seenAt, ok := p.seen[id]; ok && now.Sub(seenAt) <= ctripSeenTTL {
			continue
		}
		if p.handler == nil {
			p.seen[id] = now
			continue
		}
		if err := p.handler(ctx, p.channelCfg, msg); err != nil {
			if p.logger != nil {
				p.logger.Warn("ctrip_cs inbound handler failed",
					slog.String("config_id", strings.TrimSpace(p.channelCfg.ID)),
					slog.String("message_id", id),
					slog.Any("error", err),
				)
			}
			continue
		}
		p.seen[id] = now
	}

	return nil
}

func (p *poller) trimSeen(now time.Time) {
	if p == nil {
		return
	}
	if !p.lastTrim.IsZero() && now.Sub(p.lastTrim) < ctripSeenTrimInterval {
		return
	}
	for id, seenAt := range p.seen {
		if now.Sub(seenAt) > ctripSeenTTL {
			delete(p.seen, id)
		}
	}
	p.lastTrim = now
}

func (p *poller) logWarn(msg string, err error) {
	if p == nil || p.logger == nil || err == nil {
		return
	}
	p.logger.Warn(msg,
		slog.String("config_id", strings.TrimSpace(p.channelCfg.ID)),
		slog.String("browser_context_id", strings.TrimSpace(p.cfg.BrowserContextID)),
		slog.Any("error", err),
	)
}

func isHardConnectionError(err error) bool {
	return errors.Is(err, errMissingBrowserContext) || errors.Is(err, ErrLoginExpired)
}

func ctripBackoffDelay(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	if attempt >= len(ctripTransientBackoffs) {
		return ctripTransientMaxDelay
	}
	delay := ctripTransientBackoffs[attempt]
	if delay > ctripTransientMaxDelay {
		return ctripTransientMaxDelay
	}
	return delay
}

func sleepContext(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
