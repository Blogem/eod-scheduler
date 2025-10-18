package services

import (
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/blogem/eod-scheduler/models"
	"github.com/blogem/eod-scheduler/repositories/mocks"
)

// GenerateScheduleTestSuite is a test suite for the GenerateSchedule method
type GenerateScheduleTestSuite struct {
	suite.Suite
	service          ScheduleService
	mockScheduleRepo *mocks.MockScheduleRepository
	mockTeamRepo     *mocks.MockTeamRepository
	mockWorkingRepo  *mocks.MockWorkingHoursRepository
}

// SetupTest sets up the test suite before each test
func (suite *GenerateScheduleTestSuite) SetupTest() {
	suite.mockScheduleRepo = mocks.NewMockScheduleRepository(suite.T())
	suite.mockTeamRepo = mocks.NewMockTeamRepository(suite.T())
	suite.mockWorkingRepo = mocks.NewMockWorkingHoursRepository(suite.T())

	suite.service = NewScheduleService(
		suite.mockScheduleRepo,
		suite.mockTeamRepo,
		suite.mockWorkingRepo,
	)
}

// TestGenerateSchedule_ValidationFailure_NoActiveMembers tests validation failure when no active team members exist
func (suite *GenerateScheduleTestSuite) TestGenerateSchedule_ValidationFailure_NoActiveMembers() {
	// Setup: No active team members
	suite.mockTeamRepo.EXPECT().GetActiveMembers().Return([]models.TeamMember{}, nil)

	// Act
	result, err := suite.service.GenerateSchedule(false)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.False(suite.T(), result.Success)
	assert.Contains(suite.T(), result.Message, "no active team members found")
}

// TestGenerateSchedule_ValidationFailure_NoActiveDays tests validation failure when no active working days exist
func (suite *GenerateScheduleTestSuite) TestGenerateSchedule_ValidationFailure_NoActiveDays() {
	// Setup: Active team members but no working days
	activeMembers := []models.TeamMember{
		{ID: 1, Name: "John Doe", Active: true},
	}
	suite.mockTeamRepo.EXPECT().GetActiveMembers().Return(activeMembers, nil)
	suite.mockWorkingRepo.EXPECT().GetActiveDays().Return([]models.WorkingHours{}, nil)

	// Act
	result, err := suite.service.GenerateSchedule(false)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.False(suite.T(), result.Success)
	assert.Contains(suite.T(), result.Message, "no active working days found")
}

// TestGenerateSchedule_ValidationFailure_RepositoryError tests error handling when repository fails
func (suite *GenerateScheduleTestSuite) TestGenerateSchedule_ValidationFailure_RepositoryError() {
	// Setup: Repository error
	expectedError := errors.New("database connection failed")
	suite.mockTeamRepo.EXPECT().GetActiveMembers().Return(nil, expectedError)

	// Act
	result, err := suite.service.GenerateSchedule(false)

	// Assert - The validation failure returns a GenerationResult with Success=false, not an error
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.False(suite.T(), result.Success)
	assert.Contains(suite.T(), result.Message, "failed to get active team members")
}

// TestGenerateSchedule_UpToDate_RecentGeneration tests that schedule generation is skipped when recently generated
func (suite *GenerateScheduleTestSuite) TestGenerateSchedule_UpToDate_RecentGeneration() {
	// Setup: Recent generation (3 days ago, less than 7)
	recentDate := time.Now().AddDate(0, 0, -3)
	scheduleState := &models.ScheduleState{
		ID:                 1,
		LastGenerationDate: recentDate,
	}

	activeMembers := []models.TeamMember{
		{ID: 1, Name: "John Doe", Active: true},
		{ID: 2, Name: "Jane Smith", Active: true},
	}
	activeDays := []models.WorkingHours{
		{ID: 1, DayOfWeek: 0, StartTime: "09:00", EndTime: "17:00", Active: true}, // Monday
		{ID: 2, DayOfWeek: 1, StartTime: "09:00", EndTime: "17:00", Active: true}, // Tuesday
	}

	suite.mockTeamRepo.EXPECT().GetActiveMembers().Return(activeMembers, nil)
	suite.mockWorkingRepo.EXPECT().GetActiveDays().Return(activeDays, nil)
	suite.mockScheduleRepo.EXPECT().GetState().Return(scheduleState, nil)

	// Act
	result, err := suite.service.GenerateSchedule(false)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.True(suite.T(), result.Success)
	assert.Equal(suite.T(), "Schedule is up to date", result.Message)
	assert.Equal(suite.T(), recentDate, result.GenerationDate)
	assert.Equal(suite.T(), recentDate.AddDate(0, 0, 7), result.NextGenerationDue)
}

// TestGenerateSchedule_ForceGeneration tests forced schedule generation even when recently generated
func (suite *GenerateScheduleTestSuite) TestGenerateSchedule_ForceGeneration() {
	// Setup: Recent generation but force=true
	recentDate := time.Now().AddDate(0, 0, -3)
	scheduleState := &models.ScheduleState{
		ID:                 1,
		LastGenerationDate: recentDate,
	}

	activeMembers := []models.TeamMember{
		{ID: 1, Name: "John Doe", Active: true},
		{ID: 2, Name: "Jane Smith", Active: true},
	}
	activeDays := []models.WorkingHours{
		{ID: 1, DayOfWeek: 0, StartTime: "09:00", EndTime: "17:00", Active: true}, // Monday
	}

	// Mock expectations for successful generation
	suite.mockTeamRepo.EXPECT().GetActiveMembers().Return(activeMembers, nil)
	suite.mockWorkingRepo.EXPECT().GetActiveDays().Return(activeDays, nil)
	suite.mockScheduleRepo.EXPECT().GetState().Return(scheduleState, nil)

	// Mock getting existing entries and deleting non-overrides
	today := time.Now()
	tomorrow := today.AddDate(0, 0, 1)
	futureEnd := today.AddDate(0, 3, 0)
	existingEntries := []models.ScheduleEntry{
		{ID: 1, Date: today.AddDate(0, 0, 1), IsManualOverride: false},
		{ID: 2, Date: today.AddDate(0, 0, 2), IsManualOverride: true}, // Should not be deleted
	}
	suite.mockScheduleRepo.EXPECT().GetByDateRange(mock.MatchedBy(func(from time.Time) bool {
		// Cleanup always starts from tomorrow
		return from.Year() == tomorrow.Year() && from.Month() == tomorrow.Month() && from.Day() == tomorrow.Day()
	}), mock.MatchedBy(func(to time.Time) bool {
		return to.Year() == futureEnd.Year() && to.Month() == futureEnd.Month()
	})).Return(existingEntries, nil)

	// Expect deletion of non-override entries only
	suite.mockScheduleRepo.EXPECT().Delete(1).Return(nil) // Delete non-override entry
	// ID 2 (override) should NOT be deleted

	// Mock creation of new entries - we'll expect at least one Monday in the next 3 months
	suite.mockScheduleRepo.EXPECT().GetByDate(mock.AnythingOfType("time.Time")).Return([]models.ScheduleEntry{}, nil).Maybe()
	suite.mockScheduleRepo.EXPECT().Create(mock.AnythingOfType("*models.ScheduleEntry")).Return(nil).Maybe()

	// Mock state update
	suite.mockScheduleRepo.EXPECT().UpdateState(mock.MatchedBy(func(state *models.ScheduleState) bool {
		return state.ID == 1
	})).Return(nil)

	// Act
	result, err := suite.service.GenerateSchedule(true)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.True(suite.T(), result.Success)
	assert.Contains(suite.T(), result.Message, "Successfully generated schedule")
}

// TestGenerateSchedule_SuccessfulGeneration tests successful schedule generation
func (suite *GenerateScheduleTestSuite) TestGenerateSchedule_SuccessfulGeneration() {
	// Setup: Old generation (8 days ago, more than 7)
	oldDate := time.Now().AddDate(0, 0, -8)
	scheduleState := &models.ScheduleState{
		ID:                 1,
		LastGenerationDate: oldDate,
	}

	activeMembers := []models.TeamMember{
		{ID: 1, Name: "John Doe", Active: true},
		{ID: 2, Name: "Jane Smith", Active: true},
		{ID: 3, Name: "Bob Wilson", Active: true},
	}
	activeDays := []models.WorkingHours{
		{ID: 1, DayOfWeek: 0, StartTime: "09:00", EndTime: "17:00", Active: true}, // Monday
		{ID: 2, DayOfWeek: 2, StartTime: "10:00", EndTime: "18:00", Active: true}, // Wednesday
		{ID: 3, DayOfWeek: 4, StartTime: "09:00", EndTime: "17:00", Active: true}, // Friday
	}

	// Mock expectations
	suite.mockTeamRepo.EXPECT().GetActiveMembers().Return(activeMembers, nil)
	suite.mockWorkingRepo.EXPECT().GetActiveDays().Return(activeDays, nil)
	suite.mockScheduleRepo.EXPECT().GetState().Return(scheduleState, nil)

	// Mock getting existing entries (empty for simplicity)
	suite.mockScheduleRepo.EXPECT().GetByDateRange(mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.ScheduleEntry{}, nil)

	// Mock entry creation - expect multiple calls for different days
	suite.mockScheduleRepo.EXPECT().GetByDate(mock.AnythingOfType("time.Time")).Return([]models.ScheduleEntry{}, nil).Maybe()
	suite.mockScheduleRepo.EXPECT().Create(mock.AnythingOfType("*models.ScheduleEntry")).Return(nil).Maybe()

	// Mock state update
	suite.mockScheduleRepo.EXPECT().UpdateState(mock.MatchedBy(func(state *models.ScheduleState) bool {
		return state.ID == 1
	})).Return(nil)

	// Act
	result, err := suite.service.GenerateSchedule(false)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.True(suite.T(), result.Success)
	assert.Contains(suite.T(), result.Message, "Successfully generated schedule")
	assert.True(suite.T(), result.EntriesCreated >= 0)
	assert.Equal(suite.T(), result.NextGenerationDue, result.GenerationDate.AddDate(0, 0, 7))
}

// TestGenerateSchedule_RoundRobinLogic tests the deterministic round-robin assignment logic
func (suite *GenerateScheduleTestSuite) TestGenerateSchedule_RoundRobinLogic() {
	// Setup: Old generation to force regeneration
	scheduleState := &models.ScheduleState{
		ID:                 1,
		LastGenerationDate: time.Now().AddDate(0, 0, -8),
	}

	activeMembers := []models.TeamMember{
		{ID: 10, Name: "John Doe", Active: true},
		{ID: 20, Name: "Jane Smith", Active: true},
		{ID: 30, Name: "Bob Wilson", Active: true},
	}
	activeDays := []models.WorkingHours{
		{ID: 1, DayOfWeek: 0, StartTime: "09:00", EndTime: "17:00", Active: true}, // Monday only
	}

	suite.mockTeamRepo.EXPECT().GetActiveMembers().Return(activeMembers, nil)
	suite.mockWorkingRepo.EXPECT().GetActiveDays().Return(activeDays, nil)
	suite.mockScheduleRepo.EXPECT().GetState().Return(scheduleState, nil)
	suite.mockScheduleRepo.EXPECT().GetByDateRange(mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.ScheduleEntry{}, nil)

	// Track the order of team member assignments
	var assignedMembers []int
	suite.mockScheduleRepo.EXPECT().GetByDate(mock.AnythingOfType("time.Time")).Return([]models.ScheduleEntry{}, nil).Maybe()
	suite.mockScheduleRepo.EXPECT().Create(mock.MatchedBy(func(entry *models.ScheduleEntry) bool {
		assignedMembers = append(assignedMembers, entry.TeamMemberID)
		return entry.TeamMemberID == 20 || entry.TeamMemberID == 30 || entry.TeamMemberID == 10 // Expect round-robin
	})).Return(nil).Maybe()

	// Mock state update - expect the index to wrap around correctly
	suite.mockScheduleRepo.EXPECT().UpdateState(mock.AnythingOfType("*models.ScheduleState")).Return(nil)

	// Act
	result, err := suite.service.GenerateSchedule(false)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.True(suite.T(), result.Success)

	// Verify that round-robin assignment works (the exact order depends on the deterministic calculation)
	// The key is that we should see different people assigned in a round-robin pattern
	if len(assignedMembers) >= 3 {
		// Check that we're doing round-robin - each person appears multiple times in sequence
		memberCounts := make(map[int]int)
		for _, memberID := range assignedMembers {
			memberCounts[memberID]++
		}

		// All members should be used if we have enough assignments
		expectedMembers := []int{10, 20, 30}
		for _, memberID := range expectedMembers {
			assert.True(suite.T(), memberCounts[memberID] > 0, "Member %d should have at least one assignment", memberID)
		}
	}
}

// TestGenerateSchedule_SkipManualOverrides tests that manual overrides are preserved
func (suite *GenerateScheduleTestSuite) TestGenerateSchedule_SkipManualOverrides() {
	scheduleState := &models.ScheduleState{
		ID:                 1,
		LastGenerationDate: time.Now().AddDate(0, 0, -8),
	}

	activeMembers := []models.TeamMember{
		{ID: 1, Name: "John Doe", Active: true},
	}
	activeDays := []models.WorkingHours{
		{ID: 1, DayOfWeek: 0, StartTime: "09:00", EndTime: "17:00", Active: true}, // Monday
	}

	suite.mockTeamRepo.EXPECT().GetActiveMembers().Return(activeMembers, nil)
	suite.mockWorkingRepo.EXPECT().GetActiveDays().Return(activeDays, nil)
	suite.mockScheduleRepo.EXPECT().GetState().Return(scheduleState, nil)
	suite.mockScheduleRepo.EXPECT().GetByDateRange(mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.ScheduleEntry{}, nil)

	// Mock GetByDate to return a manual override for a specific Monday
	nextMonday := getNextMonday()
	manualOverride := []models.ScheduleEntry{
		{ID: 100, Date: nextMonday, TeamMemberID: 999, IsManualOverride: true},
	}

	suite.mockScheduleRepo.EXPECT().GetByDate(nextMonday).Return(manualOverride, nil).Maybe()
	suite.mockScheduleRepo.EXPECT().GetByDate(mock.MatchedBy(func(date time.Time) bool {
		return !date.Equal(nextMonday)
	})).Return([]models.ScheduleEntry{}, nil).Maybe()

	// Expect creation of entries but NOT for the day with manual override
	suite.mockScheduleRepo.EXPECT().Create(mock.MatchedBy(func(entry *models.ScheduleEntry) bool {
		return !entry.Date.Equal(nextMonday) // Should not create entry for manual override day
	})).Return(nil).Maybe()

	suite.mockScheduleRepo.EXPECT().UpdateState(mock.AnythingOfType("*models.ScheduleState")).Return(nil)

	// Act
	result, err := suite.service.GenerateSchedule(false)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.True(suite.T(), result.Success)
}

// TestGenerateSchedule_ErrorHandling tests various error scenarios
func (suite *GenerateScheduleTestSuite) TestGenerateSchedule_ErrorHandling() {
	testCases := []struct {
		name          string
		setupMocks    func()
		expectedError string
	}{
		{
			name: "GetState error",
			setupMocks: func() {
				suite.mockTeamRepo.EXPECT().GetActiveMembers().Return([]models.TeamMember{{ID: 1}}, nil)
				suite.mockWorkingRepo.EXPECT().GetActiveDays().Return([]models.WorkingHours{{DayOfWeek: 0}}, nil)
				suite.mockScheduleRepo.EXPECT().GetState().Return(nil, errors.New("state error"))
			},
			expectedError: "failed to get schedule state",
		},
		{
			name: "GetByDateRange error",
			setupMocks: func() {
				suite.mockTeamRepo.EXPECT().GetActiveMembers().Return([]models.TeamMember{{ID: 1}}, nil)
				suite.mockWorkingRepo.EXPECT().GetActiveDays().Return([]models.WorkingHours{{DayOfWeek: 0}}, nil)
				suite.mockScheduleRepo.EXPECT().GetState().Return(&models.ScheduleState{LastGenerationDate: time.Now().AddDate(0, 0, -8)}, nil)
				suite.mockScheduleRepo.EXPECT().GetByDateRange(mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("range error"))
			},
			expectedError: "failed to get existing entries",
		},
		{
			name: "UpdateState error",
			setupMocks: func() {
				suite.mockTeamRepo.EXPECT().GetActiveMembers().Return([]models.TeamMember{{ID: 1}}, nil)
				suite.mockWorkingRepo.EXPECT().GetActiveDays().Return([]models.WorkingHours{{DayOfWeek: 0}}, nil)
				suite.mockScheduleRepo.EXPECT().GetState().Return(&models.ScheduleState{LastGenerationDate: time.Now().AddDate(0, 0, -8)}, nil)
				suite.mockScheduleRepo.EXPECT().GetByDateRange(mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.ScheduleEntry{}, nil)
				suite.mockScheduleRepo.EXPECT().GetByDate(mock.AnythingOfType("time.Time")).Return([]models.ScheduleEntry{}, nil).Maybe()
				suite.mockScheduleRepo.EXPECT().Create(mock.AnythingOfType("*models.ScheduleEntry")).Return(nil).Maybe()
				suite.mockScheduleRepo.EXPECT().UpdateState(mock.AnythingOfType("*models.ScheduleState")).Return(errors.New("update error"))
			},
			expectedError: "failed to update schedule state",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Setup fresh mocks for each test case
			suite.SetupTest()
			tc.setupMocks()

			// Act
			result, err := suite.service.GenerateSchedule(false)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

// TestGenerateSchedule_DeterministicGenerationWithTwoMembers tests deterministic assignment works correctly with small team
func (suite *GenerateScheduleTestSuite) TestGenerateSchedule_DeterministicGenerationWithTwoMembers() {
	scheduleState := &models.ScheduleState{
		ID:                 1,
		LastGenerationDate: time.Now().AddDate(0, 0, -8),
	}

	activeMembers := []models.TeamMember{
		{ID: 1, Name: "John Doe", Active: true},
		{ID: 2, Name: "Jane Smith", Active: true},
	}
	activeDays := []models.WorkingHours{
		{ID: 1, DayOfWeek: 0, StartTime: "09:00", EndTime: "17:00", Active: true},
	}

	suite.mockTeamRepo.EXPECT().GetActiveMembers().Return(activeMembers, nil)
	suite.mockWorkingRepo.EXPECT().GetActiveDays().Return(activeDays, nil)
	suite.mockScheduleRepo.EXPECT().GetState().Return(scheduleState, nil)
	suite.mockScheduleRepo.EXPECT().GetByDateRange(mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.ScheduleEntry{}, nil)

	// Expect entries to be created based on deterministic date-based assignment
	suite.mockScheduleRepo.EXPECT().GetByDate(mock.AnythingOfType("time.Time")).Return([]models.ScheduleEntry{}, nil).Maybe()
	suite.mockScheduleRepo.EXPECT().Create(mock.AnythingOfType("*models.ScheduleEntry")).Return(nil).Maybe()

	suite.mockScheduleRepo.EXPECT().UpdateState(mock.AnythingOfType("*models.ScheduleState")).Return(nil)

	// Act
	result, err := suite.service.GenerateSchedule(false)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.True(suite.T(), result.Success)
}

// Helper function to get the next Monday
func getNextMonday() time.Time {
	now := time.Now()
	days := (7 - int(now.Weekday()) + 1) % 7
	if days == 0 {
		days = 7
	}
	return now.AddDate(0, 0, days)
}

// TestGenerateSchedule_DeterministicAssignment tests that the assignment is deterministic regardless of generation start day
func (suite *GenerateScheduleTestSuite) TestGenerateSchedule_DeterministicAssignment() {
	// Store original timeNow function to restore later
	originalTimeNow := timeNow
	defer func() { timeNow = originalTimeNow }()

	activeMembers := []models.TeamMember{
		{ID: 1, Name: "Alice", Active: true},
		{ID: 2, Name: "Bob", Active: true},
		{ID: 3, Name: "Charlie", Active: true},
	}

	// Working days: Monday, Wednesday (using models.GetWeekdayNumber numbering: 0=Monday, 2=Wednesday)
	// With 3 people and 2 working days, we should see rotation between people on each day
	activeDays := []models.WorkingHours{
		{ID: 1, DayOfWeek: 0, StartTime: "09:00", EndTime: "17:00", Active: true}, // Monday (0)
		{ID: 2, DayOfWeek: 2, StartTime: "09:00", EndTime: "17:00", Active: true}, // Wednesday (2)
	}

	scheduleState := &models.ScheduleState{
		ID:                 1,
		LastGenerationDate: time.Now().AddDate(0, 0, -8), // Force regeneration
	}

	// Test the actual service method on different start days
	testCases := []struct {
		name      string
		startDate time.Time
	}{
		{
			name:      "Generate starting Monday",
			startDate: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), // Monday
		},
		{
			name:      "Generate starting Tuesday",
			startDate: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC), // Tuesday
		},
	}

	// Store results from both runs to compare
	var allRunResults [][]struct {
		date     time.Time
		memberID int
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Reset mocks for each test case
			suite.SetupTest()

			// Mock the current time to be the test date
			timeNow = func() time.Time { return tc.startDate }

			// Set up mocks
			suite.mockTeamRepo.EXPECT().GetActiveMembers().Return(activeMembers, nil)
			suite.mockWorkingRepo.EXPECT().GetActiveDays().Return(activeDays, nil)
			suite.mockScheduleRepo.EXPECT().GetState().Return(scheduleState, nil)
			suite.mockScheduleRepo.EXPECT().GetByDateRange(mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]models.ScheduleEntry{}, nil)

			// Track assignments by day of week AND chronological sequence
			assignments := make(map[time.Weekday][]int)
			var chronologicalAssignments []struct {
				date     time.Time
				memberID int
			}

			suite.mockScheduleRepo.EXPECT().GetByDate(mock.AnythingOfType("time.Time")).Return([]models.ScheduleEntry{}, nil).Maybe()
			suite.mockScheduleRepo.EXPECT().Create(mock.MatchedBy(func(entry *models.ScheduleEntry) bool {
				weekday := entry.Date.Weekday()
				assignments[weekday] = append(assignments[weekday], entry.TeamMemberID)
				chronologicalAssignments = append(chronologicalAssignments, struct {
					date     time.Time
					memberID int
				}{entry.Date, entry.TeamMemberID})
				return true
			})).Return(nil).Maybe()

			suite.mockScheduleRepo.EXPECT().UpdateState(mock.AnythingOfType("*models.ScheduleState")).Return(nil)

			// Act - Call the actual GenerateSchedule method
			result, err := suite.service.GenerateSchedule(true) // Force generation

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.True(t, result.Success)

			// Store results for comparison
			allRunResults = append(allRunResults, chronologicalAssignments)
		})
	}

	// Compare assignments between both runs
	if len(allRunResults) == 2 {
		run1 := allRunResults[0]
		run2 := allRunResults[1]

		// Create maps of date -> memberID for easy comparison
		run1Map := make(map[string]int)
		run2Map := make(map[string]int)

		for _, assignment := range run1 {
			dateKey := assignment.date.Format("2006-01-02")
			run1Map[dateKey] = assignment.memberID
		}

		for _, assignment := range run2 {
			dateKey := assignment.date.Format("2006-01-02")
			run2Map[dateKey] = assignment.memberID
		}

		// Compare assignments for same dates
		matchingDates := 0
		differentAssignments := 0

		for dateKey, run1Member := range run1Map {
			if run2Member, exists := run2Map[dateKey]; exists {
				matchingDates++
				if run1Member != run2Member {
					differentAssignments++
					suite.T().Errorf("DETERMINISM FAILURE: Date %s assigned to Member %d in run 1 but Member %d in run 2",
						dateKey, run1Member, run2Member)
				}
			}
		}

		if differentAssignments == 0 && matchingDates > 0 {
			suite.T().Logf("SUCCESS: Algorithm is deterministic! %d matching dates all have consistent assignments", matchingDates)
		} else if matchingDates == 0 {
			suite.T().Errorf("FAILURE: No overlapping dates between runs (different generation periods)")
		} else {
			suite.T().Errorf("FAILURE: %d out of %d matching dates have different assignments!", differentAssignments, matchingDates)
		}
	}
}

// TestGenerateSchedule_UnfairRoundRobin demonstrates that the current algorithm
// doesn't provide fair round-robin scheduling across weeks - it continues the absolute day count
// instead of restarting the rotation each week or using a proper working-day-based rotation
func (suite *GenerateScheduleTestSuite) TestGenerateSchedule_UnfairRoundRobin() {
	// Setup: Override time to a specific Monday to make test deterministic
	originalTimeNow := timeNow
	defer func() { timeNow = originalTimeNow }()

	// Start on a Monday (2023-10-02 was a Monday)
	testStartDate := time.Date(2023, 10, 2, 0, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return testStartDate }

	// Setup: 3 active team members
	activeMembers := []models.TeamMember{
		{ID: 1, Name: "Alice", Active: true},
		{ID: 2, Name: "Bob", Active: true},
		{ID: 3, Name: "Charlie", Active: true},
	}

	// Setup: Working days Monday-Friday only
	activeDays := []models.WorkingHours{
		{DayOfWeek: 0, StartTime: "09:00", EndTime: "17:00", Active: true}, // Monday
		{DayOfWeek: 1, StartTime: "09:00", EndTime: "17:00", Active: true}, // Tuesday
		{DayOfWeek: 3, StartTime: "09:00", EndTime: "17:00", Active: true}, // Thursday
		{DayOfWeek: 4, StartTime: "09:00", EndTime: "17:00", Active: true}, // Friday
	}

	// Mock setup
	suite.mockTeamRepo.EXPECT().GetActiveMembers().Return(activeMembers, nil)
	suite.mockWorkingRepo.EXPECT().GetActiveDays().Return(activeDays, nil)

	// Mock schedule state
	initialState := &models.ScheduleState{
		ID:                 1,
		LastGenerationDate: testStartDate.AddDate(0, 0, -30), // 30 days ago
	}
	suite.mockScheduleRepo.EXPECT().GetState().Return(initialState, nil)

	// Mock existing entries - no existing entries
	suite.mockScheduleRepo.EXPECT().GetByDateRange(mock.Anything, mock.Anything).Return([]models.ScheduleEntry{}, nil)

	// Mock GetByDate calls for override checks - no overrides
	// Need to be more generous with the number of calls since it generates for 3 months
	suite.mockScheduleRepo.EXPECT().GetByDate(mock.Anything).Return([]models.ScheduleEntry{}, nil).Maybe()

	// Track created entries
	var createdEntries []models.ScheduleEntry
	suite.mockScheduleRepo.EXPECT().Create(mock.AnythingOfType("*models.ScheduleEntry")).RunAndReturn(
		func(entry *models.ScheduleEntry) error {
			entry.ID = len(createdEntries) + 1
			createdEntries = append(createdEntries, *entry)
			return nil
		},
	).Maybe() // Let it create as many as needed

	// Mock state update
	suite.mockScheduleRepo.EXPECT().UpdateState(mock.Anything).Return(nil)

	// Act
	result, err := suite.service.GenerateSchedule(true)

	// Assert generation succeeded
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.True(suite.T(), result.Success)

	suite.T().Logf("Generated %d entries total", len(createdEntries))

	// Sort entries by date to check the sequence
	sort.Slice(createdEntries, func(i, j int) bool {
		return createdEntries[i].Date.Before(createdEntries[j].Date)
	})

	// The FAILING assertion: This should demonstrate the unfairness
	// Check that each team member is in the same order

	teamMap := make(map[int]int)
	suite.T().Logf("Checking consecutive assignments in chronological order:")
	for i := 0; i < len(createdEntries); i++ {
		entry := createdEntries[i]

		if i > 0 {
			prevEntry := createdEntries[i-1]
			if tmid, ok := teamMap[prevEntry.TeamMemberID]; ok {
				assert.Equal(suite.T(), tmid, entry.TeamMemberID)
			} else {
				teamMap[prevEntry.TeamMemberID] = entry.TeamMemberID
			}
		}
	}
}

// TestRunGenerateScheduleTestSuite runs the test suite
func TestRunGenerateScheduleTestSuite(t *testing.T) {
	suite.Run(t, new(GenerateScheduleTestSuite))
}
