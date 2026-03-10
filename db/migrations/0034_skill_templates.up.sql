-- skill_templates: global admin-managed skill template library
CREATE TABLE IF NOT EXISTS skill_templates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  slug TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  category TEXT NOT NULL DEFAULT 'general',
  content TEXT NOT NULL,
  version INTEGER NOT NULL DEFAULT 1,
  author TEXT NOT NULL DEFAULT '',
  icon TEXT NOT NULL DEFAULT '',
  tags TEXT[] NOT NULL DEFAULT '{}',
  is_published BOOLEAN NOT NULL DEFAULT false,
  install_count INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT skill_templates_slug_unique UNIQUE (slug)
);

CREATE INDEX IF NOT EXISTS idx_skill_templates_category ON skill_templates(category);
CREATE INDEX IF NOT EXISTS idx_skill_templates_published ON skill_templates(is_published) WHERE is_published = true;

-- bot_skill_installs: tracks which templates are installed per bot
CREATE TABLE IF NOT EXISTS bot_skill_installs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
  template_id UUID REFERENCES skill_templates(id) ON DELETE SET NULL,
  installed_version INTEGER NOT NULL,
  skill_name TEXT NOT NULL,
  customized BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT bot_skill_installs_bot_template UNIQUE (bot_id, template_id)
);

CREATE INDEX IF NOT EXISTS idx_bot_skill_installs_bot_id ON bot_skill_installs(bot_id);
