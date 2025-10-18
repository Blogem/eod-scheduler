package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/blogem/eod-scheduler/models"
	"github.com/blogem/eod-scheduler/userctx"
)

// TeamRepository interface defines team member database operations
type TeamRepository interface {
	GetAll(ctx context.Context) ([]models.TeamMember, error)
	GetByID(ctx context.Context, id int) (*models.TeamMember, error)
	GetActiveMembers(ctx context.Context) ([]models.TeamMember, error)
	Create(ctx context.Context, member *models.TeamMember) error
	Update(ctx context.Context, member *models.TeamMember) error
	Delete(ctx context.Context, id int) error
	Count(ctx context.Context) (int, error)
}

// teamRepository implements TeamRepository interface
type teamRepository struct {
	db *sql.DB
}

// NewTeamRepository creates a new team repository
func NewTeamRepository(db *sql.DB) TeamRepository {
	return &teamRepository{db: db}
}

// GetAll retrieves all team members
func (r *teamRepository) GetAll(ctx context.Context) ([]models.TeamMember, error) {
	query := `
		SELECT id, name, slack_handle, active, date_added, 
		       created_by, modified_by, modified_at
		FROM team_members 
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query team members: %w", err)
	}
	defer rows.Close()

	var members []models.TeamMember
	for rows.Next() {
		var member models.TeamMember
		var modifiedBy sql.NullString
		var modifiedAt sql.NullTime

		err := rows.Scan(
			&member.ID,
			&member.Name,
			&member.SlackHandle,
			&member.Active,
			&member.DateAdded,
			&member.CreatedBy,
			&modifiedBy,
			&modifiedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan team member: %w", err)
		}

		// Convert NULL values to empty string/nil
		if modifiedBy.Valid {
			member.ModifiedBy = modifiedBy.String
		}
		if modifiedAt.Valid {
			member.ModifiedAt = &modifiedAt.Time
		}

		members = append(members, member)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating team members: %w", err)
	}

	return members, nil
}

// GetByID retrieves a team member by ID
func (r *teamRepository) GetByID(ctx context.Context, id int) (*models.TeamMember, error) {
	query := `
		SELECT id, name, slack_handle, active, date_added,
		       created_by, modified_by, modified_at
		FROM team_members 
		WHERE id = ?
	`

	var member models.TeamMember
	var modifiedBy sql.NullString
	var modifiedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&member.ID,
		&member.Name,
		&member.SlackHandle,
		&member.Active,
		&member.DateAdded,
		&member.CreatedBy,
		&modifiedBy,
		&modifiedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("team member with ID %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get team member: %w", err)
	}

	// Convert NULL values to empty string/nil
	if modifiedBy.Valid {
		member.ModifiedBy = modifiedBy.String
	}
	if modifiedAt.Valid {
		member.ModifiedAt = &modifiedAt.Time
	}

	return &member, nil
}

// GetActiveMembers retrieves only active team members
func (r *teamRepository) GetActiveMembers(ctx context.Context) ([]models.TeamMember, error) {
	query := `
		SELECT id, name, slack_handle, active, date_added,
		       created_by, modified_by, modified_at
		FROM team_members 
		WHERE active = 1 
		ORDER BY date_added ASC, name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active team members: %w", err)
	}
	defer rows.Close()

	var members []models.TeamMember
	for rows.Next() {
		var member models.TeamMember
		var modifiedBy sql.NullString
		var modifiedAt sql.NullTime

		err := rows.Scan(
			&member.ID,
			&member.Name,
			&member.SlackHandle,
			&member.Active,
			&member.DateAdded,
			&member.CreatedBy,
			&modifiedBy,
			&modifiedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan active team member: %w", err)
		}

		// Convert NULL values to empty string/nil
		if modifiedBy.Valid {
			member.ModifiedBy = modifiedBy.String
		}
		if modifiedAt.Valid {
			member.ModifiedAt = &modifiedAt.Time
		}

		members = append(members, member)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating active team members: %w", err)
	}

	return members, nil
}

// Create creates a new team member
func (r *teamRepository) Create(ctx context.Context, member *models.TeamMember) error {
	query := `
		INSERT INTO team_members (name, slack_handle, active, date_added, created_by) 
		VALUES (?, ?, ?, ?, ?)
	`

	// Set default values
	if member.DateAdded.IsZero() {
		member.DateAdded = time.Now()
	}

	// Get user from context
	userEmail := userctx.GetUserEmail(ctx)

	result, err := r.db.ExecContext(ctx, query,
		member.Name,
		member.SlackHandle,
		member.Active,
		member.DateAdded,
		userEmail,
	)
	if err != nil {
		return fmt.Errorf("failed to create team member: %w", err)
	}

	// Get the inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get inserted ID: %w", err)
	}

	member.ID = int(id)
	member.CreatedBy = userEmail
	return nil
}

// Update updates an existing team member
func (r *teamRepository) Update(ctx context.Context, member *models.TeamMember) error {
	query := `
		UPDATE team_members 
		SET name = ?, slack_handle = ?, active = ?,
		    modified_by = ?, modified_at = ?
		WHERE id = ?
	`

	// Get user from context
	userEmail := userctx.GetUserEmail(ctx)
	now := time.Now()

	result, err := r.db.ExecContext(ctx, query,
		member.Name,
		member.SlackHandle,
		member.Active,
		userEmail,
		now,
		member.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update team member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("team member with ID %d not found", member.ID)
	}

	member.ModifiedBy = userEmail
	member.ModifiedAt = &now
	return nil
}

// Delete deletes a team member by ID
func (r *teamRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM team_members WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete team member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("team member with ID %d not found", id)
	}

	return nil
}

// Count returns the total number of team members
func (r *teamRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM team_members`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count team members: %w", err)
	}

	return count, nil
}
