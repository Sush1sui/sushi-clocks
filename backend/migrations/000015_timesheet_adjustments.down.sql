ALTER TABLE timesheets DROP CONSTRAINT IF EXISTS timesheets_status_check;
ALTER TABLE timesheets ADD CONSTRAINT timesheets_status_check CHECK (status IN ('active', 'completed', 'flagged_for_review'));

ALTER TABLE timesheets DROP COLUMN IF EXISTS reviewed_at;
ALTER TABLE timesheets DROP COLUMN IF EXISTS reviewed_by;
ALTER TABLE timesheets DROP COLUMN IF EXISTS adjustment_reason;
