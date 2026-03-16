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
