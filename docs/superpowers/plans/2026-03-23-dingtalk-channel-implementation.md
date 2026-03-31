# DingTalk Channel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a production-ready DingTalk channel adapter for Memoh with Stream-mode inbound chat, session replies, proactive sends, configurable group routing, session reset commands, media attachments, and AI Card streaming.

**Architecture:** Implement a Go-native adapter in `internal/channel/adapters/dingtalk/` that maps DingTalk protocol behavior onto Memoh's existing channel contracts. Follow the product semantics proven by `dingtalk-openclaw-connector`, but keep all DingTalk transport, card, and media details isolated inside the adapter package so the core channel framework stays unchanged.

**Tech Stack:** Go 1.25, existing Memoh `internal/channel` contracts, DingTalk enterprise app Stream APIs, DingTalk HTTP APIs for token/send/media/card operations, `httptest`, Go standard library JSON/HTTP, optional official DingTalk Go SDK only if it materially reduces Stream client risk.

---

## File Structure

### Files to create

- `internal/channel/adapters/dingtalk/dingtalk.go`
  - Adapter type, constructor, descriptor, interface delegation.
- `internal/channel/adapters/dingtalk/config.go`
  - Config parsing, alias normalization, session-scope settings, AI Card settings.
- `internal/channel/adapters/dingtalk/target.go`
  - `NormalizeTarget`, parse target families, `ResolveTarget` from user bindings.
- `internal/channel/adapters/dingtalk/client.go`
  - Shared HTTP client, token acquisition, self lookup, proactive send, media upload/download, AI Card create/update/finalize helpers.
- `internal/channel/adapters/dingtalk/protocol.go`
  - Request/response structs for auth, self identity, inbound events, media, proactive send, and AI Card.
- `internal/channel/adapters/dingtalk/inbound.go`
  - Pure mapping from DingTalk events to `channel.InboundMessage`, route-key helpers, mention gating, command detection.
- `internal/channel/adapters/dingtalk/stream.go`
  - Long-lived Stream connection management, event ACK, callback dispatch.
- `internal/channel/adapters/dingtalk/send.go`
  - Plain send path for webhook replies and proactive user/conversation sends.
- `internal/channel/adapters/dingtalk/card_stream.go`
  - `OpenStream` implementation, throttled AI Card session state, fallback behavior.
- `internal/channel/adapters/dingtalk/attachments.go`
  - Attachment upload/download and conversion helpers.
- `internal/channel/adapters/dingtalk/commands.go`
  - Session reset command matching and execution helpers.
- `internal/channel/adapters/dingtalk/config_test.go`
- `internal/channel/adapters/dingtalk/target_test.go`
- `internal/channel/adapters/dingtalk/inbound_test.go`
- `internal/channel/adapters/dingtalk/send_test.go`
- `internal/channel/adapters/dingtalk/card_stream_test.go`
- `internal/channel/adapters/dingtalk/attachments_test.go`

### Files to modify

- `cmd/agent/main.go`
  - Register the DingTalk adapter.
- `cmd/memoh/serve.go`
  - Register the DingTalk adapter in the secondary entrypoint.
- `go.mod`
  - Add DingTalk dependency only if needed after a prototype pass.
- `go.sum`

### Reference files to follow closely

- `internal/channel/adapter.go`
- `internal/channel/types.go`
- `internal/channel/adapters/qq/qq.go`
- `internal/channel/adapters/qq/send.go`
- `internal/channel/adapters/qq/receive.go`
- `internal/channel/adapters/qq/send_test.go`
- `internal/channel/adapters/qq/receive_test.go`
- `internal/channel/adapters/feishu/feishu.go`
- `internal/channel/adapters/feishu/stream.go`
- `internal/channel/adapters/feishu/config.go`
- `internal/channel/adapters/feishu/bot_identity.go`
- `docs/superpowers/specs/2026-03-23-dingtalk-channel-design.md`

---

## Task 1: Scaffold the adapter boundary and registration

**Files:**
- Create: `internal/channel/adapters/dingtalk/dingtalk.go`
- Create: `internal/channel/adapters/dingtalk/protocol.go`
- Modify: `cmd/agent/main.go`
- Modify: `cmd/memoh/serve.go`

- [ ] **Step 1: Write the failing compile target**

Run: `go test ./internal/channel/...`
Expected: PASS before any new DingTalk code exists, establishing the current baseline.

- [ ] **Step 2: Create the adapter shell**

Add `internal/channel/adapters/dingtalk/dingtalk.go` with:
- `const Type channel.ChannelType = "dingtalk"`
- `type Adapter struct { ... }`
- `func NewDingTalkAdapter(log *slog.Logger) *Adapter`
- `Descriptor()` advertising:
  - `Text: true`
  - `Attachments: true`
  - `Media: true`
  - `Reply: true`
  - `Streaming: true`
  - `BlockStreaming: true`
- interface stubs for config normalization, target handling, self discovery, send, stream send, and receive

- [ ] **Step 3: Add the first protocol structs**

Create `internal/channel/adapters/dingtalk/protocol.go` with the smallest structs needed for:
- access token lookup
- self identity lookup
- inbound event envelopes
- text reply payloads
- proactive send payloads

- [ ] **Step 4: Register the adapter in the server entrypoints**

Update `cmd/agent/main.go` and `cmd/memoh/serve.go` to register `dingtalk.NewDingTalkAdapter(log)`.

- [ ] **Step 5: Re-run the compile target**

Run: `go test ./internal/channel/...`
Expected: PASS with the new adapter package linked into the channel registry.

- [ ] **Step 6: Commit**

```bash
git add internal/channel/adapters/dingtalk/dingtalk.go internal/channel/adapters/dingtalk/protocol.go cmd/agent/main.go cmd/memoh/serve.go
git commit -m "feat: scaffold dingtalk channel adapter"
```

---

## Task 2: Implement config normalization, target resolution, and self discovery

**Files:**
- Create: `internal/channel/adapters/dingtalk/config.go`
- Create: `internal/channel/adapters/dingtalk/target.go`
- Create: `internal/channel/adapters/dingtalk/client.go`
- Create: `internal/channel/adapters/dingtalk/config_test.go`
- Create: `internal/channel/adapters/dingtalk/target_test.go`

- [ ] **Step 1: Write failing config and target tests**

Add tests covering:
- missing `appKey` / `appSecret`
- alias normalization for `app_key`, `app_secret`, `clientId`, `clientSecret` where supported
- default `groupSessionScope`
- default `requireMentionInGroup`
- default `enableAICard`
- accepted user config fields (`conversationId`, `staffId`, `unionId`)
- target normalization for `webhook:`, `conversation:`, and `user:`
- rejection of malformed or unknown targets

- [ ] **Step 2: Run the tests to verify failure**

Run: `go test ./internal/channel/adapters/dingtalk -run 'TestNormalizeConfig|TestNormalizeUserConfig|TestNormalizeTarget|TestResolveTarget' -v`
Expected: FAIL because parsing and normalization are not implemented yet.

- [ ] **Step 3: Implement normalized config parsing**

In `config.go`:
- define the internal `Config` model
- trim and validate credentials
- normalize session and AI Card settings
- normalize reset commands
- keep YAGNI boundaries aligned with the spec

- [ ] **Step 4: Implement target parsing and resolution**

In `target.go`:
- parse the 3 supported target families
- keep parsing pure and deterministic
- implement `ResolveTarget` from user config without hidden lookup behavior

- [ ] **Step 5: Add low-level client auth/self helpers**

In `client.go` implement:
- access token acquisition
- self identity lookup for `DiscoverSelf(...)`
- injectable base URL and HTTP client for tests

- [ ] **Step 6: Implement `DiscoverSelf(...)`**

Return:
- normalized self identity map containing all reliable bot identifiers
- stable external identity string when DingTalk exposes one

- [ ] **Step 7: Re-run the tests**

Run: `go test ./internal/channel/adapters/dingtalk -run 'TestNormalizeConfig|TestNormalizeUserConfig|TestNormalizeTarget|TestResolveTarget|TestDiscoverSelf' -v`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/channel/adapters/dingtalk/config.go internal/channel/adapters/dingtalk/target.go internal/channel/adapters/dingtalk/client.go internal/channel/adapters/dingtalk/config_test.go internal/channel/adapters/dingtalk/target_test.go
git commit -m "feat: add dingtalk config and target handling"
```

---

## Task 3: Implement inbound event mapping, group mention gating, and session commands

**Files:**
- Create: `internal/channel/adapters/dingtalk/inbound.go`
- Create: `internal/channel/adapters/dingtalk/commands.go`
- Create: `internal/channel/adapters/dingtalk/inbound_test.go`

- [ ] **Step 1: Write failing inbound tests**

Cover:
- private text message maps to the expected `InboundMessage`
- group text with mention is accepted
- group text without mention is ignored
- `group` route key generation
- `group_sender` route key generation
- `/new` and `新会话` command detection
- command path marks the current route for reset instead of normal LLM processing
- reply target is normalized to `webhook:<sessionWebhook>`
- stable sender subject ID selection order

- [ ] **Step 2: Run the inbound tests to verify failure**

Run: `go test ./internal/channel/adapters/dingtalk -run 'TestEventToInboundMessage|TestRouteKey|TestSessionCommand' -v`
Expected: FAIL.

- [ ] **Step 3: Implement pure event mapping helpers**

In `inbound.go`:
- parse supported message types
- extract text and attachments
- generate route keys based on config
- populate DingTalk metadata fields
- enforce mention gating

- [ ] **Step 4: Implement session-command helpers**

In `commands.go`:
- normalize reset commands
- detect command messages early
- expose enough signal for the stream layer to invoke session reset behavior

- [ ] **Step 5: Re-run the inbound tests**

Run: `go test ./internal/channel/adapters/dingtalk -run 'TestEventToInboundMessage|TestRouteKey|TestSessionCommand' -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/channel/adapters/dingtalk/inbound.go internal/channel/adapters/dingtalk/commands.go internal/channel/adapters/dingtalk/inbound_test.go
git commit -m "feat: add dingtalk inbound message mapping"
```

---

## Task 4: Implement the Stream receiver lifecycle

**Files:**
- Create: `internal/channel/adapters/dingtalk/stream.go`
- Modify: `internal/channel/adapters/dingtalk/inbound.go`
- Modify: `internal/channel/adapters/dingtalk/client.go`
- Modify: `internal/channel/adapters/dingtalk/protocol.go`
- Modify: `internal/channel/adapters/dingtalk/inbound_test.go`

- [ ] **Step 1: Write failing receiver tests**

Cover:
- `Connect(...)` starts a long-lived receiver
- supported message events are ACKed and passed to the handler
- unsupported events are ignored but ACKed safely
- command messages are intercepted for reset behavior
- `Stop(...)` closes the receiver gracefully

- [ ] **Step 2: Run the receiver tests to verify failure**

Run: `go test ./internal/channel/adapters/dingtalk -run 'TestConnect|TestConnectionStop' -v`
Expected: FAIL.

- [ ] **Step 3: Implement the Stream client lifecycle**

In `stream.go`:
- create/start the Stream client
- decode event envelopes
- dispatch through the pure mapper
- ACK and log errors conservatively

- [ ] **Step 4: Wire command-reset behavior into the receiver path**

Before handing a message to the normal inbound handler:
- detect reset commands
- invoke the Memoh-compatible reset flow
- send confirmation output through the current reply path

- [ ] **Step 5: Re-run the receiver tests**

Run: `go test ./internal/channel/adapters/dingtalk -run 'TestConnect|TestConnectionStop' -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/channel/adapters/dingtalk/stream.go internal/channel/adapters/dingtalk/inbound.go internal/channel/adapters/dingtalk/client.go internal/channel/adapters/dingtalk/protocol.go internal/channel/adapters/dingtalk/inbound_test.go
git commit -m "feat: add dingtalk stream receiver"
```

---

## Task 5: Implement plain replies and proactive text sending

**Files:**
- Create: `internal/channel/adapters/dingtalk/send.go`
- Create: `internal/channel/adapters/dingtalk/send_test.go`
- Modify: `internal/channel/adapters/dingtalk/target.go`
- Modify: `internal/channel/adapters/dingtalk/client.go`

- [ ] **Step 1: Write failing send tests**

Cover:
- webhook reply send
- proactive conversation send
- proactive user send
- rejection of malformed targets
- empty `PlainText()` rejection
- explicit error on unsupported attachment-bearing message in text-only path

- [ ] **Step 2: Run the send tests to verify failure**

Run: `go test ./internal/channel/adapters/dingtalk -run 'TestSend' -v`
Expected: FAIL.

- [ ] **Step 3: Implement plain text send routing**

In `send.go`:
- branch by target family
- send text through the matching DingTalk API
- surface API errors clearly

- [ ] **Step 4: Re-run the send tests**

Run: `go test ./internal/channel/adapters/dingtalk -run 'TestSend' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/channel/adapters/dingtalk/send.go internal/channel/adapters/dingtalk/send_test.go internal/channel/adapters/dingtalk/target.go internal/channel/adapters/dingtalk/client.go
git commit -m "feat: add dingtalk text sending"
```

---

## Task 6: Implement AI Card streaming with fallback

**Files:**
- Create: `internal/channel/adapters/dingtalk/card_stream.go`
- Create: `internal/channel/adapters/dingtalk/card_stream_test.go`
- Modify: `internal/channel/adapters/dingtalk/client.go`
- Modify: `internal/channel/adapters/dingtalk/protocol.go`
- Modify: `internal/channel/adapters/dingtalk/dingtalk.go`

- [ ] **Step 1: Write failing AI Card tests**

Cover:
- `OpenStream(...)` creates a card session for supported targets
- delta events are buffered and throttled
- final events flush final content
- error events trigger fallback behavior
- unsupported targets return explicit errors

- [ ] **Step 2: Run the AI Card tests to verify failure**

Run: `go test ./internal/channel/adapters/dingtalk -run 'TestOpenStream|TestCardStream' -v`
Expected: FAIL.

- [ ] **Step 3: Implement the card session state machine**

In `card_stream.go`:
- create a stream object that satisfies `channel.OutboundStream`
- buffer deltas
- update the card on throttle cadence
- finalize or fail deterministically

- [ ] **Step 4: Implement plain-text fallback**

If card create/update fails:
- prefer current webhook reply
- otherwise fall back to supported proactive text send
- otherwise return the send error

- [ ] **Step 5: Re-run the AI Card tests**

Run: `go test ./internal/channel/adapters/dingtalk -run 'TestOpenStream|TestCardStream' -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/channel/adapters/dingtalk/card_stream.go internal/channel/adapters/dingtalk/card_stream_test.go internal/channel/adapters/dingtalk/client.go internal/channel/adapters/dingtalk/protocol.go internal/channel/adapters/dingtalk/dingtalk.go
git commit -m "feat: add dingtalk ai card streaming"
```

---

## Task 7: Implement inbound and outbound media handling

**Files:**
- Create: `internal/channel/adapters/dingtalk/attachments.go`
- Create: `internal/channel/adapters/dingtalk/attachments_test.go`
- Modify: `internal/channel/adapters/dingtalk/inbound.go`
- Modify: `internal/channel/adapters/dingtalk/send.go`
- Modify: `internal/channel/adapters/dingtalk/client.go`
- Modify: `internal/channel/adapters/dingtalk/protocol.go`

- [ ] **Step 1: Write failing attachment tests**

Cover:
- inbound image maps to `channel.Attachment`
- inbound file maps to `channel.Attachment`
- inbound audio/voice maps to `channel.Attachment`
- outbound image uploads then sends
- outbound file uploads then sends
- outbound audio falls back to file when native audio send path is unavailable
- missing attachment bytes or unresolved references return explicit errors

- [ ] **Step 2: Run the attachment tests to verify failure**

Run: `go test ./internal/channel/adapters/dingtalk -run 'TestInboundAttachment|TestOutboundAttachment' -v`
Expected: FAIL.

- [ ] **Step 3: Implement attachment conversion helpers**

In `attachments.go`:
- centralize upload/download logic
- map DingTalk media metadata to Memoh attachments
- keep helper APIs reusable from both inbound and outbound code

- [ ] **Step 4: Wire attachments into inbound and send paths**

Update:
- `inbound.go` to append media attachments
- `send.go` to split text/media delivery when needed

- [ ] **Step 5: Re-run the attachment tests**

Run: `go test ./internal/channel/adapters/dingtalk -run 'TestInboundAttachment|TestOutboundAttachment' -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/channel/adapters/dingtalk/attachments.go internal/channel/adapters/dingtalk/attachments_test.go internal/channel/adapters/dingtalk/inbound.go internal/channel/adapters/dingtalk/send.go internal/channel/adapters/dingtalk/client.go internal/channel/adapters/dingtalk/protocol.go
git commit -m "feat: add dingtalk media handling"
```

---

## Task 8: Full adapter verification and cleanup

**Files:**
- Modify: `internal/channel/adapters/dingtalk/*.go`
- Modify: `internal/channel/adapters/dingtalk/*_test.go`
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Run focused DingTalk package tests**

Run: `go test ./internal/channel/adapters/dingtalk -v`
Expected: PASS.

- [ ] **Step 2: Run broader channel tests**

Run: `go test ./internal/channel/...`
Expected: PASS.

- [ ] **Step 3: Run formatting**

Run: `gofmt -w internal/channel/adapters/dingtalk/*.go internal/channel/adapters/dingtalk/*_test.go cmd/agent/main.go cmd/memoh/serve.go`
Expected: no output, files reformatted in place.

- [ ] **Step 4: Re-run the verification suite**

Run: `go test ./internal/channel/adapters/dingtalk -v && go test ./internal/channel/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/channel/adapters/dingtalk cmd/agent/main.go cmd/memoh/serve.go go.mod go.sum
git commit -m "feat: add dingtalk channel integration"
```
