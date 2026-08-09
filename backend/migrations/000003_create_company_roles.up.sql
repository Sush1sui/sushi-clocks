CREATE TABLE IF NOT EXISTS company_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    default_wage DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    wage_type VARCHAR(20) NOT NULL CHECK (wage_type IN ('hourly', 'daily', 'monthly')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_company_roles_company_id ON company_roles(company_id);
