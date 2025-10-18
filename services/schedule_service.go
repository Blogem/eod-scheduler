package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/blogem/eod-scheduler/models"
	"github.com/blogem/eod-scheduler/repositories"
)

var timeNow = func() time.Time {
	return time.Now()
}

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
	today := timeNow().Format("2006-01-02")

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
	if err := s.validateScheduleGeneration(); err != nil {
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
	if !force && s.isScheduleUpToDate(state) {
		return s.createUpToDateResult(state), nil
	}

	// Get required data for generation
	activeMembers, activeDays, err := s.getGenerationData()
	if err != nil {
		return nil, err
	}

	// Clean up existing entries and prepare for new generation
	if err := s.cleanupExistingEntries(); err != nil {
		return nil, err
	}

	// Generate new schedule entries
	entriesCreated, err := s.generateScheduleEntries(activeMembers, activeDays)
	if err != nil {
		return nil, err
	}

	// Update state and return result
	return s.finalizeGeneration(state, entriesCreated)
}

// isScheduleUpToDate checks if the schedule was generated recently
func (s *scheduleService) isScheduleUpToDate(state *models.ScheduleState) bool {
	daysSinceLastGen := int(time.Since(state.LastGenerationDate).Hours() / 24)
	return daysSinceLastGen < 7 // Don't regenerate more than once per week unless forced
}

// createUpToDateResult creates a result indicating the schedule is current
func (s *scheduleService) createUpToDateResult(state *models.ScheduleState) *models.GenerationResult {
	return &models.GenerationResult{
		Success:           true,
		Message:           "Schedule is up to date",
		GenerationDate:    state.LastGenerationDate,
		NextGenerationDue: state.LastGenerationDate.AddDate(0, 0, 7),
	}
}

// getGenerationData retrieves active members and working days needed for generation
func (s *scheduleService) getGenerationData() ([]models.TeamMember, []models.WorkingHours, error) {
	activeMembers, err := s.teamRepo.GetActiveMembers()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get active team members: %w", err)
	}

	activeDays, err := s.workingHoursRepo.GetActiveDays()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get active working days: %w", err)
	}

	return activeMembers, activeDays, nil
}

// cleanupExistingEntries removes non-override entries from the future period
func (s *scheduleService) cleanupExistingEntries() error {
	today := timeNow()
	// Always start cleanup from tomorrow to never delete today's entry
	startDate := today.AddDate(0, 0, 1)
	futureEnd := today.AddDate(0, 3, 0) // 3 months ahead

	existingEntries, err := s.scheduleRepo.GetByDateRange(startDate, futureEnd)
	if err != nil {
		return fmt.Errorf("failed to get existing entries: %w", err)
	}

	// Delete only non-override entries to preserve manual changes
	for _, entry := range existingEntries {
		if !entry.IsManualOverride {
			if err := s.scheduleRepo.Delete(entry.ID); err != nil {
				return fmt.Errorf("failed to delete existing entry: %w", err)
			}
		}
	}

	return nil
}

// generateScheduleEntries creates new schedule entries using deterministic assignment
func (s *scheduleService) generateScheduleEntries(activeMembers []models.TeamMember, activeDays []models.WorkingHours) (int, error) {
	workingDates, err := s.collectWorkingDates(activeDays)
	if err != nil {
		return 0, err
	}

	entriesCreated := 0
	for _, workingDate := range workingDates {
		entry := s.createScheduleEntry(workingDate, activeMembers, activeDays)

		if err := s.scheduleRepo.Create(entry); err != nil {
			return 0, fmt.Errorf("failed to create schedule entry: %w", err)
		}

		entriesCreated++
	}

	return entriesCreated, nil
}

// WorkingDate represents a date with its associated working hours
type WorkingDate struct {
	Date         time.Time
	WorkingHours models.WorkingHours
}

// collectWorkingDates finds all working dates in the generation period that don't have overrides
func (s *scheduleService) collectWorkingDates(activeDays []models.WorkingHours) ([]WorkingDate, error) {
	today := timeNow()

	// Check if today has any schedule entries
	todayEntries, err := s.scheduleRepo.GetByDate(today)
	if err != nil {
		return nil, fmt.Errorf("failed to check today's entries: %w", err)
	}

	// Start from tomorrow if today has entries, otherwise from today
	startDate := today.AddDate(0, 0, 1)
	if len(todayEntries) == 0 {
		startDate = today
	}

	futureEnd := today.AddDate(0, 3, 0) // 3 months ahead
	var workingDates []WorkingDate

	for date := startDate; date.Before(futureEnd); date = date.AddDate(0, 0, 1) {
		weekday := models.GetWeekdayNumber(date)

		// Find working hours for this day of week
		workingHours := s.findWorkingHoursForDay(activeDays, weekday)
		if workingHours == nil {
			continue // Skip non-working days
		}

		// Skip dates that already have manual overrides
		if hasOverride, err := s.hasManualOverride(date); err != nil {
			return nil, fmt.Errorf("failed to check existing entries for date: %w", err)
		} else if hasOverride {
			continue
		}

		workingDates = append(workingDates, WorkingDate{
			Date:         date,
			WorkingHours: *workingHours,
		})
	}

	return workingDates, nil
}

// findWorkingHoursForDay finds the working hours configuration for a specific weekday
func (s *scheduleService) findWorkingHoursForDay(activeDays []models.WorkingHours, weekday int) *models.WorkingHours {
	for _, wh := range activeDays {
		if wh.DayOfWeek == weekday {
			return &wh
		}
	}
	return nil
}

// hasManualOverride checks if a date already has a manual override entry
func (s *scheduleService) hasManualOverride(date time.Time) (bool, error) {
	existingForDay, err := s.scheduleRepo.GetByDate(date)
	if err != nil {
		return false, err
	}

	for _, entry := range existingForDay {
		if entry.IsManualOverride {
			return true, nil
		}
	}

	return false, nil
}

// createScheduleEntry creates a schedule entry with deterministic team member assignment based on working day sequence
func (s *scheduleService) createScheduleEntry(workingDate WorkingDate, activeMembers []models.TeamMember, activeDays []models.WorkingHours) *models.ScheduleEntry {
	// Calculate deterministic assignment based on working days since epoch for this specific date
	// This maintains determinism (same date always gets same assignment) while avoiding consecutive assignments
	workingDaysSinceEpoch := s.calculateWorkingDaysSinceEpoch(workingDate.Date, activeDays)
	memberIndex := workingDaysSinceEpoch % len(activeMembers)

	return &models.ScheduleEntry{
		Date:             workingDate.Date,
		TeamMemberID:     activeMembers[memberIndex].ID,
		StartTime:        workingDate.WorkingHours.StartTime,
		EndTime:          workingDate.WorkingHours.EndTime,
		IsManualOverride: false,
	}
}

// calculateWorkingDaysSinceEpoch calculates how many working days have passed since a fixed epoch
// using the actual configured working days. This ensures deterministic assignments
// while preventing consecutive assignments due to non-working days.
func (s *scheduleService) calculateWorkingDaysSinceEpoch(date time.Time, activeDays []models.WorkingHours) int {
	// Use a fixed epoch date that's a Monday to make calculation easier
	epoch := time.Date(2000, 1, 3, 0, 0, 0, 0, time.UTC) // Monday, January 3, 2000

	if date.Before(epoch) {
		return 0
	}

	// Create a map of active days for fast lookup
	// Convert from our DayOfWeek format (0=Monday) to Go's time.Weekday format (1=Monday, 0=Sunday)
	activeWeekdays := make(map[time.Weekday]bool)
	for _, workingHours := range activeDays {
		if workingHours.Active {
			// Convert from our format (0=Monday, 1=Tuesday, ..., 6=Sunday)
			// to Go's format (0=Sunday, 1=Monday, ..., 6=Saturday)
			goWeekday := time.Weekday((workingHours.DayOfWeek + 1) % 7)
			activeWeekdays[goWeekday] = true
		}
	}

	// Count working days by iterating through each day since epoch
	workingDays := 0
	for d := epoch; d.Before(date); d = d.AddDate(0, 0, 1) {
		if activeWeekdays[d.Weekday()] {
			workingDays++
		}
	}

	return workingDays
}

// finalizeGeneration updates the state and creates the final result
func (s *scheduleService) finalizeGeneration(state *models.ScheduleState, entriesCreated int) (*models.GenerationResult, error) {
	// Update state
	state.LastGenerationDate = timeNow()
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

// Helper functions for shared logic between CreateManualOverride and UpdateScheduleEntry

// validateFormAndTeamMember validates the form and checks if the team member exists
func (s *scheduleService) validateFormAndTeamMember(form *models.ScheduleEntryForm) error {
	// Validate form
	if errors := form.Validate(); len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, ", "))
	}

	// Validate team member exists
	_, err := s.teamRepo.GetByID(form.TeamMemberID)
	if err != nil {
		return fmt.Errorf("team member not found: %w", err)
	}

	return nil
}

// parseDateFromForm parses and validates the date from the form
func (s *scheduleService) parseDateFromForm(form *models.ScheduleEntryForm) (time.Time, error) {
	date, err := models.ParseDate(form.Date)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format: %w", err)
	}
	return date, nil
}

// getExistingEntryWithValidation retrieves an existing entry by ID with validation
func (s *scheduleService) getExistingEntryWithValidation(id int) (*models.ScheduleEntry, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid schedule entry ID: %d", id)
	}

	entry, err := s.scheduleRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("schedule entry not found: %w", err)
	}

	return entry, nil
}

// CreateManualOverride creates a manual schedule override
func (s *scheduleService) CreateManualOverride(entryID int, form *models.ScheduleEntryForm) (*models.ScheduleEntry, error) {
	if err := s.validateFormAndTeamMember(form); err != nil {
		return nil, err
	}

	date, err := s.parseDateFromForm(form)
	if err != nil {
		return nil, err
	}

	existingEntry, err := s.getExistingEntryWithValidation(entryID)
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
	if err := s.validateFormAndTeamMember(form); err != nil {
		return nil, err
	}

	entry, err := s.getExistingEntryWithValidation(id)
	if err != nil {
		return nil, err
	}

	date, err := s.parseDateFromForm(form)
	if err != nil {
		return nil, err
	}

	isManualOverride := entry.TeamMemberID != form.TeamMemberID

	// Update entry fields
	if isManualOverride && entry.OriginalTeamMemberID == nil {
		// Store original team member ID if this is now a manual override
		entry.OriginalTeamMemberID = &entry.TeamMemberID
	}
	entry.Date = date
	entry.TeamMemberID = form.TeamMemberID
	entry.StartTime = strings.TrimSpace(form.StartTime)
	entry.EndTime = strings.TrimSpace(form.EndTime)
	entry.IsManualOverride = isManualOverride

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

// validateScheduleGeneration checks if schedule generation is possible
func (s *scheduleService) validateScheduleGeneration() error {
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
