# Memoh Enterprise Features Architecture Design

> Target: 500-user enterprise (明生医疗) on 8C32G single server, WeCom as primary channel
> Base: Memoh v0.3.1 | Reference: OpenFang autonomous agent patterns
> Includes: AI Management Cockpit (AI效能管理驾驶舱) as integrated frontend module

---

## Table of Contents

1. [Feature Overview & Priority](#1-feature-overview--priority)
2. [F1: RBAC & Permission System](#2-f1-rbac--permission-system)
3. [F2: Audit Logging](#3-f2-audit-logging)
4. [F3: Smart Model Routing](#4-f3-smart-model-routing)
5. [F4: Cost Tracking & Budget Control](#5-f4-cost-tracking--budget-control)
6. [F5: Hands (Autonomous Agent Packages)](#6-f5-hands-autonomous-agent-packages)
7. [F6: WeCom Channel Adapter](#7-f6-wecom-channel-adapter)
8. [F7: Cockpit (AI Management Dashboard)](#8-f7-cockpit-ai-management-dashboard)
9. [Database Migration Plan](#9-database-migration-plan)
10. [Performance Budget (8C32G)](#10-performance-budget-8c32g)
11. [Implementation Sequence](#11-implementation-sequence)

---

## 1. Feature Overview & Priority

| # | Feature | Priority | Depends On | Effort |
|---|---------|----------|------------|--------|
| F1 | RBAC & Permission System | P0 | - | Medium |
| F2 | Audit Logging | P0 | F1 | Medium |
| F3 | Smart Model Routing | P1 | - | Medium |
| F4 | Cost Tracking & Budget Control | P1 | F3 | Medium |
| F5 | Hands (Autonomous Agents) | P2 | F1, F3 | Large |
| F6 | WeCom Channel Adapter | P3 | F1 | Medium |
| F7 | Cockpit (AI Management Dashboard) | P1 | F1, F4, F5 | Large |

---

## 2. F1: RBAC & Permission System

### 2.1 Current State Analysis

Memoh already has a foundation:
- `users` table with `role` enum (`member` | `admin`)
- `bot_members` table with `role` enum (`owner` | `admin` | `member`)
- `AuthorizeAccess()` in `internal/bots/service.go` — per-bot access policy
- JWT auth with `internal/auth/jwt.go`
- Middleware in `internal/server/server.go`

**Gaps for enterprise:**
- No department/team/group concept
- No fine-grained permission (tool-level, action-level)
- No bot visibility scoping (which bots a department can see)
- No sensitive operation approval flow

### 2.2 Design

#### 2.2.1 Role Hierarchy

```
super_admin          (system-wide, manages everything)
  |
org_admin            (manages departments, users, bot assignments)
  |
dept_admin           (manages own department's bot access)
  |
member               (uses assigned bots)
  |
guest                (limited access via preauth keys, existing)
```

Extend `user_role` enum: `member` | `admin` -> `member` | `dept_admin` | `org_admin` | `super_admin`

> Note: For backwards compatibility, treat existing `admin` as `super_admin` during migration.

#### 2.2.2 New Tables

```sql
-- Departments / Teams
CREATE TABLE departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT DEFAULT '',
    parent_id UUID REFERENCES departments(id) ON DELETE SET NULL,  -- tree structure
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- User <-> Department binding (many-to-many)
CREATE TABLE department_members (
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (department_id, user_id)
);

-- Bot <-> Department visibility (which departments can access a bot)
CREATE TABLE bot_department_access (
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (bot_id, department_id)
);

-- Permission definitions for fine-grained control
CREATE TABLE bot_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    role TEXT NOT NULL,  -- 'owner' | 'admin' | 'member'
    resource TEXT NOT NULL,  -- 'tools', 'skills', 'settings', 'members', 'files'
    action TEXT NOT NULL,    -- 'read', 'write', 'execute', 'admin'
    allowed BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (bot_id, role, resource, action)
);
```

#### 2.2.3 Authorization Flow

```
Request arrives
    |
    v
[JWT Middleware] -- extract user_id from token (existing)
    |
    v
[RBAC Middleware] -- NEW: enrich context with roles & permissions
    |  - Load user.role (system-wide)
    |  - Load department memberships (cached, 5min TTL)
    |  - Load bot memberships (cached, 5min TTL)
    |
    v
[Handler] -- calls authorize(ctx, botID, resource, action)
    |
    v
[Permission Resolver]
    1. super_admin? -> ALLOW ALL
    2. org_admin? -> ALLOW (org-level resources)
    3. Check bot_members.role for this user+bot
    4. Check bot_department_access for user's departments
    5. Check bot_permissions for role+resource+action
    6. DENY
```

#### 2.2.4 Go Implementation

```
internal/
  rbac/
    types.go          -- Role, Permission, DepartmentMember types
    service.go        -- CRUD for departments, permissions
    resolver.go       -- Permission resolution logic (cached)
    middleware.go     -- Echo middleware that enriches context
    cache.go          -- In-memory LRU cache for role lookups
```

**Key Interface:**

```go
// internal/rbac/resolver.go
type Resolver interface {
    // Authorize checks if user can perform action on resource within bot context
    Authorize(ctx context.Context, userID, botID string, resource, action string) error

    // AuthorizeBotAccess checks if user can access bot at all (via membership or department)
    AuthorizeBotAccess(ctx context.Context, userID, botID string) error

    // ListAccessibleBots returns bot IDs the user can access
    ListAccessibleBots(ctx context.Context, userID string) ([]string, error)
}
```

**Cache Strategy:**

```go
// internal/rbac/cache.go
type PermissionCache struct {
    mu       sync.RWMutex
    entries  map[string]*cacheEntry  // key: "user:{id}" -> roles+departments+permissions
    ttl      time.Duration           // 5 minutes for enterprise (low churn)
    maxSize  int                     // 1000 entries (covers 500 users with buffer)
}
```

#### 2.2.5 Handler Changes

Existing `internal/handlers/handler_helpers.go`:

```go
// Current:
func AuthorizeBotAccess(ctx, botService, accountService, identityID, botID, policy)

// Enhanced:
func AuthorizeBotAccess(ctx, rbacResolver, identityID, botID, resource, action) error
```

New handler file:

```go
// internal/handlers/departments.go
GET    /departments                     -- List departments (org_admin+)
POST   /departments                     -- Create department (org_admin+)
PUT    /departments/{id}                -- Update department (org_admin+)
DELETE /departments/{id}                -- Delete department (org_admin+)
GET    /departments/{id}/members        -- List members (dept_admin+)
PUT    /departments/{id}/members        -- Add/update member (dept_admin+)
DELETE /departments/{id}/members/{uid}  -- Remove member (dept_admin+)

// internal/handlers/bot_access.go
GET    /bots/{id}/department-access     -- List department access (bot owner/admin)
PUT    /bots/{id}/department-access     -- Set department access (bot owner/admin)
GET    /bots/{id}/permissions           -- List permissions (bot owner/admin)
PUT    /bots/{id}/permissions           -- Set permissions (bot owner/admin)
```

#### 2.2.6 Frontend Changes

```
packages/web/src/
  views/
    departments/
      DepartmentList.vue       -- Department tree view
      DepartmentMembers.vue    -- Member management
  components/
    bot/
      BotAccessControl.vue     -- Department assignment for bot
      BotPermissions.vue       -- Permission matrix editor
```

---

## 3. F2: Audit Logging

### 3.1 Design Principles

- **Append-only**: Audit logs are never modified or deleted
- **Structured**: Machine-parseable JSON for analytics
- **Non-blocking**: Async write via buffered channel (no request latency impact)
- **Queryable**: Indexed by user, bot, action, timestamp

### 3.2 New Table

```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    bot_id UUID REFERENCES bots(id) ON DELETE SET NULL,
    action TEXT NOT NULL,           -- 'chat', 'tool_call', 'skill_invoke', 'setting_change',
                                    -- 'member_add', 'member_remove', 'login', 'bot_create', etc.
    resource_type TEXT NOT NULL,    -- 'bot', 'user', 'department', 'model', 'schedule', etc.
    resource_id TEXT,               -- UUID of the affected resource
    detail JSONB DEFAULT '{}',     -- Action-specific details
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_bot_id ON audit_logs(bot_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- Partition by month for performance (optional, for high-volume deployments)
-- CREATE TABLE audit_logs (...) PARTITION BY RANGE (created_at);
```

### 3.3 Go Implementation

```
internal/
  audit/
    types.go       -- AuditEntry, AuditAction constants
    service.go     -- Buffered writer with background flush
    middleware.go  -- Echo middleware for automatic request logging
    query.go       -- Query builder for audit log search
```

**Core Service:**

```go
// internal/audit/service.go
type Service struct {
    queries *sqlc.Queries
    buffer  chan AuditEntry   // buffered channel, size 1000
    logger  *slog.Logger
    done    chan struct{}
}

type AuditEntry struct {
    UserID       string
    BotID        string
    Action       string
    ResourceType string
    ResourceID   string
    Detail       map[string]any
    IPAddress    string
    UserAgent    string
}

func (s *Service) Log(entry AuditEntry)  // non-blocking send to buffer
func (s *Service) Start()                 // background goroutine: batch insert every 1s or 100 entries
func (s *Service) Stop()                  // flush remaining on shutdown
```

**Middleware (auto-capture):**

```go
// internal/audit/middleware.go
func AuditMiddleware(auditService *Service) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Pre: capture request metadata
            // Post: log action based on route + status code
            // Only for mutating operations (POST/PUT/DELETE)
        }
    }
}
```

**Integration points — explicit logging in handlers:**

```go
// Example: in handlers/users.go CreateUser()
auditService.Log(audit.AuditEntry{
    UserID:       adminUserID,
    Action:       "user_create",
    ResourceType: "user",
    ResourceID:   newUser.ID,
    Detail:       map[string]any{"username": req.Username, "role": req.Role},
})
```

### 3.4 Sensitive Data Protection

Tool call audit entries must scrub:
- API keys (mask after 8 chars, reuse existing `maskAPIKey()`)
- Passwords (never logged)
- File contents over 1KB (truncate with "[truncated]")

```go
// internal/audit/scrub.go
func ScrubDetail(detail map[string]any) map[string]any
```

### 3.5 API Endpoints

```go
// internal/handlers/audit.go
GET /audit-logs                          -- Query audit logs (org_admin+)
    ?user_id=...
    &bot_id=...
    &action=...
    &from=2026-03-01&to=2026-03-05
    &page=1&per_page=50
GET /audit-logs/export                   -- CSV/JSON export (super_admin)
```

### 3.6 Retention Policy

- Default: 90 days (configurable in `config.toml`)
- Background goroutine: daily cleanup of expired logs
- Config:

```toml
[audit]
enabled = true
retention_days = 90
buffer_size = 1000
flush_interval = "1s"
```

---

## 4. F3: Smart Model Routing

### 4.1 Current State

- Model selected per-bot via `bots.chat_model_id` (single model)
- `model_variants` table exists but unused — has `weight` field
- `selectChatModel()` in `internal/conversation/flow/resolver.go` does simple lookup
- Agent uses `createModel(modelConfig)` in `packages/agent/src/model.ts` — single model

### 4.2 Design: Router Strategy

```
User message arrives
    |
    v
[Router] -- evaluate message complexity
    |
    +--> Simple (greeting, FAQ, short answer)
    |       -> Use cheap/fast model (e.g., Qwen-7B, GPT-4o-mini)
    |
    +--> Medium (analysis, summarization, moderate reasoning)
    |       -> Use balanced model (e.g., GPT-4o, Claude Sonnet)
    |
    +--> Complex (multi-step reasoning, code generation, long context)
    |       -> Use powerful model (e.g., Claude Opus, GPT-o3)
    |
    +--> Fallback: primary model unavailable
            -> Try next model in priority order
```

### 4.3 New Table

```sql
-- Model routing rules per bot
CREATE TABLE bot_model_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    name TEXT NOT NULL,                     -- 'simple', 'medium', 'complex', 'fallback'
    model_id UUID NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    priority INT NOT NULL DEFAULT 0,        -- lower = higher priority within same tier
    complexity_tier TEXT NOT NULL CHECK (complexity_tier IN ('simple', 'medium', 'complex', 'fallback')),
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (bot_id, name)
);
```

### 4.4 Complexity Classifier

Two approaches, layered:

**Approach A: Rule-based (fast, no extra LLM call)**

```go
// internal/routing/classifier.go
type ComplexityTier string

const (
    TierSimple  ComplexityTier = "simple"
    TierMedium  ComplexityTier = "medium"
    TierComplex ComplexityTier = "complex"
)

func ClassifyComplexity(message string, history []Message) ComplexityTier {
    score := 0

    // Factor 1: Message length
    if len(message) > 2000 { score += 3 }
    else if len(message) > 500 { score += 1 }

    // Factor 2: Keyword signals
    complexKeywords := []string{"analyze", "compare", "implement", "refactor",
        "debug", "architecture", "design", "evaluate", "code", "algorithm"}
    simpleKeywords := []string{"hi", "hello", "thanks", "ok", "yes", "no", "help"}

    msgLower := strings.ToLower(message)
    for _, kw := range complexKeywords {
        if strings.Contains(msgLower, kw) { score += 2 }
    }
    for _, kw := range simpleKeywords {
        if msgLower == kw { score -= 2 }
    }

    // Factor 3: Conversation depth (long conversations likely need stronger model)
    if len(history) > 20 { score += 2 }

    // Factor 4: Attachment presence (images, files -> multimodal model)
    // (checked separately for modality routing)

    switch {
    case score <= 0:  return TierSimple
    case score <= 4:  return TierMedium
    default:          return TierComplex
    }
}
```

**Approach B: LLM-based (optional, configurable)**

Use the cheap model itself to classify with a one-shot prompt:

```
Rate the complexity of this user request as SIMPLE, MEDIUM, or COMPLEX.
Only respond with one word.
Request: "{message}"
```

This costs ~50 tokens per classification. Only enable when cost savings from routing exceed classification cost.

### 4.5 Fallback & Health Check

```go
// internal/routing/router.go
type Router struct {
    queries     *sqlc.Queries
    classifier  Classifier
    healthCheck *HealthChecker
    logger      *slog.Logger
}

type HealthChecker struct {
    mu       sync.RWMutex
    status   map[string]ModelHealth  // model_id -> health
    interval time.Duration           // check every 60s
}

type ModelHealth struct {
    Available    bool
    LastCheck    time.Time
    LastLatency  time.Duration
    FailCount    int
}

func (r *Router) SelectModel(ctx context.Context, botID string, message string, history []Message) (ModelConfig, error) {
    // 1. Classify complexity
    tier := r.classifier.Classify(message, history)

    // 2. Get routes for this bot + tier, ordered by priority
    routes := r.queries.GetBotModelRoutes(ctx, botID, string(tier))

    // 3. Try each route, skip unhealthy models
    for _, route := range routes {
        if r.healthCheck.IsHealthy(route.ModelID) {
            return r.buildModelConfig(ctx, route.ModelID)
        }
    }

    // 4. Fallback tier
    fallbackRoutes := r.queries.GetBotModelRoutes(ctx, botID, "fallback")
    for _, route := range fallbackRoutes {
        if r.healthCheck.IsHealthy(route.ModelID) {
            return r.buildModelConfig(ctx, route.ModelID)
        }
    }

    // 5. Last resort: bot's default chat_model_id
    return r.getDefaultModel(ctx, botID)
}
```

### 4.6 Integration Point

Modify `internal/conversation/flow/resolver.go`:

```go
// Current: selectChatModel() -> single model lookup
// New: selectChatModel() calls router.SelectModel() if routing enabled

func (r *Resolver) selectChatModel(ctx, req, botSettings, cs) (models.GetResponse, sqlc.LlmProvider, error) {
    // If model explicitly specified in request -> use it (override)
    if req.Model != "" {
        return r.lookupModel(ctx, req.Model)
    }

    // If routing enabled for this bot -> use router
    if r.routingEnabled(botSettings) {
        return r.router.SelectModel(ctx, botSettings.BotID, req.Message, req.History)
    }

    // Fallback: existing behavior
    return r.lookupModel(ctx, botSettings.ChatModelID)
}
```

### 4.7 Configuration

```toml
[routing]
enabled = true
classifier = "rule"   # "rule" | "llm"
health_check_interval = "60s"
```

### 4.8 API & UI

```go
// internal/handlers/model_routes.go
GET    /bots/{id}/model-routes          -- List routing rules
POST   /bots/{id}/model-routes          -- Create routing rule
PUT    /bots/{id}/model-routes/{rid}    -- Update routing rule
DELETE /bots/{id}/model-routes/{rid}    -- Delete routing rule
```

Frontend: Add a "Model Routing" tab in bot settings with a visual tier-to-model mapping editor.

---

## 5. F4: Cost Tracking & Budget Control

### 5.1 Current State

- `bot_history_messages.usage` JSONB already captures: `inputTokens`, `outputTokens`, `cacheReadTokens`, `cacheWriteTokens`, `reasoningTokens`
- `bot_heartbeat_logs.usage` JSONB — same structure
- `GET /bots/{id}/token-usage` endpoint exists with daily/model aggregation
- **Missing**: pricing per model, cost calculation, budget limits, per-user attribution

### 5.2 New Tables

```sql
-- Model pricing (per million tokens)
CREATE TABLE model_pricing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id UUID NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    input_price_per_mtok NUMERIC(10, 4) NOT NULL DEFAULT 0,       -- $/1M input tokens
    output_price_per_mtok NUMERIC(10, 4) NOT NULL DEFAULT 0,      -- $/1M output tokens
    cache_read_price_per_mtok NUMERIC(10, 4) NOT NULL DEFAULT 0,  -- $/1M cache read tokens
    cache_write_price_per_mtok NUMERIC(10, 4) NOT NULL DEFAULT 0, -- $/1M cache write tokens
    reasoning_price_per_mtok NUMERIC(10, 4) NOT NULL DEFAULT 0,   -- $/1M reasoning tokens
    effective_from TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (model_id, effective_from)
);

-- Budget limits
CREATE TABLE budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope_type TEXT NOT NULL CHECK (scope_type IN ('system', 'department', 'bot', 'user')),
    scope_id TEXT NOT NULL,             -- the UUID of the scoped entity
    period TEXT NOT NULL CHECK (period IN ('daily', 'weekly', 'monthly')),
    limit_amount NUMERIC(10, 2) NOT NULL,  -- in USD
    alert_threshold NUMERIC(3, 2) NOT NULL DEFAULT 0.80,  -- alert at 80%
    action_on_exceed TEXT NOT NULL DEFAULT 'warn' CHECK (action_on_exceed IN ('warn', 'block', 'downgrade')),
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (scope_type, scope_id, period)
);
```

### 5.3 Cost Calculation

```go
// internal/cost/calculator.go
type Calculator struct {
    queries *sqlc.Queries
    cache   map[string]*ModelPricing  // model_id -> pricing (refreshed hourly)
}

type CostBreakdown struct {
    InputCost     float64
    OutputCost    float64
    CacheReadCost float64
    CacheWriteCost float64
    ReasoningCost float64
    TotalCost     float64
    Currency      string  // "USD"
}

func (c *Calculator) Calculate(modelID string, usage TokenUsage) CostBreakdown {
    pricing := c.getPricing(modelID)
    return CostBreakdown{
        InputCost:      float64(usage.InputTokens) / 1_000_000 * pricing.InputPricePerMTok,
        OutputCost:     float64(usage.OutputTokens) / 1_000_000 * pricing.OutputPricePerMTok,
        CacheReadCost:  float64(usage.CacheReadTokens) / 1_000_000 * pricing.CacheReadPricePerMTok,
        CacheWriteCost: float64(usage.CacheWriteTokens) / 1_000_000 * pricing.CacheWritePricePerMTok,
        ReasoningCost:  float64(usage.ReasoningTokens) / 1_000_000 * pricing.ReasoningPricePerMTok,
    }
    // TotalCost = sum of all
}
```

### 5.4 Budget Enforcement

```go
// internal/cost/budget.go
type BudgetChecker struct {
    queries    *sqlc.Queries
    calculator *Calculator
    logger     *slog.Logger
}

type BudgetCheckResult struct {
    Allowed    bool
    Reason     string
    Usage      float64  // current period spend
    Limit      float64  // budget limit
    Percentage float64  // usage/limit * 100
}

func (b *BudgetChecker) Check(ctx context.Context, userID, botID, deptID string) BudgetCheckResult {
    // Check all applicable budgets (user > bot > department > system)
    // Most restrictive wins
    // Returns: allowed, or reason for denial
}
```

**Integration in conversation flow:**

```go
// internal/conversation/flow/resolver.go - before sending to agent gateway
budgetResult := r.budgetChecker.Check(ctx, userID, botID, deptID)
if !budgetResult.Allowed {
    switch budget.ActionOnExceed {
    case "block":
        return error("budget exceeded")
    case "downgrade":
        // Force use of cheapest model
        modelConfig = r.router.GetCheapestModel(ctx, botID)
    case "warn":
        // Add warning to response, continue
    }
}
```

### 5.5 Enhanced API

```go
// internal/handlers/cost.go
GET /bots/{id}/cost                     -- Cost breakdown (daily/model/user)
    ?from=...&to=...&group_by=day|model|user

GET /cost/summary                       -- System-wide cost summary (org_admin+)
    ?from=...&to=...&group_by=department|bot|user

// internal/handlers/budgets.go
GET    /budgets                          -- List budgets (org_admin+)
POST   /budgets                          -- Create budget (org_admin+)
PUT    /budgets/{id}                     -- Update budget (org_admin+)
DELETE /budgets/{id}                     -- Delete budget (org_admin+)

// internal/handlers/model_pricing.go
GET    /models/{id}/pricing              -- Get model pricing
PUT    /models/{id}/pricing              -- Set model pricing (super_admin)
POST   /models/pricing/import            -- Bulk import pricing (super_admin)
```

### 5.6 Cost Dashboard (Frontend)

```
packages/web/src/views/cost/
  CostDashboard.vue          -- System-wide cost overview
    - Total spend this month (line chart)
    - Cost by department (bar chart)
    - Cost by model (pie chart)
    - Top 10 users by spend
  BotCostDetail.vue          -- Per-bot cost breakdown
  BudgetManager.vue          -- Budget CRUD with alert thresholds
```

---

## 6. F5: Hands (Autonomous Agent Packages)

### 6.1 Concept Mapping: OpenFang Hands -> Memoh

| OpenFang | Memoh Current | Memoh Enhanced |
|----------|--------------|----------------|
| HAND.toml | SKILL.md | HAND.md (extended SKILL.md with schedule + tools + triggers) |
| 7 built-in Hands | 0 | Enterprise Hands (see below) |
| FangHub marketplace | - | Internal Hand library (enterprise) |
| Autonomous execution | trigger-schedule exists | Full Hand lifecycle with monitoring |

### 6.2 HAND.md Format

Extend existing SKILL.md format (backwards compatible):

```markdown
---
name: daily-report
description: Generate daily team activity report
type: hand                              # NEW: 'skill' (default) | 'hand'
schedule:
  pattern: "0 0 18 * * 1-5"            # NEW: Cron pattern (6pm weekdays)
  max_calls: ~                           # unlimited
  timezone: "Asia/Shanghai"
triggers:                                # NEW: event-based triggers
  - event: "message_keyword"
    match: "generate report"
  - event: "webhook"
    path: "/hooks/daily-report"
tools:                                   # NEW: allowed tool whitelist
  - web_fetch
  - search_inbox
  - send
  - list_contacts
model:                                   # NEW: override bot's default model
  complexity_tier: "medium"
  reasoning: false
output:                                  # NEW: where results go
  channels: ["wecom"]
  target: "department_group"
permissions:
  require_approval: false                # human-in-the-loop for this hand
  max_tokens_per_run: 50000
metadata:
  author: "enterprise-admin"
  version: "1.0"
  tags: ["report", "daily"]
---

## Role
You are a daily report generator for the team. Your job is to:
1. Review today's inbox messages and conversation history
2. Summarize key discussions, decisions, and action items
3. Format as a structured report
4. Send to the department group chat

## Report Format
### Daily Report - {date}

**Key Discussions:**
- ...

**Decisions Made:**
- ...

**Action Items:**
- [ ] ...

**Tomorrow's Priorities:**
- ...
```

### 6.3 New Tables

```sql
-- Hand definitions (extends skills concept)
CREATE TABLE hands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    content TEXT NOT NULL,                -- Full HAND.md content
    type TEXT NOT NULL DEFAULT 'hand',
    schedule_pattern TEXT,               -- Cron pattern (null = manual/event only)
    schedule_timezone TEXT DEFAULT 'UTC',
    schedule_max_calls INT,              -- null = unlimited
    triggers JSONB DEFAULT '[]',         -- Event trigger definitions
    allowed_tools JSONB DEFAULT '[]',    -- Tool whitelist
    model_config JSONB DEFAULT '{}',     -- Model override config
    output_config JSONB DEFAULT '{}',    -- Output channel/target config
    permissions JSONB DEFAULT '{}',      -- Approval, token limits
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (bot_id, name)
);

-- Hand execution logs
CREATE TABLE hand_execution_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hand_id UUID NOT NULL REFERENCES hands(id) ON DELETE CASCADE,
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    trigger_type TEXT NOT NULL,           -- 'schedule', 'event', 'manual', 'webhook'
    trigger_detail JSONB DEFAULT '{}',
    status TEXT NOT NULL CHECK (status IN ('running', 'completed', 'failed', 'cancelled', 'approval_pending')),
    result_text TEXT,
    error_message TEXT,
    usage JSONB,                          -- Token usage
    model_id UUID REFERENCES models(id),
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_hand_execution_logs_hand_id ON hand_execution_logs(hand_id);
CREATE INDEX idx_hand_execution_logs_status ON hand_execution_logs(status);
```

### 6.4 Hand Lifecycle

```
[Hand Created/Updated]
    |
    v
[Hand Scheduler] -- register cron job if schedule_pattern exists
    |
    v
[Trigger fires] -- cron tick / event match / webhook / manual
    |
    v
[Pre-execution checks]
    |- Budget check (F4)
    |- Permission check (F1)
    |- Approval gate (if require_approval=true -> pause, notify admin)
    |
    v
[Hand Executor]
    |- Create restricted agent (only allowed_tools)
    |- Inject hand content as system prompt extension
    |- Select model based on hand's model_config or routing (F3)
    |- Execute with token limit
    |
    v
[Post-execution]
    |- Log execution result to hand_execution_logs
    |- Audit log (F2)
    |- Cost tracking (F4)
    |- Send output to configured channels
    |- If failed, retry with backoff (max 3 retries)
```

### 6.5 Go Implementation

```
internal/
  hands/
    types.go         -- Hand, HandExecution, HandConfig types
    service.go       -- CRUD for hands
    scheduler.go     -- Cron scheduling for hands (extends existing schedule service pattern)
    executor.go      -- Hand execution orchestration
    parser.go        -- Parse HAND.md frontmatter + content
```

**Key Interface:**

```go
// internal/hands/executor.go
type Executor struct {
    scheduleService *schedule.Service
    flowResolver    *flow.Resolver
    budgetChecker   *cost.BudgetChecker
    auditService    *audit.Service
    queries         *sqlc.Queries
    logger          *slog.Logger
}

func (e *Executor) Execute(ctx context.Context, hand Hand, trigger TriggerContext) (*ExecutionResult, error) {
    // 1. Pre-checks (budget, permissions, approval)
    // 2. Build restricted agent request
    // 3. Call agent gateway with hand context
    // 4. Capture result + usage
    // 5. Log execution
    // 6. Deliver output
}
```

**Agent Gateway Extension:**

```typescript
// agent/src/modules/chat.ts — new route
.post('/chat/trigger-hand', async ({ body, bearer }) => {
    const { triggerHand } = createAgent({...}, authFetcher)
    return triggerHand({
        hand: body.hand,
        messages: body.messages,
        skills: body.skills,
    })
})
```

```typescript
// packages/agent/src/agent.ts — new method
triggerHand: async (params: HandParams) => {
    const handPrompt = buildHandPrompt(params.hand)
    // Restrict tools to hand's allowed_tools
    const restrictedTools = filterTools(allTools, params.hand.allowedTools)
    const result = await generateText({
        model,
        system: [...systemPrompt, handPrompt],
        tools: restrictedTools,
        messages: params.messages,
        maxTokens: params.hand.permissions?.maxTokensPerRun,
    })
    return result
}
```

### 6.6 Enterprise Pre-built Hands

| Hand | Schedule | Description |
|------|----------|-------------|
| daily-report | 18:00 weekdays | Summarize team conversations, decisions, action items |
| knowledge-digest | 09:00 daily | Scan inbox for important updates, distill into briefing |
| meeting-minutes | On trigger | Generate meeting notes from conversation context |
| compliance-check | 02:00 daily | Scan conversations for sensitive data leaks, policy violations |
| onboarding-guide | On trigger | Guide new employees through company resources |
| weekly-review | 17:00 Friday | Weekly team productivity and project status summary |

### 6.7 API & UI

```go
// internal/handlers/hands.go
GET    /bots/{id}/hands                  -- List hands
POST   /bots/{id}/hands                  -- Create hand
GET    /bots/{id}/hands/{hid}            -- Get hand detail
PUT    /bots/{id}/hands/{hid}            -- Update hand
DELETE /bots/{id}/hands/{hid}            -- Delete hand
POST   /bots/{id}/hands/{hid}/execute    -- Manual trigger
GET    /bots/{id}/hands/{hid}/logs       -- Execution history
```

Frontend:

```
packages/web/src/views/hands/
  HandList.vue            -- Grid/list of hands with status indicators
  HandEditor.vue          -- HAND.md editor with live preview
  HandExecutionLog.vue    -- Execution history with status/cost/duration
  HandTemplates.vue       -- Pre-built hand template picker
```

---

## 7. F6: WeCom Channel Adapter

### 7.1 WeCom API Overview

WeCom (企业微信) uses a callback-based architecture:
- **Callback URL**: WeCom pushes events (messages, follows, etc.) to your server
- **Active messaging**: Your server calls WeCom API to send messages
- **Authentication**: CorpID + Secret -> AccessToken (2h TTL, cached)
- **Message encryption**: AES-256-CBC with EncodingAESKey

### 7.2 Architecture

Follow existing adapter pattern in `internal/channel/adapters/`:

```
internal/channel/adapters/wecom/
  adapter.go       -- Implements channel.Adapter interface
  client.go        -- WeCom API client (token management, message sending)
  handler.go       -- Webhook handler (message receive, event handling)
  crypto.go        -- Message encryption/decryption (AES-CBC + SHA1 signature)
  types.go         -- WeCom-specific types (message formats, events)
```

### 7.3 Adapter Implementation

```go
// internal/channel/adapters/wecom/adapter.go
type Adapter struct {
    client *Client
    logger *slog.Logger
}

func (a *Adapter) Type() string { return "wecom" }

func (a *Adapter) NormalizeConfig(config map[string]any) error {
    // Validate: corp_id, agent_id, secret, token, encoding_aes_key
}

func (a *Adapter) ResolveTarget(config map[string]any) (channel.SendTarget, error) {
    // Map to WeCom user_id or chat_id
}

func (a *Adapter) Send(ctx context.Context, target channel.SendTarget, msg channel.OutgoingMessage) error {
    // POST to https://qyapi.weixin.qq.com/cgi-bin/message/send
    // Support: text, markdown, image, file, template_card
}
```

### 7.4 WeCom API Client

```go
// internal/channel/adapters/wecom/client.go
type Client struct {
    corpID   string
    secret   string
    agentID  int
    token    string          // callback verification token
    aesKey   []byte          // EncodingAESKey (base64 decoded, 32 bytes)
    mu       sync.RWMutex
    tokenCache *AccessToken  // cached access_token with expiry
    httpClient *http.Client
}

type AccessToken struct {
    Token     string
    ExpiresAt time.Time
}

func (c *Client) GetAccessToken(ctx context.Context) (string, error) {
    // Cache with refresh: GET /gettoken?corpid=...&corpsecret=...
    // Token valid for 7200s, refresh at 80% TTL
}

func (c *Client) SendText(ctx context.Context, toUser, content string) error
func (c *Client) SendMarkdown(ctx context.Context, toUser, content string) error
func (c *Client) SendFile(ctx context.Context, toUser string, mediaID string) error
func (c *Client) UploadMedia(ctx context.Context, fileType string, data io.Reader) (string, error)
```

### 7.5 Webhook Handler

```go
// internal/channel/adapters/wecom/handler.go

// Verification endpoint (GET) - WeCom URL verification
func (h *Handler) VerifyURL(c echo.Context) error {
    // 1. Parse msg_signature, timestamp, nonce, echostr from query
    // 2. Verify signature: SHA1(sort(token, timestamp, nonce, echostr))
    // 3. Decrypt echostr with AES key
    // 4. Return decrypted echostr
}

// Message receive endpoint (POST) - WeCom pushes messages here
func (h *Handler) ReceiveMessage(c echo.Context) error {
    // 1. Parse XML body
    // 2. Verify signature
    // 3. Decrypt message content
    // 4. Parse message type (text, image, voice, video, location, link, event)
    // 5. Resolve channel identity from FromUserName
    // 6. Route to conversation flow
    // 7. Return "success" (WeCom requires immediate response)
}
```

### 7.6 Message Format Mapping

| WeCom Inbound | Memoh Internal |
|---------------|---------------|
| text | `{ role: "user", content: text }` |
| image | `{ role: "user", content: "", attachments: [{ type: "image", url }] }` |
| voice | `{ role: "user", content: "[voice message]", attachments: [{ type: "audio", url }] }` |
| event (subscribe) | System event, no conversation |
| event (click menu) | Map to skill/hand trigger |

| Memoh Outbound | WeCom Format |
|----------------|-------------|
| Plain text | text message |
| Markdown | markdown message (WeCom supports subset) |
| Long text (>2048 chars) | Split into multiple messages |
| File attachment | Upload media -> send file message |

### 7.7 Streaming Support

WeCom doesn't support SSE/streaming natively. Strategy:

1. Receive message -> respond "success" immediately (WeCom timeout: 5s)
2. Process asynchronously
3. Send result via active message API when ready
4. For long responses: send "thinking..." indicator, then final result

```go
func (h *Handler) ReceiveMessage(c echo.Context) error {
    // ...parse and decrypt...

    // Async processing (non-blocking)
    go func() {
        result, err := h.conversationFlow.Chat(ctx, chatRequest)
        if err != nil {
            h.client.SendText(ctx, fromUser, "An error occurred.")
            return
        }
        h.client.SendMarkdown(ctx, fromUser, result.Content)
    }()

    return c.String(200, "success")  // WeCom requires immediate response
}
```

### 7.8 Bot Channel Configuration

```go
// bot_channel_configs entry for WeCom
{
    "channel_type": "wecom",
    "config": {
        "corp_id": "ww1234567890",
        "agent_id": 1000002,
        "secret": "...",
        "token": "...",               // Callback verification token
        "encoding_aes_key": "...",    // 43-char base64 encoded AES key
        "welcome_message": "Hi, I'm your AI assistant.",
        "group_chat_enabled": true,
        "mention_only": true          // In group chats, only respond when @mentioned
    }
}
```

### 7.9 Route Registration

```go
// internal/server/server.go — add WeCom webhook routes to skip list
skipPaths = append(skipPaths, "/channels/wecom/webhook/*")

// internal/channel/router.go — register WeCom adapter
adapters["wecom"] = wecom.NewAdapter(...)
```

### 7.10 Configuration

```toml
[channels.wecom]
enabled = true
callback_base_url = "https://your-domain.com/channels/wecom/webhook"
```

---

## 8. F7: Cockpit (AI Management Dashboard)

> AI效能管理驾驶舱 — 原明生医疗 Demo 需求，作为 Memoh 企业版的集成前端模块实现。
> 核心理念：**"不是告诉老板AI花了多少钱，而是告诉老板AI帮了多少忙。"**

### 8.1 Architecture Position

Cockpit 不是独立服务，是 F1/F4/F5 已有 API 的**前端聚合视图** + 少量聚合查询接口。

```
F1 RBAC ─────────┐
  (departments,   │
   users, roles)  │
                  │
F4 Cost ──────────┼──> F7 Cockpit 驾驶舱
  (token usage,   │      ├── CockpitDashboard.vue   (效能总览)
   model pricing, │      ├── CockpitDailyReports.vue (AI日报聚合)
   budgets)       │      ├── CockpitProfile.vue      (个人效能画像)
                  │      └── CockpitSettings.vue     (驾驶舱配置)
F5 Hands ─────────┘
  (daily-report,
   execution logs,
   hand results)
```

### 8.2 Dual-Dimension Efficiency Model

```
                    AI Efficiency Assessment
                   /                        \
          Quantitative                  Qualitative
        (F4: Cost Tracking)          (F5: Hands Daily Report)
              |                              |
       Token consumption              What AI actually did
       Model usage distribution        Hours saved per task
       Cost trends                     Task completion quality
              \                              /
               ────────────────────────────────
                            |
                   AI Efficiency Index
                     (Cockpit aggregates both)
```

### 8.3 Data Flow

```
[Bot conversations]                     [Hands: daily-report]
     |                                        |
     v                                        v
bot_history_messages.usage (JSONB)     hand_execution_logs.result_text (JSON)
     |                                        |
     v                                        v
model_pricing (cost calc)              Structured daily report data:
     |                                   - tasks[].category
     |                                   - tasks[].estimated_manual_hours
     |                                   - tasks[].actual_ai_minutes
     |                                   - tasks[].quality_score
     |                                   - tasks[].innovation_tag
     |                                        |
     v                                        v
     +-----------+   Cockpit APIs   +---------+
                 |                  |
                 v                  v
           GET /cockpit/summary
           GET /cockpit/daily-reports
           GET /cockpit/rankings
           GET /cockpit/profile/:userId
```

### 8.4 Daily Report Hand (F5 Integration)

The Cockpit's qualitative data comes from a pre-built Hand: `daily-report`.

```markdown
---
name: daily-report
description: Automatically generate structured daily work report
type: hand
schedule:
  pattern: "0 0 8 * * 1-5"           # 08:00 weekdays
  timezone: "Asia/Shanghai"
triggers:
  - event: "message_keyword"
    match: "generate report"
tools:
  - search_inbox
  - list_contacts
  - send
model:
  complexity_tier: "medium"
  reasoning: false
output:
  channels: ["wecom"]
  format: "json"
permissions:
  require_approval: false
  max_tokens_per_run: 10000
metadata:
  author: "system"
  version: "1.0"
  tags: ["report", "daily", "cockpit"]
---

## Role
You are a daily work report generator. Review yesterday's ({date}) conversations
and work assisted by you, then output a structured JSON report.

## Output Format

Strictly output the following JSON, no additional text:

{
  "date": "{date}",
  "summary": "One sentence summary of today's work and time saved",
  "tasks": [
    {
      "category": "Category (document|analysis|research|content|decision_support|communication|meeting_prep|code|other)",
      "description": "What was done (under 50 chars)",
      "estimated_manual_hours": 4.0,
      "actual_ai_minutes": 35,
      "status": "completed",
      "output_type": "document|analysis|research|content|communication|code|other",
      "quality_score": 4.5,
      "innovation_tag": false
    }
  ],
  "pending_tasks": ["Unfinished task 1", "Unfinished task 2"]
}

## Guidelines
- estimated_manual_hours: reasonable estimate based on industry average
- innovation_tag: only true for genuinely innovative AI applications
- If no work yesterday, return empty tasks array
- quality_score: self-assessment 1-5
```

### 8.5 New Tables

```sql
-- Parsed daily report data (extracted from hand_execution_logs.result_text)
CREATE TABLE cockpit_daily_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    hand_execution_id UUID REFERENCES hand_execution_logs(id) ON DELETE SET NULL,
    report_date DATE NOT NULL,
    summary TEXT DEFAULT '',
    tasks JSONB NOT NULL DEFAULT '[]',
    pending_tasks JSONB DEFAULT '[]',
    total_estimated_saved_hours NUMERIC(6, 2) DEFAULT 0,
    total_ai_time_minutes INT DEFAULT 0,
    efficiency_multiplier NUMERIC(4, 1) DEFAULT 0,
    innovation_count INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (bot_id, user_id, report_date)
);

CREATE INDEX idx_cockpit_daily_reports_date ON cockpit_daily_reports(report_date DESC);
CREATE INDEX idx_cockpit_daily_reports_user ON cockpit_daily_reports(user_id);

-- Cockpit configuration (per-system settings)
CREATE TABLE cockpit_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT NOT NULL UNIQUE,
    value JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Pre-populate with defaults
INSERT INTO cockpit_config (key, value) VALUES
    ('efficiency', '{"labor_cost_hourly_cny": 200, "categories": ["document","analysis","research","content","decision_support","communication","meeting_prep","code","other"]}'),
    ('display', '{"default_period_days": 7, "innovation_highlight_count": 5}');
```

### 8.6 Backend: Cockpit API

```go
// internal/handlers/cockpit.go

// === Summary API ===

// GET /cockpit/summary?from=2026-02-26&to=2026-03-04&department_id=...
// Returns aggregated efficiency metrics for the period
type CockpitSummaryResponse struct {
    Period struct {
        Start string `json:"start"`
        End   string `json:"end"`
    } `json:"period"`
    TotalTasksCompleted       int     `json:"total_tasks_completed"`
    TotalSavedHours           float64 `json:"total_saved_hours"`
    AverageEfficiencyMultiplier float64 `json:"average_efficiency_multiplier"`
    TotalAITimeHours          float64 `json:"total_ai_time_hours"`
    EquivalentLaborCostSavedCNY float64 `json:"equivalent_labor_cost_saved_cny"`
    TotalTokenCostUSD         float64 `json:"total_token_cost_usd"`
    InnovationTasksCount      int     `json:"innovation_tasks_count"`
    ActiveUsers               int     `json:"active_users"`
    DailyAvgTasksPerUser      float64 `json:"daily_avg_tasks_per_user"`
    // Trends (vs previous period)
    SavedHoursTrend           float64 `json:"saved_hours_trend"`       // percentage change
    EfficiencyTrend           float64 `json:"efficiency_trend"`
    TasksTrend                float64 `json:"tasks_trend"`
    CostSavedTrend            float64 `json:"cost_saved_trend"`
}

// === Daily Trend API ===

// GET /cockpit/trend?from=...&to=...&department_id=...
type CockpitDailyTrend struct {
    Date         string  `json:"date"`
    Tasks        int     `json:"tasks"`
    SavedHours   float64 `json:"saved_hours"`
    Multiplier   float64 `json:"multiplier"`
    Tokens       int64   `json:"tokens"`
    CostUSD      float64 `json:"cost_usd"`
}

// === Category Distribution API ===

// GET /cockpit/categories?from=...&to=...
type CockpitCategoryDistribution struct {
    Category   string  `json:"category"`
    Count      int     `json:"count"`
    SavedHours float64 `json:"saved_hours"`
    Percentage float64 `json:"percentage"`
}

// === Daily Reports API ===

// GET /cockpit/daily-reports?date=2026-03-04&user_id=...&department_id=...
// Returns parsed daily reports for the date
type CockpitDailyReportResponse struct {
    UserID               string              `json:"user_id"`
    UserName             string              `json:"user_name"`
    UserRole             string              `json:"user_role"`
    Department           string              `json:"department"`
    Date                 string              `json:"date"`
    Summary              string              `json:"summary"`
    Tasks                []CockpitTask       `json:"tasks"`
    PendingTasks         []string            `json:"pending_tasks"`
    TotalEstimatedSavedH float64             `json:"total_estimated_saved_hours"`
    TotalAITimeMinutes   int                 `json:"total_ai_time_minutes"`
    EfficiencyMultiplier float64             `json:"efficiency_multiplier"`
}

type CockpitTask struct {
    Category             string  `json:"category"`
    Description          string  `json:"description"`
    EstimatedManualHours float64 `json:"estimated_manual_hours"`
    ActualAIMinutes      int     `json:"actual_ai_minutes"`
    Status               string  `json:"status"`
    OutputType           string  `json:"output_type"`
    QualityScore         float64 `json:"quality_score"`
    InnovationTag        bool    `json:"innovation_tag"`
}

// === User Rankings API ===

// GET /cockpit/rankings?from=...&to=...&department_id=...&limit=10
type CockpitUserRanking struct {
    UserID         string  `json:"user_id"`
    UserName       string  `json:"user_name"`
    Department     string  `json:"department"`
    TotalTasks     int     `json:"total_tasks"`
    SavedHours     float64 `json:"saved_hours"`
    AvgMultiplier  float64 `json:"avg_multiplier"`
    InnovationCount int    `json:"innovation_count"`
}

// === User Profile API ===

// GET /cockpit/profile/:userId?from=...&to=...
type CockpitProfileResponse struct {
    User struct {
        ID         string `json:"id"`
        Name       string `json:"name"`
        Role       string `json:"role"`
        Department string `json:"department"`
        DaysUsing  int    `json:"days_using"`
    } `json:"user"`
    Summary struct {
        TotalSavedHours    float64 `json:"total_saved_hours"`
        AvgMultiplier      float64 `json:"avg_multiplier"`
        TotalTasks         int     `json:"total_tasks"`
        InnovationCount    int     `json:"innovation_count"`
    } `json:"summary"`
    DailyTrend             []CockpitDailyTrend           `json:"daily_trend"`
    CategoryDistribution   []CockpitCategoryDistribution `json:"category_distribution"`
    ModelPreference        []ModelUsageShare              `json:"model_preference"`
    TokenTrend             []DailyTokenByModel            `json:"token_trend"`
    RecentTasks            []CockpitTaskWithDate          `json:"recent_tasks"`
}

type ModelUsageShare struct {
    ModelName  string  `json:"model_name"`
    Percentage float64 `json:"percentage"`
    TokenCount int64   `json:"token_count"`
}

type DailyTokenByModel struct {
    Date   string           `json:"date"`
    Models []ModelTokenDay  `json:"models"`
}

type ModelTokenDay struct {
    ModelName string `json:"model_name"`
    Tokens    int64  `json:"tokens"`
}

type CockpitTaskWithDate struct {
    Date string `json:"date"`
    CockpitTask
}

// === Innovation Highlights API ===

// GET /cockpit/innovations?from=...&to=...&limit=5
type CockpitInnovation struct {
    UserName    string `json:"user_name"`
    Date        string `json:"date"`
    Description string `json:"description"`
    Category    string `json:"category"`
}

// === Config API ===

// GET /cockpit/config                    -- Get cockpit settings (org_admin+)
// PUT /cockpit/config                    -- Update cockpit settings (org_admin+)
```

### 8.7 Backend: Service & Query Layer

```go
// internal/cockpit/service.go
type Service struct {
    queries       *sqlc.Queries
    costCalc      *cost.Calculator
    rbacResolver  *rbac.Resolver
    logger        *slog.Logger
}

// GetSummary aggregates efficiency data across users/departments for the period
func (s *Service) GetSummary(ctx context.Context, req SummaryRequest) (CockpitSummaryResponse, error) {
    // 1. Query cockpit_daily_reports for the period
    //    SUM(total_estimated_saved_hours), SUM(tasks count), AVG(efficiency_multiplier)
    // 2. Query token usage from bot_history_messages + model_pricing for cost
    // 3. Calculate trends vs previous period of same length
    // 4. Apply department filter if specified (via F1 RBAC)
}

// GetDailyReports returns parsed daily reports for a specific date
func (s *Service) GetDailyReports(ctx context.Context, req DailyReportsRequest) ([]CockpitDailyReportResponse, error) {
    // 1. Query cockpit_daily_reports for the date
    // 2. Join with users table for name/role
    // 3. Join with departments for department name
    // 4. Apply RBAC: org_admin sees all, dept_admin sees own department, member sees self only
}

// ProcessHandResult parses a daily-report Hand execution result and stores it
func (s *Service) ProcessHandResult(ctx context.Context, executionLog HandExecutionLog) error {
    // 1. Parse result_text as JSON -> DailyReportData
    // 2. Calculate aggregates (total_saved_hours, efficiency_multiplier, innovation_count)
    // 3. Upsert into cockpit_daily_reports
    // Called by hands.Executor after successful daily-report hand execution
}
```

### 8.8 SQL Queries

```sql
-- db/queries/cockpit.sql

-- name: GetCockpitSummary :one
SELECT
    COUNT(DISTINCT user_id) as active_users,
    COALESCE(SUM(jsonb_array_length(tasks)), 0) as total_tasks,
    COALESCE(SUM(total_estimated_saved_hours), 0) as total_saved_hours,
    COALESCE(AVG(efficiency_multiplier), 0) as avg_multiplier,
    COALESCE(SUM(total_ai_time_minutes), 0) as total_ai_minutes,
    COALESCE(SUM(innovation_count), 0) as innovation_count
FROM cockpit_daily_reports
WHERE report_date BETWEEN @from_date AND @to_date
    AND (@department_id::uuid IS NULL OR user_id IN (
        SELECT user_id FROM department_members WHERE department_id = @department_id
    ));

-- name: GetCockpitDailyTrend :many
SELECT
    report_date::text as date,
    COALESCE(SUM(jsonb_array_length(tasks)), 0)::int as tasks,
    COALESCE(SUM(total_estimated_saved_hours), 0) as saved_hours,
    COALESCE(AVG(efficiency_multiplier), 0) as multiplier
FROM cockpit_daily_reports
WHERE report_date BETWEEN @from_date AND @to_date
    AND (@department_id::uuid IS NULL OR user_id IN (
        SELECT user_id FROM department_members WHERE department_id = @department_id
    ))
GROUP BY report_date
ORDER BY report_date;

-- name: GetCockpitCategoryDistribution :many
SELECT
    task->>'category' as category,
    COUNT(*) as count,
    COALESCE(SUM((task->>'estimated_manual_hours')::numeric), 0) as saved_hours
FROM cockpit_daily_reports,
     jsonb_array_elements(tasks) as task
WHERE report_date BETWEEN @from_date AND @to_date
    AND (@department_id::uuid IS NULL OR user_id IN (
        SELECT user_id FROM department_members WHERE department_id = @department_id
    ))
GROUP BY task->>'category'
ORDER BY count DESC;

-- name: GetCockpitDailyReports :many
SELECT
    r.id, r.bot_id, r.user_id, r.report_date, r.summary,
    r.tasks, r.pending_tasks, r.total_estimated_saved_hours,
    r.total_ai_time_minutes, r.efficiency_multiplier, r.innovation_count,
    u.display_name as user_name, u.role as user_role
FROM cockpit_daily_reports r
LEFT JOIN users u ON r.user_id = u.id
WHERE r.report_date = @report_date
    AND (@user_id::uuid IS NULL OR r.user_id = @user_id)
ORDER BY r.total_estimated_saved_hours DESC;

-- name: GetCockpitUserRankings :many
SELECT
    r.user_id,
    u.display_name as user_name,
    COALESCE(SUM(jsonb_array_length(r.tasks)), 0)::int as total_tasks,
    COALESCE(SUM(r.total_estimated_saved_hours), 0) as saved_hours,
    COALESCE(AVG(r.efficiency_multiplier), 0) as avg_multiplier,
    COALESCE(SUM(r.innovation_count), 0)::int as innovation_count
FROM cockpit_daily_reports r
LEFT JOIN users u ON r.user_id = u.id
WHERE r.report_date BETWEEN @from_date AND @to_date
    AND (@department_id::uuid IS NULL OR r.user_id IN (
        SELECT user_id FROM department_members WHERE department_id = @department_id
    ))
GROUP BY r.user_id, u.display_name
ORDER BY saved_hours DESC
LIMIT @max_results;

-- name: GetCockpitInnovations :many
SELECT
    u.display_name as user_name,
    r.report_date::text as date,
    task->>'description' as description,
    task->>'category' as category
FROM cockpit_daily_reports r,
     jsonb_array_elements(r.tasks) as task
LEFT JOIN users u ON r.user_id = u.id
WHERE r.report_date BETWEEN @from_date AND @to_date
    AND (task->>'innovation_tag')::boolean = true
ORDER BY r.report_date DESC
LIMIT @max_results;

-- name: UpsertCockpitDailyReport :one
INSERT INTO cockpit_daily_reports (
    bot_id, user_id, hand_execution_id, report_date,
    summary, tasks, pending_tasks,
    total_estimated_saved_hours, total_ai_time_minutes,
    efficiency_multiplier, innovation_count
) VALUES (
    @bot_id, @user_id, @hand_execution_id, @report_date,
    @summary, @tasks, @pending_tasks,
    @total_estimated_saved_hours, @total_ai_time_minutes,
    @efficiency_multiplier, @innovation_count
)
ON CONFLICT (bot_id, user_id, report_date)
DO UPDATE SET
    hand_execution_id = EXCLUDED.hand_execution_id,
    summary = EXCLUDED.summary,
    tasks = EXCLUDED.tasks,
    pending_tasks = EXCLUDED.pending_tasks,
    total_estimated_saved_hours = EXCLUDED.total_estimated_saved_hours,
    total_ai_time_minutes = EXCLUDED.total_ai_time_minutes,
    efficiency_multiplier = EXCLUDED.efficiency_multiplier,
    innovation_count = EXCLUDED.innovation_count,
    created_at = now()
RETURNING *;

-- name: GetCockpitUserProfile :many
-- (Reuses GetCockpitDailyReports filtered by user_id + date range)
SELECT
    r.report_date::text as date,
    r.tasks,
    r.total_estimated_saved_hours,
    r.efficiency_multiplier,
    r.innovation_count
FROM cockpit_daily_reports r
WHERE r.user_id = @user_id
    AND r.report_date BETWEEN @from_date AND @to_date
ORDER BY r.report_date;
```

### 8.9 Frontend: Visual Design (Deep Space Theme)

The Cockpit pages use the Deep Space dark theme adapted for Memoh's Vue 3 + TailwindCSS stack.

#### 8.9.1 Theme Variables

Add to `packages/web/src/styles/cockpit.css`:

```css
:root {
  /* Deep Space base palette */
  --cockpit-surface-base:     #0a0e1a;
  --cockpit-surface-card:     #0f1629;
  --cockpit-surface-elevated: #1a2040;

  /* Accent colors */
  --cockpit-accent-cyan:      #06b6d4;
  --cockpit-accent-purple:    #8b5cf6;
  --cockpit-accent-green:     #10b981;
  --cockpit-accent-amber:     #f59e0b;
  --cockpit-accent-red:       #ef4444;
  --cockpit-accent-glow:      rgba(6, 182, 212, 0.2);

  /* Glass morphism */
  --cockpit-glass-light:      rgba(255, 255, 255, 0.03);
  --cockpit-glass-medium:     rgba(255, 255, 255, 0.06);
  --cockpit-glass-heavy:      rgba(255, 255, 255, 0.10);
  --cockpit-glass-border:     rgba(255, 255, 255, 0.06);

  /* Text hierarchy */
  --cockpit-text-primary:     #f1f5f9;
  --cockpit-text-secondary:   #94a3b8;
  --cockpit-text-muted:       #64748b;

  /* Labor cost for equivalent calculation */
  --cockpit-labor-cost-hourly: 200;  /* CNY, configurable via cockpit_config */
}
```

#### 8.9.2 Shared Components

```
packages/web/src/components/cockpit/
  GlassPanel.vue          -- Reusable glass morphism container
  StatCard.vue            -- Key metric card with glow effect
  AmbientBackground.vue   -- Floating orb background effect
  TrendBadge.vue          -- Up/down trend percentage indicator
```

**StatCard.vue Props:**

```typescript
interface StatCardProps {
  title: string           // "本周节省工时"
  value: string           // "142.5h"
  trend: number           // 12.3 (percentage vs previous period)
  icon: string            // Lucide icon name
  glowColor: 'cyan' | 'purple' | 'green' | 'amber'
}
```

**GlassPanel.vue Props:**

```typescript
interface GlassPanelProps {
  variant: 'light' | 'medium' | 'solid'  // glass intensity
  hover: boolean                          // enable glow-on-hover
  glowColor?: 'cyan' | 'purple' | 'green'
}
```

#### 8.9.3 Chart Theme (ECharts/Chart.js)

Memoh uses Vue 3, so adapt chart theme for ECharts or Chart.js instead of Recharts:

```typescript
// packages/web/src/components/cockpit/chartTheme.ts
export const cockpitChartTheme = {
  backgroundColor: 'transparent',
  textStyle: { color: '#94a3b8', fontSize: 12 },
  grid: {
    borderColor: 'rgba(255, 255, 255, 0.06)',
  },
  categoryAxis: {
    axisLine: { lineStyle: { color: 'rgba(255,255,255,0.06)' } },
    axisLabel: { color: '#94a3b8' },
    splitLine: { lineStyle: { color: 'rgba(255,255,255,0.04)', type: 'dashed' } },
  },
  valueAxis: {
    axisLine: { lineStyle: { color: 'rgba(255,255,255,0.06)' } },
    axisLabel: { color: '#94a3b8' },
    splitLine: { lineStyle: { color: 'rgba(255,255,255,0.04)', type: 'dashed' } },
  },
  tooltip: {
    backgroundColor: '#1a2040',
    borderColor: 'rgba(255,255,255,0.1)',
    textStyle: { color: '#f1f5f9' },
  },
  color: ['#06b6d4', '#8b5cf6', '#10b981', '#f59e0b', '#ef4444', '#3b82f6', '#ec4899'],
}
```

### 8.10 Frontend: Page Designs

#### 8.10.1 Route Structure

```typescript
// packages/web/src/router/
{
  path: '/cockpit',
  component: CockpitLayout,      // Dark themed layout with ambient background
  children: [
    { path: '',              component: CockpitDashboard },     // Default: efficiency overview
    { path: 'daily-reports', component: CockpitDailyReports },  // AI daily report aggregation
    { path: 'profile/:userId', component: CockpitProfile },     // Individual efficiency profile
    { path: 'settings',     component: CockpitSettings },       // Cockpit configuration
  ]
}
```

#### 8.10.2 Navigation (Sidebar Extension)

```
Sidebar
├── AI Cockpit                    <-- NEW section (cockpit-accent-cyan highlight)
│   ├── Efficiency Overview       <-- /cockpit
│   ├── AI Daily Reports          <-- /cockpit/daily-reports
│   └── Team Members              <-- /cockpit/profile (list, click to individual)
├── Bots                          <-- existing
├── Models                        <-- existing
├── Settings                      <-- existing
│   └── Cockpit Settings          <-- /cockpit/settings (NEW)
```

#### 8.10.3 CockpitDashboard.vue (Efficiency Overview)

```
┌──────────────────────────────────────────────────────────────┐
│  [AmbientBackground: 2 floating orbs]                        │
│                                                              │
│  AI Efficiency Cockpit        [Period: Last 7 days ▼]        │
│                                [Department: All ▼]            │
│                                                              │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌──────────┐ │
│  │ StatCard   │ │ StatCard   │ │ StatCard   │ │ StatCard │ │
│  │ Hours Saved│ │ Efficiency │ │ Tasks Done │ │ Cost     │ │
│  │ 142.5h     │ │ 6.2x       │ │ 68         │ │ ¥28,500  │ │
│  │ ↑12.3%     │ │ ↑0.4       │ │ ↑15%       │ │ ↑18.2%   │ │
│  │ glow:cyan  │ │ glow:purple│ │ glow:green │ │ glow:amber│
│  └────────────┘ └────────────┘ └────────────┘ └──────────┘ │
│                                                              │
│  ┌────────────────────────────────┐ ┌──────────────────────┐│
│  │ GlassPanel                    │ │ GlassPanel           ││
│  │ Daily Efficiency Trend         │ │ Task Category Dist.  ││
│  │ [Area chart + line chart]      │ │ [Donut chart]        ││
│  │  Y: saved hours               │ │  with center total   ││
│  │  Y2: efficiency multiplier    │ │                      ││
│  │  X: date                      │ │                      ││
│  └────────────────────────────────┘ └──────────────────────┘│
│                                                              │
│  ┌────────────────────────────────┐ ┌──────────────────────┐│
│  │ GlassPanel                    │ │ GlassPanel           ││
│  │ Team Rankings                  │ │ Innovation Highlights││
│  │  1. 李明 62.0h 6.5x ████▊    │ │  ★ 李明 - Auto cost  ││
│  │  2. 王芳 48.5h 6.0x ███▊     │ │    analysis report   ││
│  │  3. 陈总 32.0h 6.1x ██▊      │ │  ★ 王芳 - Competitor ││
│  │  [Click -> /cockpit/profile]  │ │    pricing analysis  ││
│  └────────────────────────────────┘ └──────────────────────┘│
└──────────────────────────────────────────────────────────────┘
```

**Data source mapping:**

| Component | API Endpoint | Backend Source |
|-----------|-------------|---------------|
| StatCards (4x) | `GET /cockpit/summary` | cockpit_daily_reports aggregate + model_pricing cost |
| Efficiency Trend | `GET /cockpit/trend` | cockpit_daily_reports grouped by date |
| Category Donut | `GET /cockpit/categories` | cockpit_daily_reports.tasks JSONB unnest |
| Team Rankings | `GET /cockpit/rankings` | cockpit_daily_reports grouped by user |
| Innovation Highlights | `GET /cockpit/innovations` | cockpit_daily_reports.tasks where innovation_tag=true |

#### 8.10.4 CockpitDailyReports.vue (AI Daily Reports)

```
┌──────────────────────────────────────────────────────────────┐
│  AI Daily Reports     [Date: 2026-03-04 ◀ ▶]  [User: All ▼]│
│                       [Department: All ▼]                     │
│                                                              │
│  ┌─ GlassPanel: 李明 ─────────────────────────────────────┐ │
│  │  👤 李明 · Project Director · Strategic Development     │ │
│  │  📅 2026-03-04  ⏱ AI: 65min  💡 Saved: 7.5h  ⚡ 6.9x  │ │
│  │                                                          │ │
│  │  ┌ TaskItem ──────────────────────────────────────────┐ │ │
│  │  │ [cyan badge] Document · Draft Q1 marketing plan    │ │ │
│  │  │ Manual est: 4.0h → AI: 35min  ⚡ 6.9x             │ │ │
│  │  ├────────────────────────────────────────────────────┤ │ │
│  │  │ [purple badge] Analysis · Cost comparison ★Innov   │ │ │
│  │  │ Manual est: 2.0h → AI: 18min  ⚡ 6.7x             │ │ │
│  │  ├────────────────────────────────────────────────────┤ │ │
│  │  │ [pink badge] Communication · Hospital invitations  │ │ │
│  │  │ Manual est: 1.5h → AI: 12min  ⚡ 7.5x             │ │ │
│  │  └────────────────────────────────────────────────────┘ │ │
│  │                                                          │ │
│  │  📌 Pending: Complete competitor analysis section | ...  │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌─ GlassPanel: 陈总 ─────────────────────────────────────┐ │
│  │  ... (same structure)                                    │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌─ GlassPanel: 王芳 ─────────────────────────────────────┐ │
│  │  ... (same structure)                                    │ │
│  └──────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

**Category color mapping (TailwindCSS):**

| Category | Color | Badge Classes |
|----------|-------|--------------|
| document | Cyan | `bg-cyan-500/20 text-cyan-400` |
| analysis | Purple | `bg-violet-500/20 text-violet-400` |
| research | Green | `bg-emerald-500/20 text-emerald-400` |
| content | Amber | `bg-amber-500/20 text-amber-400` |
| decision_support | Blue | `bg-blue-500/20 text-blue-400` |
| communication | Pink | `bg-pink-500/20 text-pink-400` |
| meeting_prep | Indigo | `bg-indigo-500/20 text-indigo-400` |
| code | Orange | `bg-orange-500/20 text-orange-400` |

**EfficiencyBadge visual rules:**

| Multiplier | Color | Style |
|------------|-------|-------|
| >= 8x | Gold | `text-amber-300 bg-amber-500/20 ring-1 ring-amber-500/30` |
| >= 5x | Cyan | `text-cyan-400 bg-cyan-500/20` |
| < 5x | Gray | `text-slate-400 bg-slate-500/20` |

#### 8.10.5 CockpitProfile.vue (Individual Profile)

```
┌──────────────────────────────────────────────────────────────┐
│  ◀ Back   Individual AI Efficiency Profile                    │
│                                                              │
│  ┌─ GlassPanel: User Header ──────────────────────────────┐ │
│  │  [Avatar] 李明                                          │ │
│  │           Project Director · Strategic Development      │ │
│  │           Using AI for 32 days                          │ │
│  │                                                          │ │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │ │
│  │  │ 62.0h    │ │ 6.5x     │ │ 28 tasks │ │ 5 innov  │  │ │
│  │  │ Saved    │ │ Avg mult │ │ Done     │ │ Items    │  │ │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘  │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌───────────────────────┐ ┌──────────────────────────────┐ │
│  │ Efficiency Trend (7d) │ │ Task Category Distribution   │ │
│  │ [Line chart]          │ │ [Donut chart]                │ │
│  └───────────────────────┘ └──────────────────────────────┘ │
│                                                              │
│  ┌───────────────────────┐ ┌──────────────────────────────┐ │
│  │ Model Preference      │ │ Token Consumption Trend      │ │
│  │ gpt-4o      68% ████ │ │ [Stacked area, by model]     │ │
│  │ claude-3.5  32% ██   │ │                              │ │
│  └───────────────────────┘ └──────────────────────────────┘ │
│                                                              │
│  ┌─ Recent Tasks ──────────────────────────────────────────┐ │
│  │ Date    Category    Description         Est → Actual    │ │
│  │ 03-04   Document    Draft Q1 plan       4.0h → 35min   │ │
│  │ 03-04   Analysis    Cost comparison ★   2.0h → 18min   │ │
│  │ 03-03   Document    Feasibility report  3.0h → 28min   │ │
│  │ ...                  [Pagination]                       │ │
│  └──────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

### 8.11 RBAC Integration

Cockpit respects the F1 RBAC permission system:

| Role | Dashboard View | Daily Reports View | Profile View |
|------|---------------|-------------------|-------------|
| super_admin | All departments, all users | All reports | Any user |
| org_admin | All departments, all users | All reports | Any user |
| dept_admin | Own department only | Own department members | Own department members |
| member | Personal stats only | Own report only | Self only |

```go
// Permission check in cockpit handlers
func (h *CockpitHandler) GetSummary(c echo.Context) error {
    userID := auth.GetUserID(c)
    deptFilter := c.QueryParam("department_id")

    // RBAC: restrict department_id filter based on role
    allowedDepts, err := h.rbacResolver.ListAccessibleDepartments(ctx, userID)
    if err != nil { return err }

    if deptFilter != "" && !contains(allowedDepts, deptFilter) {
        return echo.ErrForbidden
    }

    // For member role: force filter to own data only
    role := h.rbacResolver.GetUserRole(ctx, userID)
    if role == "member" {
        // Override: only return personal data
        return h.service.GetPersonalSummary(ctx, userID, req)
    }

    return h.service.GetSummary(ctx, req)
}
```

### 8.12 Cockpit Configuration

```go
// internal/handlers/cockpit_settings.go

// Configurable parameters stored in cockpit_config table
type CockpitEfficiencyConfig struct {
    LaborCostHourlyCNY float64  `json:"labor_cost_hourly_cny"`  // Default: 200
    Categories         []string `json:"categories"`              // Task category list
}

type CockpitDisplayConfig struct {
    DefaultPeriodDays        int `json:"default_period_days"`         // Default: 7
    InnovationHighlightCount int `json:"innovation_highlight_count"`  // Default: 5
}
```

This allows the enterprise admin to customize:
- **Labor cost rate**: Different companies have different hourly rates. Affects "equivalent cost saved" metric.
- **Categories**: Add/remove task categories (e.g., add "code_review" for tech teams).
- **Display**: Default dashboard period, number of innovation highlights shown.

---

## 9. Database Migration Plan

All changes in a single migration file `db/migrations/0026_enterprise_features.up.sql`:

```sql
-- 0026_enterprise_features.up.sql

-- F1: RBAC
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'dept_admin';
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'org_admin';
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'super_admin';

CREATE TABLE IF NOT EXISTS departments (...);
CREATE TABLE IF NOT EXISTS department_members (...);
CREATE TABLE IF NOT EXISTS bot_department_access (...);
CREATE TABLE IF NOT EXISTS bot_permissions (...);

-- F2: Audit
CREATE TABLE IF NOT EXISTS audit_logs (...);

-- F3: Routing
CREATE TABLE IF NOT EXISTS bot_model_routes (...);

-- F4: Cost
CREATE TABLE IF NOT EXISTS model_pricing (...);
CREATE TABLE IF NOT EXISTS budgets (...);

-- F5: Hands
CREATE TABLE IF NOT EXISTS hands (...);
CREATE TABLE IF NOT EXISTS hand_execution_logs (...);

-- F7: Cockpit
CREATE TABLE IF NOT EXISTS cockpit_daily_reports (...);
CREATE TABLE IF NOT EXISTS cockpit_config (...);
```

Update `0001_init.up.sql` with all new tables for fresh installations.

---

## 10. Performance Budget (8C32G)

### 9.1 Memory Allocation

| Component | Allocation | Notes |
|-----------|-----------|-------|
| PostgreSQL | 4 GB | shared_buffers=1GB, effective_cache=3GB |
| Qdrant | 4 GB | Vector index for 500 users' memories |
| Go Server | 500 MB | With RBAC cache, audit buffer, routing health checks |
| Agent Gateway (Bun) | 500 MB | Single process, handles all agent calls |
| Web UI (Nginx+static) | 50 MB | Static files, minimal |
| OS + Buffer | 3 GB | Filesystem cache, OS overhead |
| Bot Containers | ~20 GB | ~40MB per active container, ~500 containers max |
| **Total** | **~32 GB** | Tight but viable |

### 9.2 CPU Allocation

| Component | Cores | Notes |
|-----------|-------|-------|
| PostgreSQL | 1-2 | Burst for complex queries |
| Go Server | 2-3 | HTTP handling, RBAC, routing, audit |
| Agent Gateway | 1-2 | LLM API calls (mostly I/O wait) |
| Containers | shared | Lightweight, most CPU is LLM API I/O |

### 9.3 Optimization Strategies

1. **Container sharing**: For 500 users, don't create 500 containers. Use shared bot containers (e.g., 10-20 bots, each accessible by multiple users via RBAC).

2. **Connection pooling**: PostgreSQL `max_connections=200`, use pgbouncer if needed.

3. **RBAC cache**: In-memory LRU (1000 entries, 5min TTL) avoids DB queries on every request.

4. **Audit buffering**: Batch insert every 1s or 100 entries (not per-request).

5. **Model health cache**: Check every 60s, not per-request.

6. **Qdrant indexing**: Use scalar quantization to reduce memory for 500-user vector indices.

---

## 11. Implementation Sequence

```
Phase 1: Foundation (F1 + F2)                    ~2 weeks
  |
  |- Migration 0026 (RBAC + audit tables)
  |- internal/rbac/ (service, resolver, middleware, cache)
  |- internal/audit/ (service, middleware, scrub)
  |- handlers (departments, audit-logs)
  |- Frontend (department management, audit log viewer)
  |- Update existing handlers to use RBAC resolver
  |
Phase 2: Intelligence (F3 + F4)                  ~2 weeks
  |
  |- Migration 0027 (routing + cost tables)
  |- internal/routing/ (classifier, router, health checker)
  |- internal/cost/ (calculator, budget checker)
  |- Modify resolver.go to integrate routing
  |- handlers (model-routes, cost, budgets, model-pricing)
  |- Frontend (routing config, cost dashboard, budget manager)
  |
Phase 3: Autonomy (F5)                           ~2 weeks
  |
  |- Migration 0028 (hands + execution logs tables)
  |- internal/hands/ (service, scheduler, executor, parser)
  |- Agent gateway: /chat/trigger-hand endpoint
  |- packages/agent: triggerHand() method + hand prompt
  |- handlers (hands CRUD, execution logs)
  |- Frontend (hand editor, execution log, templates)
  |- Pre-built enterprise hands
  |
Phase 4: Channel (F6)                            ~1 week
  |
  |- internal/channel/adapters/wecom/ (adapter, client, handler, crypto)
  |- Webhook route registration
  |- Bot channel config UI for WeCom
  |- End-to-end testing with WeCom sandbox
  |
Phase 5: Cockpit (F7)                            ~2 weeks
  |
  |- Migration (cockpit_daily_reports, cockpit_config tables)
  |- internal/cockpit/ (service, queries)
  |- handlers/cockpit.go (summary, trend, categories, rankings, daily-reports, profile, innovations)
  |- Pre-built daily-report Hand template
  |- Deep Space theme CSS (cockpit.css, chart theme)
  |- Shared components (GlassPanel, StatCard, AmbientBackground, TrendBadge)
  |- CockpitDashboard.vue (4 stat cards, trend chart, category donut, rankings, innovations)
  |- CockpitDailyReports.vue (report cards, task items, date navigator)
  |- CockpitProfile.vue (user header, personal charts, recent tasks table)
  |- CockpitSettings.vue (labor cost, categories, display preferences)
  |- Sidebar navigation extension
  |- RBAC integration (dept_admin sees own dept, member sees self)
  |
Phase 6: Integration Testing & Hardening         ~1 week
  |
  |- Full flow testing: WeCom -> RBAC -> Routing -> Hand -> Cost -> Cockpit -> Audit
  |- Load testing: 500 concurrent users on 8C32G
  |- Security review: WeCom crypto, audit log integrity
  |- Cockpit data accuracy validation (cross-check token usage vs daily reports)
  |- Documentation
```

---

## Appendix: File Structure (New Files)

```
internal/
  rbac/
    types.go
    service.go
    resolver.go
    middleware.go
    cache.go
  audit/
    types.go
    service.go
    middleware.go
    scrub.go
    query.go
  routing/
    types.go
    classifier.go
    router.go
    health.go
  cost/
    types.go
    calculator.go
    budget.go
  hands/
    types.go
    service.go
    scheduler.go
    executor.go
    parser.go
  channel/adapters/wecom/
    adapter.go
    client.go
    handler.go
    crypto.go
    types.go
  cockpit/
    types.go
    service.go
  handlers/
    departments.go
    audit.go
    model_routes.go
    cost.go
    budgets.go
    model_pricing.go
    hands.go
    wecom_webhook.go
    cockpit.go
    cockpit_settings.go

db/
  migrations/
    0026_enterprise_features.up.sql
    0026_enterprise_features.down.sql
  queries/
    departments.sql
    department_members.sql
    bot_department_access.sql
    bot_permissions.sql
    audit_logs.sql
    bot_model_routes.sql
    model_pricing.sql
    budgets.sql
    hands.sql
    hand_execution_logs.sql
    cockpit.sql
    cockpit_config.sql

packages/
  agent/src/
    prompts/
      hand.ts              -- Hand prompt builder
    types/
      hand.ts              -- Hand type definitions
  web/src/
    views/
      departments/
        DepartmentList.vue
        DepartmentMembers.vue
      cost/
        CostDashboard.vue
        BudgetManager.vue
      hands/
        HandList.vue
        HandEditor.vue
        HandExecutionLog.vue
        HandTemplates.vue
      cockpit/
        CockpitDashboard.vue
        CockpitDailyReports.vue
        CockpitProfile.vue
        CockpitSettings.vue
    components/
      bot/
        BotAccessControl.vue
        BotPermissions.vue
        ModelRouting.vue
        WeComConfig.vue
      cockpit/
        GlassPanel.vue
        StatCard.vue
        AmbientBackground.vue
        TrendBadge.vue
        TaskItem.vue
        EfficiencyBadge.vue
        InnovationTag.vue
        DateNavigator.vue
        chartTheme.ts
    styles/
      cockpit.css
```
