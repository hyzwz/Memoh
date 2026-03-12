# Add Bot Department Templates Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the existing department system with skill template associations and directory templates, enabling batch skill deployment and uniform directory structures across all bots in a department.

**Architecture:** Build on the existing `departments` table (migration 0026) and `bot_department_access` many-to-many. Add a `department_skill_templates` join table and a `directory_templates` JSONB column on `departments`. Extend `DepartmentHandler` and `DepartmentService` with new endpoints. Reuse the existing skill install logic from `SkillTemplateHandler.Install()`.

**Tech Stack:** Go 1.25 + Echo + Uber FX + sqlc, PostgreSQL, Vue 3 + Vite + TailwindCSS, Reka UI

---

## File Structure

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `db/migrations/0001_init.up.sql` | Add `department_skill_templates` table and `directory_templates` column to canonical schema |
| Create | `db/migrations/0036_department_templates.up.sql` | Incremental migration (add table + column) |
| Create | `db/migrations/0036_department_templates.down.sql` | Rollback migration |
| Modify | `db/queries/departments.sql` | Add 5 new sqlc queries for skill template and directory template operations |
| Modify | `internal/enterprise/department_service.go` | Add skill template CRUD, directory template CRUD, sync methods, OnBotAddedToDepartment |
| Modify | `internal/handlers/departments.go` | Add DTOs, interface methods, new endpoint handlers with Swagger annotations |
| Modify | `cmd/agent/main.go` | Update `provideDepartmentHandler` to inject `mcp.Manager` and `*dbsqlc.Queries` |
| Modify | `packages/web/src/pages/enterprise/departments/index.vue` | Extend with skill template + directory template management UI |

---

## Chunk 1: Database Layer

### Task 1: Add department_skill_templates table and directory_templates column

**Files:**
- Modify: `db/migrations/0001_init.up.sql` (after the `bot_department_access` table, around line where departments section ends in 0026 content)
- Create: `db/migrations/0036_department_templates.up.sql`
- Create: `db/migrations/0036_department_templates.down.sql`

- [ ] **Step 1: Add to canonical schema `0001_init.up.sql`**

Find the `departments` table definition and add the `directory_templates` column. Then add the new join table after `bot_department_access`.

In `0001_init.up.sql`, locate the departments CREATE TABLE (from 0026 content merged into 0001) and:
1. Add `directory_templates JSONB NOT NULL DEFAULT '[]'` to the `departments` table
2. Add after `bot_department_access`:

```sql
-- department_skill_templates: skill templates associated with a department
CREATE TABLE IF NOT EXISTS department_skill_templates (
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    template_id UUID NOT NULL REFERENCES skill_templates(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (department_id, template_id)
);

CREATE INDEX IF NOT EXISTS idx_dept_skill_templates_template ON department_skill_templates(template_id);
```

- [ ] **Step 2: Create incremental up migration**

Create `db/migrations/0036_department_templates.up.sql`:

```sql
-- Add directory templates column to departments
ALTER TABLE departments ADD COLUMN IF NOT EXISTS directory_templates JSONB NOT NULL DEFAULT '[]';

-- Department skill template associations
CREATE TABLE IF NOT EXISTS department_skill_templates (
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    template_id UUID NOT NULL REFERENCES skill_templates(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (department_id, template_id)
);

CREATE INDEX IF NOT EXISTS idx_dept_skill_templates_template ON department_skill_templates(template_id);
```

- [ ] **Step 3: Create incremental down migration**

Create `db/migrations/0036_department_templates.down.sql`:

```sql
DROP TABLE IF EXISTS department_skill_templates;
ALTER TABLE departments DROP COLUMN IF EXISTS directory_templates;
```

- [ ] **Step 4: Verify migration files exist**

```bash
ls -la db/migrations/0036_department_templates.*.sql
```

Expected: Two files listed.

### Task 2: Add sqlc queries

**Files:**
- Modify: `db/queries/departments.sql`

- [ ] **Step 1: Add queries to `db/queries/departments.sql`**

Append to the end of the file:

```sql
-- name: AddDepartmentSkillTemplate :exec
INSERT INTO department_skill_templates (department_id, template_id)
VALUES ($1, $2)
ON CONFLICT (department_id, template_id) DO NOTHING;

-- name: RemoveDepartmentSkillTemplate :exec
DELETE FROM department_skill_templates
WHERE department_id = $1 AND template_id = $2;

-- name: ListDepartmentSkillTemplates :many
SELECT st.*
FROM skill_templates st
JOIN department_skill_templates dst ON dst.template_id = st.id
WHERE dst.department_id = $1
ORDER BY st.name;

-- name: GetDepartmentDirectoryTemplates :one
SELECT directory_templates FROM departments WHERE id = $1;

-- name: UpdateDepartmentDirectoryTemplates :exec
UPDATE departments
SET directory_templates = $2, updated_at = now()
WHERE id = $1;
```

- [ ] **Step 2: Run sqlc generate**

```bash
mise run sqlc-generate
```

Expected: No errors, regenerated files in `internal/db/sqlc/`.

- [ ] **Step 3: Run database migration**

```bash
mise run db-up
```

Expected: Migration 0036 applied successfully.

- [ ] **Step 4: Commit database changes**

```bash
git add db/migrations/0036_department_templates.up.sql db/migrations/0036_department_templates.down.sql db/queries/departments.sql db/migrations/0001_init.up.sql internal/db/sqlc/
git commit -m "feat(db): add department skill templates table and directory templates column"
```

---

## Chunk 2: Go Backend — Service Layer

### Task 3: Extend DepartmentServiceInterface and DTOs

**Files:**
- Modify: `internal/handlers/departments.go` (DTOs and interface only, not handlers yet)

- [ ] **Step 1: Add new DTOs and request/response types**

Add after `CreateDepartmentRequest`:

```go
// SkillTemplateBriefDTO is a lightweight skill template reference.
type SkillTemplateBriefDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     int32  `json:"version"`
}

// AddSkillTemplateRequest is the body for associating a skill template.
type AddSkillTemplateRequest struct {
	TemplateID string `json:"template_id"`
}

// DirectoryTemplatesRequest is the body for setting directory templates.
type DirectoryTemplatesRequest struct {
	Paths []string `json:"paths"`
}

// SyncResultDTO is the response for bulk sync operations.
type SyncResultDTO struct {
	TotalBots       int              `json:"total_bots"`
	Installed       int              `json:"installed"`
	Skipped         int              `json:"skipped"`
	Errors          []BotSyncError   `json:"errors,omitempty"`
}

// BotSyncError reports a per-bot sync failure.
type BotSyncError struct {
	BotID   string `json:"bot_id"`
	Message string `json:"message"`
}
```

- [ ] **Step 2: Extend DepartmentServiceInterface**

Add new methods to the interface:

```go
type DepartmentServiceInterface interface {
	ListDepartments(ctx context.Context, botID string) ([]DepartmentDTO, error)
	CreateDepartment(ctx context.Context, botID string, req CreateDepartmentRequest) (*DepartmentDTO, error)
	// Skill template management
	AddSkillTemplate(ctx context.Context, departmentID, templateID string) error
	RemoveSkillTemplate(ctx context.Context, departmentID, templateID string) error
	ListSkillTemplates(ctx context.Context, departmentID string) ([]SkillTemplateBriefDTO, error)
	// Directory template management
	GetDirectoryTemplates(ctx context.Context, departmentID string) ([]string, error)
	UpdateDirectoryTemplates(ctx context.Context, departmentID string, paths []string) error
	// Bulk sync
	SyncSkills(ctx context.Context, departmentID string) (*SyncResultDTO, error)
	SyncDirectories(ctx context.Context, departmentID string) (*SyncResultDTO, error)
}
```

- [ ] **Step 3: Commit interface changes**

```bash
git add internal/handlers/departments.go
git commit -m "feat(handlers): extend DepartmentServiceInterface with template management"
```

### Task 4: Implement service methods

**Files:**
- Modify: `internal/enterprise/department_service.go`

- [ ] **Step 1: Write failing test for AddSkillTemplate**

Create `internal/enterprise/department_service_test.go` with a test that calls `AddSkillTemplate` and expects it to succeed. Use a test database or mock `*sqlc.Queries`.

- [ ] **Step 2: Add skill template CRUD to DepartmentService**

Add `mcp.Manager` to `DepartmentService` struct and update constructor:

```go
type DepartmentService struct {
	queries *sqlc.Queries
	logger  *slog.Logger
	manager MCPManagerInterface
}

// MCPManagerInterface for container operations needed by sync.
type MCPManagerInterface interface {
	MCPClient(ctx context.Context, botID string) (MCPClient, error)
}

// MCPClient is the subset of mcp client methods we need.
type MCPClient interface {
	Mkdir(ctx context.Context, path string) error
	WriteFile(ctx context.Context, path string, data []byte) error
}
```

Implement:

```go
func (s *DepartmentService) AddSkillTemplate(ctx context.Context, departmentID, templateID string) error {
	deptUUID, err := db.ParseUUID(departmentID)
	if err != nil { return err }
	tmplUUID, err := db.ParseUUID(templateID)
	if err != nil { return err }
	return s.queries.AddDepartmentSkillTemplate(ctx, sqlc.AddDepartmentSkillTemplateParams{
		DepartmentID: deptUUID, TemplateID: tmplUUID,
	})
}

func (s *DepartmentService) RemoveSkillTemplate(ctx context.Context, departmentID, templateID string) error {
	deptUUID, _ := db.ParseUUID(departmentID)
	tmplUUID, _ := db.ParseUUID(templateID)
	return s.queries.RemoveDepartmentSkillTemplate(ctx, sqlc.RemoveDepartmentSkillTemplateParams{
		DepartmentID: deptUUID, TemplateID: tmplUUID,
	})
}

func (s *DepartmentService) ListSkillTemplates(ctx context.Context, departmentID string) ([]handlers.SkillTemplateBriefDTO, error) {
	deptUUID, err := db.ParseUUID(departmentID)
	if err != nil { return nil, err }
	rows, err := s.queries.ListDepartmentSkillTemplates(ctx, deptUUID)
	if err != nil { return nil, err }
	out := make([]handlers.SkillTemplateBriefDTO, len(rows))
	for i, r := range rows {
		out[i] = handlers.SkillTemplateBriefDTO{
			ID: uuidToString(r.ID), Name: r.Name, Description: r.Description, Version: r.Version,
		}
	}
	return out, nil
}
```

- [ ] **Step 3: Add directory template methods**

```go
func (s *DepartmentService) GetDirectoryTemplates(ctx context.Context, departmentID string) ([]string, error) {
	deptUUID, err := db.ParseUUID(departmentID)
	if err != nil { return nil, err }
	raw, err := s.queries.GetDepartmentDirectoryTemplates(ctx, deptUUID)
	if err != nil { return nil, err }
	var paths []string
	if err := json.Unmarshal(raw, &paths); err != nil { return nil, err }
	return paths, nil
}

func (s *DepartmentService) UpdateDirectoryTemplates(ctx context.Context, departmentID string, paths []string) error {
	for _, p := range paths {
		if !strings.HasPrefix(p, "/data/") || strings.Contains(p, "..") {
			return fmt.Errorf("invalid path: %s (must start with /data/ and not contain ..)", p)
		}
	}
	deptUUID, _ := db.ParseUUID(departmentID)
	data, _ := json.Marshal(paths)
	return s.queries.UpdateDepartmentDirectoryTemplates(ctx, sqlc.UpdateDepartmentDirectoryTemplatesParams{
		ID: deptUUID, DirectoryTemplates: data,
	})
}
```

- [ ] **Step 4: Add SyncSkills method**

```go
func (s *DepartmentService) SyncSkills(ctx context.Context, departmentID string) (*handlers.SyncResultDTO, error) {
	deptUUID, err := db.ParseUUID(departmentID)
	if err != nil { return nil, err }

	// Get department skill templates
	templates, err := s.queries.ListDepartmentSkillTemplates(ctx, deptUUID)
	if err != nil { return nil, err }

	// Get all bots in department
	bots, err := s.queries.ListDepartmentBots(ctx, deptUUID)
	if err != nil { return nil, err }

	result := &handlers.SyncResultDTO{TotalBots: len(bots)}

	for _, bot := range bots {
		botIDStr := uuidToString(bot.ID)
		botUUID := bot.ID

		// Get existing installs for this bot
		installs, err := s.queries.ListBotSkillInstalls(ctx, botUUID)
		if err != nil {
			result.Errors = append(result.Errors, handlers.BotSyncError{BotID: botIDStr, Message: err.Error()})
			continue
		}

		installMap := make(map[string]bool)
		customizedMap := make(map[string]bool)
		for _, inst := range installs {
			tid := uuidToString(inst.TemplateID)
			installMap[tid] = true
			if inst.Customized { customizedMap[tid] = true }
		}

		client, err := s.manager.MCPClient(ctx, botIDStr)
		if err != nil {
			result.Errors = append(result.Errors, handlers.BotSyncError{BotID: botIDStr, Message: "container not reachable"})
			continue
		}

		for _, tmpl := range templates {
			tid := uuidToString(tmpl.ID)
			if customizedMap[tid] { result.Skipped++; continue }
			if installMap[tid] { continue }

			// Install skill into container
			parsed := parseSkillFile(tmpl.Content, tmpl.Slug)
			dirPath := path.Join("/data/.skills", parsed.Name)
			_ = client.Mkdir(ctx, dirPath)
			_ = client.WriteFile(ctx, path.Join(dirPath, "SKILL.md"), []byte(tmpl.Content))

			s.queries.CreateBotSkillInstall(ctx, sqlc.CreateBotSkillInstallParams{
				BotID: botUUID, TemplateID: tmpl.ID,
				InstalledVersion: tmpl.Version, SkillName: parsed.Name,
			})
			result.Installed++
		}
	}

	return result, nil
}
```

- [ ] **Step 5: Add SyncDirectories method**

Similar pattern: iterate bots, get MCP client, mkdir for each path.

- [ ] **Step 6: Run tests**

```bash
go test ./internal/enterprise/... -v -run TestDepartment
```

Expected: All tests pass.

- [ ] **Step 7: Commit service layer**

```bash
git add internal/enterprise/department_service.go internal/enterprise/department_service_test.go
git commit -m "feat(enterprise): implement department skill/directory template service methods"
```

---

## Chunk 3: Go Backend — Handler Layer + Route Registration

### Task 5: Add handler endpoints

**Files:**
- Modify: `internal/handlers/departments.go`

- [ ] **Step 1: Add ListSkillTemplates handler**

```go
// ListDepartmentSkillTemplates godoc
// @Summary List skill templates for a department
// @Tags departments
// @Param department_id path string true "Department ID"
// @Success 200 {array} SkillTemplateBriefDTO
// @Failure 400 {object} ErrorResponse
// @Router /departments/{department_id}/skill-templates [get]
func (h *DepartmentHandler) ListDepartmentSkillTemplates(c echo.Context) error {
	deptID := strings.TrimSpace(c.Param("department_id"))
	if deptID == "" { return echo.NewHTTPError(http.StatusBadRequest, "department_id required") }
	templates, err := h.svc.ListSkillTemplates(c.Request().Context(), deptID)
	if err != nil { return echo.NewHTTPError(http.StatusInternalServerError, err.Error()) }
	return c.JSON(http.StatusOK, templates)
}
```

- [ ] **Step 2: Add AddSkillTemplate handler**

- [ ] **Step 3: Add RemoveSkillTemplate handler**

- [ ] **Step 4: Add GetDirectoryTemplates and UpdateDirectoryTemplates handlers**

- [ ] **Step 5: Add SyncSkills and SyncDirectories handlers**

- [ ] **Step 6: Register new routes in Register method**

Update the `Register` method:

```go
func (h *DepartmentHandler) Register(e *echo.Echo) {
	// Existing bot-scoped routes
	g := e.Group("/bots/:bot_id/departments", h.middleware...)
	g.GET("", h.ListDepartments)
	g.POST("", h.CreateDepartment)

	// New department-scoped routes
	d := e.Group("/departments/:department_id", h.middleware...)
	d.GET("/skill-templates", h.ListDepartmentSkillTemplates)
	d.POST("/skill-templates", h.AddDepartmentSkillTemplate)
	d.DELETE("/skill-templates/:template_id", h.RemoveDepartmentSkillTemplate)
	d.GET("/directory-templates", h.GetDirectoryTemplates)
	d.PUT("/directory-templates", h.UpdateDirectoryTemplates)
	d.POST("/sync-skills", h.SyncSkills)
	d.POST("/sync-directories", h.SyncDirectories)
}
```

- [ ] **Step 7: Run handler tests**

```bash
go test ./internal/handlers/... -v -run TestDepartment
```

- [ ] **Step 8: Commit handler changes**

```bash
git add internal/handlers/departments.go
git commit -m "feat(handlers): add department skill/directory template endpoints"
```

### Task 6: Update DI wiring in main.go

**Files:**
- Modify: `cmd/agent/main.go`

- [ ] **Step 1: Update provideDepartmentHandler to inject manager**

```go
func provideDepartmentHandler(log *slog.Logger, queries *dbsqlc.Queries, al *enterprise.AuditLogger, resolver *authz.Resolver, manager *mcp.Manager) *handlers.DepartmentHandler {
	svc := enterprise.NewDepartmentService(log, queries, manager)
	// ... rest unchanged
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./cmd/agent/...
```

Expected: No errors.

- [ ] **Step 3: Commit DI changes**

```bash
git add cmd/agent/main.go
git commit -m "feat(main): inject mcp.Manager into department handler for sync operations"
```

---

## Chunk 4: API Spec + SDK Regeneration

### Task 7: Regenerate API spec and SDK

**Files:**
- Regenerate: `spec/swagger.json`
- Regenerate: `packages/sdk/src/`

- [ ] **Step 1: Generate swagger**

```bash
mise run swagger-generate
```

Expected: No errors, `spec/swagger.json` updated with new department endpoints.

- [ ] **Step 2: Generate SDK**

```bash
mise run sdk-generate
```

Expected: `packages/sdk/src/types.gen.ts` updated with new types.

- [ ] **Step 3: Verify SDK types exist**

Check that `packages/sdk/src/types.gen.ts` contains `SkillTemplateBriefDTO`, `DirectoryTemplatesRequest`, `SyncResultDTO`.

- [ ] **Step 4: Commit generated files**

```bash
git add spec/swagger.json packages/sdk/src/
git commit -m "chore: regenerate swagger spec and TypeScript SDK"
```

---

## Chunk 5: Frontend Extension

### Task 8: Extend department management UI

**Files:**
- Modify: `packages/web/src/pages/enterprise/departments/index.vue`

- [ ] **Step 1: Add skill template management section**

After the existing department card grid, add a department detail panel that shows:
- List of associated skill templates (fetched from `GET /departments/{id}/skill-templates`)
- "Add Skill Template" button that opens a picker dialog (list from `GET /skill-templates`)
- Remove button per template (calls `DELETE /departments/{id}/skill-templates/{tid}`)

- [ ] **Step 2: Add directory template management section**

Below skill templates:
- Editable list of directory paths
- Add path input + button
- Remove button per path
- Save button (calls `PUT /departments/{id}/directory-templates`)
- Path validation: must start with `/data/`, no `..`

- [ ] **Step 3: Add sync buttons**

Two buttons in the department detail panel:
- "Sync Skills to All Bots" → `POST /departments/{id}/sync-skills`
- "Sync Directories to All Bots" → `POST /departments/{id}/sync-directories`
- Show result toast with summary (installed, skipped, errors)

- [ ] **Step 4: Test the UI manually**

```bash
mise run //packages/web:dev
```

Navigate to the departments page, create/select a department, verify:
1. Can add/remove skill templates
2. Can set/clear directory templates
3. Sync buttons work and show results

- [ ] **Step 5: Run lint**

```bash
pnpm lint
```

Expected: No errors.

- [ ] **Step 6: Commit frontend changes**

```bash
git add packages/web/src/pages/enterprise/departments/
git commit -m "feat(web): extend department page with skill/directory template management and sync"
```

---

## Chunk 6: Integration Testing

### Task 9: Write Go integration tests

**Files:**
- Modify: `internal/db/sqlc/enterprise_integration_test.go`

- [ ] **Step 1: Add TestDepartmentSkillTemplates**

Test adding/removing/listing skill templates for a department.

- [ ] **Step 2: Add TestDepartmentDirectoryTemplates**

Test set/get directory templates with path validation.

- [ ] **Step 3: Add TestSyncSkills**

Test bulk sync: create department with templates, add bots, sync, verify `bot_skill_installs` records created. Verify customized skills are skipped.

- [ ] **Step 4: Add TestSyncDirectories**

Test bulk directory provisioning (may require mock MCP client).

- [ ] **Step 5: Run all tests**

```bash
go test ./... -v -count=1
```

Expected: All tests pass.

- [ ] **Step 6: Commit tests**

```bash
git add internal/db/sqlc/enterprise_integration_test.go
git commit -m "test: add department skill/directory template integration tests"
```
