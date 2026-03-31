# DingTalk Channel Design

**Date:** 2026-03-23

**Goal:** Add a full-featured DingTalk messaging channel to Memoh using DingTalk enterprise application Robot Stream mode, covering inbound chat, context replies, proactive sends, configurable group session routing, session reset commands, media attachments, and AI Card streaming without changing Memoh's core channel framework.

**Recommendation:** Implement this as a Go-native Memoh channel adapter whose product behavior closely follows `DingTalk-Real-AI/dingtalk-openclaw-connector`, while keeping all DingTalk-specific protocol details inside `internal/channel/adapters/dingtalk/`.

---

## Context

Memoh already exposes the right extension points for an IM integration:

- `Adapter` for metadata and schemas
- `ConfigNormalizer` and `SelfDiscoverer` for persisted config lifecycle
- `Sender` and `StreamSender` for outbound text, attachments, and streaming sessions
- `Receiver` for long-lived inbound transport
- `AttachmentResolver` and processing-status callbacks for richer message lifecycles

Existing adapters such as QQ and Feishu confirm the intended shape:

- adapters register themselves in startup wiring
- adapters normalize configuration and target syntax
- adapters translate platform events into `channel.InboundMessage`
- adapters perform platform-specific delivery while Memoh core remains platform-agnostic

The reference project `DingTalk-Real-AI/dingtalk-openclaw-connector` shows the correct DingTalk product model for this feature set:

- DingTalk enterprise internal app with robot capability
- Stream mode for inbound messages and callbacks
- `sessionWebhook` as the preferred in-context reply path
- proactive send APIs for user or conversation targets
- AI Card as the primary streaming UX
- message-session controls such as `/new`
- image/file/audio support as message-channel features

That behavior should be reproduced in Memoh, but the implementation must remain aligned with Memoh's native Go adapter architecture rather than embedding or proxying an external connector runtime.

---

## Scope

### In scope

- enterprise app credentials (`appKey`, `appSecret`)
- Stream mode inbound connection and lifecycle management
- private chat inbound messages
- group chat inbound messages gated by `@bot`
- session replies via `sessionWebhook`
- proactive outbound messaging to conversations and users
- configurable group session routing:
  - shared group session
  - per-sender group session
- session reset commands:
  - `/new`
  - `新会话`
- AI Card streaming as the primary streaming UX
- plain text fallback when AI Card creation/update is unavailable
- inbound image, file, and audio/voice attachment ingestion
- outbound image, file, and audio/file delivery
- config validation and normalization
- self identity discovery for mention matching
- adapter registration in both server entrypoints
- unit tests for config, target handling, inbound mapping, AI Card streaming, and media transport helpers

### Out of scope

- DingTalk document API integration
- multi-account channel fan-out inside one adapter instance
- multi-agent routing policy surfaces beyond Memoh's existing bot/channel model
- message editing after final send outside AI Card update lifecycle
- reactions
- org directory sync or user search UI
- async ack mode and separate "task accepted" delivery flow

---

## Chosen Approach

### Approach 1: Go-native adapter with reference-behavior parity

This is the chosen approach.

Why it fits:

- matches Memoh's existing adapter contracts
- preserves one runtime and one source of truth for routing and message state
- lets us copy the proven DingTalk product behavior from the reference connector without importing its plugin architecture
- keeps future capabilities inside the same adapter package

Trade-offs:

- more protocol work must be implemented directly in Go
- AI Card lifecycle and media transport require more up-front code than a text-only adapter

### Approach 2: Minimal Phase A text-only adapter

Rejected.

Why not:

- it does not meet the approved scope
- it would force a second redesign for AI Card, proactive send, and media

### Approach 3: Bridge to an external DingTalk connector

Rejected.

Why not:

- splits state and observability across runtimes
- weak fit for Memoh's channel lifecycle and attachment pipeline
- likely to require later migration back into a native adapter

---

## Architecture

Add a new adapter package:

- `internal/channel/adapters/dingtalk/`

The package is split into two layers:

### Adapter layer

This layer speaks Memoh contracts:

- `Descriptor()`
- `NormalizeConfig()`
- `NormalizeUserConfig()`
- `NormalizeTarget()`
- `ResolveTarget()`
- `DiscoverSelf()`
- `Connect()`
- `Send()`
- `OpenStream()`
- attachment resolution helpers if required by media flow

### DingTalk protocol layer

This layer speaks DingTalk:

- token acquisition
- self identity lookup
- Stream connection lifecycle
- inbound event decode and ACK
- `sessionWebhook` replies
- proactive send endpoints
- media upload/download
- AI Card create/update/finalize

Boundary rule:

- Memoh core only sees normalized `InboundMessage`, `OutboundMessage`, `StreamEvent`, and `Attachment` values.
- All DingTalk request/response structs, target parsing rules, and card payload details stay inside the adapter package.

---

## Message Model

### Inbound message model

Every accepted DingTalk event is converted into one `channel.InboundMessage`.

Required fields:

- `Channel = dingtalk`
- `ReplyTarget = webhook:<sessionWebhook>` when a session webhook is present
- `Conversation.ID = <conversationId>`
- `Conversation.Type = private | group`
- `Sender.SubjectID = stable sender identifier`
- `Message.Text` from normalized text content
- `Message.Attachments` for image/file/audio inputs

Metadata should preserve DingTalk-specific fields needed for diagnostics and future behavior:

- `session_webhook`
- `session_webhook_expired_time`
- `conversation_type`
- `sender_staff_id`
- `sender_union_id`
- `robot_code`
- `chatbot_user_id`
- `is_mentioned`
- `at_users`
- `raw_msgtype`
- `card_supported`

### Outbound target model

The adapter supports exactly three target families:

- `webhook:<sessionWebhook>`
- `conversation:<conversationId>`
- `user:<staffId>` or `user:<unionId>`

Priority rules:

1. Replies to inbound messages use `ReplyTarget` when available.
2. Explicit `conversation:` and `user:` targets use proactive send APIs.
3. AI Card streaming prefers in-context session targets and only opens against proactive targets when DingTalk supports that path explicitly.

The adapter must not infer missing reply context from route metadata during `Send()`. Outbound behavior depends on the normalized target string alone.

---

## Session Routing

### Private chats

Private chats always map to a per-user session.

Route key:

- `dingtalk:<botID>:<conversationId>`

### Group chats

Group chats are configurable via adapter config:

- `groupSessionScope = group`
- `groupSessionScope = group_sender`

Route key behavior:

- `group`: `dingtalk:<botID>:<conversationId>`
- `group_sender`: `dingtalk:<botID>:<conversationId>:<senderSubjectID>`

### Sender identity selection

For route keys and stable sender identity, choose the first non-empty identifier in this order:

1. `staffId`
2. `unionId`
3. other DingTalk stable sender/chatbot IDs
4. display name only as a last-resort fallback

### Group gating

Group chat messages are processed only when the bot is mentioned.

Default config:

- `requireMentionInGroup = true`

This applies to both normal messages and session reset commands to avoid accidental bot activation in busy groups.

---

## Session Commands

Supported commands:

- `/new`
- `新会话`

Behavior:

- command detection happens before LLM processing
- the current route/session is reset
- the adapter sends a confirmation reply using the current reply path
- in group mode:
  - `group` resets the shared group session
  - `group_sender` resets only the sender-scoped session

These commands belong to the adapter integration design because they directly affect route/session boundaries for DingTalk conversations.

---

## AI Card Streaming

AI Card is the primary streaming UX for DingTalk.

The adapter should implement `channel.StreamSender` by exposing `OpenStream(...)`.

### Stream lifecycle

For supported targets:

- create a card session
- buffer and throttle delta updates
- render tool/status/phase information conservatively
- finalize the card on `StreamEventFinal`
- mark failure or downgrade cleanly on `StreamEventError`

### Event mapping

Supported first-release event handling:

- `StreamEventStatus`
- `StreamEventDelta`
- `StreamEventFinal`
- `StreamEventError`
- `StreamEventToolCallStart`
- `StreamEventToolCallEnd`
- `StreamEventPhaseStart`
- `StreamEventPhaseEnd`

Attachment events may be summarized in the card or deferred to follow-up message delivery if card payload constraints make inline rendering unreliable.

### Fallback rules

If AI Card creation or update fails:

1. try a plain text reply through `webhook:<sessionWebhook>`
2. otherwise try supported proactive text delivery
3. otherwise return an explicit send error

This ensures the channel remains usable when the card path is degraded.

---

## Media and Attachments

### Inbound

Support:

- images
- files
- audio/voice

Flow:

- decode the DingTalk message type
- resolve or download the media bytes from DingTalk
- map to `channel.Attachment`
- preserve DingTalk media identifiers in `PlatformKey` and metadata

### Outbound

Support:

- image sends
- file sends
- audio sends when the DingTalk API path is stable enough; otherwise fall back to file delivery

If an outbound message contains both text and media, the adapter may split delivery into multiple DingTalk messages while preserving order.

If an attachment cannot be resolved to bytes or a known uploaded media token, return an explicit error rather than silently dropping it.

---

## Config Design

### Channel config

Required:

- `appKey`
- `appSecret`

Optional:

- `robotCode`
- `streamEndpoint`
- `apiBaseURL`
- `groupSessionScope`
- `requireMentionInGroup`
- `enableSessionCommands`
- `sessionResetCommands`
- `enableAICard`
- `cardUpdateThrottleMs`

Accepted aliases should include underscore and camelCase forms where the rest of Memoh already uses that normalization style.

### User config

User binding config should support future proactive delivery without introducing wider routing policy:

- `conversationId`
- `staffId`
- `unionId`

### Self identity

`DiscoverSelf(...)` should persist the bot's DingTalk identity fields needed for mention matching and diagnostics:

- bot/chatbot user id when available
- staff or union identifiers when available
- robot code when returned by DingTalk

The adapter should store all reliable identifiers and use the most specific available one during mention checks.

---

## File Plan

### New files

- `internal/channel/adapters/dingtalk/dingtalk.go`
  - adapter entrypoint, descriptor, interface delegation
- `internal/channel/adapters/dingtalk/config.go`
  - config parse and normalization
- `internal/channel/adapters/dingtalk/target.go`
  - target parsing and resolution
- `internal/channel/adapters/dingtalk/client.go`
  - HTTP/API helpers, token lookup, proactive send, card operations, media operations
- `internal/channel/adapters/dingtalk/protocol.go`
  - DingTalk request and response structs
- `internal/channel/adapters/dingtalk/inbound.go`
  - inbound event to `InboundMessage` mapping
- `internal/channel/adapters/dingtalk/stream.go`
  - Stream connection lifecycle and event dispatch
- `internal/channel/adapters/dingtalk/send.go`
  - plain send and media send entrypoints
- `internal/channel/adapters/dingtalk/card_stream.go`
  - `OpenStream` implementation and AI Card session lifecycle
- `internal/channel/adapters/dingtalk/attachments.go`
  - media upload/download and attachment conversion helpers
- `internal/channel/adapters/dingtalk/commands.go`
  - `/new` and reset-command handling
- `internal/channel/adapters/dingtalk/config_test.go`
- `internal/channel/adapters/dingtalk/target_test.go`
- `internal/channel/adapters/dingtalk/inbound_test.go`
- `internal/channel/adapters/dingtalk/send_test.go`
- `internal/channel/adapters/dingtalk/card_stream_test.go`
- `internal/channel/adapters/dingtalk/attachments_test.go`

### Modified files

- `cmd/agent/main.go`
- `cmd/memoh/serve.go`
- `go.mod`
- `go.sum`

### Files intentionally not modified

- `internal/channel/registry.go`
- `internal/channel/lifecycle.go`
- `internal/channel/manager.go`

The adapter must fit the existing channel extension points instead of forcing core framework changes.

---

## Testing Strategy

Required coverage:

- config normalization and validation
- target normalization and parsing
- self discovery
- group mention gating
- session route-key generation
- session reset command interception
- reply delivery through `sessionWebhook`
- proactive send through `conversation:` and `user:` targets
- AI Card lifecycle:
  - create
  - throttled updates
  - finalize
  - fallback on failure
- inbound media mapping
- outbound media upload/send

Prefer `httptest`, table-driven unit tests, and pure mapping helpers that keep most behavior testable without a live DingTalk environment.

---

## Risks and Mitigations

- **AI Card API volatility**
  - Mitigation: isolate card payloads and lifecycle in `card_stream.go`; provide plain-text fallback.
- **Mention matching ambiguity**
  - Mitigation: store multiple self identifiers and keep mention logic test-driven with realistic payloads.
- **Media API differences by message type**
  - Mitigation: centralize upload/download behavior in one helper layer rather than spreading it across send/inbound code.
- **Session routing regressions**
  - Mitigation: keep route-key generation pure and heavily unit-tested.

---

## Success Criteria

The feature is complete when:

- a DingTalk enterprise app can connect through Stream mode
- private and group `@bot` conversations work end to end
- `/new` and `新会话` reset the right session scope
- normal replies use `sessionWebhook`
- explicit proactive sends work through `conversation:` and `user:` targets
- AI Card streams token-by-token or chunk-by-chunk and finalizes cleanly
- image/file/audio inputs reach Memoh as attachments
- image/file/audio outputs can be sent back through the adapter
- both `cmd/agent` and `cmd/memoh serve` expose the DingTalk channel
