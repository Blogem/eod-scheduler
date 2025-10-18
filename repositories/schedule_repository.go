package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/blogem/eod-scheduler/models"
	"github.com/blogem/eod-scheduler/userctx"
)

// ScheduleRepository interface defines schedule database operations
type ScheduleRepository interface {
	GetByDateRange(ctx context.Context, from, to time.Time) ([]models.ScheduleEntry, error)
	GetByDate(ctx context.Context, date time.Time) ([]models.ScheduleEntry, error)
	GetByID(ctx context.Context, id int) (*models.ScheduleEntry, error)
	Create(ctx context.Context, entry *models.ScheduleEntry) error
	Update(ctx context.Context, entry *models.ScheduleEntry) error
	Delete(ctx context.Context, id int) error
	DeleteByDateRange(ctx context.Context, from, to time.Time) error
	GetState(ctx context.Context) (*models.ScheduleState, error)
	UpdateState(ctx context.Context, state *models.ScheduleState) error
	CountByTeamMember(ctx context.Context, teamMemberID int) (int, error)
	HasFutureEntries(ctx context.Context, teamMemberID int) (bool, error)
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
func (r *scheduleRepository) GetByDateRange(ctx context.Context, from, to time.Time) ([]models.ScheduleEntry, error) {
	query := `
		SELECT se.id, se.date, se.team_member_id, se.start_time, se.end_time, 
			   se.is_manual_override, se.original_team_member_id,
			   t.name as team_member_name, t.slack_handle as team_member_slack_handle
		FROM schedule_entries se
		LEFT JOIN team_members t ON se.team_member_id = t.id
		WHERE se.date >= ? AND se.date <= ?
		ORDER BY se.date, se.start_time
		`

	rows, err := r.db.Query(query, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to query schedule entries: %w", err)
	}
	defer rows.Close()

	var entries []models.ScheduleEntry
	for rows.Next() {
		var entry models.ScheduleEntry
		var teamMemberName, teamMemberSlackHandle sql.NullString

		err := rows.Scan(
			&entry.ID,
			&entry.Date,
			&entry.TeamMemberID,
			&entry.StartTime,
			&entry.EndTime,
			&entry.IsManualOverride,
			&entry.OriginalTeamMemberID,
			&teamMemberName,
			&teamMemberSlackHandle,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule entry: %w", err)
		}

		// Handle nullable fields
		if teamMemberName.Valid {
			entry.TeamMemberName = teamMemberName.String
		}
		if teamMemberSlackHandle.Valid {
			entry.TeamMemberSlackHandle = teamMemberSlackHandle.String
		}

		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schedule entries: %w", err)
	}

	return entries, nil
}

// GetByDate retrieves schedule entries for a specific date
func (r *scheduleRepository) GetByDate(ctx context.Context, date time.Time) ([]models.ScheduleEntry, error) {
	return r.GetByDateRange(ctx, date, date)
}

// GetByID retrieves a single schedule entry by ID with team member info
func (r *scheduleRepository) GetByID(ctx context.Context, id int) (*models.ScheduleEntry, error) {
	query := `
		SELECT 
			s.id, s.date, s.team_member_id, s.start_time, s.end_time, s.is_manual_override, s.original_team_member_id,
			t.name as team_member_name, t.slack_handle as team_member_slack_handle
		FROM schedule_entries s
		LEFT JOIN team_members t ON s.team_member_id = t.id
		WHERE s.id = ?
	`

	var entry models.ScheduleEntry
	var teamMemberName, teamMemberSlackHandle sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&entry.ID,
		&entry.Date,
		&entry.TeamMemberID,
		&entry.StartTime,
		&entry.EndTime,
		&entry.IsManualOverride,
		&entry.OriginalTeamMemberID,
		&teamMemberName,
		&teamMemberSlackHandle,
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
	if teamMemberSlackHandle.Valid {
		entry.TeamMemberSlackHandle = teamMemberSlackHandle.String
	}

	return &entry, nil
}

// Create creates a new schedule entry
func (r *scheduleRepository) Create(ctx context.Context, entry *models.ScheduleEntry) error {
	// Get user email from context for audit
	userEmail := userctx.GetUserEmail(ctx)

	fmt.Println("Creating schedule entry:", entry)
	query := `
		INSERT INTO schedule_entries (date, team_member_id, start_time, end_time, is_manual_override, original_team_member_id, created_by) 
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query,
		entry.Date.Format("2006-01-02"),
		entry.TeamMemberID,
		entry.StartTime,
		entry.EndTime,
		entry.IsManualOverride,
		entry.OriginalTeamMemberID,
		userEmail,
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

// Update updates an existing schedule entry with audit fields
func (r *scheduleRepository) Update(ctx context.Context, entry *models.ScheduleEntry) error {
	// Get user email from context for audit
	userEmail := userctx.GetUserEmail(ctx)
	now := time.Now()

	query := `
		UPDATE schedule_entries 
		SET date = ?, team_member_id = ?, start_time = ?, end_time = ?, is_manual_override = ?, original_team_member_id = ?,
		    modified_by = ?, modified_at = ?
		WHERE id = ?
	`

	result, err := r.db.Exec(query,
		entry.Date.Format("2006-01-02"),
		entry.TeamMemberID,
		entry.StartTime,
		entry.EndTime,
		entry.IsManualOverride,
		entry.OriginalTeamMemberID,
		userEmail,
		now,
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
func (r *scheduleRepository) Delete(ctx context.Context, id int) error {
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
func (r *scheduleRepository) DeleteByDateRange(ctx context.Context, from, to time.Time) error {
	query := `DELETE FROM schedule_entries WHERE date >= ? AND date <= ?`

	_, err := r.db.Exec(query, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("failed to delete schedule entries in range: %w", err)
	}

	return nil
}

// GetState retrieves the current schedule state
func (r *scheduleRepository) GetState(ctx context.Context) (*models.ScheduleState, error) {
	query := `
		SELECT id, last_generation_date 
		FROM schedule_state 
		WHERE id = 1
	`

	var state models.ScheduleState
	err := r.db.QueryRow(query).Scan(
		&state.ID,
		&state.LastGenerationDate,
	)

	if err == sql.ErrNoRows {
		// Initialize default state if not exists
		defaultState := &models.ScheduleState{
			ID:                 1,
			LastGenerationDate: time.Now(),
		}
		if err := r.UpdateState(ctx, defaultState); err != nil {
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
func (r *scheduleRepository) UpdateState(ctx context.Context, state *models.ScheduleState) error {
	query := `
		INSERT OR REPLACE INTO schedule_state (id, last_generation_date) 
		VALUES (1, ?)
	`

	_, err := r.db.Exec(query, state.LastGenerationDate.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("failed to update schedule state: %w", err)
	}

	return nil
}

// CountByTeamMember counts schedule entries for a specific team member
func (r *scheduleRepository) CountByTeamMember(ctx context.Context, teamMemberID int) (int, error) {
	query := `SELECT COUNT(*) FROM schedule_entries WHERE team_member_id = ?`

	var count int
	err := r.db.QueryRow(query, teamMemberID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count schedule entries for team member: %w", err)
	}

	return count, nil
}

// HasFutureEntries checks if a team member has future schedule entries
func (r *scheduleRepository) HasFutureEntries(ctx context.Context, teamMemberID int) (bool, error) {
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
