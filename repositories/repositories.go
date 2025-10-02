package repositories

import (
	"database/sql"
)

// Repositories struct holds all repository interfaces
type Repositories struct {
	Team         TeamRepository
	WorkingHours WorkingHoursRepository
	Schedule     ScheduleRepository
}

// NewRepositories creates and initializes all repositories
func NewRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		Team:         NewTeamRepository(db),
		WorkingHours: NewWorkingHoursRepository(db),
		Schedule:     NewScheduleRepository(db),
	}
}
