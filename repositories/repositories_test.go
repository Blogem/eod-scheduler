package repositories

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/blogem/eod-scheduler/database"
	"github.com/blogem/eod-scheduler/models"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	// Create a temporary database for testing
	dbPath := "test_" + time.Now().Format("20060102150405") + ".db"

	// Clean up function
	t.Cleanup(func() {
		os.Remove(dbPath)
	})

	// Initialize test database using the actual migration system
	if err := database.InitializeDatabase(dbPath); err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	return database.GetDB()
}

func TestTeamRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTeamRepository(db)

	// Test Create
	member := &models.TeamMember{
		Name:        "Test User",
		SlackHandle: "@test.user",
		Active:      true,
	}

	err := repo.Create(member)
	if err != nil {
		t.Fatalf("Failed to create team member: %v", err)
	}

	if member.ID == 0 {
		t.Error("Expected member ID to be set after creation")
	}

	// Test GetByID
	retrieved, err := repo.GetByID(member.ID)
	if err != nil {
		t.Fatalf("Failed to get team member by ID: %v", err)
	}

	if retrieved.Name != member.Name {
		t.Errorf("Expected name %s, got %s", member.Name, retrieved.Name)
	}

	// Test GetAll
	members, err := repo.GetAll()
	if err != nil {
		t.Fatalf("Failed to get all team members: %v", err)
	}

	if len(members) != 1 {
		t.Errorf("Expected 1 team member, got %d", len(members))
	}

	// Test GetActiveMembers
	activeMembers, err := repo.GetActiveMembers()
	if err != nil {
		t.Fatalf("Failed to get active team members: %v", err)
	}

	if len(activeMembers) != 1 {
		t.Errorf("Expected 1 active team member, got %d", len(activeMembers))
	}

	// Test Update
	member.Name = "Updated Name"
	err = repo.Update(member)
	if err != nil {
		t.Fatalf("Failed to update team member: %v", err)
	}

	updated, err := repo.GetByID(member.ID)
	if err != nil {
		t.Fatalf("Failed to get updated team member: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Expected updated name 'Updated Name', got %s", updated.Name)
	}

	// Test Count
	count, err := repo.Count()
	if err != nil {
		t.Fatalf("Failed to count team members: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	// Test Delete
	err = repo.Delete(member.ID)
	if err != nil {
		t.Fatalf("Failed to delete team member: %v", err)
	}

	// Verify deletion
	_, err = repo.GetByID(member.ID)
	if err == nil {
		t.Error("Expected error when getting deleted team member")
	}
}

func TestWorkingHoursRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkingHoursRepository(db)

	// Test GetAll (should have default data from migration)
	hours, err := repo.GetAll()
	if err != nil {
		t.Fatalf("Failed to get all working hours: %v", err)
	}

	if len(hours) != 7 {
		t.Errorf("Expected 7 working hours entries, got %d", len(hours))
	}

	// Test GetByDay
	monday, err := repo.GetByDay(0) // Monday
	if err != nil {
		t.Fatalf("Failed to get Monday working hours: %v", err)
	}

	if monday.StartTime != "09:00" || monday.EndTime != "17:00" {
		t.Errorf("Expected Monday 09:00-17:00, got %s-%s", monday.StartTime, monday.EndTime)
	}

	// Test GetActiveDays
	activeDays, err := repo.GetActiveDays()
	if err != nil {
		t.Fatalf("Failed to get active days: %v", err)
	}

	if len(activeDays) != 5 { // Monday-Friday
		t.Errorf("Expected 5 active days, got %d", len(activeDays))
	}

	// Test UpdateByDay
	err = repo.UpdateByDay(0, "08:00", "16:00", true)
	if err != nil {
		t.Fatalf("Failed to update Monday working hours: %v", err)
	}

	// Verify update
	updated, err := repo.GetByDay(0)
	if err != nil {
		t.Fatalf("Failed to get updated Monday working hours: %v", err)
	}

	if updated.StartTime != "08:00" || updated.EndTime != "16:00" {
		t.Errorf("Expected updated Monday 08:00-16:00, got %s-%s", updated.StartTime, updated.EndTime)
	}
}

func TestScheduleRepository(t *testing.T) {
	db := setupTestDB(t)
	scheduleRepo := NewScheduleRepository(db)
	teamRepo := NewTeamRepository(db)

	// Create a test team member first
	member := &models.TeamMember{
		Name:        "Test User",
		SlackHandle: "@test.user",
		Active:      true,
	}
	err := teamRepo.Create(member)
	if err != nil {
		t.Fatalf("Failed to create test team member: %v", err)
	}

	// Test Create schedule entry
	tomorrow := time.Now().AddDate(0, 0, 1)
	entry := &models.ScheduleEntry{
		Date:             tomorrow,
		TeamMemberID:     member.ID,
		StartTime:        "09:00",
		EndTime:          "17:00",
		IsManualOverride: false,
	}

	err = scheduleRepo.Create(entry)
	if err != nil {
		t.Fatalf("Failed to create schedule entry: %v", err)
	}

	if entry.ID == 0 {
		t.Error("Expected entry ID to be set after creation")
	}

	// Test GetByID
	retrieved, err := scheduleRepo.GetByID(entry.ID)
	if err != nil {
		t.Fatalf("Failed to get schedule entry by ID: %v", err)
	}

	if retrieved.TeamMemberName != member.Name {
		t.Errorf("Expected team member name %s, got %s", member.Name, retrieved.TeamMemberName)
	}

	// Test GetByDateRange
	entries, err := scheduleRepo.GetByDateRange(tomorrow, tomorrow)
	if err != nil {
		t.Fatalf("Failed to get schedule entries by date range: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 schedule entry, got %d", len(entries))
	}

	// Test GetState
	state, err := scheduleRepo.GetState()
	if err != nil {
		t.Fatalf("Failed to get schedule state: %v", err)
	}

	if state.ID != 1 {
		t.Errorf("Expected state ID 1, got %d", state.ID)
	}

	// Test UpdateState - update the generation date
	newDate := time.Now().AddDate(0, 0, 1)
	state.LastGenerationDate = newDate
	err = scheduleRepo.UpdateState(state)
	if err != nil {
		t.Fatalf("Failed to update schedule state: %v", err)
	}

	// Verify state update
	updatedState, err := scheduleRepo.GetState()
	if err != nil {
		t.Fatalf("Failed to get updated schedule state: %v", err)
	}

	// Compare dates (allowing for minor time differences)
	expectedDate := newDate.Format("2006-01-02")
	actualDate := updatedState.LastGenerationDate.Format("2006-01-02")
	if actualDate != expectedDate {
		t.Errorf("Expected updated last generation date %s, got %s", expectedDate, actualDate)
	}
}
