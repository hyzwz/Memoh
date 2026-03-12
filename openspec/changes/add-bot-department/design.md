## Context

The department system is already implemented in migration 0026 with:
- `departments` table (with hierarchical parent_id support)
- `department_members` (user-department with admin/member roles)
- `bot_department_access` (many-to-many bot-department association)
- RBAC integration in `internal/rbac/` that uses departments for authorization
- Handler at `internal/handlers/departments.go` (currently: list and create departments scoped to a bot)
- Service at `internal/enterprise/department_service.go`
- Frontend page at `packages/web/src/pages/enterprise/departments/index.vue`

What's missing: the ability to define skill templates and directory structures at the department level and sync them to bots.

## Goals / Non-Goals

**Goals:**
- Associate skill templates with a department
- Define directory structure templates per department
- Auto-install skills and provision directories when a bot is added to a department
- Bulk-sync skills and directories to all existing bots in a department
- Extend the existing department UI to manage these new capabilities

**Non-Goals:**
- Changing the existing department CRUD, membership, or RBAC logic
- Changing the bot_department_access model (keep many-to-many)
- Auto-sync on every department template change (explicit sync only)
- Department-level model/channel configuration inheritance

## Decisions

### 1. New `department_skill_templates` join table

**Decision**: Create a `department_skill_templates` table (department_id, template_id, created_at) to track which skill templates belong to a department.

**Rationale**: Clean relational model with foreign keys to both `departments` and `skill_templates`. Enables querying "which skills does this department have" and "which departments use this template" efficiently.

**Alternatives considered**:
- JSONB array of template IDs on departments → No referential integrity, can't JOIN easily
- Reuse bot_skill_installs with a department marker → Semantic mismatch

### 2. Directory templates as JSONB on departments table

**Decision**: Add a `directory_templates JSONB DEFAULT '[]'` column to the existing `departments` table rather than a separate table.

**Rationale**: Directory templates are a simple list of path strings. A separate table adds complexity for no benefit. JSONB on the department row keeps it simple and atomic. Example: `["/data/reports", "/data/shared", "/data/logs"]`.

**Alternatives considered**:
- Separate `department_directory_templates` table → Over-normalized for a simple path list
- Store in departments.metadata → Mixes operational config with arbitrary metadata

### 3. Sync triggered by explicit API call, not automatic

**Decision**: Provide `POST /departments/{department_id}/sync-skills` and `POST /departments/{department_id}/sync-directories` endpoints. Do NOT auto-sync when templates change.

**Rationale**: Auto-sync could overwrite customized skills or disrupt running bots. Explicit sync gives admins control. The sync endpoints skip skills marked `customized=true` in `bot_skill_installs`.

### 4. Extend existing handler and service rather than new files

**Decision**: Add new methods to the existing `DepartmentHandler` and `DepartmentService` rather than creating new files.

**Rationale**: The department handler already has the right middleware, audit logging, and DI wiring. Adding methods keeps the codebase cohesive. New routes registered alongside existing ones.

### 5. Reuse existing skill installation logic

**Decision**: Call the same `installSkillTemplate()` code path used by the individual skill install API (`internal/handlers/skill_templates.go`).

**Rationale**: Avoids duplicating container file operations and `bot_skill_installs` record management. The sync simply iterates bots and calls the existing install for each missing skill.

## Risks / Trade-offs

- **[Risk] Bulk sync to large departments could be slow** → Mitigation: Process bots sequentially, return summary with per-bot results. Future: parallelize with goroutine pool.
- **[Risk] Sync may fail partway through** → Mitigation: Process all bots, collect errors per bot, return complete report. Don't stop on first error.
- **[Risk] Race between sync and individual skill edits** → Mitigation: Sync skips `customized=true` skills. Per-bot operations use database transactions.
- **[Trade-off] Directory templates on departments column vs separate table** → Simpler but less queryable. Acceptable since we only need to read templates for one department at a time.
