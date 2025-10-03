package services

import (
	"errors"
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
		NextPersonIndex:    2,
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
		NextPersonIndex:    1,
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
	futureEnd := today.AddDate(0, 3, 0)
	existingEntries := []models.ScheduleEntry{
		{ID: 1, Date: today.AddDate(0, 0, 1), IsManualOverride: false},
		{ID: 2, Date: today.AddDate(0, 0, 2), IsManualOverride: true}, // Should not be deleted
	}
	suite.mockScheduleRepo.EXPECT().GetByDateRange(mock.MatchedBy(func(from time.Time) bool {
		return from.Year() == today.Year() && from.Month() == today.Month() && from.Day() == today.Day()
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
		NextPersonIndex:    0,
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
		return state.ID == 1 && state.NextPersonIndex >= 0 && state.NextPersonIndex < len(activeMembers)
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

// TestGenerateSchedule_RoundRobinLogic tests the round-robin assignment logic
func (suite *GenerateScheduleTestSuite) TestGenerateSchedule_RoundRobinLogic() {
	// Setup: Start with member index 1 (second member)
	scheduleState := &models.ScheduleState{
		ID:                 1,
		NextPersonIndex:    1, // Start with second member
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

	// Verify round-robin logic: should start with member 20 (index 1), then 30 (index 2), then 10 (index 0)
	if len(assignedMembers) >= 3 {
		assert.Equal(suite.T(), 20, assignedMembers[0]) // Jane Smith (index 1)
		assert.Equal(suite.T(), 30, assignedMembers[1]) // Bob Wilson (index 2)
		assert.Equal(suite.T(), 10, assignedMembers[2]) // John Doe (index 0, wrapped around)
	}
}

// TestGenerateSchedule_SkipManualOverrides tests that manual overrides are preserved
func (suite *GenerateScheduleTestSuite) TestGenerateSchedule_SkipManualOverrides() {
	scheduleState := &models.ScheduleState{
		ID:                 1,
		NextPersonIndex:    0,
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

// TestGenerateSchedule_IndexBoundsHandling tests handling of out-of-bounds member index
func (suite *GenerateScheduleTestSuite) TestGenerateSchedule_IndexBoundsHandling() {
	// Setup: Index is out of bounds (larger than team size)
	scheduleState := &models.ScheduleState{
		ID:                 1,
		NextPersonIndex:    10, // Out of bounds
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

	// Expect entries to be created starting from the first member (index 0, ID 1)
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

// TestRunGenerateScheduleTestSuite runs the test suite
func TestRunGenerateScheduleTestSuite(t *testing.T) {
	suite.Run(t, new(GenerateScheduleTestSuite))
}
