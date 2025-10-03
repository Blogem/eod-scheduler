package models

import (
	"time"
)

// TeamMember represents a team member in the EoD scheduler
type TeamMember struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	SlackHandle string    `json:"slack_handle" db:"slack_handle"`
	Active      bool      `json:"active" db:"active"`
	DateAdded   time.Time `json:"date_added" db:"date_added"`
}

// TeamMemberForm represents form data for creating/updating team members
type TeamMemberForm struct {
	Name        string `json:"name"`
	SlackHandle string `json:"slack_handle"`
	Active      bool   `json:"active"`
}

// Validate validates the team member form data
func (f *TeamMemberForm) Validate() []string {
	var errors []string

	if f.Name == "" {
		errors = append(errors, "Name is required")
	}

	if len(f.Name) > 100 {
		errors = append(errors, "Name must be less than 100 characters")
	}

	if f.SlackHandle != "" && len(f.SlackHandle) > 255 {
		errors = append(errors, "Slack handle must be less than 255 characters")
	}

	// Basic slack handle validation
	if f.SlackHandle != "" && !isValidSlackHandle(f.SlackHandle) {
		errors = append(errors, "Slack handle format is invalid (should start with @)")
	}

	return errors
}

// isValidSlackHandle performs basic slack handle validation
func isValidSlackHandle(handle string) bool {
	// Simple validation: must start with @ and be at least 2 characters
	if len(handle) < 2 {
		return false
	}

	if handle[0] != '@' {
		return false
	}

	// Check that the rest contains only valid characters (alphanumeric, dots, hyphens, underscores)
	for i := 1; i < len(handle); i++ {
		c := handle[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '-' || c == '_') {
			return false
		}
	}

	return true
}
