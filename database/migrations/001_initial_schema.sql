-- Initial schema for EoD Scheduler
-- Creates all required tables with proper relationships

CREATE TABLE team_members (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    active BOOLEAN DEFAULT 1,
    date_added DATE DEFAULT CURRENT_DATE
);

CREATE TABLE working_hours (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    day_of_week INTEGER NOT NULL CHECK (day_of_week >= 0 AND day_of_week <= 6), -- 0=Monday, 6=Sunday
    start_time TEXT NOT NULL, -- "09:00" format
    end_time TEXT NOT NULL,   -- "17:00" format
    active BOOLEAN DEFAULT 1
);

CREATE TABLE schedule_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date DATE NOT NULL,
    team_member_id INTEGER,
    start_time TEXT NOT NULL,
    end_time TEXT NOT NULL,
    is_manual_override BOOLEAN DEFAULT 0,
    original_team_member_id INTEGER REFERENCES team_members(id),
    FOREIGN KEY (team_member_id) REFERENCES team_members(id) ON DELETE CASCADE
);

CREATE TABLE schedule_state (
    id INTEGER PRIMARY KEY CHECK (id = 1), -- Ensure only one record
    last_generation_date DATE
);

-- Create indexes for better performance
CREATE INDEX idx_schedule_entries_date ON schedule_entries(date);
CREATE INDEX idx_schedule_entries_team_member ON schedule_entries(team_member_id);
CREATE INDEX idx_working_hours_day ON working_hours(day_of_week);
CREATE UNIQUE INDEX idx_working_hours_day_unique ON working_hours(day_of_week) WHERE active = 1;

-- Insert default working hours (Monday-Friday, 9-5)
INSERT INTO working_hours (day_of_week, start_time, end_time, active) VALUES
(0, '09:00', '17:00', 1), -- Monday
(1, '09:00', '17:00', 1), -- Tuesday
(2, '09:00', '17:00', 1), -- Wednesday
(3, '09:00', '17:00', 1), -- Thursday
(4, '09:00', '17:00', 1), -- Friday
(5, '00:00', '00:00', 0), -- Saturday (inactive)
(6, '00:00', '00:00', 0); -- Sunday (inactive)

-- Initialize schedule state
INSERT INTO schedule_state (id, last_generation_date) VALUES
(1, date('now'));