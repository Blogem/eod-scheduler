package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/blogem/eod-scheduler/models"
)

// ScheduleRepository interface defines schedule database operations
type ScheduleRepository interface {
	GetByDateRange(from, to time.Time) ([]models.ScheduleEntry, error)
	GetByDate(date time.Time) ([]models.ScheduleEntry, error)
	GetByID(id int) (*models.ScheduleEntry, error)
	Create(entry *models.ScheduleEntry) error
	Update(entry *models.ScheduleEntry) error
	Delete(id int) error
	DeleteByDateRange(from, to time.Time) error
	GetState() (*models.ScheduleState, error)
	UpdateState(state *models.ScheduleState) error
	CountByTeamMember(teamMemberID int) (int, error)
	HasFutureEntries(teamMemberID int) (bool, error)
}

// scheduleRepository implements ScheduleRepository interface
type scheduleRepository struct {
	db *sql.DB
}

// NewScheduleRepository creates a new schedule repository
func NewScheduleRepository(db *sql.DB) ScheduleRepository {
	return &scheduleRepository{db: db}
}

// GetByDateRange retrieves schedule entries within a date range with team member info
func (r *scheduleRepository) GetByDateRange(from, to time.Time) ([]models.ScheduleEntry, error) {
	query := `
		SELECT 
			s.id, s.date, s.team_member_id, s.start_time, s.end_time, s.is_manual_override,
			t.name as team_member_name, t.email as team_member_email
		FROM schedule_entries s
		LEFT JOIN team_members t ON s.team_member_id = t.id
		WHERE s.date >= ? AND s.date <= ?
		ORDER BY s.date ASC, s.start_time ASC
	`

	rows, err := r.db.Query(query, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to query schedule entries: %w", err)
	}
	defer rows.Close()

	var entries []models.ScheduleEntry
	for rows.Next() {
		var entry models.ScheduleEntry
		var teamMemberName, teamMemberEmail sql.NullString

		err := rows.Scan(
			&entry.ID,
			&entry.Date,
			&entry.TeamMemberID,
			&entry.StartTime,
			&entry.EndTime,
			&entry.IsManualOverride,
			&teamMemberName,
			&teamMemberEmail,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule entry: %w", err)
		}

		// Handle nullable fields
		if teamMemberName.Valid {
			entry.TeamMemberName = teamMemberName.String
		}
		if teamMemberEmail.Valid {
			entry.TeamMemberEmail = teamMemberEmail.String
		}

		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schedule entries: %w", err)
	}

	return entries, nil
}

// GetByDate retrieves schedule entries for a specific date
func (r *scheduleRepository) GetByDate(date time.Time) ([]models.ScheduleEntry, error) {
	return r.GetByDateRange(date, date)
}

// GetByID retrieves a schedule entry by ID
func (r *scheduleRepository) GetByID(id int) (*models.ScheduleEntry, error) {
	query := `
		SELECT 
			s.id, s.date, s.team_member_id, s.start_time, s.end_time, s.is_manual_override, s.original_team_member_id,
			t.name as team_member_name, t.email as team_member_email
		FROM schedule_entries s
		LEFT JOIN team_members t ON s.team_member_id = t.id
		WHERE s.id = ?
	`

	var entry models.ScheduleEntry
	var teamMemberName, teamMemberEmail sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&entry.ID,
		&entry.Date,
		&entry.TeamMemberID,
		&entry.StartTime,
		&entry.EndTime,
		&entry.IsManualOverride,
		&entry.OriginalTeamMemberID,
		&teamMemberName,
		&teamMemberEmail,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schedule entry with ID %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule entry: %w", err)
	}

	// Handle nullable fields
	if teamMemberName.Valid {
		entry.TeamMemberName = teamMemberName.String
	}
	if teamMemberEmail.Valid {
		entry.TeamMemberEmail = teamMemberEmail.String
	}

	return &entry, nil
}

// Create creates a new schedule entry
func (r *scheduleRepository) Create(entry *models.ScheduleEntry) error {

	fmt.Println("Creating schedule entry:", entry)
	query := `
		INSERT INTO schedule_entries (date, team_member_id, start_time, end_time, is_manual_override, original_team_member_id) 
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query,
		entry.Date.Format("2006-01-02"),
		entry.TeamMemberID,
		entry.StartTime,
		entry.EndTime,
		entry.IsManualOverride,
		entry.OriginalTeamMemberID,
	)
	if err != nil {
		return fmt.Errorf("failed to create schedule entry: %w", err)
	}

	// Get the inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get inserted ID: %w", err)
	}

	entry.ID = int(id)
	return nil
}

// Update updates an existing schedule entry
func (r *scheduleRepository) Update(entry *models.ScheduleEntry) error {
	query := `
		UPDATE schedule_entries 
		SET date = ?, team_member_id = ?, start_time = ?, end_time = ?, is_manual_override = ?
		WHERE id = ?
	`

	result, err := r.db.Exec(query,
		entry.Date.Format("2006-01-02"),
		entry.TeamMemberID,
		entry.StartTime,
		entry.EndTime,
		entry.IsManualOverride,
		entry.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update schedule entry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("schedule entry with ID %d not found", entry.ID)
	}

	return nil
}

// Delete deletes a schedule entry by ID
func (r *scheduleRepository) Delete(id int) error {
	query := `DELETE FROM schedule_entries WHERE id = ?`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete schedule entry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("schedule entry with ID %d not found", id)
	}

	return nil
}

// DeleteByDateRange deletes schedule entries within a date range
func (r *scheduleRepository) DeleteByDateRange(from, to time.Time) error {
	query := `DELETE FROM schedule_entries WHERE date >= ? AND date <= ?`

	_, err := r.db.Exec(query, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("failed to delete schedule entries in range: %w", err)
	}

	return nil
}

// GetState retrieves the current schedule state
func (r *scheduleRepository) GetState() (*models.ScheduleState, error) {
	query := `
		SELECT id, next_person_index, last_generation_date 
		FROM schedule_state 
		WHERE id = 1
	`

	var state models.ScheduleState
	err := r.db.QueryRow(query).Scan(
		&state.ID,
		&state.NextPersonIndex,
		&state.LastGenerationDate,
	)

	if err == sql.ErrNoRows {
		// Initialize default state if not exists
		defaultState := &models.ScheduleState{
			ID:                 1,
			NextPersonIndex:    0,
			LastGenerationDate: time.Now(),
		}
		if err := r.UpdateState(defaultState); err != nil {
			return nil, fmt.Errorf("failed to initialize schedule state: %w", err)
		}
		return defaultState, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule state: %w", err)
	}

	return &state, nil
}

// UpdateState updates the schedule state
func (r *scheduleRepository) UpdateState(state *models.ScheduleState) error {
	query := `
		INSERT OR REPLACE INTO schedule_state (id, next_person_index, last_generation_date) 
		VALUES (1, ?, ?)
	`

	_, err := r.db.Exec(query, state.NextPersonIndex, state.LastGenerationDate.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("failed to update schedule state: %w", err)
	}

	return nil
}

// CountByTeamMember counts schedule entries for a specific team member
func (r *scheduleRepository) CountByTeamMember(teamMemberID int) (int, error) {
	query := `SELECT COUNT(*) FROM schedule_entries WHERE team_member_id = ?`

	var count int
	err := r.db.QueryRow(query, teamMemberID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count schedule entries for team member: %w", err)
	}

	return count, nil
}

// HasFutureEntries checks if a team member has future schedule entries
func (r *scheduleRepository) HasFutureEntries(teamMemberID int) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM schedule_entries 
		WHERE team_member_id = ? AND date > date('now')
	`

	var count int
	err := r.db.QueryRow(query, teamMemberID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check future entries for team member: %w", err)
	}

	return count > 0, nil
}
