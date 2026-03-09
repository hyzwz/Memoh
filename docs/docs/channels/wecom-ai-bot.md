# WeCom AI Bot Channel Configuration

Memoh supports integrating with the Enterprise WeChat intelligent bot platform through the long-lived WebSocket channel model. This is the same Bot ID + Secret style flow used by the official OpenClaw WeCom plugin.

## Step 1: Create a WeCom AI Bot

1. Open the Enterprise WeChat client.
2. Go to **Workbench** > **Intelligent Bot**.
3. Create a bot in **API mode**.
4. Choose the **long connection** access method.
5. Copy the generated **Bot ID** and **Secret**.

## Step 2: Configure Memoh

1. Open your Bot in the Memoh Web UI.
2. Go to the **Channels** tab.
3. Click **Add Channel** and select **WeCom AI Bot**.
4. Fill in:
   - `Bot ID`
   - `Secret`
   - `WebSocket URL` only if Tencent gives you a non-default endpoint
5. Save and enable the channel.

By default, Memoh connects to:

```text
wss://openws.work.weixin.qq.com
```

## Step 3: Talk to the Bot

After the channel is enabled, send a message to the Enterprise WeChat bot. Memoh will receive the inbound callback over WebSocket and reply on the same request stream.

## Step 4: Optional User Binding

If you want proactive outbound delivery or stricter identity mapping, create a user binding for this channel with one of:

- `chatid` for a direct chat or group chat target
- `userid` when you only need identity matching

For proactive outbound delivery, `chatid` is preferred.

## CLI Configuration

Memoh CLI also supports this channel:

```bash
memoh channel config set <bot_id> --type wecom_ai_bot --bot_id <wecom_bot_id> --secret <wecom_secret>
memoh channel bind set --type wecom_ai_bot --chat_id <chatid>
```

## Notes

- Streaming replies are supported.
- Attachment ingest from inbound messages is supported.
- Proactive outbound send currently requires a `chatid` target.
