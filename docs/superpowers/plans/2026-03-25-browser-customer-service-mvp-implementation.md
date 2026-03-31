# Browser Customer Service MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first runnable browser-backed customer-service channel in Memoh, starting with `ctrip_cs`, so one bot can receive and reply to Ctrip web customer-service sessions through a managed browser context.

**Architecture:** Keep the MVP inside Memoh's existing channel model. Add one dedicated `ctripcs` adapter that uses the current Browser Gateway HTTP action API as a polling transport, converts Ctrip page snapshots into `channel.InboundMessage`, and sends replies by driving the page UI. Do not change database schema, generic channel routing, or Browser Gateway protocol in this phase.

**Tech Stack:** Go channel adapters and tests, existing Browser Gateway HTTP endpoints, Echo/FX service wiring, Playwright runtime behind `apps/browser`, Go test.

---

## File Map

### New files

- `internal/channel/adapters/ctripcs/ctripcs.go`
- `internal/channel/adapters/ctripcs/config.go`
- `internal/channel/adapters/ctripcs/config_test.go`
- `internal/channel/adapters/ctripcs/browser_gateway.go`
- `internal/channel/adapters/ctripcs/browser_gateway_test.go`
- `internal/channel/adapters/ctripcs/parser.go`
- `internal/channel/adapters/ctripcs/parser_test.go`
- `internal/channel/adapters/ctripcs/connection.go`
- `internal/channel/adapters/ctripcs/connection_test.go`
- `internal/channel/adapters/ctripcs/send.go`
- `internal/channel/adapters/ctripcs/send_test.go`
- `internal/channel/adapters/ctripcs/testdata/ctrip_inbox_snapshot.json`
- `internal/channel/adapters/ctripcs/testdata/ctrip_login_expired_snapshot.json`
- `internal/channel/adapters/ctripcs/testdata/ctrip_reply_target_snapshot.json`

### Modified files

- `cmd/agent/main.go`
- `cmd/memoh/serve.go`

### Existing references to inspect before coding

- `docs/superpowers/specs/2026-03-25-browser-customer-service-mvp-design.md`
- `internal/channel/adapter.go`
- `internal/channel/types.go`
- `internal/channel/connection.go`
- `internal/channel/service.go`
- `internal/channel/route/service.go`
- `internal/channel/inbound/channel.go`
- `internal/channel/adapters/wecombot/wecombot.go`
- `internal/mcp/providers/browser/provider.go`
- `apps/browser/src/modules/action.ts`
- `internal/browsercontexts/service.go`

### Explicit non-goals for this plan

- No SQL migration
- No frontend configuration UI change
- No Browser Gateway websocket/event-stream protocol change
- No multi-account support for the same platform

---

### Task 1: Scaffold `ctrip_cs` adapter config, descriptor, and Browser Gateway client

**Files:**
- Create: `internal/channel/adapters/ctripcs/ctripcs.go`
- Create: `internal/channel/adapters/ctripcs/config.go`
- Create: `internal/channel/adapters/ctripcs/config_test.go`
- Create: `internal/channel/adapters/ctripcs/browser_gateway.go`
- Create: `internal/channel/adapters/ctripcs/browser_gateway_test.go`

- [ ] **Step 1: Write the failing config and client tests**

Add table-driven tests for:

```go
func TestNormalizeConfigDefaults(t *testing.T) {}
func TestNormalizeConfigRequiresBrowserContextID(t *testing.T) {}
func TestNormalizeConfigRejectsNonCtripEntryURL(t *testing.T) {}
func TestDiscoverSelfUsesAccountLabel(t *testing.T) {}
func TestBrowserGatewayClientBuildsContextActionRequest(t *testing.T) {}
```

Expected behaviors:
- config normalizes `browserContextId`, `entryUrl`, `accountLabel`, `pollIntervalMs`
- `browserContextId` is required
- `entryUrl` must be an HTTP(S) URL under a Ctrip host
- `DiscoverSelf()` uses `accountLabel` as the MVP external identity and mirrors it into self identity metadata
- the gateway client targets `POST /context/{id}/action` with the exact JSON payload Memoh needs

- [ ] **Step 2: Run targeted tests to verify they fail**

Run:

```bash
mkdir -p /tmp/memoh-gocache && GOCACHE=/tmp/memoh-gocache go test ./internal/channel/adapters/ctripcs -run 'TestNormalizeConfigDefaults|TestNormalizeConfigRequiresBrowserContextID|TestNormalizeConfigRejectsNonCtripEntryURL|TestDiscoverSelfUsesAccountLabel|TestBrowserGatewayClientBuildsContextActionRequest'
```

Expected: FAIL because the package does not exist yet.

- [ ] **Step 3: Write the minimal adapter scaffold**

Implement:

```go
const Type channel.ChannelType = "ctrip_cs"

type Config struct {
	BrowserContextID string
	EntryURL         string
	AccountLabel     string
	PollIntervalMS   int
	InboxPageURL     string
}

type browserGatewayClient struct {
	baseURL string
	client  *http.Client
}

func NewAdapter(log *slog.Logger, gatewayCfg config.BrowserGatewayConfig) *Adapter
func (a *Adapter) Descriptor() channel.Descriptor
func (a *Adapter) NormalizeConfig(raw map[string]any) (map[string]any, error)
func (a *Adapter) DiscoverSelf(ctx context.Context, credentials map[string]any) (map[string]any, string, error)
```

Implementation constraints:
- keep the adapter package self-contained for the MVP; do not refactor `internal/mcp/providers/browser/provider.go`
- default `pollIntervalMs` to a conservative value such as `1500`
- keep `TargetSpec` simple: reply-target strings are opaque adapter-managed session identifiers

- [ ] **Step 4: Run the targeted tests to verify they pass**

Run:

```bash
mkdir -p /tmp/memoh-gocache && GOCACHE=/tmp/memoh-gocache go test ./internal/channel/adapters/ctripcs -run 'TestNormalizeConfigDefaults|TestNormalizeConfigRequiresBrowserContextID|TestNormalizeConfigRejectsNonCtripEntryURL|TestDiscoverSelfUsesAccountLabel|TestBrowserGatewayClientBuildsContextActionRequest'
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/channel/adapters/ctripcs
git commit -m "feat: scaffold ctrip customer service adapter"
```

---

### Task 2: Add Ctrip snapshot parsing and inbound message normalization

**Files:**
- Create: `internal/channel/adapters/ctripcs/parser.go`
- Create: `internal/channel/adapters/ctripcs/parser_test.go`
- Create: `internal/channel/adapters/ctripcs/testdata/ctrip_inbox_snapshot.json`
- Create: `internal/channel/adapters/ctripcs/testdata/ctrip_login_expired_snapshot.json`
- Create: `internal/channel/adapters/ctripcs/testdata/ctrip_reply_target_snapshot.json`

- [ ] **Step 1: Write the failing parser tests**

Add tests for:

```go
func TestParseInboxSnapshotBuildsInboundMessages(t *testing.T) {}
func TestParseInboxSnapshotSkipsAgentAuthoredMessages(t *testing.T) {}
func TestParseInboxSnapshotDetectsLoginExpired(t *testing.T) {}
func TestBuildReplyTargetUsesStableConversationHandle(t *testing.T) {}
```

Expected behaviors:
- parser turns one representative Ctrip snapshot into one or more normalized inbound candidates
- only customer-authored messages become `channel.InboundMessage`
- login-expired pages return a sentinel error such as `ErrLoginExpired`
- reply target is stable for the same conversation across polling rounds

- [ ] **Step 2: Run parser tests to verify they fail**

Run:

```bash
mkdir -p /tmp/memoh-gocache && GOCACHE=/tmp/memoh-gocache go test ./internal/channel/adapters/ctripcs -run 'TestParseInboxSnapshotBuildsInboundMessages|TestParseInboxSnapshotSkipsAgentAuthoredMessages|TestParseInboxSnapshotDetectsLoginExpired|TestBuildReplyTargetUsesStableConversationHandle'
```

Expected: FAIL because the parser does not exist yet.

- [ ] **Step 3: Write the minimal parser**

Implement a parser that:
- accepts raw JSON extracted from `evaluate()` results instead of raw HTML
- returns a typed snapshot model plus normalized inbound candidates
- fills `Message.ID`, `Conversation.ID`, `ReplyTarget`, sender subject, text, timestamps, and metadata
- keeps raw platform identifiers in metadata:

```go
metadata := map[string]any{
	"page_url": raw.PageURL,
	"account_label": cfg.AccountLabel,
	"raw_conversation_id": raw.ConversationID,
	"raw_message_id": raw.MessageID,
	"source_transport": "dom_poll",
}
```

- [ ] **Step 4: Run parser tests to verify they pass**

Run:

```bash
mkdir -p /tmp/memoh-gocache && GOCACHE=/tmp/memoh-gocache go test ./internal/channel/adapters/ctripcs -run 'TestParseInboxSnapshotBuildsInboundMessages|TestParseInboxSnapshotSkipsAgentAuthoredMessages|TestParseInboxSnapshotDetectsLoginExpired|TestBuildReplyTargetUsesStableConversationHandle'
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/channel/adapters/ctripcs
git commit -m "feat: add ctrip snapshot parser"
```

---

### Task 3: Implement the long-running polling connection with dedupe and reconnect behavior

**Files:**
- Create: `internal/channel/adapters/ctripcs/connection.go`
- Create: `internal/channel/adapters/ctripcs/connection_test.go`
- Modify: `internal/channel/adapters/ctripcs/ctripcs.go`

- [ ] **Step 1: Write the failing connection tests**

Add tests for:

```go
func TestConnectPollsAndDispatchesNewMessages(t *testing.T) {}
func TestConnectDeduplicatesMessageIDsAcrossPollingRounds(t *testing.T) {}
func TestConnectStopsOnMissingBrowserContext(t *testing.T) {}
func TestConnectReturnsLoginExpiredAsConnectionError(t *testing.T) {}
```

Expected behaviors:
- `Connect()` starts one worker loop and hands new messages to the inbound handler
- the same platform message ID is emitted once even if it appears in multiple snapshots
- configuration-grade failures such as missing browser context stop the connection
- login expiry surfaces as a hard error instead of silent retries

- [ ] **Step 2: Run connection tests to verify they fail**

Run:

```bash
mkdir -p /tmp/memoh-gocache && GOCACHE=/tmp/memoh-gocache go test ./internal/channel/adapters/ctripcs -run 'TestConnectPollsAndDispatchesNewMessages|TestConnectDeduplicatesMessageIDsAcrossPollingRounds|TestConnectStopsOnMissingBrowserContext|TestConnectReturnsLoginExpiredAsConnectionError'
```

Expected: FAIL because `Connect()` is not implemented yet.

- [ ] **Step 3: Write the minimal polling worker**

Implement:

```go
type poller struct {
	cfg        Config
	gateway    browserActionRunner
	seen       map[string]time.Time
	handler    channel.InboundHandler
}

func (a *Adapter) Connect(ctx context.Context, cfg channel.ChannelConfig, handler channel.InboundHandler) (channel.Connection, error)
```

Worker requirements:
- ensure the browser context exists before the first poll
- navigate to `entryUrl` on startup
- call a single `evaluate` script that returns normalized JSON for the current inbox/session view
- parse the JSON, emit new messages, and trim the dedupe map periodically
- use bounded backoff for transient page errors
- expose shutdown through `channel.NewConnection(...)`

- [ ] **Step 4: Run connection tests to verify they pass**

Run:

```bash
mkdir -p /tmp/memoh-gocache && GOCACHE=/tmp/memoh-gocache go test ./internal/channel/adapters/ctripcs -run 'TestConnectPollsAndDispatchesNewMessages|TestConnectDeduplicatesMessageIDsAcrossPollingRounds|TestConnectStopsOnMissingBrowserContext|TestConnectReturnsLoginExpiredAsConnectionError'
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/channel/adapters/ctripcs
git commit -m "feat: add ctrip polling receiver"
```

---

### Task 4: Implement outbound send and collapsed streaming replies

**Files:**
- Create: `internal/channel/adapters/ctripcs/send.go`
- Create: `internal/channel/adapters/ctripcs/send_test.go`
- Modify: `internal/channel/adapters/ctripcs/ctripcs.go`

- [ ] **Step 1: Write the failing outbound tests**

Add tests for:

```go
func TestSendUsesReplyTargetToFocusConversationAndSubmit(t *testing.T) {}
func TestSendRejectsEmptyText(t *testing.T) {}
func TestOpenStreamBuffersChunksAndSendsFinalMessageOnClose(t *testing.T) {}
func TestOpenStreamPropagatesSendFailureOnClose(t *testing.T) {}
```

Expected behaviors:
- `Send()` uses the reply target to reopen or focus the correct session before filling the input
- empty outbound text is rejected
- streaming buffers text locally and performs one final page send on `Close()`
- close returns the underlying send error if page submission fails

- [ ] **Step 2: Run outbound tests to verify they fail**

Run:

```bash
mkdir -p /tmp/memoh-gocache && GOCACHE=/tmp/memoh-gocache go test ./internal/channel/adapters/ctripcs -run 'TestSendUsesReplyTargetToFocusConversationAndSubmit|TestSendRejectsEmptyText|TestOpenStreamBuffersChunksAndSendsFinalMessageOnClose|TestOpenStreamPropagatesSendFailureOnClose'
```

Expected: FAIL because outbound send and stream handling do not exist yet.

- [ ] **Step 3: Write the minimal outbound implementation**

Implement:

```go
func (a *Adapter) Send(ctx context.Context, cfg channel.ChannelConfig, msg channel.OutboundMessage) error
func (a *Adapter) OpenStream(ctx context.Context, cfg channel.ChannelConfig, target string, opts channel.StreamOptions) (channel.OutboundStream, error)
```

Implementation constraints:
- use adapter-owned reply-target parsing instead of exposing raw selectors to Memoh core
- keep the MVP send path text-only
- use a page-level success heuristic such as cleared input or echoed assistant bubble before returning success
- do not implement proactive send outside an existing session

- [ ] **Step 4: Run outbound tests to verify they pass**

Run:

```bash
mkdir -p /tmp/memoh-gocache && GOCACHE=/tmp/memoh-gocache go test ./internal/channel/adapters/ctripcs -run 'TestSendUsesReplyTargetToFocusConversationAndSubmit|TestSendRejectsEmptyText|TestOpenStreamBuffersChunksAndSendsFinalMessageOnClose|TestOpenStreamPropagatesSendFailureOnClose'
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/channel/adapters/ctripcs
git commit -m "feat: add ctrip outbound reply handling"
```

---

### Task 5: Register the adapter and verify end-to-end channel wiring

**Files:**
- Modify: `cmd/agent/main.go`
- Modify: `cmd/memoh/serve.go`
- Modify: `internal/channel/adapters/ctripcs/connection_test.go`

- [ ] **Step 1: Write the failing registration/integration tests**

Add tests for:

```go
func TestAdapterDescriptorExposesExpectedCapabilities(t *testing.T) {}
func TestAdapterImplementsReceiverAndSender(t *testing.T) {}
```

Expected behaviors:
- descriptor advertises text, reply, and streaming support for `ctrip_cs`
- the concrete adapter satisfies `channel.Adapter`, `channel.Receiver`, `channel.Sender`, and `channel.StreamSender`

- [ ] **Step 2: Run the focused adapter tests to verify they fail**

Run:

```bash
mkdir -p /tmp/memoh-gocache && GOCACHE=/tmp/memoh-gocache go test ./internal/channel/adapters/ctripcs -run 'TestAdapterDescriptorExposesExpectedCapabilities|TestAdapterImplementsReceiverAndSender'
```

Expected: FAIL until the concrete adapter type is fully wired.

- [ ] **Step 3: Register the adapter in both binaries**

Implementation notes:
- import `github.com/memohai/memoh/internal/channel/adapters/ctripcs` in both files
- call `registry.MustRegister(ctripcs.NewAdapter(log, cfg.BrowserGateway))` alongside the other adapters
- keep registration order simple; no special hooks are required in the inbound pipeline

- [ ] **Step 4: Run the focused package tests and channel smoke tests**

Run:

```bash
mkdir -p /tmp/memoh-gocache && GOCACHE=/tmp/memoh-gocache go test ./internal/channel/adapters/ctripcs ./internal/channel/...
```

Expected: PASS

- [ ] **Step 5: Run manual MVP verification**

Manual checklist:
1. Start the dev environment and Browser Gateway.
2. Create or reuse one browser context in Memoh admin.
3. Log that context into the real Ctrip customer-service page.
4. Upsert one `ctrip_cs` config for a test bot with `browserContextId`, `entryUrl`, and `accountLabel`.
5. Confirm the channel manager shows the connection as running.
6. Send one customer message in the Ctrip page and verify Memoh creates or reuses the expected route.
7. Verify the bot reply lands back in the same Ctrip session.
8. Refresh the page once and verify the poller recovers without duplicating the prior inbound message.

- [ ] **Step 6: Commit**

```bash
git add cmd/agent/main.go cmd/memoh/serve.go internal/channel/adapters/ctripcs
git commit -m "feat: register ctrip customer service channel"
```

---

## Final Verification

Run the full focused verification before calling the MVP complete:

```bash
mkdir -p /tmp/memoh-gocache && GOCACHE=/tmp/memoh-gocache go test ./internal/channel/adapters/ctripcs ./internal/channel/...
```

If Browser Gateway code was touched during execution despite this MVP plan, also run:

```bash
pnpm test --run apps/browser
```

Expected: PASS

## Notes for the Implementer

- Keep selectors and page scripts inside `internal/channel/adapters/ctripcs`; do not leak them into generic channel code.
- Prefer `evaluate()` returning normalized JSON objects over scraping full HTML blobs.
- Treat `login expired`, `missing browser context`, and `selector contract broken` as operator-visible errors.
- Do not expand into JD/Taobao until this one adapter is stable.
