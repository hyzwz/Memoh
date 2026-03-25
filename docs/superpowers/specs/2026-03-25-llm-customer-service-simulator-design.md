# LLM Customer Service Simulator Design

**Date:** 2026-03-25

**Goal:** Design a standalone web-based customer-service simulator that looks and behaves like a real e-commerce support console, can be taken over through Playwright like a real third-party platform, and uses configurable OpenAI-compatible models to simulate customer behavior while allowing Memoh AI and a human operator to handle escalation.

**Recommendation:** Build this as a separate small web application with a single high-fidelity customer-service workspace, a lightweight backend, a scenario engine, and an LLM-driven customer agent. The page should expose a stable browser automation contract for Memoh, but replies must still be entered through the real on-screen composer and send controls so the full browser-takeover path stays realistic.

---

## Context

The target use case is not a fake JSON API test harness. It is a realistic browser-facing system for two purposes:

- demoing the browser-customer-service concept to internal and external stakeholders
- testing Memoh's browser-driven customer-service channel against a controllable but believable environment

The real production constraint is important:

- some target platforms such as Ctrip do not provide a suitable public API for this workflow
- Memoh must therefore operate by taking over a live web page with Playwright
- any test system that hides behind a synthetic backend API would validate the wrong thing

This tool should therefore behave like a small real platform:

- the interface should feel like a genuine customer-service console
- the customer should behave like a real, inconsistent, multilingual human
- Memoh should interact with the page the same way it would on Ctrip
- a human operator should be able to step in when AI handling fails or risk rises

At the same time, the system must still be controllable enough to support repeatable testing:

- customer scenarios must remain bounded
- upgrades to human handling must be deterministic enough to verify
- the page structure must be stable enough for Playwright automation

---

## Scope

### In scope

- a completely standalone project, separate from the Memoh repository runtime
- one high-fidelity browser customer-service workspace modeled after the provided screenshot
- three roles in one system: LLM customer, Memoh customer-service agent, and human operator
- Memoh takeover via browser automation only
- customer simulation driven by `scenario state + LLM generation`
- configurable OpenAI-compatible model providers
- support for at least OpenAI-compatible OpenAI endpoints and MiniMax endpoints
- scenario controls for demo and testing
- conversation persistence and event timeline
- deterministic escalation and handoff rules
- stable DOM and browser snapshot contract for Playwright consumers

### Out of scope

- pretending to be the real Ctrip backend or inventing a platform API
- multi-tenant enterprise administration
- multiple concurrent human seats in the same workspace
- attachment upload, voice, or video support
- analytics dashboards beyond basic timeline and run state
- separate skins for many different e-commerce platforms in the MVP
- replacing Memoh's real browser channel implementation

---

## Chosen Approach

### Approach 1: Realistic workspace UI plus internal scenario engine plus LLM customer agent

This is the chosen approach.

Why it fits:

- matches the real browser-takeover constraint
- produces a strong demo because the tool looks alive rather than scripted
- validates the actual Memoh integration path instead of a surrogate API
- remains testable because scenario state constrains the LLM's behavior

Trade-offs:

- larger MVP than a static mockup
- requires careful boundaries between visual UI, automation contract, and simulation logic

### Approach 2: Realistic UI with state-machine customer and optional LLM wording

Rejected for the first cut.

Why not:

- easier to control, but less convincing during demos
- reduces the "feels real" quality the user explicitly wants
- can still be borrowed internally by making the LLM run inside scenario boundaries

### Approach 3: Manual-trigger demo page with LLM-generated next replies

Rejected.

Why not:

- too close to mock behavior
- does not exercise ongoing customer initiative
- weakens the test value of the system

---

## Product Model

The simulator has three active actors.

### LLM customer

The LLM customer behaves like a user on a third-party support platform:

- initiates conversations
- asks follow-up questions
- changes tone and language
- becomes impatient or satisfied depending on responses
- can explicitly request a human

The LLM customer is not free-running. It operates inside a bounded scenario state.

### Memoh customer-service agent

Memoh acts as the primary automated service agent:

- reads the conversation from the browser page
- sends replies through the same visible UI controls a human would use
- remains subject to page timing, handoff status, and session context

### Human operator

The human operator is the final shield:

- can take over conversations when escalation rules fire
- can reply directly from the same workspace
- can release the conversation back to AI handling if appropriate

This gives the system the "spear and shield" behavior the user requested:

- the customer agent attacks the system with realistic demand and variation
- the Memoh agent and human operator defend and absorb failure cases

---

## System Architecture

The standalone tool should be split into five clear units.

### 1. Workspace frontend

Responsibilities:

- render the customer-service workspace
- show session list, chat timeline, and customer/product metadata
- allow human takeover and send actions
- expose a stable browser automation contract

The workspace should visually follow the screenshot's shape:

- left column for conversation queue
- center column for the active chat
- right column for customer and product detail

### 2. Simulation backend

Responsibilities:

- persist conversations, messages, and actor state
- coordinate scenario progression
- accept human actions
- publish state updates to the frontend

This backend is the source of truth for the simulator's state.

### 3. Scenario engine

Responsibilities:

- define why the customer is contacting support
- maintain the current stage of the issue
- decide whether the next move is a follow-up, a satisfaction signal, a complaint, or an escalation trigger
- bound the LLM's next-turn choices

The scenario engine is what prevents the simulator from degenerating into untestable free chat.

### 4. LLM adapter layer

Responsibilities:

- provide a single OpenAI-compatible client abstraction
- support per-run model configuration
- handle provider-specific base URLs, API keys, model names, and timeouts
- format prompts for the customer role only

This layer must work with both OpenAI-compatible OpenAI endpoints and MiniMax-compatible endpoints as configured by the operator.

### 5. Browser takeover contract

Responsibilities:

- provide stable selectors for Playwright
- publish a normalized page snapshot for reliable message reads
- keep the send path realistic by using the visible composer and send button

This is the key design boundary: page reading may use a stable normalized snapshot, but page writing must still go through real user-visible controls.

---

## Workspace Design

The workspace should look close enough to the provided screenshot to be immediately legible as a customer-service console.

### Left rail: conversation queue

Each queue item should show:

- customer display name
- short preview of the latest message
- timestamp
- status badge such as `pending`, `in_progress`, `waiting_customer`, `escalated`, `resolved`
- optional source or scenario tag

The queue should support:

- selecting the active conversation
- visually highlighting waiting or escalated conversations
- showing unread activity while another conversation is open

### Center panel: active conversation

The active conversation area should include:

- a small top bar with agent identity and current handling mode
- a scrollable message timeline
- visually distinct bubbles for customer, Memoh, and human messages
- composer area with text input and send action
- buttons for `Take Over`, `Return to AI`, `Pause Customer`, and optional scenario controls in debug mode

### Right rail: customer and product detail

The detail panel should show:

- customer language
- customer region
- source page or entry page
- product name
- product or order ID
- price
- risk level
- short scenario summary

This panel should be realistic enough to support demos and also useful enough to drive test assertions.

---

## Browser Automation Contract

The simulator should be realistic for Playwright without being brittle.

### Stable selectors

All critical elements must expose stable `data-testid` markers, including:

- conversation list
- conversation row
- active conversation ID
- latest customer message ID
- message timeline container
- composer input
- send button
- takeover controls
- handling mode indicator

### Normalized read snapshot

The frontend should maintain a normalized read-only object on `window.__CS_SNAPSHOT__`.

That snapshot should include:

- current workspace state
- active conversation ID
- conversation summaries
- normalized messages with actor role, message ID, timestamps, and raw content
- handling mode and takeover status

This allows browser readers to consume reliable structured state instead of reverse-engineering every visual detail from the DOM.

### Real write path

Outbound replies must still be performed by:

- focusing the real composer
- typing into the real input area
- clicking the real send control

This ensures the simulator still exercises the same operational path Memoh needs on Ctrip-like platforms.

---

## Conversation and State Model

The simulator needs a small but explicit state model.

### Conversation

Each conversation should store:

- conversation ID
- status
- active handling mode
- customer profile
- product context
- scenario ID
- scenario run ID
- current scenario stage
- timestamps for created, updated, last customer message, last agent reply

### Message

Each message should store:

- message ID
- conversation ID
- actor role: `customer`, `memoh`, or `human`
- content
- locale
- created timestamp
- optional metadata such as scenario step or escalation reason

### Handling mode

Handling mode should be explicit and visible:

- `ai_active`
- `human_takeover`
- `paused`
- `closed`

This mode directly affects whether Memoh is allowed to send.

### Status vs handling mode

The simulator must treat `status` and `handling_mode` as separate fields.

- `status` is the queue-level lifecycle used for list badges and operator triage
- `handling_mode` is the control authority over who is allowed to reply

`status` should be one of:

- `pending`
- `in_progress`
- `waiting_customer`
- `escalated`
- `resolved`

`handling_mode` should be one of:

- `ai_active`
- `human_takeover`
- `paused`
- `closed`

Source-of-truth rule:

- backend state owns both fields
- the frontend only renders them
- Playwright readers should consume both from the normalized snapshot, not infer one from the other

Expected mapping:

- new inbound customer message with no reply yet: `status=pending`, `handling_mode=ai_active`
- Memoh is actively working the case: `status=in_progress`, `handling_mode=ai_active`
- Memoh has replied and the system is waiting for the customer: `status=waiting_customer`, `handling_mode=ai_active`
- escalation or manual takeover: `status=escalated`, `handling_mode=human_takeover`
- temporary stop for demos or debugging: `status=in_progress`, `handling_mode=paused`
- conversation finished: `status=resolved`, `handling_mode=closed`

Write permission rule:

- Memoh may send only when `handling_mode=ai_active`
- the human may send only when `handling_mode=human_takeover`
- no one sends when `handling_mode=paused` or `closed`

This separation avoids the state collision where queue badges and send authority accidentally drift apart.

---

## Scenario Model

The scenario system needs a minimal explicit schema so it can drive both realism and repeatability.

### Scenario definition

Each reusable scenario definition should include:

- `scenario_id`
- `title`
- `category`, for example `product_inquiry`, `activation_issue`, `refund_request`
- `customer_profile`
- `product_context`
- `entry_context`, such as source page and channel language
- `opening_goal`
- ordered `stages`
- escalation policy
- terminal success and failure conditions

### Stage definition

Each stage should include:

- `stage_id`
- `intent`
- `customer_goal`
- `allowed_next_actions`
- `llm_instructions`
- optional language or tone constraints
- transition conditions

`allowed_next_actions` should be drawn from a small fixed set such as:

- `ask_follow_up`
- `provide_detail`
- `repeat_question`
- `express_confusion`
- `become_impatient`
- `request_human`
- `accept_resolution`
- `end_conversation`

### Scenario run

When the operator starts a test conversation, the backend creates a `scenario_run` from a scenario definition.

Each run should store:

- `scenario_run_id`
- `scenario_id`
- active `stage_id`
- run seed or deterministic init metadata
- satisfaction or frustration score
- escalation flags
- terminal state

### Run initialization flow

1. Operator chooses a scenario template or a random scenario pack.
2. Backend creates a new conversation and a linked `scenario_run`.
3. Stage 1 becomes active.
4. The customer LLM receives the stage context and generates the opening message.
5. Each later customer turn is generated from the current stage plus the latest Memoh or human reply.

### Stage transition rules

The scenario engine, not the LLM, decides stage transitions.

Transitions are triggered by:

- a reply matching a stage resolution condition
- timeout or no reply
- escalation trigger
- operator-forced takeover
- customer accepting resolution

The LLM may phrase the next move, but it may not invent a new stage outside the allowed transition graph.

### Terminal states

A scenario run should end in one of:

- `resolved_by_ai`
- `resolved_by_human`
- `abandoned`
- `escalated_unresolved`

This is the minimum structure needed to keep the simulator believable while still plannable and testable.

---

## LLM Customer Design

The customer simulator should not be a blind prompt loop. It should be a controlled agent.

### Customer generation model

Each customer turn should be produced from:

- current scenario stage
- customer profile
- previous conversation turns
- satisfaction or frustration state
- hard bounds on what the customer may do next

The generation prompt should instruct the model to behave as a real customer, not as an assistant.

### Provider model

The simulator should expose configurable fields for:

- provider label
- base URL
- API key
- model name
- optional headers if needed
- temperature
- timeout

The system should treat both OpenAI and MiniMax as OpenAI-compatible providers behind the same interface.

### Guardrails

The customer agent should obey the scenario engine on:

- whether to ask a follow-up
- whether to switch language
- whether to become upset
- whether to request a human
- whether to end the conversation

This keeps the dialogue believable without losing test repeatability.

---

## Escalation and Human Handoff

Escalation is part of the product, not an error path.

### Escalation triggers

The MVP should support deterministic triggers such as:

- Memoh reply timeout
- too many unresolved turns
- explicit customer request for a human
- risk-tagged scenario branch such as refund, policy dispute, or complaint
- operator-forced takeover

### Human takeover behavior

When takeover starts:

- the workspace status changes visibly
- Memoh may continue reading but must not write
- the human operator can reply through the same composer

When the human releases the conversation:

- handling mode returns to `ai_active`
- the event timeline records the handoff boundary

This makes both demos and debugging easier because responsibility shifts stay explicit.

---

## Control Surfaces

The simulator should include a small operator surface for testing and demo use.

### Run controls

The operator should be able to:

- start a new scenario
- pause customer generation
- resume customer generation
- force escalation
- close or reset a conversation

### Debug visibility

The operator should be able to inspect:

- current scenario stage
- latest normalized snapshot
- event timeline
- LLM provider and model in use
- recent generation or delivery failures

This does not need a large admin system. A slim debug panel is enough for the MVP.

---

## Error Handling

The simulator must fail in ways that are visible and recoverable.

### Model errors

If customer generation fails:

- the conversation should enter a visible degraded state
- the operator may retry generation or switch providers
- the system should not silently fabricate customer messages

### Browser-side timing issues

If Memoh types or sends at the wrong time:

- the UI should preserve state coherently
- invalid sends during `human_takeover` should be rejected visibly
- timeline events should record the attempt for debugging

### Frontend/backend drift

If live state and rendered state diverge:

- the normalized snapshot should still be derived from the source-of-truth state
- the operator should have a visible refresh or resync option

---

## MVP Technology Direction

The exact stack can be finalized in planning, but the design assumes:

- a standalone frontend application
- a lightweight backend process
- local persistence suitable for demo and test workflows
- WebSocket or SSE updates from backend to frontend for near-live conversation updates

The simplest likely implementation is:

- React-based frontend
- Node-based backend
- SQLite or a small local database

These are recommendations, not hard architectural requirements. The non-negotiable design requirement is the browser-facing behavior, not a specific framework.

---

## Testing Strategy

The tool exists partly to be tested, so testability must be explicit.

### Unit-level testing

Cover:

- scenario engine transitions
- escalation rule evaluation
- LLM adapter configuration and request shaping
- snapshot normalization

### Integration testing

Cover:

- frontend rendering from backend conversation state
- customer generation leading to rendered messages
- mode changes across `ai_active` and `human_takeover`
- provider switching between OpenAI-compatible endpoints

### Browser automation testing

Cover:

- reading the normalized snapshot from Playwright
- selecting a conversation
- typing into the real composer
- sending replies
- respecting takeover boundaries

### Demo-readiness testing

Cover a few seeded scenarios such as:

- product inquiry that resolves cleanly
- multilingual clarification
- frustrated customer requiring takeover
- timeout-based escalation

---

## MVP Success Criteria

The MVP is successful when all of the following are true:

- a viewer can mistake the tool for a real customer-service workspace at first glance
- the customer side behaves credibly enough that demos do not feel scripted
- Memoh can take over the page through Playwright and conduct the conversation through visible UI controls
- the simulator can deterministically trigger at least one escalation path to a human operator
- the operator can demonstrate both testing and presentation workflows from the same tool

---

## Future Expansion

After the MVP, the most natural extensions are:

- alternate visual skins for different platforms
- scenario packs for different industries
- richer metrics and replay tools
- multi-seat human collaboration
- attachment and order-detail simulation

These are intentionally deferred so the first version stays focused on the one thing that matters: a believable, browser-takeover-friendly customer-service simulator.
