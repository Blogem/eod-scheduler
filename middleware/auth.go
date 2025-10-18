package middleware

import (
	"net/http"

	"gitea.com/go-chi/session"
	"github.com/blogem/eod-scheduler/userctx"
)

// GetUserIDFromSession retrieves the user ID from session
func GetUserIDFromSession(r *http.Request) string {
	sess := session.GetSession(r)
	if userID := sess.Get("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

// GetUserEmailFromSession retrieves the user email from session
func GetUserEmailFromSession(r *http.Request) string {
	sess := session.GetSession(r)
	if email := sess.Get("user_email"); email != nil {
		if e, ok := email.(string); ok {
			return e
		}
	}
	return ""
}

// UserContext middleware extracts user from session and adds to context
func UserContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user email from session
		email := GetUserEmailFromSession(r)
		if email != "" {
			ctx := userctx.SetUserEmail(r.Context(), email)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
} // RequireAuth ensures the user is authenticated
// If not authenticated, redirects to /login and stores the intended destination
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := session.GetSession(r)
		userID := sess.Get("user_id")

		if userID == nil {
			// Store the intended destination for redirect after login
			sess.Set("redirect_after_login", r.URL.Path)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Add user ID to request context for use in handlers
		if id, ok := userID.(string); ok {
			ctx := userctx.SetUserID(r.Context(), id)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}
