package controllers

import (
	"net/http"
	"strconv"

	"github.com/blogem/eod-scheduler/models"
	"github.com/blogem/eod-scheduler/services"
)

// WorkingHoursController handles working hours configuration requests
type WorkingHoursController struct {
	services *services.Services
}

// NewWorkingHoursController creates a new working hours controller
func NewWorkingHoursController(services *services.Services) *WorkingHoursController {
	return &WorkingHoursController{
		services: services,
	}
}

// Index handles GET /hours
func (c *WorkingHoursController) Index(w http.ResponseWriter, r *http.Request) {
	workingHours, err := c.services.WorkingHours.GetAllWorkingHours()
	if err != nil {
		http.Error(w, "Failed to load working hours: "+err.Error(), http.StatusInternalServerError)
		return
	}

	dayNames := c.services.WorkingHours.GetDayNames()

	templateData := struct {
		Title        string
		CurrentPage  string
		Error        string
		Success      string
		WorkingHours []models.WorkingHours
		DayNames     map[int]string
	}{
		Title:        "Working Hours Configuration",
		CurrentPage:  "hours",
		Error:        "",
		Success:      "",
		WorkingHours: workingHours,
		DayNames:     dayNames,
	}

	renderTemplate(w, "hours", "templates/hours.html", templateData)
}

// Update handles POST /hours
func (c *WorkingHoursController) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Parse form data into working hours forms for each day
	forms := make(map[int]*models.WorkingHoursForm)
	dayNames := c.services.WorkingHours.GetDayNames()

	for dayNum := range dayNames {
		activeKey := "active_" + strconv.Itoa(dayNum)
		startKey := "start_time_" + strconv.Itoa(dayNum)
		endKey := "end_time_" + strconv.Itoa(dayNum)

		isActive := r.FormValue(activeKey) == "on"
		startTime := r.FormValue(startKey)
		endTime := r.FormValue(endKey)

		forms[dayNum] = &models.WorkingHoursForm{
			DayOfWeek: dayNum,
			Active:    isActive,
			StartTime: startTime,
			EndTime:   endTime,
		}
	}

	err := c.services.WorkingHours.UpdateAllWorkingHours(forms)
	if err != nil {
		// Reload page with error
		workingHours, loadErr := c.services.WorkingHours.GetAllWorkingHours()
		if loadErr != nil {
			http.Error(w, "Failed to load working hours: "+loadErr.Error(), http.StatusInternalServerError)
			return
		}

		templateData := struct {
			Title        string
			CurrentPage  string
			Error        string
			Success      string
			WorkingHours []models.WorkingHours
			DayNames     map[int]string
		}{
			Title:        "Working Hours Configuration",
			CurrentPage:  "hours",
			Error:        err.Error(),
			Success:      "",
			WorkingHours: workingHours,
			DayNames:     dayNames,
		}

		renderTemplateWithStatus(w, http.StatusBadRequest, "hours_update_error", "templates/hours.html", templateData)
		return
	}

	// Redirect to hours page after successful update
	http.Redirect(w, r, "/hours", http.StatusSeeOther)
}
