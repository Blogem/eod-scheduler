package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/blogem/eod-scheduler/models"
)

// TeamRepository interface defines team member database operations
type TeamRepository interface {
	GetAll() ([]models.TeamMember, error)
	GetByID(id int) (*models.TeamMember, error)
	GetActiveMembers() ([]models.TeamMember, error)
	Create(member *models.TeamMember) error
	Update(member *models.TeamMember) error
	Delete(id int) error
	Count() (int, error)
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
func (r *teamRepository) GetAll() ([]models.TeamMember, error) {
	query := `
		SELECT id, name, email, active, date_added 
		FROM team_members 
		ORDER BY name ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query team members: %w", err)
	}
	defer rows.Close()

	var members []models.TeamMember
	for rows.Next() {
		var member models.TeamMember
		err := rows.Scan(
			&member.ID,
			&member.Name,
			&member.Email,
			&member.Active,
			&member.DateAdded,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan team member: %w", err)
		}
		members = append(members, member)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating team members: %w", err)
	}

	return members, nil
}

// GetByID retrieves a team member by ID
func (r *teamRepository) GetByID(id int) (*models.TeamMember, error) {
	query := `
		SELECT id, name, email, active, date_added 
		FROM team_members 
		WHERE id = ?
	`

	var member models.TeamMember
	err := r.db.QueryRow(query, id).Scan(
		&member.ID,
		&member.Name,
		&member.Email,
		&member.Active,
		&member.DateAdded,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("team member with ID %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get team member: %w", err)
	}

	return &member, nil
}

// GetActiveMembers retrieves only active team members
func (r *teamRepository) GetActiveMembers() ([]models.TeamMember, error) {
	query := `
		SELECT id, name, email, active, date_added 
		FROM team_members 
		WHERE active = 1 
		ORDER BY date_added ASC, name ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active team members: %w", err)
	}
	defer rows.Close()

	var members []models.TeamMember
	for rows.Next() {
		var member models.TeamMember
		err := rows.Scan(
			&member.ID,
			&member.Name,
			&member.Email,
			&member.Active,
			&member.DateAdded,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan team member: %w", err)
		}
		members = append(members, member)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating active team members: %w", err)
	}

	return members, nil
}

// Create creates a new team member
func (r *teamRepository) Create(member *models.TeamMember) error {
	query := `
		INSERT INTO team_members (name, email, active, date_added) 
		VALUES (?, ?, ?, ?)
	`

	// Set default values
	if member.DateAdded.IsZero() {
		member.DateAdded = time.Now()
	}

	result, err := r.db.Exec(query, member.Name, member.Email, member.Active, member.DateAdded)
	if err != nil {
		return fmt.Errorf("failed to create team member: %w", err)
	}

	// Get the inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get inserted ID: %w", err)
	}

	member.ID = int(id)
	return nil
}

// Update updates an existing team member
func (r *teamRepository) Update(member *models.TeamMember) error {
	query := `
		UPDATE team_members 
		SET name = ?, email = ?, active = ? 
		WHERE id = ?
	`

	result, err := r.db.Exec(query, member.Name, member.Email, member.Active, member.ID)
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

	return nil
}

// Delete deletes a team member by ID
func (r *teamRepository) Delete(id int) error {
	query := `DELETE FROM team_members WHERE id = ?`

	result, err := r.db.Exec(query, id)
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
func (r *teamRepository) Count() (int, error) {
	query := `SELECT COUNT(*) FROM team_members`

	var count int
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count team members: %w", err)
	}

	return count, nil
}
