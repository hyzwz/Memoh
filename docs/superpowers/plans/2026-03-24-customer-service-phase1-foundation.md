# Customer Service Phase 1 Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first runnable customer-service slice on Memoh: customer-service metadata flow, minimal handoff lifecycle, read APIs for conversation workspace, and a basic web workspace shell.

**Architecture:** Keep Phase 1 configuration-first. Reuse existing chat, streaming, message history, departments, and subagent capabilities. Add a small `internal/customerservice` package for metadata, handoff state, and DTO normalization, then wire it into conversation flow and new handler endpoints without introducing queue/ticket formal schema yet.

**Tech Stack:** Go + Echo + sqlc backend, Bun/TypeScript agent gateway, Vue 3 + Vite frontend, Vitest for web tests, Go test for backend.

---

## File Map

### New files

- `internal/customerservice/types.go`
- `internal/customerservice/metadata.go`
- `internal/customerservice/handoff.go`
- `internal/customerservice/types_test.go`
- `internal/customerservice/handoff_test.go`
- `internal/handlers/customer_service.go`
- `internal/handlers/customer_service_test.go`
- `apps/web/src/pages/customer-service/index.vue`
- `apps/web/src/pages/customer-service/conversations/index.vue`
- `apps/web/src/pages/customer-service/conversations/detail.vue`
- `apps/web/src/pages/customer-service/types.ts`
- `apps/web/src/pages/customer-service/api.ts`

### Modified files

- `internal/conversation/flow/resolver.go`
- `internal/conversation/flow/resolver_test.go`
- `internal/server/server.go`
- `apps/web/src/router.ts`

### Existing references to inspect before coding

- `docs/architecture/智能客服开发蓝图.md`
- `docs/architecture/智能客服前端状态机与工作台设计.md`
- `internal/conversation/flow/resolver_stream_order_test.go`
- `internal/handlers/message.go`
- `internal/handlers/departments.go`
- `apps/web/src/composables/api/useChat.chat-api.ts`
- `apps/web/src/pages/enterprise/departments/index.vue`

---

### Task 1: Add customer-service core types and handoff state helpers

**Files:**
- Create: `internal/customerservice/types.go`
- Create: `internal/customerservice/metadata.go`
- Create: `internal/customerservice/handoff.go`
- Test: `internal/customerservice/types_test.go`
- Test: `internal/customerservice/handoff_test.go`

- [ ] **Step 1: Write the failing metadata tests**

Write table-driven tests for:

```go
func TestNormalizeMetadataDefaults(t *testing.T) {}
func TestNormalizeMetadataKeepsKnownFields(t *testing.T) {}
func TestNextHandoffState(t *testing.T) {}
```

Expected behaviors:
- empty metadata normalizes to `handoff_decision=none`
- invalid handoff state is rejected
- `ai_active -> awaiting_handoff -> handoff_active -> human_resolved` transitions are valid

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/customerservice/...
```

Expected: FAIL because package/files do not exist yet.

- [ ] **Step 3: Write minimal implementation**

Implement:

```go
type HandoffDecision string
type HandoffState string

type Metadata struct {
    Language            string
    Intent              string
    ServiceType         string
    ProductLine         string
    SpecialistBot       string
    DeescalationAttempts int
    HandoffDecision     HandoffDecision
    HandoffReason       string
    EmotionAssessment   EmotionAssessment
}

func NormalizeMetadata(input map[string]any) Metadata
func NextHandoffState(current HandoffState, action HandoffAction) (HandoffState, error)
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
go test ./internal/customerservice/...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/customerservice
git commit -m "feat: add customer service core types"
```

---

### Task 2: Attach customer-service metadata to conversation flow results

**Files:**
- Modify: `internal/conversation/flow/resolver.go`
- Modify: `internal/conversation/flow/resolver_test.go`
- Reference: `internal/conversation/flow/resolver_stream_order_test.go`

- [ ] **Step 1: Write the failing resolver test**

Add a focused test like:

```go
func TestStreamPreservesCustomerServiceMetadata(t *testing.T) {}
```

Expected behavior:
- when upstream agent response contains `metadata.customer_service`, resolver output preserves and normalizes it
- missing metadata does not break normal chat flow

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/conversation/flow -run TestStreamPreservesCustomerServiceMetadata
```

Expected: FAIL because resolver does not normalize or expose this metadata yet.

- [ ] **Step 3: Write minimal implementation**

Implementation notes:
- import `internal/customerservice`
- normalize `customer_service` metadata before emitting final payload
- do not change stream event names in Phase 1
- keep backward compatibility for existing consumers

- [ ] **Step 4: Run targeted resolver tests**

Run:

```bash
go test ./internal/conversation/flow -run 'TestStreamPreservesCustomerServiceMetadata|TestStreamForwardsEvents'
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/conversation/flow/resolver.go internal/conversation/flow/resolver_test.go
git commit -m "feat: preserve customer service metadata in resolver"
```

---

### Task 3: Add minimal customer-service workspace APIs

**Files:**
- Create: `internal/handlers/customer_service.go`
- Create: `internal/handlers/customer_service_test.go`
- Modify: `internal/server/server.go`
- Reference: `internal/handlers/message.go`
- Reference: `internal/handlers/departments.go`

- [ ] **Step 1: Write the failing handler tests**

Add tests for:

```go
func TestListCustomerServiceConversations(t *testing.T) {}
func TestGetCustomerServiceConversation(t *testing.T) {}
func TestClaimCustomerServiceConversation(t *testing.T) {}
```

Expected behavior:
- list returns placeholder DTOs derived from existing message/history data source or stub service
- detail returns customer-service DTO with phase and summary fields
- claim validates conversation id and returns updated state

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/handlers -run 'TestListCustomerServiceConversations|TestGetCustomerServiceConversation|TestClaimCustomerServiceConversation'
```

Expected: FAIL because handler/routes do not exist yet.

- [ ] **Step 3: Write minimal implementation**

Add endpoints:

```text
GET  /customer-service/conversations
GET  /customer-service/conversations/:id
POST /customer-service/conversations/:id/claim
POST /customer-service/conversations/:id/handoff
```

Implementation constraints:
- Phase 1 can use an in-memory or adapter-backed service interface if durable storage is not ready
- DTO must include `phase`, `emotion_level`, `handoff_reason`, `summary`
- register routes in `internal/server/server.go`

- [ ] **Step 4: Run handler tests**

Run:

```bash
go test ./internal/handlers -run 'TestListCustomerServiceConversations|TestGetCustomerServiceConversation|TestClaimCustomerServiceConversation'
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/customer_service.go internal/handlers/customer_service_test.go internal/server/server.go
git commit -m "feat: add customer service workspace handlers"
```

---

### Task 4: Add customer-service workspace shell to the web app

**Files:**
- Create: `apps/web/src/pages/customer-service/index.vue`
- Create: `apps/web/src/pages/customer-service/conversations/index.vue`
- Create: `apps/web/src/pages/customer-service/conversations/detail.vue`
- Create: `apps/web/src/pages/customer-service/types.ts`
- Create: `apps/web/src/pages/customer-service/api.ts`
- Modify: `apps/web/src/router.ts`

- [ ] **Step 1: Write the failing router/UI test**

Add or extend a router test with:

```ts
it('registers customer service routes', () => {})
```

Expected behavior:
- `/customer-service`
- `/customer-service/conversations`
- `/customer-service/conversations/:id`

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
pnpm vitest apps/web/src/router.enterprise-routes.test.ts
```

Expected: FAIL because routes do not exist.

- [ ] **Step 3: Write minimal implementation**

Implementation notes:
- start with workspace shell, conversation list, and detail skeleton
- keep API wrapper isolated in `apps/web/src/pages/customer-service/api.ts`
- use existing UI patterns from enterprise pages
- no queue/ticket UI yet

- [ ] **Step 4: Run frontend tests**

Run:

```bash
pnpm vitest apps/web/src/router.enterprise-routes.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add apps/web/src/router.ts apps/web/src/pages/customer-service
git commit -m "feat: add customer service workspace shell"
```

---

### Task 5: Verify the end-to-end Phase 1 foundation slice

**Files:**
- Verify: `internal/customerservice/*`
- Verify: `internal/conversation/flow/*`
- Verify: `internal/handlers/customer_service*`
- Verify: `apps/web/src/pages/customer-service/*`

- [ ] **Step 1: Run backend verification**

Run:

```bash
go test ./internal/customerservice/... ./internal/conversation/flow ./internal/handlers
```

Expected: PASS

- [ ] **Step 2: Run frontend verification**

Run:

```bash
pnpm vitest apps/web/src/router.enterprise-routes.test.ts
```

Expected: PASS

- [ ] **Step 3: Sanity-check route registration and DTO shape**

Confirm manually in code:
- handler routes are registered once
- frontend route names are unique
- DTO fields align with `docs/architecture/智能客服开发蓝图.md`

- [ ] **Step 4: Commit verification if any follow-up fixes were required**

```bash
git add .
git commit -m "test: verify customer service phase1 foundation"
```

---

## Notes

- Do not add queue/ticket formal database schema in this slice.
- Do not invent new SSE event names in this slice.
- Keep all customer-service-specific state behind small DTOs and helper types.
- If handler storage becomes ambiguous, create a narrow `internal/customerservice.Service` interface instead of leaking sqlc into handlers.
- Reviewer subagent step from the skill is intentionally skipped here because this session has no explicit user authorization for subagent delegation.
