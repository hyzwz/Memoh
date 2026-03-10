-- name: CreateSkillTemplate :one
INSERT INTO skill_templates (
  name, slug, description, category, content, author, icon, tags, is_published
)
VALUES (
  sqlc.arg(name),
  sqlc.arg(slug),
  sqlc.arg(description),
  sqlc.arg(category),
  sqlc.arg(content),
  sqlc.arg(author),
  sqlc.arg(icon),
  sqlc.arg(tags),
  sqlc.arg(is_published)
)
RETURNING *;

-- name: GetSkillTemplateByID :one
SELECT * FROM skill_templates WHERE id = sqlc.arg(id);

-- name: GetSkillTemplateBySlug :one
SELECT * FROM skill_templates WHERE slug = sqlc.arg(slug);

-- name: ListSkillTemplates :many
SELECT * FROM skill_templates ORDER BY install_count DESC, created_at DESC;

-- name: ListPublishedSkillTemplates :many
SELECT * FROM skill_templates WHERE is_published = true ORDER BY install_count DESC, created_at DESC;

-- name: ListSkillTemplatesByCategory :many
SELECT * FROM skill_templates WHERE is_published = true AND category = sqlc.arg(category) ORDER BY install_count DESC, created_at DESC;

-- name: UpdateSkillTemplate :one
UPDATE skill_templates
SET
  name = sqlc.arg(name),
  slug = sqlc.arg(slug),
  description = sqlc.arg(description),
  category = sqlc.arg(category),
  content = sqlc.arg(content),
  author = sqlc.arg(author),
  icon = sqlc.arg(icon),
  tags = sqlc.arg(tags),
  is_published = sqlc.arg(is_published),
  version = version + 1,
  updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteSkillTemplate :exec
DELETE FROM skill_templates WHERE id = sqlc.arg(id);

-- name: IncrementInstallCount :exec
UPDATE skill_templates SET install_count = install_count + 1, updated_at = now() WHERE id = sqlc.arg(id);

-- name: DecrementInstallCount :exec
UPDATE skill_templates SET install_count = GREATEST(install_count - 1, 0), updated_at = now() WHERE id = sqlc.arg(id);

-- name: CountSkillTemplates :one
SELECT count(*) FROM skill_templates;

-- name: CreateBotSkillInstall :one
INSERT INTO bot_skill_installs (
  bot_id, template_id, installed_version, skill_name
)
VALUES (
  sqlc.arg(bot_id),
  sqlc.arg(template_id),
  sqlc.arg(installed_version),
  sqlc.arg(skill_name)
)
RETURNING *;

-- name: ListBotSkillInstalls :many
SELECT
  bsi.*,
  st.version AS latest_version,
  st.name AS template_name
FROM bot_skill_installs bsi
LEFT JOIN skill_templates st ON st.id = bsi.template_id
WHERE bsi.bot_id = sqlc.arg(bot_id)
ORDER BY bsi.created_at DESC;

-- name: GetBotSkillInstall :one
SELECT * FROM bot_skill_installs WHERE bot_id = sqlc.arg(bot_id) AND template_id = sqlc.arg(template_id);

-- name: GetBotSkillInstallBySkillName :one
SELECT * FROM bot_skill_installs WHERE bot_id = sqlc.arg(bot_id) AND skill_name = sqlc.arg(skill_name);

-- name: DeleteBotSkillInstall :exec
DELETE FROM bot_skill_installs WHERE bot_id = sqlc.arg(bot_id) AND template_id = sqlc.arg(template_id);

-- name: UpdateBotSkillInstallVersion :one
UPDATE bot_skill_installs
SET installed_version = sqlc.arg(installed_version), customized = false, updated_at = now()
WHERE bot_id = sqlc.arg(bot_id) AND template_id = sqlc.arg(template_id)
RETURNING *;

-- name: MarkBotSkillCustomized :exec
UPDATE bot_skill_installs
SET customized = true, updated_at = now()
WHERE bot_id = sqlc.arg(bot_id) AND skill_name = sqlc.arg(skill_name);
