package models

import (
	"testing"
	"time"
)

// Test TeamMemberForm validation
func TestTeamMemberFormValidation(t *testing.T) {
	// Test valid form
	validForm := TeamMemberForm{
		Name:  "John Doe",
		Email: "john@example.com",
	}
	errors := validForm.Validate()
	if len(errors) != 0 {
		t.Errorf("Expected no errors for valid form, got: %v", errors)
	}

	// Test invalid form
	invalidForm := TeamMemberForm{
		Name:  "", // Empty name
		Email: "invalid-email",
	}
	errors = invalidForm.Validate()
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors for invalid form, got: %v", errors)
	}
}

// Test WorkingHoursForm validation
func TestWorkingHoursFormValidation(t *testing.T) {
	// Test valid form
	validForm := WorkingHoursForm{
		DayOfWeek: 0, // Monday
		StartTime: "09:00",
		EndTime:   "17:00",
		Active:    true,
	}
	errors := validForm.Validate()
	if len(errors) != 0 {
		t.Errorf("Expected no errors for valid form, got: %v", errors)
	}

	// Test invalid form
	invalidForm := WorkingHoursForm{
		DayOfWeek: 8,       // Invalid day
		StartTime: "25:00", // Invalid time
		EndTime:   "08:00", // End before start
		Active:    true,
	}
	errors = invalidForm.Validate()
	if len(errors) < 2 {
		t.Errorf("Expected at least 2 errors for invalid form, got: %v", errors)
	}
}

// Test time validation functions
func TestTimeValidation(t *testing.T) {
	// Test valid times
	validTimes := []string{"00:00", "09:00", "17:30", "23:59"}
	for _, timeStr := range validTimes {
		if !isValidTimeFormat(timeStr) {
			t.Errorf("Expected %s to be valid time format", timeStr)
		}
	}

	// Test invalid times
	invalidTimes := []string{"", "9:00", "25:00", "12:60", "ab:cd", "12:3"}
	for _, timeStr := range invalidTimes {
		if isValidTimeFormat(timeStr) {
			t.Errorf("Expected %s to be invalid time format", timeStr)
		}
	}

	// Test start before end
	if !isStartBeforeEnd("09:00", "17:00") {
		t.Error("Expected 09:00 to be before 17:00")
	}

	if isStartBeforeEnd("17:00", "09:00") {
		t.Error("Expected 17:00 not to be before 09:00")
	}
}

// Test date utilities
func TestDateUtilities(t *testing.T) {
	now := time.Now()

	// Test date range functions
	currentWeek := GetCurrentWeek()
	if currentWeek.Start.After(now) {
		t.Error("Current week start should not be after now")
	}

	next3Months := GetNext3Months()
	if next3Months.End.Before(now.AddDate(0, 2, 0)) {
		t.Error("Next 3 months should extend at least 2 months from now")
	}

	// Test weekday number conversion
	monday := time.Date(2025, 10, 6, 0, 0, 0, 0, time.UTC) // This is a Monday
	if GetWeekdayNumber(monday) != 0 {
		t.Errorf("Expected Monday to be weekday 0, got %d", GetWeekdayNumber(monday))
	}

	sunday := time.Date(2025, 10, 5, 0, 0, 0, 0, time.UTC) // This is a Sunday
	if GetWeekdayNumber(sunday) != 6 {
		t.Errorf("Expected Sunday to be weekday 6, got %d", GetWeekdayNumber(sunday))
	}
}
