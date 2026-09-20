ALTER TABLE companies
ADD COLUMN IF NOT EXISTS proration_basis VARCHAR(20) NOT NULL DEFAULT 'calendar_days'
    CHECK (proration_basis IN ('calendar_days', 'working_days', 'fixed_22', 'fixed_26'));

ALTER TABLE companies
ADD COLUMN IF NOT EXISTS work_days_mask INT NOT NULL DEFAULT 62
    CHECK (work_days_mask BETWEEN 1 AND 127);
