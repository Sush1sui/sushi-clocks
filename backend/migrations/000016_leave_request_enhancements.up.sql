-- 1. Add company_id to leave_requests if not exists
ALTER TABLE leave_requests 
ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;

-- Backfill company_id from users table for any existing records
UPDATE leave_requests lr
SET company_id = u.company_id
FROM users u
WHERE lr.user_id = u.id AND lr.company_id IS NULL;

-- Make company_id NOT NULL after backfill
ALTER TABLE leave_requests
ALTER COLUMN company_id SET NOT NULL;

-- 2. Add reason, reviewed_at, and review_notes to leave_requests
ALTER TABLE leave_requests
ADD COLUMN IF NOT EXISTS reason TEXT,
ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS review_notes TEXT;

-- 3. Indexes for fast tenant-scoped queries and overlap checks
CREATE INDEX IF NOT EXISTS idx_leave_requests_company_id ON leave_requests(company_id);
CREATE INDEX IF NOT EXISTS idx_leave_requests_user_dates ON leave_requests(user_id, start_date, end_date);

-- 4. Add dynamic leave policy configuration columns to companies
ALTER TABLE companies
ADD COLUMN IF NOT EXISTS leave_reset_month INT NOT NULL DEFAULT 1,
ADD COLUMN IF NOT EXISTS leave_reset_day INT NOT NULL DEFAULT 1,
ADD COLUMN IF NOT EXISTS allow_leave_carryover BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS max_carryover_days INT NOT NULL DEFAULT 0;

-- 5. Add per-leave-type carryover override columns to leave_types
ALTER TABLE leave_types
ADD COLUMN IF NOT EXISTS allow_carryover BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS max_carryover_days INT NOT NULL DEFAULT 0;
