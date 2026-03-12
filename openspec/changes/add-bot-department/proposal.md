## Why

Department management already exists (migration 0026): `departments` table with hierarchy, `department_members` for user membership, and `bot_department_access` for bot-department association. RBAC already uses departments for access control.

However, there is no way to leverage departments for **operational management** of bots — specifically, batch-deploying skills and enforcing uniform directory structures across all bots in a department. Administrators must still set up each bot individually, which is repetitive and error-prone when managing dozens of bots in the same team.

This change extends the existing department system with skill template associations and directory templates, so departments become not just an access control mechanism but also an operational management unit.

## What Changes

- Add a `department_skill_templates` join table to associate skill templates with departments
- Add a `directory_templates` JSONB column (or separate table) to departments for defining standard directory structures
- When a bot is added to a department (via existing `bot_department_access`), optionally trigger skill installation and directory provisioning
- Provide API endpoints to bulk-sync skills and directories to all bots in a department
- Extend the existing department UI to manage skill templates and directory templates
- Extend the existing department detail page to show sync actions

## Capabilities

### New Capabilities
- `department-skill-sync`: Associating skill templates with a department and bulk-installing/syncing them across all department bots
- `department-directory-templates`: Defining directory structure templates per department and provisioning them into bot containers on association or on-demand sync

### Modified Capabilities

## Impact

- **Database**: New `department_skill_templates` table, possible new column on `departments` for directory templates; existing tables unchanged
- **Go Backend**: Extend `internal/handlers/departments.go` and `internal/enterprise/department_service.go` with skill/directory template management and sync endpoints
- **API**: New endpoints under existing `/bots/{bot_id}/departments` or new `/departments/{department_id}/...` routes
- **Frontend**: Extend existing department management page with skill template and directory template management sections, plus sync buttons
- **Agent Gateway**: No changes expected
