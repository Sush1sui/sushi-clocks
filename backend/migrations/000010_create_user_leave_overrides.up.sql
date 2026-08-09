CREATE TABLE IF NOT EXISTS user_leave_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    leave_type_id UUID NOT NULL REFERENCES leave_types(id) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    max_days INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_user_leave_overrides UNIQUE (user_id, leave_type_id)
);

CREATE INDEX IF NOT EXISTS idx_user_leave_overrides_leave_type_id ON user_leave_overrides(leave_type_id);
