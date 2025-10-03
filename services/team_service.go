package services

import (
	"fmt"
	"strings"

	"github.com/blogem/eod-scheduler/models"
	"github.com/blogem/eod-scheduler/repositories"
)

// TeamService interface defines team management business logic
type TeamService interface {
	GetAllMembers() ([]models.TeamMember, error)
	GetMemberByID(id int) (*models.TeamMember, error)
	GetActiveMembers() ([]models.TeamMember, error)
	CreateMember(form *models.TeamMemberForm) (*models.TeamMember, error)
	UpdateMember(id int, form *models.TeamMemberForm) (*models.TeamMember, error)
	DeleteMember(id int) error
	DeactivateMember(id int) error
	ActivateMember(id int) error
	GetMemberCount() (int, error)
	ValidateDeleteMember(id int) error
}

// teamService implements TeamService interface
type teamService struct {
	teamRepo     repositories.TeamRepository
	scheduleRepo repositories.ScheduleRepository
}

// NewTeamService creates a new team service
func NewTeamService(teamRepo repositories.TeamRepository, scheduleRepo repositories.ScheduleRepository) TeamService {
	return &teamService{
		teamRepo:     teamRepo,
		scheduleRepo: scheduleRepo,
	}
}

// GetAllMembers retrieves all team members
func (s *teamService) GetAllMembers() ([]models.TeamMember, error) {
	return s.teamRepo.GetAll()
}

// GetMemberByID retrieves a team member by ID
func (s *teamService) GetMemberByID(id int) (*models.TeamMember, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid team member ID: %d", id)
	}
	return s.teamRepo.GetByID(id)
}

// GetActiveMembers retrieves only active team members
func (s *teamService) GetActiveMembers() ([]models.TeamMember, error) {
	return s.teamRepo.GetActiveMembers()
}

// CreateMember creates a new team member with validation
func (s *teamService) CreateMember(form *models.TeamMemberForm) (*models.TeamMember, error) {
	// Validate form
	if errors := form.Validate(); len(errors) > 0 {
		return nil, fmt.Errorf("validation failed: %s", strings.Join(errors, ", "))
	}

	// Check for duplicate slack handle (if slack handle provided)
	if form.SlackHandle != "" {
		existing, err := s.findMemberBySlackHandle(form.SlackHandle)
		if err == nil && existing != nil {
			return nil, fmt.Errorf("team member with slack handle %s already exists", form.SlackHandle)
		}
	}

	// Create new member
	member := &models.TeamMember{
		Name:        strings.TrimSpace(form.Name),
		SlackHandle: strings.TrimSpace(form.SlackHandle),
		Active:      form.Active,
	}

	if err := s.teamRepo.Create(member); err != nil {
		return nil, fmt.Errorf("failed to create team member: %w", err)
	}

	return member, nil
}

// UpdateMember updates an existing team member
func (s *teamService) UpdateMember(id int, form *models.TeamMemberForm) (*models.TeamMember, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid team member ID: %d", id)
	}

	// Validate form
	if errors := form.Validate(); len(errors) > 0 {
		return nil, fmt.Errorf("validation failed: %s", strings.Join(errors, ", "))
	}

	// Get existing member
	member, err := s.teamRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("team member not found: %w", err)
	}

	// Check for duplicate slack handle (if slack handle changed and provided)
	if form.SlackHandle != "" && form.SlackHandle != member.SlackHandle {
		existing, err := s.findMemberBySlackHandle(form.SlackHandle)
		if err == nil && existing != nil && existing.ID != id {
			return nil, fmt.Errorf("team member with slack handle %s already exists", form.SlackHandle)
		}
	}

	// Update member fields
	member.Name = strings.TrimSpace(form.Name)
	member.SlackHandle = strings.TrimSpace(form.SlackHandle)
	member.Active = form.Active

	if err := s.teamRepo.Update(member); err != nil {
		return nil, fmt.Errorf("failed to update team member: %w", err)
	}

	return member, nil
}

// DeleteMember permanently deletes a team member
func (s *teamService) DeleteMember(id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid team member ID: %d", id)
	}

	// Validate deletion is allowed
	if err := s.ValidateDeleteMember(id); err != nil {
		return err
	}

	if err := s.teamRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete team member: %w", err)
	}

	return nil
}

// DeactivateMember deactivates a team member (soft delete)
func (s *teamService) DeactivateMember(id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid team member ID: %d", id)
	}

	member, err := s.teamRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("team member not found: %w", err)
	}

	if !member.Active {
		return fmt.Errorf("team member is already inactive")
	}

	member.Active = false
	if err := s.teamRepo.Update(member); err != nil {
		return fmt.Errorf("failed to deactivate team member: %w", err)
	}

	return nil
}

// ActivateMember activates a team member
func (s *teamService) ActivateMember(id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid team member ID: %d", id)
	}

	member, err := s.teamRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("team member not found: %w", err)
	}

	if member.Active {
		return fmt.Errorf("team member is already active")
	}

	member.Active = true
	if err := s.teamRepo.Update(member); err != nil {
		return fmt.Errorf("failed to activate team member: %w", err)
	}

	return nil
}

// GetMemberCount returns the total number of team members
func (s *teamService) GetMemberCount() (int, error) {
	return s.teamRepo.Count()
}

// ValidateDeleteMember checks if a team member can be safely deleted
func (s *teamService) ValidateDeleteMember(id int) error {
	// Check if member exists
	_, err := s.teamRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("team member not found: %w", err)
	}

	// Check if member has future schedule entries
	hasFuture, err := s.scheduleRepo.HasFutureEntries(id)
	if err != nil {
		return fmt.Errorf("failed to check future entries: %w", err)
	}

	if hasFuture {
		return fmt.Errorf("cannot delete team member with future schedule assignments. Consider deactivating instead")
	}

	// Check if this is the last active member
	activeMembers, err := s.teamRepo.GetActiveMembers()
	if err != nil {
		return fmt.Errorf("failed to check active members: %w", err)
	}

	// Count active members excluding the one being deleted
	activeCount := 0
	for _, member := range activeMembers {
		if member.ID != id {
			activeCount++
		}
	}

	if activeCount == 0 {
		return fmt.Errorf("cannot delete the last team member. At least one team member must remain")
	}

	return nil
}

// findMemberBySlackHandle finds a team member by slack handle (helper function)
func (s *teamService) findMemberBySlackHandle(slackHandle string) (*models.TeamMember, error) {
	if slackHandle == "" {
		return nil, fmt.Errorf("slack handle is empty")
	}

	members, err := s.teamRepo.GetAll()
	if err != nil {
		return nil, err
	}

	for _, member := range members {
		if strings.EqualFold(member.SlackHandle, slackHandle) {
			return &member, nil
		}
	}

	return nil, fmt.Errorf("no team member found with slack handle: %s", slackHandle)
}
