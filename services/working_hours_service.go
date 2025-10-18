package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/blogem/eod-scheduler/models"
	"github.com/blogem/eod-scheduler/repositories"
)

// WorkingHoursService interface defines working hours business logic
type WorkingHoursService interface {
	GetAllWorkingHours(ctx context.Context) ([]models.WorkingHours, error)
	GetWorkingHoursByDay(ctx context.Context, dayOfWeek int) (*models.WorkingHours, error)
	GetActiveDays(ctx context.Context) ([]models.WorkingHours, error)
	UpdateWorkingHours(ctx context.Context, dayOfWeek int, form *models.WorkingHoursForm) (*models.WorkingHours, error)
	UpdateAllWorkingHours(ctx context.Context, forms map[int]*models.WorkingHoursForm) error
	IsWorkingDay(ctx context.Context, dayOfWeek int) (bool, error)
	GetDayNames() map[int]string
}

// workingHoursService implements WorkingHoursService interface
type workingHoursService struct {
	workingHoursRepo repositories.WorkingHoursRepository
}

// NewWorkingHoursService creates a new working hours service
func NewWorkingHoursService(workingHoursRepo repositories.WorkingHoursRepository) WorkingHoursService {
	return &workingHoursService{
		workingHoursRepo: workingHoursRepo,
	}
}

// GetAllWorkingHours retrieves all working hours configurations
func (s *workingHoursService) GetAllWorkingHours(ctx context.Context) ([]models.WorkingHours, error) {
	return s.workingHoursRepo.GetAll(ctx)
}

// GetWorkingHoursByDay retrieves working hours for a specific day
func (s *workingHoursService) GetWorkingHoursByDay(ctx context.Context, dayOfWeek int) (*models.WorkingHours, error) {
	if dayOfWeek < 0 || dayOfWeek > 6 {
		return nil, fmt.Errorf("invalid day of week: %d (must be 0-6)", dayOfWeek)
	}
	return s.workingHoursRepo.GetByDay(ctx, dayOfWeek)
}

// GetActiveDays retrieves only active working days
func (s *workingHoursService) GetActiveDays(ctx context.Context) ([]models.WorkingHours, error) {
	return s.workingHoursRepo.GetActiveDays(ctx)
}

// UpdateWorkingHours updates working hours for a specific day
func (s *workingHoursService) UpdateWorkingHours(ctx context.Context, dayOfWeek int, form *models.WorkingHoursForm) (*models.WorkingHours, error) {
	if dayOfWeek < 0 || dayOfWeek > 6 {
		return nil, fmt.Errorf("invalid day of week: %d (must be 0-6)", dayOfWeek)
	}

	// Validate form
	if errors := form.Validate(); len(errors) > 0 {
		return nil, fmt.Errorf("validation failed: %s", strings.Join(errors, ", "))
	}

	// Get existing working hours
	existing, err := s.workingHoursRepo.GetByDay(ctx, dayOfWeek)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing working hours: %w", err)
	}

	// Update fields
	existing.StartTime = strings.TrimSpace(form.StartTime)
	existing.EndTime = strings.TrimSpace(form.EndTime)
	existing.Active = form.Active

	// If deactivating, set times to 00:00
	if !form.Active {
		existing.StartTime = "00:00"
		existing.EndTime = "00:00"
	}

	if err := s.workingHoursRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update working hours: %w", err)
	}

	return existing, nil
}

// UpdateAllWorkingHours updates working hours for multiple days
func (s *workingHoursService) UpdateAllWorkingHours(ctx context.Context, forms map[int]*models.WorkingHoursForm) error {
	// Validate all forms first
	for dayOfWeek, form := range forms {
		if dayOfWeek < 0 || dayOfWeek > 6 {
			return fmt.Errorf("invalid day of week: %d (must be 0-6)", dayOfWeek)
		}

		if errors := form.Validate(); len(errors) > 0 {
			dayName := models.DayNames[dayOfWeek]
			return fmt.Errorf("validation failed for %s: %s", dayName, strings.Join(errors, ", "))
		}
	}

	// Check that at least one day is active
	hasActiveDay := false
	for _, form := range forms {
		if form.Active {
			hasActiveDay = true
			break
		}
	}

	if !hasActiveDay {
		return fmt.Errorf("at least one working day must be active")
	}

	// Update all working hours
	for dayOfWeek, form := range forms {
		_, err := s.UpdateWorkingHours(ctx, dayOfWeek, form)
		if err != nil {
			dayName := models.DayNames[dayOfWeek]
			return fmt.Errorf("failed to update %s: %w", dayName, err)
		}
	}

	return nil
}

// IsWorkingDay checks if a specific day is a working day
func (s *workingHoursService) IsWorkingDay(ctx context.Context, dayOfWeek int) (bool, error) {
	if dayOfWeek < 0 || dayOfWeek > 6 {
		return false, fmt.Errorf("invalid day of week: %d (must be 0-6)", dayOfWeek)
	}

	workingHours, err := s.workingHoursRepo.GetByDay(ctx, dayOfWeek)
	if err != nil {
		return false, fmt.Errorf("failed to get working hours: %w", err)
	}

	return workingHours.Active, nil
}

// GetDayNames returns the mapping of day numbers to names
func (s *workingHoursService) GetDayNames() map[int]string {
	return models.DayNames
}
