package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/blogem/eod-scheduler/models"
	"github.com/blogem/eod-scheduler/repositories"
)

// AuditLogger middleware logs all POST/PUT/DELETE requests
func AuditLogger(auditRepo repositories.AuditRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only log mutation operations
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete {
				// Create audit log entry
				entry := &models.AuditLogEntry{
					UserEmail: GetUserEmail(r.Context()),
					Method:    r.Method,
					Path:      r.URL.Path,
					UserAgent: r.UserAgent(),
					IPAddress: getIPAddress(r),
					FormData:  captureFormData(r),
				}

				// Log asynchronously to avoid blocking request
				go func() {
					err := auditRepo.Create(entry)
					if err != nil {
						log.Printf("Failed to create audit log: %v", err)
					}
				}()
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getIPAddress extracts IP address from request, checking X-Forwarded-For first
func getIPAddress(r *http.Request) string {
	// Check X-Forwarded-For header (proxy/load balancer)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take first IP if multiple
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// captureFormData captures form data as JSON string
func captureFormData(r *http.Request) string {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		return ""
	}

	// Convert to map
	formMap := make(map[string]interface{})
	for key, values := range r.Form {
		if len(values) == 1 {
			formMap[key] = values[0]
		} else {
			formMap[key] = values
		}
	}

	// Convert to JSON
	jsonData, err := json.Marshal(formMap)
	if err != nil {
		return ""
	}

	return string(jsonData)
}
