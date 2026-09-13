ALTER TABLE timesheets DROP CONSTRAINT IF EXISTS timesheets_status_check;
ALTER TABLE timesheets ADD CONSTRAINT timesheets_status_check CHECK (status IN ('active', 'completed', 'flagged_for_review', 'rejected'));

ALTER TABLE timesheets ADD COLUMN IF NOT EXISTS adjustment_reason TEXT;
ALTER TABLE timesheets ADD COLUMN IF NOT EXISTS reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE timesheets ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ;
