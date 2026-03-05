-- 0026_enterprise_features.up.sql
-- Enterprise features: RBAC departments, audit logs, model routing, cost tracking, hands, cockpit

-- F1: RBAC - Extend user roles
DO $$ BEGIN
    ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'dept_admin';
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
    ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'org_admin';
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
    ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'super_admin';
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- F1: Departments
CREATE TABLE IF NOT EXISTS departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    parent_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS department_members (
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (department_id, user_id)
);

CREATE TABLE IF NOT EXISTS bot_department_access (
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (bot_id, department_id)
);

CREATE TABLE IF NOT EXISTS bot_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    resource TEXT NOT NULL CHECK (resource IN ('chat', 'tools', 'skills', 'settings', 'members', 'files')),
    action TEXT NOT NULL CHECK (action IN ('read', 'write', 'execute', 'admin')),
    allowed BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (bot_id, role, resource, action)
);

-- F1: Missing indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_departments_parent_id ON departments(parent_id);
CREATE INDEX IF NOT EXISTS idx_department_members_user_id ON department_members(user_id);
CREATE INDEX IF NOT EXISTS idx_bot_department_access_department_id ON bot_department_access(department_id);

-- F2: Audit logs
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    bot_id UUID REFERENCES bots(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    detail JSONB NOT NULL DEFAULT '{}',
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_bot_id ON audit_logs(bot_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- F3: Model routing
CREATE TABLE IF NOT EXISTS bot_model_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    model_id UUID NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    priority INT NOT NULL DEFAULT 0,
    complexity_tier TEXT NOT NULL CHECK (complexity_tier IN ('simple', 'medium', 'complex', 'fallback')),
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (bot_id, name)
);

CREATE INDEX IF NOT EXISTS idx_bot_model_routes_tier ON bot_model_routes(bot_id, complexity_tier, is_enabled, priority DESC);

-- F4: Cost tracking
CREATE TABLE IF NOT EXISTS model_pricing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id UUID NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    input_price_per_mtok NUMERIC(10, 4) NOT NULL DEFAULT 0,
    output_price_per_mtok NUMERIC(10, 4) NOT NULL DEFAULT 0,
    cache_read_price_per_mtok NUMERIC(10, 4) NOT NULL DEFAULT 0,
    cache_write_price_per_mtok NUMERIC(10, 4) NOT NULL DEFAULT 0,
    reasoning_price_per_mtok NUMERIC(10, 4) NOT NULL DEFAULT 0,
    effective_from TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (model_id, effective_from)
);

CREATE TABLE IF NOT EXISTS budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope_type TEXT NOT NULL CHECK (scope_type IN ('system', 'department', 'bot', 'user')),
    scope_id TEXT NOT NULL,
    period TEXT NOT NULL CHECK (period IN ('daily', 'weekly', 'monthly')),
    limit_amount NUMERIC(10, 2) NOT NULL,
    alert_threshold NUMERIC(3, 2) NOT NULL DEFAULT 0.80,
    action_on_exceed TEXT NOT NULL DEFAULT 'warn' CHECK (action_on_exceed IN ('warn', 'block', 'downgrade')),
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (scope_type, scope_id, period)
);

-- F5: Hands
CREATE TABLE IF NOT EXISTS hands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'hand',
    schedule_pattern TEXT,
    schedule_timezone TEXT DEFAULT 'UTC',
    schedule_max_calls INT,
    triggers JSONB NOT NULL DEFAULT '[]',
    allowed_tools JSONB NOT NULL DEFAULT '[]',
    model_config JSONB NOT NULL DEFAULT '{}',
    output_config JSONB NOT NULL DEFAULT '{}',
    permissions JSONB NOT NULL DEFAULT '{}',
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (bot_id, name)
);

CREATE TABLE IF NOT EXISTS hand_execution_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hand_id UUID NOT NULL REFERENCES hands(id) ON DELETE CASCADE,
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    trigger_type TEXT NOT NULL,
    trigger_detail JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL CHECK (status IN ('running', 'completed', 'failed', 'cancelled', 'approval_pending')),
    result_text TEXT,
    error_message TEXT,
    usage JSONB,
    model_id UUID REFERENCES models(id),
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_hand_execution_logs_hand_id ON hand_execution_logs(hand_id);
CREATE INDEX IF NOT EXISTS idx_hand_execution_logs_status ON hand_execution_logs(status);

-- F5: Trigger to ensure hand_execution_logs.bot_id matches hands.bot_id
CREATE OR REPLACE FUNCTION check_hand_bot_consistency() RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM hands WHERE id = NEW.hand_id AND bot_id = NEW.bot_id) THEN
        RAISE EXCEPTION 'hand_execution_logs.bot_id (%) does not match hands.bot_id for hand %', NEW.bot_id, NEW.hand_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_hand_execution_bot_consistency
    BEFORE INSERT OR UPDATE ON hand_execution_logs
    FOR EACH ROW EXECUTE FUNCTION check_hand_bot_consistency();

-- F7: Cockpit
-- Use NOT NULL user_id with a sentinel UUID for system-generated reports to avoid NULL unique constraint issues.
CREATE TABLE IF NOT EXISTS cockpit_daily_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    user_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    hand_execution_id UUID REFERENCES hand_execution_logs(id) ON DELETE SET NULL,
    report_date DATE NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    tasks JSONB NOT NULL DEFAULT '[]',
    pending_tasks JSONB NOT NULL DEFAULT '[]',
    total_estimated_saved_hours NUMERIC(6, 2) NOT NULL DEFAULT 0,
    total_ai_time_minutes INT NOT NULL DEFAULT 0,
    efficiency_multiplier NUMERIC(4, 1) NOT NULL DEFAULT 0,
    innovation_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (bot_id, user_id, report_date)
);

CREATE INDEX IF NOT EXISTS idx_cockpit_daily_reports_bot_date ON cockpit_daily_reports(bot_id, report_date DESC);
CREATE INDEX IF NOT EXISTS idx_cockpit_daily_reports_user_date ON cockpit_daily_reports(user_id, report_date DESC);

CREATE TABLE IF NOT EXISTS cockpit_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT NOT NULL UNIQUE,
    value JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO cockpit_config (key, value) VALUES
    ('efficiency', '{"labor_cost_hourly_cny": 200, "categories": ["document","analysis","research","content","decision_support","communication","meeting_prep","code","other"]}'),
    ('display', '{"default_period_days": 7, "innovation_highlight_count": 5}')
ON CONFLICT (key) DO NOTHING;
