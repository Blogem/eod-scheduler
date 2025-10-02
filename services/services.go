package services

import (
	"github.com/blogem/eod-scheduler/repositories"
)

// Services holds all service instances
type Services struct {
	Team         TeamService
	WorkingHours WorkingHoursService
	Schedule     ScheduleService
}

// NewServices creates and initializes all service instances
func NewServices(repos *repositories.Repositories) *Services {
	return &Services{
		Team:         NewTeamService(repos.Team, repos.Schedule),
		WorkingHours: NewWorkingHoursService(repos.WorkingHours),
		Schedule:     NewScheduleService(repos.Schedule, repos.Team, repos.WorkingHours),
	}
}
