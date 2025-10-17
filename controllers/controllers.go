package controllers

import (
	"html/template"
	"net/http"

	"github.com/blogem/eod-scheduler/services"
)

// renderTemplate creates a template set and renders it with the provided data
func renderTemplate(w http.ResponseWriter, templateName string, pageTemplate string, data interface{}) error {
	return renderTemplateWithStatus(w, http.StatusOK, templateName, pageTemplate, data)
}

// renderTemplateWithStatus creates a template set and renders it with the provided data and status code
func renderTemplateWithStatus(w http.ResponseWriter, statusCode int, templateName string, pageTemplate string, data interface{}) error {
	// Create a new template set with only the templates we need
	tmpl := template.New(templateName)
	tmpl.Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"eq":  func(a, b interface{}) bool { return a == b },
	})

	// Parse layout and page template
	_, err := tmpl.ParseFiles("templates/layout.html", pageTemplate)
	if err != nil {
		http.Error(w, "Failed to parse template: "+err.Error(), http.StatusInternalServerError)
		return err
	}

	// Set status code if not OK
	if statusCode != http.StatusOK {
		w.WriteHeader(statusCode)
	}

	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, "Failed to render template: "+err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil
}

// Controllers holds all controller instances
type Controllers struct {
	Auth         *AuthController
	Dashboard    *DashboardController
	Team         *TeamController
	WorkingHours *WorkingHoursController
	Schedule     *ScheduleController
}

// NewControllers creates and initializes all controller instances
func NewControllers(services *services.Services) *Controllers {
	return &Controllers{
		Auth:         NewAuthController(),
		Dashboard:    NewDashboardController(services),
		Team:         NewTeamController(services),
		WorkingHours: NewWorkingHoursController(services),
		Schedule:     NewScheduleController(services),
	}
}
