package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/blogem/eod-scheduler/models"
	"github.com/blogem/eod-scheduler/services"
	"github.com/go-chi/chi/v5"
)

// ScheduleController handles schedule-related requests
type ScheduleController struct {
	services *services.Services
}

// NewScheduleController creates a new schedule controller
func NewScheduleController(services *services.Services) *ScheduleController {
	return &ScheduleController{
		services: services,
	}
}

// Index handles GET /schedule
func (c *ScheduleController) Index(w http.ResponseWriter, r *http.Request) {
	// Get current week by default
	currentWeek := models.GetCurrentWeek()
	weeklySchedule, err := c.services.Schedule.GetWeeklySchedule(currentWeek.Start)
	if err != nil {
		http.Error(w, "Failed to load schedule: "+err.Error(), http.StatusInternalServerError)
		return
	}

	templateData := struct {
		Title       string
		CurrentPage string
		Error       string
		Success     string
		Schedule    *models.WeekView
		CurrentURL  string
		User        string
	}{
		Title:       "Schedule",
		CurrentPage: "schedule",
		Error:       r.URL.Query().Get("error"),
		Success:     r.URL.Query().Get("success"),
		Schedule:    weeklySchedule,
		CurrentURL:  r.URL.Path,
		User:        getUserNickname(r),
	}

	renderTemplate(w, "schedule", "templates/schedule.html", templateData)
}

// Week handles GET /schedule/week/{date}
func (c *ScheduleController) Week(w http.ResponseWriter, r *http.Request) {
	dateStr := chi.URLParam(r, "date")

	date, err := models.ParseDate(dateStr)
	if err != nil {
		http.Error(w, "Invalid date format: "+err.Error(), http.StatusBadRequest)
		return
	}

	weeklySchedule, err := c.services.Schedule.GetWeeklySchedule(date)
	if err != nil {
		http.Error(w, "Failed to load schedule: "+err.Error(), http.StatusInternalServerError)
		return
	}

	templateData := struct {
		Title       string
		CurrentPage string
		Error       string
		Success     string
		Schedule    *models.WeekView
		CurrentURL  string
		User        string
	}{
		Title:       "Schedule - Week of " + models.FormatDate(date),
		CurrentPage: "schedule",
		Error:       "",
		Success:     "",
		Schedule:    weeklySchedule,
		CurrentURL:  r.URL.Path,
		User:        getUserNickname(r),
	}

	renderTemplate(w, "schedule_week", "templates/schedule.html", templateData)
}

// Generate handles POST /schedule/generate
func (c *ScheduleController) Generate(w http.ResponseWriter, r *http.Request) {
	// Always force regenerate - simplifies the interface
	result, err := c.services.Schedule.GenerateSchedule(true)
	if err != nil {
		http.Error(w, "Failed to generate schedule: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect back to schedule page with generation result
	redirectURL := "/schedule"
	if !result.Success {
		redirectURL += "?error=" + result.Message
	} else {
		redirectURL += "?success=" + result.Message
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// ShowTakeoverForm handles GET /schedule/takeover
func (c *ScheduleController) ShowTakeoverForm(w http.ResponseWriter, r *http.Request) {
	// Get all active team members for the dropdown
	teamMembers, err := c.services.Team.GetActiveMembers()
	if err != nil {
		http.Error(w, "Failed to load team members: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get existing schedule entries from today for the next 14 days
	today := time.Now().Truncate(24 * time.Hour)
	endDate := today.AddDate(0, 0, 14)
	entries, err := c.services.Schedule.GetScheduleByDateRange(today, endDate)
	if err != nil {
		http.Error(w, "Failed to load schedule entries: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if there's a specific entry to prefill
	form := &models.TakeoverForm{}
	entryParam := r.URL.Query().Get("entry")
	if entryParam != "" {
		if entryID, err := strconv.Atoi(entryParam); err == nil {
			// Validate that this entry exists in our upcoming entries
			for _, entry := range entries {
				if entry.ID == entryID {
					form.ScheduleEntryID = entryID
					break
				}
			}
		}
	}

	templateData := struct {
		Title       string
		CurrentPage string
		Error       string
		Success     string
		TeamMembers []models.TeamMember
		Entries     []models.ScheduleEntry
		Form        *models.TakeoverForm
		Redirect    string
		User        string
	}{
		Title:       "Take Over Shift",
		CurrentPage: "schedule",
		Error:       "",
		Success:     "",
		TeamMembers: teamMembers,
		Entries:     entries,
		Form:        form,
		Redirect:    r.URL.Query().Get("redirect"),
		User:        getUserNickname(r),
	}

	renderTemplate(w, "schedule_takeover", "templates/schedule_takeover.html", templateData)
}

// CreateTakeover handles POST /schedule/takeover
func (c *ScheduleController) CreateTakeover(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	scheduleEntryID, err := strconv.Atoi(r.FormValue("schedule_entry_id"))
	if err != nil {
		http.Error(w, "Invalid schedule entry ID", http.StatusBadRequest)
		return
	}

	newTeamMemberID, err := strconv.Atoi(r.FormValue("new_team_member_id"))
	if err != nil {
		http.Error(w, "Invalid team member ID", http.StatusBadRequest)
		return
	}

	form := &models.TakeoverForm{
		ScheduleEntryID: scheduleEntryID,
		NewTeamMemberID: newTeamMemberID,
	}

	// Validate the form
	if errors := form.Validate(); len(errors) > 0 {
		// Reload form with error
		teamMembers, loadErr := c.services.Team.GetActiveMembers()
		if loadErr != nil {
			http.Error(w, "Failed to load team members: "+loadErr.Error(), http.StatusInternalServerError)
			return
		}

		today := time.Now().Truncate(24 * time.Hour)
		endDate := today.AddDate(0, 0, 14)
		entries, loadErr := c.services.Schedule.GetScheduleByDateRange(today, endDate)
		if loadErr != nil {
			http.Error(w, "Failed to load schedule entries: "+loadErr.Error(), http.StatusInternalServerError)
			return
		}

		templateData := struct {
			Title       string
			CurrentPage string
			Error       string
			Success     string
			TeamMembers []models.TeamMember
			Entries     []models.ScheduleEntry
			Form        *models.TakeoverForm
			Redirect    string
			User        string
		}{
			Title:       "Take Over Shift",
			CurrentPage: "schedule",
			Error:       strings.Join(errors, ", "),
			Success:     "",
			TeamMembers: teamMembers,
			Entries:     entries,
			Form:        form,
			Redirect:    r.FormValue("redirect"),
			User:        getUserNickname(r),
		}

		renderTemplateWithStatus(w, http.StatusBadRequest, "schedule_takeover_error", "templates/schedule_takeover.html", templateData)
		return
	}

	// Process the takeover by updating the existing schedule entry
	entry, err := c.services.Schedule.GetScheduleEntry(scheduleEntryID)
	if err != nil {
		http.Error(w, "Schedule entry not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Update the schedule entry with the new team member
	updateForm := &models.ScheduleEntryForm{
		Date:         entry.Date.Format("2006-01-02"),
		TeamMemberID: newTeamMemberID,
		StartTime:    entry.StartTime,
		EndTime:      entry.EndTime,
	}

	_, err = c.services.Schedule.CreateManualOverride(scheduleEntryID, updateForm)
	if err != nil {
		http.Error(w, "Failed to process takeover: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to originating page or schedule page by default after successful takeover
	redirectURL := r.FormValue("redirect")
	if redirectURL == "" {
		redirectURL = "/schedule"
	}
	// Add success message as URL parameter
	if redirectURL == "/" {
		redirectURL += "?success=Shift takeover completed successfully"
	} else {
		redirectURL += "?success=Shift takeover completed successfully"
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// ShowEditForm handles GET /schedule/edit/{id}
func (c *ScheduleController) ShowEditForm(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid schedule entry ID", http.StatusBadRequest)
		return
	}

	entry, err := c.services.Schedule.GetScheduleEntry(id)
	if err != nil {
		http.Error(w, "Schedule entry not found: "+err.Error(), http.StatusNotFound)
		return
	}

	teamMembers, err := c.services.Team.GetActiveMembers()
	if err != nil {
		http.Error(w, "Failed to load team members: "+err.Error(), http.StatusInternalServerError)
		return
	}

	form := &models.ScheduleEntryForm{
		Date:         models.FormatDate(entry.Date),
		TeamMemberID: entry.TeamMemberID,
		StartTime:    entry.StartTime,
		EndTime:      entry.EndTime,
	}

	templateData := struct {
		Title       string
		CurrentPage string
		Error       string
		Success     string
		Entry       *models.ScheduleEntry
		TeamMembers []models.TeamMember
		Form        *models.ScheduleEntryForm
		Redirect    string
		User        string
	}{
		Title:       "Edit Schedule Entry",
		CurrentPage: "schedule",
		Error:       "",
		Success:     "",
		Entry:       entry,
		TeamMembers: teamMembers,
		Form:        form,
		Redirect:    r.URL.Query().Get("redirect"),
		User:        getUserNickname(r),
	}

	renderTemplate(w, "schedule_edit", "templates/schedule_edit.html", templateData)
}

// UpdateEntry handles POST /schedule/edit/{id}
func (c *ScheduleController) UpdateEntry(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid schedule entry ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	teamMemberID, err := strconv.Atoi(r.FormValue("team_member_id"))
	if err != nil {
		http.Error(w, "Invalid team member ID", http.StatusBadRequest)
		return
	}

	form := &models.ScheduleEntryForm{
		Date:         r.FormValue("date"),
		TeamMemberID: teamMemberID,
		StartTime:    r.FormValue("start_time"),
		EndTime:      r.FormValue("end_time"),
	}

	_, err = c.services.Schedule.UpdateScheduleEntry(id, form)
	if err != nil {
		// Reload form with error
		entry, loadErr := c.services.Schedule.GetScheduleEntry(id)
		if loadErr != nil {
			http.Error(w, "Schedule entry not found: "+loadErr.Error(), http.StatusNotFound)
			return
		}

		teamMembers, loadErr := c.services.Team.GetActiveMembers()
		if loadErr != nil {
			http.Error(w, "Failed to load team members: "+loadErr.Error(), http.StatusInternalServerError)
			return
		}

		templateData := struct {
			Title       string
			CurrentPage string
			Error       string
			Success     string
			Entry       *models.ScheduleEntry
			TeamMembers []models.TeamMember
			Form        *models.ScheduleEntryForm
			Redirect    string
			User        string
		}{
			Title:       "Edit Schedule Entry",
			CurrentPage: "schedule",
			Error:       err.Error(),
			Success:     "",
			Entry:       entry,
			TeamMembers: teamMembers,
			Form:        form,
			Redirect:    r.FormValue("redirect"),
			User:        getUserNickname(r),
		}

		renderTemplateWithStatus(w, http.StatusBadRequest, "schedule_edit_error", "templates/schedule_edit.html", templateData)
		return
	}

	// Redirect to originating page or schedule page by default
	redirectURL := r.FormValue("redirect")
	if redirectURL == "" {
		redirectURL = "/schedule"
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// RemoveOverride handles POST /schedule/remove/{id}
func (c *ScheduleController) RemoveOverride(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid schedule entry ID", http.StatusBadRequest)
		return
	}

	if err := c.services.Schedule.RemoveManualOverride(id); err != nil {
		// Redirect back with error to originating page or schedule page
		redirectURL := r.FormValue("redirect")
		if redirectURL == "" {
			redirectURL = "/schedule"
		}
		http.Redirect(w, r, redirectURL+"?error="+err.Error(), http.StatusSeeOther)
		return
	}

	// Redirect to originating page or schedule page by default after successful removal
	redirectURL := r.FormValue("redirect")
	if redirectURL == "" {
		redirectURL = "/schedule"
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
