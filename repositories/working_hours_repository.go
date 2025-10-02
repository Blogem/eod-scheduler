package repositories

import (
	"database/sql"
	"fmt"

	"github.com/blogem/eod-scheduler/models"
)

// WorkingHoursRepository interface defines working hours database operations
type WorkingHoursRepository interface {
	GetAll() ([]models.WorkingHours, error)
	GetByDay(dayOfWeek int) (*models.WorkingHours, error)
	GetActiveDays() ([]models.WorkingHours, error)
	Update(hours *models.WorkingHours) error
	UpdateByDay(dayOfWeek int, startTime, endTime string, active bool) error
}

// workingHoursRepository implements WorkingHoursRepository interface
type workingHoursRepository struct {
	db *sql.DB
}

// NewWorkingHoursRepository creates a new working hours repository
func NewWorkingHoursRepository(db *sql.DB) WorkingHoursRepository {
	return &workingHoursRepository{db: db}
}

// GetAll retrieves all working hours configurations
func (r *workingHoursRepository) GetAll() ([]models.WorkingHours, error) {
	query := `
		SELECT id, day_of_week, start_time, end_time, active 
		FROM working_hours 
		ORDER BY day_of_week ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query working hours: %w", err)
	}
	defer rows.Close()

	var hours []models.WorkingHours
	for rows.Next() {
		var hour models.WorkingHours
		err := rows.Scan(
			&hour.ID,
			&hour.DayOfWeek,
			&hour.StartTime,
			&hour.EndTime,
			&hour.Active,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan working hours: %w", err)
		}
		hours = append(hours, hour)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating working hours: %w", err)
	}

	return hours, nil
}

// GetByDay retrieves working hours for a specific day
func (r *workingHoursRepository) GetByDay(dayOfWeek int) (*models.WorkingHours, error) {
	query := `
		SELECT id, day_of_week, start_time, end_time, active 
		FROM working_hours 
		WHERE day_of_week = ?
	`

	var hour models.WorkingHours
	err := r.db.QueryRow(query, dayOfWeek).Scan(
		&hour.ID,
		&hour.DayOfWeek,
		&hour.StartTime,
		&hour.EndTime,
		&hour.Active,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("working hours for day %d not found", dayOfWeek)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get working hours: %w", err)
	}

	return &hour, nil
}

// GetActiveDays retrieves only active working days
func (r *workingHoursRepository) GetActiveDays() ([]models.WorkingHours, error) {
	query := `
		SELECT id, day_of_week, start_time, end_time, active 
		FROM working_hours 
		WHERE active = 1 
		ORDER BY day_of_week ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active working hours: %w", err)
	}
	defer rows.Close()

	var hours []models.WorkingHours
	for rows.Next() {
		var hour models.WorkingHours
		err := rows.Scan(
			&hour.ID,
			&hour.DayOfWeek,
			&hour.StartTime,
			&hour.EndTime,
			&hour.Active,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan working hours: %w", err)
		}
		hours = append(hours, hour)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating active working hours: %w", err)
	}

	return hours, nil
}

// Update updates existing working hours
func (r *workingHoursRepository) Update(hours *models.WorkingHours) error {
	query := `
		UPDATE working_hours 
		SET start_time = ?, end_time = ?, active = ? 
		WHERE day_of_week = ?
	`

	result, err := r.db.Exec(query, hours.StartTime, hours.EndTime, hours.Active, hours.DayOfWeek)
	if err != nil {
		return fmt.Errorf("failed to update working hours: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("working hours for day %d not found", hours.DayOfWeek)
	}

	return nil
}

// UpdateByDay updates working hours for a specific day
func (r *workingHoursRepository) UpdateByDay(dayOfWeek int, startTime, endTime string, active bool) error {
	query := `
		UPDATE working_hours 
		SET start_time = ?, end_time = ?, active = ? 
		WHERE day_of_week = ?
	`

	result, err := r.db.Exec(query, startTime, endTime, active, dayOfWeek)
	if err != nil {
		return fmt.Errorf("failed to update working hours for day %d: %w", dayOfWeek, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("working hours for day %d not found", dayOfWeek)
	}

	return nil
}
