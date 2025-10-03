package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/blogem/eod-scheduler/models"
	"github.com/blogem/eod-scheduler/services"
	"github.com/go-chi/chi/v5"
)

// TeamController handles team management requests
type TeamController struct {
	services *services.Services
}

// NewTeamController creates a new team controller
func NewTeamController(services *services.Services) *TeamController {
	return &TeamController{
		services: services,
	}
}

// Index handles GET /team
func (c *TeamController) Index(w http.ResponseWriter, r *http.Request) {
	members, err := c.services.Team.GetAllMembers()
	if err != nil {
		http.Error(w, "Failed to load team members: "+err.Error(), http.StatusInternalServerError)
		return
	}

	templateData := struct {
		Title       string
		CurrentPage string
		Error       string
		Success     string
		Members     []models.TeamMember
		Form        *models.TeamMemberForm
	}{
		Title:       "Team Management",
		CurrentPage: "team",
		Error:       "",
		Success:     "",
		Members:     members,
		Form:        &models.TeamMemberForm{Active: true}, // Default to active for new members
	}

	renderTemplate(w, "team", "templates/team.html", templateData)
}

// Create handles POST /team
func (c *TeamController) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get the last value for 'active' (checkbox will override hidden field if checked)
	activeValues := r.Form["active"]
	isActive := len(activeValues) > 0 && activeValues[len(activeValues)-1] == "on"

	form := &models.TeamMemberForm{
		Name:        r.FormValue("name"),
		SlackHandle: r.FormValue("slack_handle"),
		Active:      isActive,
	}

	_, err := c.services.Team.CreateMember(form)
	if err != nil {
		// Reload page with form data and error
		members, loadErr := c.services.Team.GetAllMembers()
		if loadErr != nil {
			http.Error(w, "Failed to load team members: "+loadErr.Error(), http.StatusInternalServerError)
			return
		}

		templateData := struct {
			Title       string
			CurrentPage string
			Error       string
			Success     string
			Members     []models.TeamMember
			Form        *models.TeamMemberForm
		}{
			Title:       "Team Management",
			CurrentPage: "team",
			Error:       err.Error(),
			Success:     "",
			Members:     members,
			Form:        form,
		}

		renderTemplateWithStatus(w, http.StatusBadRequest, "team_create_error", "templates/team.html", templateData)
		return
	}

	// Redirect to team page after successful creation
	http.Redirect(w, r, "/team", http.StatusSeeOther)
}

// Edit handles GET /team/{id}/edit
func (c *TeamController) Edit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid team member ID", http.StatusBadRequest)
		return
	}

	member, err := c.services.Team.GetMemberByID(id)
	if err != nil {
		http.Error(w, "Team member not found: "+err.Error(), http.StatusNotFound)
		return
	}

	form := &models.TeamMemberForm{
		Name:        member.Name,
		SlackHandle: member.SlackHandle,
		Active:      member.Active,
	}

	templateData := struct {
		Title       string
		CurrentPage string
		Error       string
		Success     string
		Member      *models.TeamMember
		Form        *models.TeamMemberForm
	}{
		Title:       "Edit Team Member",
		CurrentPage: "team",
		Error:       "",
		Success:     "",
		Member:      member,
		Form:        form,
	}

	renderTemplate(w, "team_edit", "templates/team_edit.html", templateData)
}

// Update handles POST /team/{id}
func (c *TeamController) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid team member ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Debug: Print all form values to see what's being sent
	fmt.Printf("Debug - All form values: %+v\n", r.Form)

	// Get the last value for 'active' (checkbox will override hidden field if checked)
	activeValues := r.Form["active"]
	isActive := len(activeValues) > 0 && activeValues[len(activeValues)-1] == "on"

	fmt.Printf("Debug - Active values: %v, isActive: %v\n", activeValues, isActive)

	form := &models.TeamMemberForm{
		Name:        r.FormValue("name"),
		SlackHandle: r.FormValue("slack_handle"),
		Active:      isActive,
	}

	_, err = c.services.Team.UpdateMember(id, form)
	if err != nil {
		// Reload edit page with form data and error
		member, loadErr := c.services.Team.GetMemberByID(id)
		if loadErr != nil {
			http.Error(w, "Team member not found: "+loadErr.Error(), http.StatusNotFound)
			return
		}

		templateData := struct {
			Title       string
			CurrentPage string
			Error       string
			Success     string
			Member      *models.TeamMember
			Form        *models.TeamMemberForm
		}{
			Title:       "Edit Team Member",
			CurrentPage: "team",
			Error:       err.Error(),
			Success:     "",
			Member:      member,
			Form:        form,
		}

		renderTemplateWithStatus(w, http.StatusBadRequest, "team_update_error", "templates/team_edit.html", templateData)
		return
	}

	// Redirect to team page after successful update
	http.Redirect(w, r, "/team", http.StatusSeeOther)
}

// Delete handles POST /team/{id}/delete
func (c *TeamController) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid team member ID", http.StatusBadRequest)
		return
	}

	if err := c.services.Team.DeleteMember(id); err != nil {
		// For delete errors, we'll redirect back with error in URL params
		// (In a real app, you might want to use sessions/flash messages)
		http.Redirect(w, r, "/team?error="+err.Error(), http.StatusSeeOther)
		return
	}

	// Redirect to team page after successful deletion
	http.Redirect(w, r, "/team", http.StatusSeeOther)
}
