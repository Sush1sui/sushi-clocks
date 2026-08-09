CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    mobile_number VARCHAR(20),
    system_role VARCHAR(20) NOT NULL DEFAULT 'employee' CHECK (system_role IN ('admin', 'hr', 'employee')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_users_company_email UNIQUE (company_id, email)
);

CREATE INDEX IF NOT EXISTS idx_users_company_id ON users(company_id);
