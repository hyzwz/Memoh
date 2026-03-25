# LLM Customer Service Simulator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone browser-based customer-service simulator that looks like a real support console, lets Memoh take it over through Playwright, uses OpenAI-compatible models to simulate customer behavior, and supports human escalation.

**Architecture:** Create a separate TypeScript project at `/Users/murunkun/MeishuSourceCode/customer-service-simulator` with a React + Vite frontend, a Node + Fastify backend, SQLite persistence, WebSocket updates, a bounded scenario engine, and an OpenAI-compatible customer LLM adapter. The frontend owns the realistic workspace UI and the browser automation contract, while the backend remains the single source of truth for conversations, scenario runs, and handoff state.

**Tech Stack:** React 19, Vite, TypeScript, Tailwind CSS 4, Fastify, `ws`, `better-sqlite3`, Zod, OpenAI-compatible HTTP client via `openai`, Vitest, Playwright.

---

## File Map

### New project root

- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/package.json`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/pnpm-lock.yaml`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/tsconfig.json`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/tsconfig.node.json`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/vite.config.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/postcss.config.js`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/tailwind.config.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/index.html`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/.env.example`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/README.md`

### Frontend files

- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/main.tsx`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/app/App.tsx`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/app/layout.css`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/components/ConversationList.tsx`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/components/ChatPanel.tsx`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/components/DetailPanel.tsx`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/components/ControlPanel.tsx`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/components/StatusBadge.tsx`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/components/Composer.tsx`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/hooks/useWorkspaceSocket.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/hooks/useWorkspaceSnapshot.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/lib/testids.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/lib/snapshot.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/types/workspace.ts`

### Backend files

- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/index.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/app.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/routes/workspace.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/routes/control.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/ws/broker.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/db/database.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/db/schema.sql`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/db/repositories/conversations.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/db/repositories/messages.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/db/repositories/scenarios.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/domain/scenario-types.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/domain/state-machine.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/domain/escalation.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/domain/workspace-service.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/llm/client.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/llm/customer-agent.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/fixtures/scenarios/product-inquiry.json`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/fixtures/scenarios/esim-activation-delay.json`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/fixtures/scenarios/refund-escalation.json`

### Test files

- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/lib/snapshot.test.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/domain/state-machine.test.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/domain/escalation.test.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/llm/client.test.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/domain/workspace-service.test.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/routes/workspace.test.ts`
- `/Users/murunkun/MeishuSourceCode/customer-service-simulator/tests/e2e/workspace.spec.ts`

### Existing references to inspect before coding

- `/Users/murunkun/MeishuSourceCode/Memoh/docs/superpowers/specs/2026-03-25-llm-customer-service-simulator-design.md`
- `/Users/murunkun/MeishuSourceCode/Memoh/internal/channel/adapters/ctripcs/`
- `/Users/murunkun/MeishuSourceCode/Memoh/apps/browser/`

### Explicit non-goals for this plan

- No Memoh server-side code changes in this project
- No fake Ctrip backend or synthetic REST channel between Memoh and the simulator
- No multi-seat collaboration
- No attachment or voice support
- No multi-platform skin system

---

### Task 1: Scaffold the standalone simulator project

**Files:**
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/package.json`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/tsconfig.json`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/tsconfig.node.json`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/vite.config.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/postcss.config.js`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/tailwind.config.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/index.html`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/.env.example`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/README.md`

- [ ] **Step 1: Initialize the directory and write the package manifest**

Create `package.json` with scripts:

```json
{
  "scripts": {
    "dev": "concurrently \"pnpm dev:web\" \"pnpm dev:server\"",
    "dev:web": "vite",
    "dev:server": "tsx watch server/index.ts",
    "build": "pnpm build:web && pnpm build:server",
    "build:web": "vite build",
    "build:server": "tsup server/index.ts --format esm --out-dir dist-server",
    "test": "vitest run",
    "test:e2e": "playwright test"
  }
}
```

- [ ] **Step 2: Install dependencies**

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm install
```

Expected: dependencies install cleanly and scripts are available.

- [ ] **Step 3: Add the minimal frontend entrypoint**

Create:

```tsx
// src/main.tsx
import React from "react";
import ReactDOM from "react-dom/client";
import { App } from "./app/App";
import "./app/layout.css";

ReactDOM.createRoot(document.getElementById("root")!).render(<App />);
```

- [ ] **Step 4: Add the minimal server entrypoint and verify the first build**

Create:

```ts
// server/index.ts
import { createApp } from "./app";

const app = await createApp();
await app.listen({ port: 4318, host: "0.0.0.0" });
```

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm build
```

Expected: build completes with an empty app shell.

- [ ] **Step 5: Commit**

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
git init
git add .
git commit -m "chore: scaffold simulator project"
```

---

### Task 2: Define shared workspace types, state enums, and scenario schema

**Files:**
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/types/workspace.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/domain/scenario-types.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/domain/state-machine.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/domain/state-machine.test.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/fixtures/scenarios/product-inquiry.json`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/fixtures/scenarios/esim-activation-delay.json`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/fixtures/scenarios/refund-escalation.json`

- [ ] **Step 1: Write the failing state-machine tests**

Add tests for:

```ts
it("moves from pending to waiting_customer after an AI reply", () => {});
it("moves into escalated and human_takeover after a human request", () => {});
it("rejects AI sends when handling_mode is not ai_active", () => {});
```

- [ ] **Step 2: Run the state-machine tests and confirm they fail**

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm vitest run server/domain/state-machine.test.ts
```

Expected: FAIL because the state machine does not exist yet.

- [ ] **Step 3: Implement the minimal shared schema**

Define exact enums:

```ts
export type ConversationStatus =
  | "pending"
  | "in_progress"
  | "waiting_customer"
  | "escalated"
  | "resolved";

export type HandlingMode =
  | "ai_active"
  | "human_takeover"
  | "paused"
  | "closed";
```

Create a Zod-backed `ScenarioDefinition` and `ScenarioRunState` with:
- `scenario_id`
- `title`
- `category`
- `customer_profile`
- `product_context`
- `entry_context`
- `opening_goal`
- `stages`
- `escalation_policy`
- `terminal_states`

Make the stage schema explicit:
- `stage_id`
- `intent`
- `customer_goal`
- `allowed_next_actions`
- `llm_instructions`
- `tone_constraints`
- `language_constraints`
- `transition_conditions`

Make the run schema explicit:
- `scenario_run_id`
- `scenario_id`
- `active_stage_id`
- `seed_metadata`
- `satisfaction_score`
- `frustration_score`
- `escalation_flags`
- `terminal_state`

- [ ] **Step 4: Re-run the state-machine tests and confirm they pass**

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm vitest run server/domain/state-machine.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
git add src/types server/domain server/fixtures
git commit -m "feat: add workspace state model and scenario schema"
```

---

### Task 3: Add SQLite persistence and repository boundaries

**Files:**
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/db/schema.sql`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/db/database.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/db/repositories/conversations.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/db/repositories/messages.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/db/repositories/scenarios.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/domain/workspace-service.test.ts`

- [ ] **Step 1: Write failing repository tests**

Add tests for:

```ts
it("creates a conversation with a linked scenario run", () => {});
it("appends messages in timeline order", () => {});
it("persists handling_mode and status transitions", () => {});
```

- [ ] **Step 2: Run the repository tests and confirm they fail**

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm vitest run server/domain/workspace-service.test.ts
```

Expected: FAIL because persistence is not implemented.

- [ ] **Step 3: Implement the schema and repositories**

Use SQLite tables:

```sql
CREATE TABLE conversations (...);
CREATE TABLE messages (...);
CREATE TABLE scenario_runs (...);
CREATE TABLE event_log (...);
```

Keep repository APIs narrow:
- `createConversationFromScenario`
- `appendMessage`
- `setHandlingMode`
- `setConversationStatus`
- `getWorkspaceSnapshot`

- [ ] **Step 4: Re-run the repository tests and confirm they pass**

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm vitest run server/domain/workspace-service.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
git add server/db server/domain/workspace-service.test.ts
git commit -m "feat: add simulator persistence layer"
```

---

### Task 4: Implement escalation rules and the workspace orchestration service

**Files:**
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/domain/escalation.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/domain/escalation.test.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/domain/workspace-service.ts`
- Modify: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/domain/workspace-service.test.ts`

- [ ] **Step 1: Write failing escalation tests**

Add tests for:

```ts
it("escalates after too many unresolved turns", () => {});
it("escalates immediately when the customer requests a human", () => {});
it("blocks AI sends during human_takeover", () => {});
```

- [ ] **Step 2: Run the escalation tests and confirm they fail**

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm vitest run server/domain/escalation.test.ts server/domain/workspace-service.test.ts
```

Expected: FAIL because orchestration rules are not implemented.

- [ ] **Step 3: Implement the orchestration layer**

Implement a `WorkspaceService` that owns:
- conversation creation
- message append logic
- stage transitions
- escalation decisions
- AI/human send permission checks
- event log writes

- [ ] **Step 4: Re-run the orchestration tests and confirm they pass**

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm vitest run server/domain/escalation.test.ts server/domain/workspace-service.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
git add server/domain
git commit -m "feat: add escalation and workspace orchestration"
```

---

### Task 5: Implement the OpenAI-compatible LLM adapter and customer agent

**Files:**
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/llm/client.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/llm/customer-agent.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/llm/client.test.ts`
- Modify: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/domain/workspace-service.ts`

- [ ] **Step 1: Write failing LLM client tests**

Add tests for:

```ts
it("builds an OpenAI-compatible client from provider config", () => {});
it("supports non-default base URLs for MiniMax-compatible endpoints", () => {});
it("rejects missing api keys and model names", () => {});
```

- [ ] **Step 2: Run the LLM tests and confirm they fail**

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm vitest run server/llm/client.test.ts
```

Expected: FAIL because the client does not exist yet.

- [ ] **Step 3: Implement the LLM adapter and customer turn generator**

Expose:

```ts
type ProviderConfig = {
  label: string;
  baseUrl: string;
  apiKey: string;
  model: string;
  headers?: Record<string, string>;
  temperature: number;
  timeoutMs: number;
};
```

Implement `generateCustomerTurn()` from:
- current stage
- recent message history
- scenario constraints
- satisfaction/frustration score

The LLM may phrase the turn, but the scenario engine decides allowed next actions.

Persist provider selection at the scenario-run level so each run stores:
- `provider_label`
- `base_url`
- `model`
- `headers`
- `timeout_ms`

Load the active provider config from the scenario run when generating each customer turn.

- [ ] **Step 4: Re-run the LLM tests and confirm they pass**

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm vitest run server/llm/client.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
git add server/llm server/domain/workspace-service.ts
git commit -m "feat: add llm customer agent"
```

---

### Task 6: Expose workspace HTTP and WebSocket APIs

**Files:**
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/app.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/routes/workspace.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/routes/control.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/ws/broker.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/routes/workspace.test.ts`
- Modify: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/index.ts`

- [ ] **Step 1: Write failing API integration tests**

Add tests for:

```ts
it("returns the workspace snapshot for the active conversation", () => {});
it("accepts human replies only during human_takeover", () => {});
it("publishes websocket updates after new messages", () => {});
```

- [ ] **Step 2: Run the API tests and confirm they fail**

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm vitest run server/routes/workspace.test.ts
```

Expected: FAIL because the routes and broker are missing.

- [ ] **Step 3: Implement the backend surface**

Provide:
- `GET /api/workspace`
- `POST /api/control/start-scenario`
- `POST /api/control/take-over`
- `POST /api/control/return-to-ai`
- `POST /api/control/human-reply`
- `POST /api/control/pause-customer`
- `GET /api/debug/snapshot`

Broadcast live updates over WebSocket after every state mutation.

- [ ] **Step 4: Re-run the API tests and confirm they pass**

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm vitest run server/routes/workspace.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
git add server
git commit -m "feat: add workspace backend api"
```

---

### Task 7: Build the high-fidelity workspace UI and browser snapshot contract

**Files:**
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/app/App.tsx`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/app/layout.css`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/components/ConversationList.tsx`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/components/ChatPanel.tsx`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/components/DetailPanel.tsx`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/components/ControlPanel.tsx`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/components/StatusBadge.tsx`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/components/Composer.tsx`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/hooks/useWorkspaceSocket.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/hooks/useWorkspaceSnapshot.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/lib/testids.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/lib/snapshot.ts`
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/lib/snapshot.test.ts`

- [ ] **Step 1: Write failing snapshot tests**

Add tests for:

```ts
it("builds window.__CS_SNAPSHOT__ from workspace state", () => {});
it("includes stable conversation and message ids", () => {});
it("reflects handling mode and send permissions", () => {});
```

- [ ] **Step 2: Run the snapshot tests and confirm they fail**

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm vitest run src/lib/snapshot.test.ts
```

Expected: FAIL because the snapshot builder and UI are missing.

- [ ] **Step 3: Implement the workspace UI**

Requirements:
- recreate the screenshot’s three-column rhythm
- render customer, Memoh, and human messages with distinct bubbles
- show queue badges and product details
- expose stable `data-testid` markers for every Playwright-critical control
- assign the normalized snapshot to `window.__CS_SNAPSHOT__`
- keep sending bound to the real composer and send button

Required `data-testid` values:
- `conversation-list`
- `conversation-row-<conversationId>`
- `active-conversation-id`
- `latest-customer-message-id`
- `message-timeline`
- `composer-input`
- `send-button`
- `take-over-button`
- `return-to-ai-button`
- `pause-customer-button`
- `handling-mode-indicator`

Required `window.__CS_SNAPSHOT__` fields:
- `activeConversationId`
- `handlingMode`
- `status`
- `sendAllowed`
- `takeoverStatus`
- `conversationSummaries`
- `messages`
- `latestCustomerMessageId`
- `customerProfile`
- `productContext`

- [ ] **Step 4: Re-run the snapshot tests and a frontend build**

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm vitest run src/lib/snapshot.test.ts
pnpm build:web
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
git add src
git commit -m "feat: build simulator workspace ui"
```

---

### Task 8: Wire customer generation, operator controls, and human takeover UX

**Files:**
- Modify: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/domain/workspace-service.ts`
- Modify: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/server/routes/control.ts`
- Modify: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/components/ChatPanel.tsx`
- Modify: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/components/ControlPanel.tsx`
- Modify: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/src/components/Composer.tsx`

- [ ] **Step 1: Write failing orchestration tests for role handoff**

Add tests for:

```ts
it("creates a customer opening message when a scenario starts", () => {});
it("switches the composer to human mode after takeover", () => {});
it("returns control to ai_active when the operator releases the conversation", () => {});
```

- [ ] **Step 2: Run the orchestration tests and confirm they fail**

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm vitest run server/domain/workspace-service.test.ts src/lib/snapshot.test.ts
```

Expected: FAIL because the role handoff wiring is incomplete.

- [ ] **Step 3: Implement end-to-end role flow**

Wire:
- scenario start -> customer opening message
- Memoh-visible queue update
- operator takeover -> human-only sending
- operator release -> AI resumes
- pause customer -> no new LLM turns until resumed

- [ ] **Step 4: Re-run the orchestration tests and full unit suite**

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm test
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
git add server src
git commit -m "feat: wire customer generation and handoff flow"
```

---

### Task 9: Add browser-takeover end-to-end coverage and developer docs

**Files:**
- Create: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/tests/e2e/workspace.spec.ts`
- Modify: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/README.md`
- Modify: `/Users/murunkun/MeishuSourceCode/customer-service-simulator/.env.example`

- [ ] **Step 1: Write the failing Playwright spec**

Add a test that:
- loads the workspace
- starts a scenario
- reads `window.__CS_SNAPSHOT__`
- types a reply into the real composer
- verifies the timeline updates

```ts
test("supports browser takeover through the visible workspace", async ({ page }) => {});
```

- [ ] **Step 2: Run the end-to-end spec and confirm it fails**

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm test:e2e -- --grep "supports browser takeover through the visible workspace"
```

Expected: FAIL because the e2e flow has not been completed yet.

- [ ] **Step 3: Complete the e2e wiring and write docs**

Document:
- how to start the backend and frontend
- how to configure OpenAI and MiniMax-compatible providers
- how to run a seeded scenario
- how Memoh should target the page with Playwright
- how to use the debug snapshot endpoint

- [ ] **Step 4: Run the final verification suite**

Run:

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
pnpm test
pnpm test:e2e
pnpm build
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/murunkun/MeishuSourceCode/customer-service-simulator
git add tests README.md .env.example
git commit -m "feat: finish browser simulator mvp"
```
