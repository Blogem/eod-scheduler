package models

import (
	"time"
)

// TeamMember represents a team member in the EoD scheduler
type TeamMember struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Email     string    `json:"email" db:"email"`
	Active    bool      `json:"active" db:"active"`
	DateAdded time.Time `json:"date_added" db:"date_added"`
}

// TeamMemberForm represents form data for creating/updating team members
type TeamMemberForm struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Active bool   `json:"active"`
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

	if f.Email != "" && len(f.Email) > 255 {
		errors = append(errors, "Email must be less than 255 characters")
	}

	// Basic email validation (simple regex would be overkill for this simple app)
	if f.Email != "" && !isValidEmail(f.Email) {
		errors = append(errors, "Email format is invalid")
	}

	return errors
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	// Simple validation: must contain @ and at least one dot after @
	atIndex := -1
	for i, char := range email {
		if char == '@' {
			if atIndex != -1 {
				return false // Multiple @ symbols
			}
			atIndex = i
		}
	}

	if atIndex == -1 || atIndex == 0 || atIndex == len(email)-1 {
		return false // No @, or @ at start/end
	}

	// Check for dot after @
	for i := atIndex + 1; i < len(email); i++ {
		if email[i] == '.' && i < len(email)-1 {
			return true
		}
	}

	return false
}
