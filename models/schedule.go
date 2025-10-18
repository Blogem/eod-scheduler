package models

import (
	"time"
)

// ScheduleEntry represents a single schedule entry
type ScheduleEntry struct {
	ID                   int       `json:"id" db:"id"`
	Date                 time.Time `json:"date" db:"date"`
	TeamMemberID         int       `json:"team_member_id" db:"team_member_id"`
	StartTime            string    `json:"start_time" db:"start_time"`
	EndTime              string    `json:"end_time" db:"end_time"`
	IsManualOverride     bool      `json:"is_manual_override" db:"is_manual_override"`
	OriginalTeamMemberID *int      `json:"original_team_member_id,omitempty" db:"original_team_member_id"`
	TakeoverReason       string    `json:"takeover_reason,omitempty" db:"takeover_reason"`

	// Joined fields (populated from joins with team_members table)
	TeamMemberName        string `json:"team_member_name,omitempty" db:"team_member_name"`
	TeamMemberSlackHandle string `json:"team_member_slack_handle,omitempty" db:"team_member_slack_handle"`

	AuditFields // Embedded audit fields
}

// ScheduleState represents the current state of schedule generation
type ScheduleState struct {
	ID                 int       `json:"id" db:"id"`
	LastGenerationDate time.Time `json:"last_generation_date" db:"last_generation_date"`
}

// ScheduleEntryForm represents form data for manual overrides
type ScheduleEntryForm struct {
	Date         string `json:"date"` // "2025-10-01" format
	TeamMemberID int    `json:"team_member_id"`
	StartTime    string `json:"start_time"` // "09:00" format
	EndTime      string `json:"end_time"`   // "17:00" format
}

// WeekView represents a week's worth of schedule entries for display
type WeekView struct {
	StartDate time.Time
	EndDate   time.Time
	Days      []DayView
}

// DayView represents a single day in the week view
type DayView struct {
	Date    time.Time
	Entries []ScheduleEntry
	IsToday bool
}

// GetFormattedDate returns the date in YYYY-MM-DD format
func (s *ScheduleEntry) GetFormattedDate() string {
	return s.Date.Format("2006-01-02")
}

// GetWeekday returns the weekday name
func (s *ScheduleEntry) GetWeekday() string {
	return s.Date.Weekday().String()
}

// IsToday checks if the entry is for today
func (s *ScheduleEntry) IsToday() bool {
	today := time.Now().Format("2006-01-02")
	entryDate := s.Date.Format("2006-01-02")
	return today == entryDate
}

// IsPast checks if the entry is in the past
func (s *ScheduleEntry) IsPast() bool {
	today := time.Now()
	return s.Date.Before(today)
}

// IsFuture checks if the entry is in the future
func (s *ScheduleEntry) IsFuture() bool {
	today := time.Now()
	return s.Date.After(today)
}

// Validate validates the schedule entry form data
func (f *ScheduleEntryForm) Validate() []string {
	var errors []string

	// Validate date format
	if f.Date == "" {
		errors = append(errors, "Date is required")
	} else {
		if _, err := time.Parse("2006-01-02", f.Date); err != nil {
			errors = append(errors, "Date must be in YYYY-MM-DD format")
		}
	}

	// Validate team member ID
	if f.TeamMemberID <= 0 {
		errors = append(errors, "Team member must be selected")
	}

	// Validate times using the same validation as working hours
	if !isValidTimeFormat(f.StartTime) {
		errors = append(errors, "Start time must be in HH:MM format (e.g., 09:00)")
	}

	if !isValidTimeFormat(f.EndTime) {
		errors = append(errors, "End time must be in HH:MM format (e.g., 17:00)")
	}

	// Check that start time is before end time
	if isValidTimeFormat(f.StartTime) && isValidTimeFormat(f.EndTime) {
		if !isStartBeforeEnd(f.StartTime, f.EndTime) {
			errors = append(errors, "Start time must be before end time")
		}
	}

	return errors
}

// GenerationRequest represents a request to generate the schedule
type GenerationRequest struct {
	Force bool `json:"force"` // Force regeneration even if already up to date
}

// GenerationResult represents the result of schedule generation
type GenerationResult struct {
	Success           bool      `json:"success"`
	Message           string    `json:"message"`
	EntriesCreated    int       `json:"entries_created"`
	GenerationDate    time.Time `json:"generation_date"`
	NextGenerationDue time.Time `json:"next_generation_due"`
}

// TakeoverForm represents form data for taking over a shift
type TakeoverForm struct {
	ScheduleEntryID int    `json:"schedule_entry_id"`
	NewTeamMemberID int    `json:"new_team_member_id"`
	Reason          string `json:"reason"`
}

// Validate validates the takeover form data
func (f *TakeoverForm) Validate() []string {
	var errors []string

	if f.ScheduleEntryID <= 0 {
		errors = append(errors, "Please select a schedule entry to take over")
	}

	if f.NewTeamMemberID <= 0 {
		errors = append(errors, "Please select a team member to take over the shift")
	}

	return errors
}
