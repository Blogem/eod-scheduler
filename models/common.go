package models

import (
	"time"
)

// AuditFields contains common audit tracking fields
type AuditFields struct {
	CreatedBy  string     `json:"created_by,omitempty"`
	ModifiedBy string     `json:"modified_by,omitempty"`
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
}

// Common validation functions and utilities used across models

// FlashMessage represents a flash message for user feedback
type FlashMessage struct {
	Type    string `json:"type"` // "success", "error", "warning", "info"
	Message string `json:"message"`
}

// PageData represents common data passed to templates
type PageData struct {
	Title        string        `json:"title"`
	CurrentPage  string        `json:"current_page"`
	FlashMessage *FlashMessage `json:"flash_message,omitempty"`
	Data         interface{}   `json:"data,omitempty"`
}

// DateRange represents a range of dates
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// GetNext3Months returns a date range for the next 3 months
func GetNext3Months() DateRange {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 3, 0)
	return DateRange{Start: start, End: end}
}

// GetCurrentWeek returns a date range for the current week (Monday to Sunday)
func GetCurrentWeek() DateRange {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 { // Sunday
		weekday = 7
	}

	// Calculate days since Monday
	daysSinceMonday := weekday - 1

	// Get Monday of current week
	monday := now.AddDate(0, 0, -daysSinceMonday)
	start := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())

	// Get Sunday of current week
	end := start.AddDate(0, 0, 6)

	return DateRange{Start: start, End: end}
}

// GetWeekStartingFrom returns a date range for a week starting from the given date
func GetWeekStartingFrom(date time.Time) DateRange {
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	end := start.AddDate(0, 0, 6)
	return DateRange{Start: start, End: end}
}

// FormatDate formats a time as YYYY-MM-DD
func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// FormatDateTime formats a time as YYYY-MM-DD HH:MM
func FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

// ParseDate parses a YYYY-MM-DD string into a time.Time
func ParseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

// IsWeekend checks if a given time is a weekend (Saturday or Sunday)
func IsWeekend(t time.Time) bool {
	weekday := t.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

// GetWeekdayNumber returns the weekday as a number (0=Monday, 6=Sunday)
func GetWeekdayNumber(t time.Time) int {
	weekday := int(t.Weekday())
	if weekday == 0 { // Sunday
		return 6
	}
	return weekday - 1 // Monday=1 becomes 0, Tuesday=2 becomes 1, etc.
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

// HasErrors returns true if there are validation errors
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// GetMessages returns all error messages as a slice of strings
func (ve ValidationErrors) GetMessages() []string {
	messages := make([]string, len(ve))
	for i, err := range ve {
		messages[i] = err.Message
	}
	return messages
}
