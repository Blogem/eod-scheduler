package models

// WorkingHours represents working hours configuration for a day of the week
type WorkingHours struct {
	ID          int    `json:"id" db:"id"`
	DayOfWeek   int    `json:"day_of_week" db:"day_of_week"` // 0=Monday, 6=Sunday
	StartTime   string `json:"start_time" db:"start_time"`   // "09:00" format
	EndTime     string `json:"end_time" db:"end_time"`       // "17:00" format
	Active      bool   `json:"active" db:"active"`
	AuditFields        // Embedded audit fields
}

// WorkingHoursForm represents form data for updating working hours
type WorkingHoursForm struct {
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Active    bool   `json:"active"`
}

// DayNames maps day numbers to readable names
var DayNames = map[int]string{
	0: "Monday",
	1: "Tuesday",
	2: "Wednesday",
	3: "Thursday",
	4: "Friday",
	5: "Saturday",
	6: "Sunday",
}

// GetDayName returns the readable name for a day of week
func (w *WorkingHours) GetDayName() string {
	if name, ok := DayNames[w.DayOfWeek]; ok {
		return name
	}
	return "Unknown"
}

// Validate validates the working hours form data
func (f *WorkingHoursForm) Validate() []string {
	var errors []string

	// Validate day of week
	if f.DayOfWeek < 0 || f.DayOfWeek > 6 {
		errors = append(errors, "Day of week must be between 0 (Monday) and 6 (Sunday)")
	}

	// Only validate times if the day is active
	if f.Active {
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
	}

	return errors
}

// isValidTimeFormat validates HH:MM format
func isValidTimeFormat(timeStr string) bool {
	if len(timeStr) != 5 {
		return false
	}

	if timeStr[2] != ':' {
		return false
	}

	// Parse hours
	hours := timeStr[0:2]
	if !isNumeric(hours) {
		return false
	}
	h := parseNumber(hours)
	if h < 0 || h > 23 {
		return false
	}

	// Parse minutes
	minutes := timeStr[3:5]
	if !isNumeric(minutes) {
		return false
	}
	m := parseNumber(minutes)
	if m < 0 || m > 59 {
		return false
	}

	return true
}

// isStartBeforeEnd checks if start time is before end time
func isStartBeforeEnd(start, end string) bool {
	startMinutes := timeToMinutes(start)
	endMinutes := timeToMinutes(end)
	return startMinutes < endMinutes
}

// timeToMinutes converts HH:MM to total minutes
func timeToMinutes(timeStr string) int {
	hours := parseNumber(timeStr[0:2])
	minutes := parseNumber(timeStr[3:5])
	return hours*60 + minutes
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	for _, char := range s {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// parseNumber converts a numeric string to int (assumes valid input)
func parseNumber(s string) int {
	result := 0
	for _, char := range s {
		result = result*10 + int(char-'0')
	}
	return result
}
