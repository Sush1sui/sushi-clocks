DROP INDEX IF EXISTS idx_leave_requests_user_dates;
DROP INDEX IF EXISTS idx_leave_requests_company_id;

ALTER TABLE leave_types
DROP COLUMN IF EXISTS max_carryover_days,
DROP COLUMN IF EXISTS allow_carryover;

ALTER TABLE companies
DROP COLUMN IF EXISTS max_carryover_days,
DROP COLUMN IF EXISTS allow_leave_carryover,
DROP COLUMN IF EXISTS leave_reset_day,
DROP COLUMN IF EXISTS leave_reset_month;

ALTER TABLE leave_requests
DROP COLUMN IF EXISTS review_notes,
DROP COLUMN IF EXISTS reviewed_at,
DROP COLUMN IF EXISTS reason,
DROP COLUMN IF EXISTS company_id;
