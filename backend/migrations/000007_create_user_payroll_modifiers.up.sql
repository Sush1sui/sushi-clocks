CREATE TABLE IF NOT EXISTS user_payroll_modifiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    modifier_id UUID NOT NULL REFERENCES payroll_modifiers(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_user_payroll_modifiers UNIQUE (user_id, modifier_id)
);

CREATE INDEX IF NOT EXISTS idx_user_payroll_modifiers_modifier_id ON user_payroll_modifiers(modifier_id);
