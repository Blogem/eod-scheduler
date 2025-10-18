package controllers

import (
	"net/http"

	"github.com/blogem/eod-scheduler/services"
)

// DashboardController handles dashboard-related requests
type DashboardController struct {
	services *services.Services
}

// NewDashboardController creates a new dashboard controller
func NewDashboardController(services *services.Services) *DashboardController {
	return &DashboardController{
		services: services,
	}
}

// Index handles GET /
func (c *DashboardController) Index(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated
	user := getUserNickname(r)

	// If not authenticated, show a landing page
	if user == "" {
		c.showLandingPage(w, r)
		return
	}

	// User is authenticated, show dashboard
	data, err := c.services.Schedule.GetDashboardData(r.Context())
	if err != nil {
		http.Error(w, "Failed to load dashboard data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Check for logout success message
	successMsg := ""
	if r.URL.Query().Get("logged_out") == "true" {
		successMsg = "You have been logged out successfully"
	}

	templateData := struct {
		Title       string
		CurrentPage string
		Error       string
		Success     string
		Data        *services.DashboardData
		User        string
	}{
		Title:       "EoD Scheduler Dashboard",
		CurrentPage: "dashboard",
		Error:       "",
		Success:     successMsg,
		Data:        data,
		User:        user,
	}

	renderTemplate(w, "dashboard", "templates/dashboard.html", templateData)
}

// showLandingPage displays a landing page for unauthenticated users
func (c *DashboardController) showLandingPage(w http.ResponseWriter, r *http.Request) {
	// Check for logout success message
	successMsg := ""
	if r.URL.Query().Get("logged_out") == "true" {
		successMsg = "You have been logged out successfully"
	}

	templateData := struct {
		Title       string
		CurrentPage string
		Error       string
		Success     string
		Data        interface{}
		User        string
	}{
		Title:       "Welcome to EoD Scheduler",
		CurrentPage: "home",
		Error:       "",
		Success:     successMsg,
		Data:        nil,
		User:        "",
	}

	renderTemplate(w, "landing", "templates/landing.html", templateData)
}
