-- name: CreateDepartment :one
INSERT INTO departments (name, description, parent_id, metadata)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetDepartmentByID :one
SELECT * FROM departments WHERE id = $1;

-- name: GetDepartmentByName :one
SELECT * FROM departments WHERE name = $1;

-- name: ListDepartments :many
SELECT * FROM departments ORDER BY name;

-- name: ListChildDepartments :many
SELECT * FROM departments WHERE parent_id = $1 ORDER BY name;

-- name: UpdateDepartment :one
UPDATE departments
SET name = $2, description = $3, parent_id = $4, metadata = $5, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteDepartment :exec
DELETE FROM departments WHERE id = $1;

-- name: AddDepartmentMember :exec
INSERT INTO department_members (department_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (department_id, user_id) DO UPDATE SET role = $3;

-- name: RemoveDepartmentMember :exec
DELETE FROM department_members WHERE department_id = $1 AND user_id = $2;

-- name: ListDepartmentMembers :many
SELECT dm.*, u.username, u.display_name, u.role AS user_role
FROM department_members dm
JOIN users u ON u.id = dm.user_id
WHERE dm.department_id = $1
ORDER BY dm.created_at;

-- name: ListUserDepartments :many
SELECT d.*, dm.role AS member_role
FROM departments d
JOIN department_members dm ON dm.department_id = d.id
WHERE dm.user_id = $1
ORDER BY d.name;

-- name: SetBotDepartmentAccess :exec
INSERT INTO bot_department_access (bot_id, department_id)
VALUES ($1, $2)
ON CONFLICT (bot_id, department_id) DO NOTHING;

-- name: RemoveBotDepartmentAccess :exec
DELETE FROM bot_department_access WHERE bot_id = $1 AND department_id = $2;

-- name: ListBotDepartments :many
SELECT d.*
FROM departments d
JOIN bot_department_access bda ON bda.department_id = d.id
WHERE bda.bot_id = $1
ORDER BY d.name;

-- name: ListDepartmentBots :many
SELECT b.id, b.owner_user_id, b.type, b.display_name, b.avatar_url, b.is_active, b.status, b.metadata, b.created_at, b.updated_at
FROM bots b
JOIN bot_department_access bda ON bda.bot_id = b.id
WHERE bda.department_id = $1
ORDER BY b.display_name;

-- name: CheckBotDepartmentAccess :one
SELECT EXISTS(
    SELECT 1 FROM bot_department_access
    WHERE bot_id = $1 AND department_id = $2
) AS has_access;

-- name: CheckUserDepartmentMembership :one
SELECT EXISTS(
    SELECT 1 FROM department_members
    WHERE department_id = $1 AND user_id = $2
) AS is_member;

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
