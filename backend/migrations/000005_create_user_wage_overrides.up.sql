CREATE TABLE IF NOT EXISTS user_wage_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    override_wage DECIMAL(10,2) NOT NULL,
    wage_type VARCHAR(20) NOT NULL CHECK (wage_type IN ('hourly', 'daily', 'monthly')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
