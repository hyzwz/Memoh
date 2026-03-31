package dingtalk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	dingtalkclient "github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
	dingtalkhandler "github.com/open-dingtalk/dingtalk-stream-sdk-go/handler"
	dingtalkpayload "github.com/open-dingtalk/dingtalk-stream-sdk-go/payload"
	dingtalkutils "github.com/open-dingtalk/dingtalk-stream-sdk-go/utils"

	"github.com/memohai/memoh/internal/channel"
)

const streamTopicChatbotMessage = dingtalkpayload.BotMessageCallbackTopic

type streamAck struct {
	Success bool
	Message string
}

type streamEventHandler func(context.Context, StreamCallbackEnvelope) streamAck

type streamClient interface {
	Start(ctx context.Context) error
	Close() error
	Started() <-chan struct{}
}

type sdkStreamClient struct {
	client  *dingtalkclient.StreamClient
	started chan struct{}
	once    sync.Once
}

func newSDKStreamClient(cfg Config, handler streamEventHandler) (streamClient, error) {
	if handler == nil {
		return nil, errors.New("dingtalk stream handler is required")
	}

	frameHandler := dingtalkhandler.IFrameHandler(func(ctx context.Context, df *dingtalkpayload.DataFrame) (*dingtalkpayload.DataFrameResponse, error) {
		ack := handler(ctx, StreamCallbackEnvelope{
			Headers: StreamCallbackHeaders{
				Topic:       df.GetTopic(),
				MessageID:   df.GetHeader(dingtalkpayload.DataFrameHeaderKMessageId),
				ContentType: df.GetHeader(dingtalkpayload.DataFrameHeaderKContentType),
				Time:        df.GetHeader(dingtalkpayload.DataFrameHeaderKTime),
			},
			Data: json.RawMessage(df.Data),
		})

		resp := dingtalkpayload.NewSuccessDataFrameResponse()
		if !ack.Success {
			resp = dingtalkpayload.NewDataFrameResponse(dingtalkpayload.DataFrameResponseStatusCodeKInternalError)
			resp.Message = strings.TrimSpace(ack.Message)
		}
		return resp, nil
	})

	options := []dingtalkclient.ClientOption{
		dingtalkclient.WithAppCredential(dingtalkclient.NewAppCredentialConfig(cfg.AppKey, cfg.AppSecret)),
		dingtalkclient.WithAutoReconnect(false),
		dingtalkclient.WithSubscription(dingtalkutils.SubscriptionTypeKCallback, streamTopicChatbotMessage, frameHandler),
	}
	if value := strings.TrimSpace(cfg.StreamEndpoint); value != "" {
		options = append(options, dingtalkclient.WithOpenApiHost(value))
	}

	return &sdkStreamClient{
		client:  dingtalkclient.NewStreamClient(options...),
		started: make(chan struct{}),
	}, nil
}

func (c *sdkStreamClient) Start(ctx context.Context) error {
	if c == nil || c.client == nil {
		return errors.New("dingtalk stream client not configured")
	}
	if err := c.client.Start(ctx); err != nil {
		return err
	}
	c.once.Do(func() {
		close(c.started)
	})

	<-ctx.Done()
	c.client.Close()
	return nil
}

func (c *sdkStreamClient) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	c.client.Close()
	return nil
}

func (c *sdkStreamClient) Started() <-chan struct{} {
	if c == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return c.started
}

func (a *Adapter) Connect(ctx context.Context, cfg channel.ChannelConfig, handler channel.InboundHandler) (channel.Connection, error) {
	if handler == nil {
		return nil, errors.New("dingtalk inbound handler is required")
	}

	parsed, err := parseConfig(cfg.Credentials)
	if err != nil {
		return nil, err
	}

	clientFactory := newClient
	if a != nil && a.newClient != nil {
		clientFactory = a.newClient
	}
	apiClient := clientFactory(parsed)

	streamFactory := newSDKStreamClient
	if a != nil && a.newStreamClient != nil {
		streamFactory = a.newStreamClient
	}
	receiver, err := streamFactory(parsed, func(eventCtx context.Context, envelope StreamCallbackEnvelope) streamAck {
		return a.handleStreamEnvelope(eventCtx, cfg, parsed, apiClient, handler, envelope)
	})
	if err != nil {
		return nil, err
	}

	connCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	startErrCh := make(chan error, 1)
	go func() {
		defer close(done)
		if err := receiver.Start(connCtx); err != nil && !errors.Is(err, context.Canceled) {
			startErrCh <- err
			if a != nil && a.logger != nil {
				a.logger.Warn("dingtalk receiver stopped", slog.String("config_id", cfg.ID), slog.Any("error", err))
			}
		}
	}()

	select {
	case <-receiver.Started():
	case err := <-startErrCh:
		cancel()
		_ = receiver.Close()
		<-done
		return nil, fmt.Errorf("start dingtalk receiver: %w", err)
	case <-ctx.Done():
		cancel()
		_ = receiver.Close()
		<-done
		return nil, ctx.Err()
	}

	return channel.NewConnection(cfg, func(stopCtx context.Context) error {
		cancel()
		if err := receiver.Close(); err != nil {
			return err
		}
		select {
		case <-done:
			return nil
		case <-stopCtx.Done():
			return stopCtx.Err()
		}
	}), nil
}

func (a *Adapter) handleStreamEnvelope(ctx context.Context, cfg channel.ChannelConfig, parsed Config, apiClient *client, handler channel.InboundHandler, envelope StreamCallbackEnvelope) streamAck {
	ack := streamAck{Success: true}

	result, ok, err := streamEnvelopeToInboundEvent(parsed, strings.TrimSpace(cfg.BotID), envelope)
	if err != nil {
		a.logStreamError("decode inbound event", cfg, envelope, err)
		return failedStreamAck(err)
	}
	if !ok {
		return ack
	}

	if result.Command != nil {
		if err := a.handleSessionReset(ctx, cfg, apiClient, result); err != nil {
			a.logStreamError("handle session reset", cfg, envelope, err)
			return failedStreamAck(err)
		}
		return ack
	}

	if err := handler(ctx, cfg, result.Message); err != nil {
		a.logStreamError("dispatch inbound event", cfg, envelope, err)
		return failedStreamAck(err)
	}
	return ack
}

func (a *Adapter) handleSessionReset(ctx context.Context, cfg channel.ChannelConfig, apiClient *client, result InboundEventResult) error {
	if result.Command == nil {
		return nil
	}
	if a == nil || a.sessionCommandHandler == nil {
		return fmt.Errorf("dingtalk session reset handler not configured")
	}

	confirmation, err := a.sessionCommandHandler.HandleReset(ctx, cfg, result.Message, *result.Command)
	if err != nil {
		return err
	}
	if strings.TrimSpace(confirmation) == "" {
		return nil
	}

	target, err := parseTarget(result.Command.ReplyTarget)
	if err != nil {
		return err
	}
	if target.Family != targetFamilyWebhook {
		return fmt.Errorf("dingtalk reset confirmation requires webhook reply target")
	}
	return apiClient.replyText(ctx, target.Value, confirmation)
}

func (a *Adapter) logStreamError(action string, cfg channel.ChannelConfig, envelope StreamCallbackEnvelope, err error) {
	if a == nil || a.logger == nil || err == nil {
		return
	}
	a.logger.Warn("dingtalk stream receiver error",
		slog.String("action", action),
		slog.String("config_id", strings.TrimSpace(cfg.ID)),
		slog.String("bot_id", strings.TrimSpace(cfg.BotID)),
		slog.String("topic", strings.TrimSpace(envelope.Headers.Topic)),
		slog.String("message_id", strings.TrimSpace(envelope.Headers.MessageID)),
		slog.Any("error", err),
	)
}

func failedStreamAck(err error) streamAck {
	if err == nil {
		return streamAck{}
	}
	return streamAck{
		Success: false,
		Message: strings.TrimSpace(err.Error()),
	}
}
