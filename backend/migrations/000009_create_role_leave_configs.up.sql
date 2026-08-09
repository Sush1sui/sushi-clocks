CREATE TABLE IF NOT EXISTS role_leave_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES company_roles(id) ON DELETE CASCADE,
    leave_type_id UUID NOT NULL REFERENCES leave_types(id) ON DELETE CASCADE,
    max_days INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_role_leave_configs UNIQUE (role_id, leave_type_id)
);

CREATE INDEX IF NOT EXISTS idx_role_leave_configs_leave_type_id ON role_leave_configs(leave_type_id);
