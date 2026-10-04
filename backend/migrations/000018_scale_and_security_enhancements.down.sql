ALTER TABLE users DROP COLUMN IF EXISTS receive_audit_archive;
DROP INDEX IF EXISTS idx_timesheets_company_clock_in;
DROP INDEX IF EXISTS idx_timesheets_active_user;
