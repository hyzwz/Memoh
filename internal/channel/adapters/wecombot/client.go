package wecombot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type inboundFrameHandler func(context.Context, wsFrame) error

type pendingAck struct {
	result chan wsFrame
}

type wsClient struct {
	logger     *slog.Logger
	cfg        Config
	dialer     *websocket.Dialer
	httpClient *http.Client
	onFrame    inboundFrameHandler

	writeMu   sync.Mutex
	stateMu   sync.RWMutex
	conn      *websocket.Conn
	connected atomic.Bool

	pendingMu sync.Mutex
	pending   map[string]pendingAck

	readyOnce sync.Once
	readyCh   chan error
}

func newWSClient(log *slog.Logger, cfg Config, handler inboundFrameHandler) *wsClient {
	dialer := *websocket.DefaultDialer
	dialer.HandshakeTimeout = connectTimeout
	return &wsClient{
		logger:     log,
		cfg:        cfg,
		dialer:     &dialer,
		httpClient: &http.Client{Timeout: 60 * time.Second},
		onFrame:    handler,
		pending:    make(map[string]pendingAck),
		readyCh:    make(chan error, 1),
	}
}

func (c *wsClient) Run(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		if err := c.serve(ctx); err != nil {
			if ctx.Err() == nil {
				c.signalReady(err)
			}
			if ctx.Err() == nil && c.logger != nil {
				c.logger.Warn("wecom ai bot reconnect", slog.Any("error", err))
			}
		}
		if ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > reconnectMaxDelay {
			backoff = reconnectMaxDelay
		}
	}
}

func (c *wsClient) Close() error {
	c.stateMu.Lock()
	conn := c.conn
	c.conn = nil
	c.connected.Store(false)
	c.stateMu.Unlock()
	c.failPending(errors.New("wecom ai bot connection closed"))
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (c *wsClient) Connected() bool {
	return c.connected.Load()
}

func (c *wsClient) WaitReady(ctx context.Context) error {
	if c == nil {
		return errors.New("wecom ai bot client is not initialized")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err, ok := <-c.readyCh:
		if !ok {
			return nil
		}
		return err
	}
}

func (c *wsClient) SendMarkdown(ctx context.Context, chatID, content string) error {
	reqID := generateReqID(wsCmdSendMessage)
	_, err := c.sendRequest(ctx, outboundFrame{
		Cmd:     wsCmdSendMessage,
		Headers: wsHeaders{ReqID: reqID},
		Body:    buildSendMarkdownBody(chatID, content),
	}, replyAckTimeout)
	return err
}

func (c *wsClient) ReplyStream(ctx context.Context, sourceReqID, streamID, content string, finish bool) error {
	if sourceReqID == "" {
		return errors.New("source req_id is required")
	}
	_, err := c.sendRequest(ctx, outboundFrame{
		Cmd:     wsCmdReply,
		Headers: wsHeaders{ReqID: sourceReqID},
		Body:    buildStreamReplyBody(streamID, content, finish),
	}, replyAckTimeout)
	return err
}

func (c *wsClient) serve(ctx context.Context) error {
	conn, resp, err := c.dialer.DialContext(ctx, c.cfg.WebSocketURL, nil)
	if err != nil {
		if c.logger != nil && resp != nil {
			c.logger.Warn("wecom ai bot websocket dial failed",
				slog.String("url", c.cfg.WebSocketURL),
				slog.Int("status_code", resp.StatusCode),
				slog.Any("error", err),
			)
		}
		return err
	}
	if c.logger != nil {
		c.logger.Info("wecom ai bot websocket connected", slog.String("url", c.cfg.WebSocketURL))
	}
	c.setConn(conn)
	defer func() {
		_ = conn.Close()
		c.clearConn(conn)
	}()

	readErrCh := make(chan error, 1)
	go c.readLoop(ctx, conn, readErrCh)

	authReqID := generateReqID(wsCmdSubscribe)
	authCh, err := c.registerPending(authReqID)
	if err != nil {
		return err
	}
	if err := c.sendFrame(outboundFrame{
		Cmd:     wsCmdSubscribe,
		Headers: wsHeaders{ReqID: authReqID},
		Body: map[string]any{
			"bot_id": c.cfg.BotID,
			"secret": c.cfg.Secret,
		},
	}); err != nil {
		c.unregisterPending(authReqID)
		return err
	}
	authCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	authResp, err := c.waitAck(authCtx, authReqID, authCh)
	if err != nil {
		return err
	}
	if authResp.ErrCode != 0 {
		return fmt.Errorf("wecom ai bot authentication failed: %s (errcode: %d)", authResp.ErrMsg, authResp.ErrCode)
	}
	c.connected.Store(true)
	c.signalReady(nil)
	if c.logger != nil {
		c.logger.Info("wecom ai bot websocket authenticated", slog.String("bot_id", c.cfg.BotID))
	}

	heartbeatStop := make(chan struct{})
	defer close(heartbeatStop)
	go c.runHeartbeat(ctx, heartbeatStop)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-readErrCh:
		return err
	}
}

func (c *wsClient) readLoop(ctx context.Context, conn *websocket.Conn, errCh chan<- error) {
	for {
		select {
		case <-ctx.Done():
			c.pushReadErr(errCh, ctx.Err())
			return
		default:
		}
		_ = conn.SetReadDeadline(time.Now().Add(readTimeout))
		_, payload, err := conn.ReadMessage()
		if err != nil {
			c.failPending(err)
			c.pushReadErr(errCh, err)
			return
		}
		var frame wsFrame
		if err := json.Unmarshal(payload, &frame); err != nil {
			if c.logger != nil {
				c.logger.Warn("wecom ai bot decode frame failed", slog.Any("error", err))
			}
			continue
		}
		if c.handleAckFrame(frame) {
			continue
		}
		c.dispatchInboundFrame(ctx, frame)
	}
}

func (c *wsClient) dispatchInboundFrame(ctx context.Context, frame wsFrame) {
	if len(frame.Body) == 0 || c.onFrame == nil {
		return
	}
	frameCtx := context.WithoutCancel(ctx)
	go func() {
		if err := c.onFrame(frameCtx, frame); err != nil && c.logger != nil {
			c.logger.Warn("wecom ai bot inbound handler failed", slog.Any("error", err))
		}
	}()
}

func (c *wsClient) pushReadErr(errCh chan<- error, err error) {
	select {
	case errCh <- err:
	default:
	}
}

func (c *wsClient) signalReady(err error) {
	c.readyOnce.Do(func() {
		c.readyCh <- err
		close(c.readyCh)
	})
}

func (c *wsClient) runHeartbeat(ctx context.Context, stop <-chan struct{}) {
	ticker := time.NewTicker(defaultHeartbeatInterval)
	defer ticker.Stop()
	missed := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-ticker.C:
			if !c.Connected() {
				continue
			}
			reqID := generateReqID(wsCmdHeartbeat)
			_, err := c.sendRequest(ctx, outboundFrame{
				Cmd:     wsCmdHeartbeat,
				Headers: wsHeaders{ReqID: reqID},
			}, heartbeatAckTimeout)
			if err != nil {
				if c.logger != nil {
					c.logger.Warn("wecom ai bot heartbeat send failed", slog.Any("error", err))
				}
				missed++
			} else {
				missed = 0
			}
			if missed >= 3 {
				_ = c.Close()
				return
			}
		}
	}
}

func (c *wsClient) handleAckFrame(frame wsFrame) bool {
	reqID := frame.Headers.ReqID
	if reqID == "" {
		return false
	}
	c.pendingMu.Lock()
	pending, ok := c.pending[reqID]
	if ok {
		delete(c.pending, reqID)
	}
	c.pendingMu.Unlock()
	if !ok {
		return false
	}
	select {
	case pending.result <- frame:
	default:
	}
	close(pending.result)
	return true
}

func (c *wsClient) sendRequest(ctx context.Context, frame outboundFrame, timeout time.Duration) (wsFrame, error) {
	ackCh, err := c.registerPending(frame.Headers.ReqID)
	if err != nil {
		return wsFrame{}, err
	}
	if err := c.sendFrame(frame); err != nil {
		c.unregisterPending(frame.Headers.ReqID)
		return wsFrame{}, err
	}
	waitCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()
	return c.waitAck(waitCtx, frame.Headers.ReqID, ackCh)
}

func (c *wsClient) waitAck(ctx context.Context, reqID string, ackCh <-chan wsFrame) (wsFrame, error) {
	select {
	case <-ctx.Done():
		c.unregisterPending(reqID)
		return wsFrame{}, ctx.Err()
	case frame, ok := <-ackCh:
		if !ok {
			return wsFrame{}, errors.New("wecom ai bot ack channel closed")
		}
		if frame.ErrCode != 0 {
			return frame, fmt.Errorf("wecom ai bot request failed: %s (errcode: %d)", frame.ErrMsg, frame.ErrCode)
		}
		return frame, nil
	}
}

func (c *wsClient) sendFrame(frame outboundFrame) error {
	conn := c.getConn()
	if conn == nil {
		return errors.New("wecom ai bot websocket is not connected")
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return conn.WriteJSON(frame)
}

func (c *wsClient) registerPending(reqID string) (<-chan wsFrame, error) {
	if reqID == "" {
		return nil, errors.New("req_id is required")
	}
	ch := make(chan wsFrame, 1)
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()
	if _, exists := c.pending[reqID]; exists {
		return nil, fmt.Errorf("pending req_id already exists: %s", reqID)
	}
	c.pending[reqID] = pendingAck{result: ch}
	return ch, nil
}

func (c *wsClient) unregisterPending(reqID string) {
	c.pendingMu.Lock()
	pending, ok := c.pending[reqID]
	if ok {
		delete(c.pending, reqID)
	}
	c.pendingMu.Unlock()
	if ok {
		close(pending.result)
	}
}

func (c *wsClient) failPending(err error) {
	c.pendingMu.Lock()
	items := c.pending
	c.pending = make(map[string]pendingAck)
	c.pendingMu.Unlock()
	for _, pending := range items {
		if err != nil {
			pending.result <- wsFrame{ErrCode: -1, ErrMsg: err.Error()}
		}
		close(pending.result)
	}
}

func (c *wsClient) setConn(conn *websocket.Conn) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	c.conn = conn
}

func (c *wsClient) clearConn(conn *websocket.Conn) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	if c.conn == conn {
		c.conn = nil
	}
	c.connected.Store(false)
}

func (c *wsClient) getConn() *websocket.Conn {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.conn
}
