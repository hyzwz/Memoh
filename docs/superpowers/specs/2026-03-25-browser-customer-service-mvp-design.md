# Browser Customer Service MVP Design

**Date:** 2026-03-25

**Goal:** Add an MVP for browser-driven customer-service channels in Memoh so one bot can operate as one customer-service brain across multiple e-commerce/service platforms, with one account per platform in the MVP, using browser automation as the transport layer while preserving Memoh's existing channel, route, and conversation model.

**Recommendation:** Implement this as platform-specific browser-backed channel adapters, starting with `ctrip_cs`, where each configured platform account is represented as one endpoint with its own browser login state and long-running listener worker. Keep browser automation behind the channel adapter boundary instead of exposing browser actions directly to Memoh core.

---

## Context

Memoh already has the right core abstraction for this feature:

- `channel.Adapter` describes a channel type and its schemas.
- `Receiver` supports long-lived inbound connections.
- `Sender` and `StreamSender` support outbound reply and streaming UX.
- `Manager` owns runtime connection lifecycle and health status.
- `route.ResolveConversation` maps external conversation identifiers to Memoh conversations.
- `ChannelInboundProcessor` already handles identity resolution, inbox creation, route resolution, and streamed reply delivery.

This means the new problem is not "how to invent a new conversation system." The real problem is "how to turn browser-observed customer-service traffic into a stable channel transport."

The product model for the target use case is:

- one bot behaves like one human customer-service agent
- that bot may need to operate on multiple platforms
- each platform account has its own login state and page behavior
- incoming platform messages should enter the same bot intelligence and memory
- replies should be sent back through the correct platform account and session

The existing browser gateway is not enough by itself for this. It exposes generic one-shot page actions and browser-context CRUD, but it does not yet provide:

- long-lived event subscription
- message de-duplication
- session-specific reply routing
- channel-grade reconnect behavior
- page-specific "new message" detection contracts

So the MVP should treat browser automation as a transport implementation detail, not as the primary product abstraction.

---

## Scope

### In scope

- MVP product model: one bot can bind multiple browser-driven customer-service platforms
- MVP account model: one account per platform per bot
- first platform target: `ctrip_cs`
- one dedicated browser login state per configured account
- long-running listener worker per configured account
- conversion from page/network/browser events into `channel.InboundMessage`
- conversion from `channel.OutboundMessage` and stream events into page send actions
- reuse of existing Memoh identity, routing, inbox, and chat execution pipeline
- persisted platform config through the existing channel configuration system
- operational status visibility through the existing channel manager lifecycle

### Out of scope

- multiple accounts on the same platform for one bot
- shared-account concurrency across multiple bots
- agent handoff / transfer to human staff
- full generic browser-channel framework for every unknown site
- visual recorder / no-code site adapter builder
- inbox co-editing between human and bot in the same platform session
- platform coverage beyond `ctrip_cs`

---

## Chosen Approach

### Approach 1: One bot as one customer-service brain, one platform account as one endpoint

This is the chosen approach.

Why it fits:

- matches the real business mental model
- keeps bot knowledge and behavior centralized
- avoids duplicating one bot into separate platform-specific bots
- aligns with Memoh's existing adapter lifecycle and route model
- creates a clean upgrade path to multiple accounts per platform later

Trade-offs:

- MVP must accept the current `bot_channel_configs` limitation of one config per platform per bot
- browser login state and listener lifecycle must be managed carefully

### Approach 2: One bot per platform account

Rejected.

Why not:

- forces duplicated bot prompts, memory, settings, and permissions
- makes cross-platform service quality drift over time
- creates unnecessary operational overhead

### Approach 3: One bot per browser page or one bot per browser instance

Rejected.

Why not:

- browser runtime is not the correct business abstraction
- pages and browser processes are ephemeral runtime details
- reconnect and crash recovery become harder when runtime identity and business identity are fused

---

## MVP Product Model

### Bot

A bot remains the customer-service intelligence unit:

- prompt and behavior
- memory
- model configuration
- tool access
- ownership and permissions

The bot should not be tied to exactly one page or one browser instance.

### Platform endpoint

In the MVP, each configured browser-driven platform account is one endpoint.

For this release, the endpoint key is effectively:

- `bot_id + channel_type`

Examples:

- bot A + `ctrip_cs`
- bot A + `jd_cs`
- bot A + `taobao_cs`

Each endpoint owns:

- one platform account identity
- one isolated browser login state
- one listener worker
- one platform-specific parser and sender

### Browser login state

The runtime isolation unit is one browser context or persistent browser profile.

Important rule:

- one login state must map to one isolated browser context/profile

This avoids cookie/session leakage between different platform accounts.

For the MVP:

- one configured endpoint uses one dedicated browser context/profile
- that context/profile may hold one or more tabs as needed by the adapter

---

## Current Constraint and MVP Decision

The current database schema enforces:

- `UNIQUE (bot_id, channel_type)` on `bot_channel_configs`

This means the current system already supports:

- one bot with one Ctrip account
- one bot with one JD account
- one bot with one Taobao account

But it does not support:

- one bot with two Taobao accounts

The MVP accepts this limitation intentionally.

Design rule for this release:

- one bot may bind multiple browser-driven platforms
- each platform may bind only one account per bot

This is sufficient for the first deployment and does not require immediate schema redesign.

---

## Architecture

Add a dedicated adapter package for each platform, starting with:

- `internal/channel/adapters/ctripcs/`

The package should have two layers.

### Channel adapter layer

This layer speaks Memoh contracts:

- `Descriptor()`
- `NormalizeConfig()`
- `NormalizeUserConfig()` if needed
- `NormalizeTarget()`
- `ResolveTarget()` if needed
- `DiscoverSelf()` where possible
- `Connect()`
- `Send()`
- `OpenStream()`

### Browser transport layer

This layer speaks browser and platform page semantics:

- browser context/profile boot
- login-state validation
- target page bootstrapping
- websocket / network / DOM observation
- new-message detection
- message parsing
- send-box interaction
- reconnect and watchdog handling

Boundary rule:

- Memoh core must only see normalized channel values and runtime errors.
- Browser selectors, page scripts, DOM parsing, and interception logic stay inside the adapter package.

---

## Configuration Model

For `ctrip_cs`, the MVP config should include at least:

- browser context or profile reference
- entry URL
- account label
- platform-specific worker settings

Possible fields:

- `browserContextId` or equivalent runtime binding
- `entryUrl`
- `accountLabel`
- `pollIntervalMs`
- `inboxPageUrl` or `messagePageUrl`

The config should not store raw credentials if a persistent browser login state is already available.

Preferred operational model:

- the operator logs into the platform account in the managed browser context
- Memoh verifies that the context is logged in and points at the required platform
- the adapter stores stable account identity in `external_identity` and `self_identity`

---

## Inbound Message Model

Each accepted platform message must be converted into one `channel.InboundMessage`.

Required fields:

- `Channel = ctrip_cs`
- `BotID = owning bot`
- `Sender.SubjectID = stable customer identifier from page or platform data`
- `Conversation.ID = stable external conversation/session identifier`
- `Conversation.Type = direct` unless the platform clearly supports another type
- `ReplyTarget = stable platform-specific reply handle for this session`
- `Message.Text` and attachments from parsed message content

Recommended metadata:

- page URL
- account label
- raw conversation/session id
- raw message id
- source transport type: `dom`, `xhr`, `websocket`
- message timestamp from platform if available

Stability rule:

- `Conversation.ID` must remain stable across refreshes and reconnects
- `ReplyTarget` must be the minimal value required to send back into the same session
- `Message.ID` should use the platform message id when available, otherwise a deterministic adapter-generated id

---

## Outbound Message Model

Outbound delivery for browser-backed customer-service channels is always session-scoped.

For the MVP:

- replies should primarily use the inbound `ReplyTarget`
- proactive send outside an existing customer session is out of scope unless the platform natively supports it and the adapter can do it safely

`Send()` behavior:

- open or focus the correct session UI
- fill the message input
- submit the message
- confirm submission with a page-level success heuristic

`OpenStream()` behavior:

- buffer stream deltas adapter-side
- optionally show typing/progress placeholder if the page UX supports it
- finalize by sending one plain text message for the MVP unless the platform supports safe partial updates

MVP simplification:

- streamed model output may be rendered in Memoh internally, but platform delivery can collapse to one final sent message

---

## Runtime Flow

### Startup

1. Channel manager refresh finds enabled `ctrip_cs` config.
2. Adapter `Connect()` starts a long-lived worker.
3. Worker opens or attaches to the dedicated browser context/profile.
4. Worker verifies login state and navigates to the platform inbox page.
5. Worker enters steady-state observation mode.

### Inbound

1. Worker detects a new incoming customer message.
2. Worker extracts normalized message payload.
3. Adapter emits `InboundMessage` through the channel manager.
4. Existing inbound pipeline resolves identity and route, creates inbox item, and triggers LLM response when appropriate.

### Outbound

1. Existing Memoh flow decides to reply.
2. Adapter receives `Send()` or `OpenStream()`.
3. Adapter focuses the correct browser session using the route's `ReplyTarget`.
4. Adapter performs page interactions to send the reply.
5. Adapter reports success or failure through the normal channel path.

---

## Identity and Route Semantics

Identity resolution should keep using the existing channel identity system.

For the MVP:

- customer identity key = platform-specific stable customer subject id
- channel type = `ctrip_cs`
- bot-side account identity = endpoint `external_identity` / `self_identity`

Route resolution should keep using existing `channel_routes` behavior:

- `platform = ctrip_cs`
- `conversation_id = external session id`
- `thread_id = empty` unless the platform exposes true thread nesting
- `reply_target = adapter-defined session reply handle`

This is one of the strongest reasons to implement the feature as a true channel adapter instead of a loose browser job.

---

## Failure Handling

The MVP must handle the following failures explicitly:

- browser context missing
- browser not running
- login expired
- inbox page structure changed
- message parser returns invalid payload
- send-box interaction fails
- duplicate inbound event after refresh/reconnect

Required behavior:

- fail the connection and surface runtime status in channel manager metadata
- retry reconnect with backoff for transient browser/page errors
- stop retrying on clear configuration errors such as missing context
- de-duplicate inbound messages by stable platform message id where available

Operational rule:

- "selector not found" and "login expired" are product-grade errors, not hidden debug noise

---

## Testing

### Unit tests

- config normalization
- external/self identity extraction
- reply-target normalization
- inbound payload parsing from representative platform samples
- de-duplication logic
- outbound script generation and send-state transitions

### Integration tests

- adapter worker boot with mocked browser transport
- inbound event to `InboundMessage` conversion
- reply send flow using mocked page primitives
- reconnect after transient page disconnect

### Manual verification for MVP

- one bot bound to one Ctrip account
- operator sends customer message through real page
- Memoh receives inbound message and creates route
- Memoh sends final reply back into the same page session
- browser refresh/restart recovers listener correctly

---

## Future-Compatible Extension Path

After the MVP proves stable, the next extension should be:

- allow multiple accounts for the same platform under one bot

That future change should introduce a first-class endpoint identity rather than overloading `channel_type`.

Likely direction:

- relax `bot_channel_configs` uniqueness from `(bot_id, channel_type)`
- keep one config row per endpoint
- let routes continue pointing to `channel_config_id`

The MVP should not fake this future by inventing synthetic channel types such as `ctrip_cs_1` or `taobao_cs_store_b`.

---

## Implementation Summary

Build the browser customer-service MVP as a real Memoh channel adapter, not as a browser macro system.

For this release:

- one bot can serve multiple platforms
- each platform can bind one account per bot
- each account gets one isolated browser login state
- browser automation stays inside the platform adapter
- Memoh core continues to own identity, routes, inbox, and conversation flow

This gives the fastest path to a usable Ctrip MVP without locking the system into the wrong abstraction.
