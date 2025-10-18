package middleware

import (
	"context"
	"net/http"

	"gitea.com/go-chi/session"
)

type contextKey string

const UserIDContextKey contextKey = "user_id"

// RequireAuth ensures the user is authenticated
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
		ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext retrieves the user ID from request context
func GetUserIDFromContext(ctx context.Context) interface{} {
	return ctx.Value(UserIDContextKey)
}
