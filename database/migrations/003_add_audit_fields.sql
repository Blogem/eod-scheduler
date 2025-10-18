-- Add audit fields to team_members
ALTER TABLE team_members ADD COLUMN created_by TEXT DEFAULT 'system';
ALTER TABLE team_members ADD COLUMN modified_by TEXT;
ALTER TABLE team_members ADD COLUMN modified_at DATETIME;

-- Add audit fields to schedule_entries
ALTER TABLE schedule_entries ADD COLUMN created_by TEXT DEFAULT 'system';
ALTER TABLE schedule_entries ADD COLUMN modified_by TEXT;
ALTER TABLE schedule_entries ADD COLUMN modified_at DATETIME;
ALTER TABLE schedule_entries ADD COLUMN takeover_reason TEXT;

-- Add audit fields to working_hours
ALTER TABLE working_hours ADD COLUMN created_by TEXT DEFAULT 'system';
ALTER TABLE working_hours ADD COLUMN modified_by TEXT;
ALTER TABLE working_hours ADD COLUMN modified_at DATETIME;
