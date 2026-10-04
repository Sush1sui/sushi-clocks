-- 1. Prevent duplicate active shifts at the database engine level (race condition guard)
CREATE UNIQUE INDEX IF NOT EXISTS idx_timesheets_active_user 
ON timesheets (user_id) 
WHERE status = 'active';

-- 2. Composite index for fast payroll & roster date range scans
CREATE INDEX IF NOT EXISTS idx_timesheets_company_clock_in 
ON timesheets (company_id, clock_in_time);

-- 3. Add flag for receiving yearly audit archive emails (optional toggle for HR, admin default)
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS receive_audit_archive BOOLEAN NOT NULL DEFAULT FALSE;
