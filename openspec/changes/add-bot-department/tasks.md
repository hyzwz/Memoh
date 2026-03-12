## 1. Database Schema

- [ ] 1.1 Add `department_skill_templates` table to `0001_init.up.sql` (department_id FK, template_id FK, created_at, PK on department_id+template_id)
- [ ] 1.2 Add `directory_templates JSONB NOT NULL DEFAULT '[]'` column to existing `departments` table in `0001_init.up.sql`
- [ ] 1.3 Create incremental migration `0027_department_templates.up.sql` and `.down.sql` with the same changes as diffs
- [ ] 1.4 Add sqlc queries in `db/queries/departments.sql`: AddDepartmentSkillTemplate, RemoveDepartmentSkillTemplate, ListDepartmentSkillTemplates, GetDepartmentDirectoryTemplates, UpdateDepartmentDirectoryTemplates
- [ ] 1.5 Run `mise run sqlc-generate` and `mise run db-up`

## 2. Go Backend — Extend Department Service

- [ ] 2.1 Add skill template management methods to `internal/enterprise/department_service.go` (AddSkillTemplate, RemoveSkillTemplate, ListSkillTemplates)
- [ ] 2.2 Add directory template methods (GetDirectoryTemplates, UpdateDirectoryTemplates with path validation)
- [ ] 2.3 Add SyncSkills method: iterate bots via ListDepartmentBots, install missing skills using existing skill install logic, skip customized, collect results
- [ ] 2.4 Add SyncDirectories method: iterate bots, create missing directories via gRPC, collect results
- [ ] 2.5 Add OnBotAddedToDepartment hook: install skills + provision directories for a single bot when associated with a department

## 3. Go Backend — Extend Department Handler

- [ ] 3.1 Add skill template endpoints to `internal/handlers/departments.go`: POST/DELETE/GET `/departments/{department_id}/skill-templates` with Swagger annotations
- [ ] 3.2 Add directory template endpoints: PUT/GET `/departments/{department_id}/directory-templates`
- [ ] 3.3 Add sync endpoints: POST `/departments/{department_id}/sync-skills`, POST `/departments/{department_id}/sync-directories`
- [ ] 3.4 Register new routes in the Echo router (cmd/agent/main.go)

## 4. Go Backend — Integration with Bot-Department Association

- [ ] 4.1 Modify the SetBotDepartmentAccess flow to trigger OnBotAddedToDepartment (install department skills + provision directories)

## 5. API Spec & SDK

- [ ] 5.1 Run `mise run swagger-generate` to regenerate spec/swagger.json
- [ ] 5.2 Run `mise run sdk-generate` to regenerate packages/sdk/

## 6. Frontend — Extend Department Management

- [ ] 6.1 Add skill template management section to department detail page (list, add from template picker, remove)
- [ ] 6.2 Add directory template management section (editable path list, save)
- [ ] 6.3 Add sync buttons (sync skills, sync directories) with progress/result display
- [ ] 6.4 Show sync results summary (bots processed, skills installed, skipped, errors)

## 7. Testing

- [ ] 7.1 Write Go tests for department skill template CRUD
- [ ] 7.2 Write Go tests for directory template CRUD with path validation
- [ ] 7.3 Write Go tests for bulk sync-skills (including customized skip logic)
- [ ] 7.4 Write Go tests for bulk sync-directories
- [ ] 7.5 Write Go tests for OnBotAddedToDepartment auto-provisioning
