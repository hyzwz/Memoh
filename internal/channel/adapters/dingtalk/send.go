package dingtalk

import (
	"context"
	"fmt"
	"strings"

	"github.com/memohai/memoh/internal/channel"
)

func (a *Adapter) Send(ctx context.Context, cfg channel.ChannelConfig, msg channel.OutboundMessage) error {
	parsedCfg, err := parseConfig(cfg.Credentials)
	if err != nil {
		return err
	}
	if strings.TrimSpace(parsedCfg.RobotCode) == "" {
		parsedCfg.RobotCode = selfIdentityRobotCode(cfg.SelfIdentity)
	}
	target, err := parseTarget(msg.Target)
	if err != nil {
		return err
	}

	text := strings.TrimSpace(msg.Message.PlainText())
	if text == "" && len(msg.Message.Attachments) == 0 {
		return fmt.Errorf("dingtalk message text is required")
	}

	if text != "" {
		switch target.Family {
		case targetFamilyWebhook:
			client := a.getClient(parsedCfg)
			if err := client.replyText(ctx, target.Value, text); err != nil {
				return err
			}
		case targetFamilyConversation:
			client := a.getClient(parsedCfg)
			if err := client.sendConversationText(ctx, parsedCfg, target.Value, text); err != nil {
				return err
			}
		case targetFamilyUser:
			client := a.getClient(parsedCfg)
			if err := client.sendUserText(ctx, parsedCfg, target, text); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported dingtalk target family %q", target.Family)
		}
	}

	for _, attachment := range msg.Message.Attachments {
		if err := a.sendAttachment(ctx, parsedCfg, cfg, target, attachment); err != nil {
			return err
		}
	}
	return nil
}

func (a *Adapter) sendAttachment(ctx context.Context, cfg Config, channelCfg channel.ChannelConfig, target Target, attachment channel.Attachment) error {
	src, err := a.prepareOutboundAttachment(ctx, cfg, channelCfg, attachment)
	if err != nil {
		return err
	}
	client := a.getClient(cfg)
	uploadType := "file"
	switch src.Type {
	case channel.AttachmentImage:
		uploadType = "image"
	default:
		uploadType = "file"
	}
	mediaID, err := client.uploadMedia(ctx, cfg, uploadType, src.FileName, src.Data)
	if err != nil {
		return err
	}
	src.MediaID = mediaID
	switch target.Family {
	case targetFamilyWebhook:
		return client.sendWebhookAttachment(ctx, target.Value, src)
	case targetFamilyConversation:
		return client.sendProactiveAttachment(ctx, cfg, target, src)
	case targetFamilyUser:
		return client.sendProactiveAttachment(ctx, cfg, target, src)
	default:
		return fmt.Errorf("unsupported dingtalk target family %q", target.Family)
	}
}

func selfIdentityRobotCode(raw map[string]any) string {
	for _, key := range []string{"robotCode", "robot_code"} {
		value, ok := raw[key]
		if !ok {
			continue
		}
		if text, ok := value.(string); ok {
			return strings.TrimSpace(text)
		}
	}
	return ""
}
