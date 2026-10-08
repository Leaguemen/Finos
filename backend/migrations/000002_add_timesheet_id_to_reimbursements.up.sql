ALTER TABLE reimbursements
ADD COLUMN timesheet_id BIGINT NOT NULL
REFERENCES timesheets(id)
ON DELETE CASCADE;

CREATE INDEX idx_reimbursements_timesheet_id
ON reimbursements(timesheet_id);