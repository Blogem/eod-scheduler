package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/blogem/eod-scheduler/models"
	"github.com/blogem/eod-scheduler/repositories"
)

// ScheduleService interface defines schedule management business logic
type ScheduleService interface {
	GetScheduleByDateRange(from, to time.Time) ([]models.ScheduleEntry, error)
	GetDashboardData() (*DashboardData, error)
	GetWeeklySchedule(startDate time.Time) (*models.WeekView, error)
	GenerateSchedule(force bool) (*models.GenerationResult, error)
	CreateManualOverride(entryID int, form *models.ScheduleEntryForm) (*models.ScheduleEntry, error)
	UpdateScheduleEntry(id int, form *models.ScheduleEntryForm) (*models.ScheduleEntry, error)
	RemoveManualOverride(id int) error
	GetScheduleEntry(id int) (*models.ScheduleEntry, error)
	GetUpcomingEntries(days int) ([]models.ScheduleEntry, error)
	ValidateScheduleGeneration() error
}

// DashboardData represents data for the dashboard view
type DashboardData struct {
	CurrentWeek   []models.ScheduleEntry `json:"current_week"`
	NextWeeks     []models.ScheduleEntry `json:"next_weeks"`
	TeamCount     int                    `json:"team_count"`
	ActiveDays    int                    `json:"active_days"`
	LastGenerated time.Time              `json:"last_generated"`
}

// scheduleService implements ScheduleService interface
type scheduleService struct {
	scheduleRepo     repositories.ScheduleRepository
	teamRepo         repositories.TeamRepository
	workingHoursRepo repositories.WorkingHoursRepository
}

// NewScheduleService creates a new schedule service
func NewScheduleService(
	scheduleRepo repositories.ScheduleRepository,
	teamRepo repositories.TeamRepository,
	workingHoursRepo repositories.WorkingHoursRepository,
) ScheduleService {
	return &scheduleService{
		scheduleRepo:     scheduleRepo,
		teamRepo:         teamRepo,
		workingHoursRepo: workingHoursRepo,
	}
}

// GetScheduleByDateRange retrieves schedule entries for a date range
func (s *scheduleService) GetScheduleByDateRange(from, to time.Time) ([]models.ScheduleEntry, error) {
	return s.scheduleRepo.GetByDateRange(from, to)
}

// GetDashboardData retrieves data for the dashboard
func (s *scheduleService) GetDashboardData() (*DashboardData, error) {
	// Get current week (Monday to Sunday)
	currentWeek := models.GetCurrentWeek()
	currentWeekEntries, err := s.scheduleRepo.GetByDateRange(currentWeek.Start, currentWeek.End)
	if err != nil {
		return nil, fmt.Errorf("failed to get current week entries: %w", err)
	}

	// Get next 2 weeks
	nextWeekStart := currentWeek.End.AddDate(0, 0, 1)
	nextWeekEnd := nextWeekStart.AddDate(0, 0, 13) // 2 weeks
	nextWeeksEntries, err := s.scheduleRepo.GetByDateRange(nextWeekStart, nextWeekEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get next weeks entries: %w", err)
	}

	// Get team count
	teamCount, err := s.teamRepo.Count()
	if err != nil {
		return nil, fmt.Errorf("failed to get team count: %w", err)
	}

	// Get active days count
	activeDays, err := s.workingHoursRepo.GetActiveDays()
	if err != nil {
		return nil, fmt.Errorf("failed to get active days: %w", err)
	}

	// Get last generation date
	state, err := s.scheduleRepo.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule state: %w", err)
	}

	return &DashboardData{
		CurrentWeek:   currentWeekEntries,
		NextWeeks:     nextWeeksEntries,
		TeamCount:     teamCount,
		ActiveDays:    len(activeDays),
		LastGenerated: state.LastGenerationDate,
	}, nil
}

// GetWeeklySchedule retrieves schedule entries for a specific week
func (s *scheduleService) GetWeeklySchedule(startDate time.Time) (*models.WeekView, error) {
	weekRange := models.GetWeekStartingFrom(startDate)
	entries, err := s.scheduleRepo.GetByDateRange(weekRange.Start, weekRange.End)
	if err != nil {
		return nil, fmt.Errorf("failed to get weekly schedule: %w", err)
	}

	// Group entries by day
	dayMap := make(map[string][]models.ScheduleEntry)
	today := time.Now().Format("2006-01-02")

	for _, entry := range entries {
		dateStr := entry.GetFormattedDate()
		dayMap[dateStr] = append(dayMap[dateStr], entry)
	}

	// Create day views
	var days []models.DayView
	for d := 0; d < 7; d++ {
		date := weekRange.Start.AddDate(0, 0, d)
		dateStr := date.Format("2006-01-02")

		days = append(days, models.DayView{
			Date:    date,
			Entries: dayMap[dateStr],
			IsToday: dateStr == today,
		})
	}

	return &models.WeekView{
		StartDate: weekRange.Start,
		EndDate:   weekRange.End,
		Days:      days,
	}, nil
}

// GenerateSchedule generates schedule for the next 3 months
func (s *scheduleService) GenerateSchedule(force bool) (*models.GenerationResult, error) {
	// Validate that generation is possible
	if err := s.ValidateScheduleGeneration(); err != nil {
		return &models.GenerationResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Get current state
	state, err := s.scheduleRepo.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule state: %w", err)
	}

	// Check if regeneration is needed
	if !force {
		lastGen := state.LastGenerationDate
		daysSinceLastGen := int(time.Since(lastGen).Hours() / 24)
		if daysSinceLastGen < 7 { // Don't regenerate more than once per week unless forced
			return &models.GenerationResult{
				Success:           true,
				Message:           "Schedule is up to date",
				GenerationDate:    lastGen,
				NextGenerationDue: lastGen.AddDate(0, 0, 7),
			}, nil
		}
	}

	// Get active team members
	activeMembers, err := s.teamRepo.GetActiveMembers()
	if err != nil {
		return nil, fmt.Errorf("failed to get active team members: %w", err)
	}

	// Get active working days
	activeDays, err := s.workingHoursRepo.GetActiveDays()
	if err != nil {
		return nil, fmt.Errorf("failed to get active working days: %w", err)
	}

	// Delete existing future entries (non-overrides)
	today := time.Now()
	futureEnd := today.AddDate(0, 3, 0) // 3 months ahead

	// We need to be careful here - only delete non-override entries
	existingEntries, err := s.scheduleRepo.GetByDateRange(today, futureEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing entries: %w", err)
	}

	// Delete non-override entries
	for _, entry := range existingEntries {
		if !entry.IsManualOverride {
			if err := s.scheduleRepo.Delete(entry.ID); err != nil {
				return nil, fmt.Errorf("failed to delete existing entry: %w", err)
			}
		}
	}

	// Generate new schedule entries
	entriesCreated := 0
	currentMemberIndex := state.NextPersonIndex

	// Ensure the index is within bounds
	if currentMemberIndex >= len(activeMembers) {
		currentMemberIndex = 0
	}

	// Generate entries for each day in the next 3 months
	for date := today; date.Before(futureEnd); date = date.AddDate(0, 0, 1) {
		weekday := models.GetWeekdayNumber(date)

		// Check if this is a working day
		var workingHours *models.WorkingHours
		for _, wh := range activeDays {
			if wh.DayOfWeek == weekday {
				workingHours = &wh
				break
			}
		}

		// Skip non-working days
		if workingHours == nil {
			continue
		}

		// Check if there's already a manual override for this day
		existingForDay, err := s.scheduleRepo.GetByDate(date)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing entries for date: %w", err)
		}

		hasOverride := false
		for _, entry := range existingForDay {
			if entry.IsManualOverride {
				hasOverride = true
				break
			}
		}

		// Skip if there's already a manual override
		if hasOverride {
			continue
		}

		// Create schedule entry
		entry := &models.ScheduleEntry{
			Date:             date,
			TeamMemberID:     activeMembers[currentMemberIndex].ID,
			StartTime:        workingHours.StartTime,
			EndTime:          workingHours.EndTime,
			IsManualOverride: false,
		}

		if err := s.scheduleRepo.Create(entry); err != nil {
			return nil, fmt.Errorf("failed to create schedule entry: %w", err)
		}

		entriesCreated++

		// Move to next team member (round-robin)
		currentMemberIndex = (currentMemberIndex + 1) % len(activeMembers)
	}

	// Update state
	state.NextPersonIndex = currentMemberIndex
	state.LastGenerationDate = time.Now()
	if err := s.scheduleRepo.UpdateState(state); err != nil {
		return nil, fmt.Errorf("failed to update schedule state: %w", err)
	}

	return &models.GenerationResult{
		Success:           true,
		Message:           fmt.Sprintf("Successfully generated schedule with %d entries", entriesCreated),
		EntriesCreated:    entriesCreated,
		GenerationDate:    state.LastGenerationDate,
		NextGenerationDue: state.LastGenerationDate.AddDate(0, 0, 7),
	}, nil
}

// CreateManualOverride creates a manual schedule override
func (s *scheduleService) CreateManualOverride(entryID int, form *models.ScheduleEntryForm) (*models.ScheduleEntry, error) {
	// Validate form
	if errors := form.Validate(); len(errors) > 0 {
		return nil, fmt.Errorf("validation failed: %s", strings.Join(errors, ", "))
	}

	// Parse date
	date, err := models.ParseDate(form.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	// Validate team member exists
	_, err = s.teamRepo.GetByID(form.TeamMemberID)
	if err != nil {
		return nil, fmt.Errorf("team member not found: %w", err)
	}

	// Check if there's already an entry for this date
	existingEntry, err := s.scheduleRepo.GetByID(entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing entries: %w", err)
	}

	// Find the original entry (non-override) to store its team member ID
	var originalTeamMemberID *int
	if !existingEntry.IsManualOverride {
		originalTeamMemberID = &existingEntry.TeamMemberID
	}

	// Delete existing non-override entries for this date
	if !existingEntry.IsManualOverride {
		if err := s.scheduleRepo.Delete(existingEntry.ID); err != nil {
			return nil, fmt.Errorf("failed to delete existing entry: %w", err)
		}
	}

	// Create manual override entry
	entry := &models.ScheduleEntry{
		Date:                 date,
		TeamMemberID:         form.TeamMemberID,
		StartTime:            strings.TrimSpace(form.StartTime),
		EndTime:              strings.TrimSpace(form.EndTime),
		IsManualOverride:     true,
		OriginalTeamMemberID: originalTeamMemberID,
	}

	if err := s.scheduleRepo.Create(entry); err != nil {
		return nil, fmt.Errorf("failed to create manual override: %w", err)
	}

	// Get the created entry with team member info
	return s.scheduleRepo.GetByID(entry.ID)
}

// UpdateScheduleEntry updates an existing schedule entry
func (s *scheduleService) UpdateScheduleEntry(id int, form *models.ScheduleEntryForm) (*models.ScheduleEntry, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid schedule entry ID: %d", id)
	}

	// Validate form
	if errors := form.Validate(); len(errors) > 0 {
		return nil, fmt.Errorf("validation failed: %s", strings.Join(errors, ", "))
	}

	// Get existing entry
	entry, err := s.scheduleRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("schedule entry not found: %w", err)
	}

	// Parse date
	date, err := models.ParseDate(form.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	// Validate team member exists
	_, err = s.teamRepo.GetByID(form.TeamMemberID)
	if err != nil {
		return nil, fmt.Errorf("team member not found: %w", err)
	}

	// Update entry fields
	entry.Date = date
	entry.TeamMemberID = form.TeamMemberID
	entry.StartTime = strings.TrimSpace(form.StartTime)
	entry.EndTime = strings.TrimSpace(form.EndTime)
	entry.IsManualOverride = true // Any update makes it a manual override

	if err := s.scheduleRepo.Update(entry); err != nil {
		return nil, fmt.Errorf("failed to update schedule entry: %w", err)
	}

	// Get the updated entry with team member info
	return s.scheduleRepo.GetByID(entry.ID)
}

// RemoveManualOverride removes a manual override and restores the original assignment
func (s *scheduleService) RemoveManualOverride(id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid schedule entry ID: %d", id)
	}

	// Get the entry to be removed
	entry, err := s.scheduleRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("schedule entry not found: %w", err)
	}

	if !entry.IsManualOverride {
		return fmt.Errorf("can only remove manual overrides")
	}

	if entry.OriginalTeamMemberID == nil {
		return fmt.Errorf("missing original team member ID")
	}

	// Delete the override
	if err := s.scheduleRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete manual override: %w", err)
	}

	// Get working hours for this date to determine start/end times
	dayOfWeek := int(entry.Date.Weekday())
	if dayOfWeek == 0 { // Sunday is 0 in Go, but we use 6
		dayOfWeek = 6
	} else {
		dayOfWeek-- // Convert to our 0=Monday system
	}

	workingHours, err := s.workingHoursRepo.GetByDay(dayOfWeek)
	if err != nil {
		return fmt.Errorf("failed to get working hours: %w", err)
	}

	// Create the restored entry
	restoredEntry := &models.ScheduleEntry{
		Date:             entry.Date,
		TeamMemberID:     *entry.OriginalTeamMemberID,
		StartTime:        workingHours.StartTime,
		EndTime:          workingHours.EndTime,
		IsManualOverride: false,
	}

	if err := s.scheduleRepo.Create(restoredEntry); err != nil {
		return fmt.Errorf("failed to restore original assignment: %w", err)
	}

	return nil
}

// GetScheduleEntry retrieves a schedule entry by ID
func (s *scheduleService) GetScheduleEntry(id int) (*models.ScheduleEntry, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid schedule entry ID: %d", id)
	}
	return s.scheduleRepo.GetByID(id)
}

// ValidateScheduleGeneration checks if schedule generation is possible
func (s *scheduleService) ValidateScheduleGeneration() error {
	// Check if there are active team members
	activeMembers, err := s.teamRepo.GetActiveMembers()
	if err != nil {
		return fmt.Errorf("failed to get active team members: %w", err)
	}

	if len(activeMembers) == 0 {
		return fmt.Errorf("no active team members found. Add team members before generating schedule")
	}

	// Check if there are active working days
	activeDays, err := s.workingHoursRepo.GetActiveDays()
	if err != nil {
		return fmt.Errorf("failed to get active working days: %w", err)
	}

	if len(activeDays) == 0 {
		return fmt.Errorf("no active working days found. Configure working hours before generating schedule")
	}

	return nil
}

// GetUpcomingEntries returns schedule entries for the next N days
func (s *scheduleService) GetUpcomingEntries(days int) ([]models.ScheduleEntry, error) {
	from := time.Now()
	to := from.AddDate(0, 0, days)

	return s.scheduleRepo.GetByDateRange(from, to)
}
