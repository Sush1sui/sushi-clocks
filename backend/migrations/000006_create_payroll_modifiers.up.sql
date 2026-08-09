CREATE TABLE IF NOT EXISTS payroll_modifiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    modifier_type VARCHAR(20) NOT NULL CHECK (modifier_type IN ('addition', 'deduction')),
    calculation_method VARCHAR(20) NOT NULL CHECK (calculation_method IN ('fixed', 'percentage')),
    value DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_payroll_modifiers_company_id ON payroll_modifiers(company_id);
