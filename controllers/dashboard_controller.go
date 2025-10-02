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
	data, err := c.services.Schedule.GetDashboardData()
	if err != nil {
		http.Error(w, "Failed to load dashboard data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	templateData := struct {
		Title       string
		CurrentPage string
		Error       string
		Success     string
		Data        *services.DashboardData
	}{
		Title:       "EoD Scheduler Dashboard",
		CurrentPage: "dashboard",
		Error:       "",
		Success:     "",
		Data:        data,
	}

	renderTemplate(w, "dashboard", "templates/dashboard.html", templateData)
}
