package controllers

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"gitea.com/go-chi/session"
	"github.com/blogem/eod-scheduler/authenticator"
)

type AuthController struct{}

func NewAuthController() *AuthController {
	return &AuthController{}
}

// Login initiates the authentication process
func (ac *AuthController) Login(auth authenticator.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Generate random state
		state, err := generateRandomState()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Save the state in the session to validate in callback
		sess := session.GetSession(r)
		sess.Set("state", state)

		// Redirect to OAuth provider login page
		http.Redirect(w, r, auth.GetAuthURL(state), http.StatusTemporaryRedirect)
	}
}

// Callback handles the callback from the OAuth provider
func (ac *AuthController) Callback(auth authenticator.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get session
		sess := session.GetSession(r)

		// Verify state
		storedState := sess.Get("state")
		if storedState == nil {
			http.Error(w, "State not found in session", http.StatusBadRequest)
			return
		}

		if r.URL.Query().Get("state") != storedState.(string) {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		// Exchange the code for a token
		token, err := auth.ExchangeCode(r.Context(), r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "Failed to exchange authorization code: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Get user claims
		claims, err := auth.GetClaims(r.Context(), token)
		if err != nil {
			http.Error(w, "Failed to get user claims: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Store the user session
		sess.Set("user_id", claims["sub"].(string))

		// Try to get display name from claims
		var displayName string
		for _, key := range []string{"nickname", "name", "email", "sub"} {
			if val, ok := claims[key].(string); ok && val != "" {
				displayName = val
				break
			}
		}
		sess.Set("user_nickname", displayName)

		// Clear the state from session
		sess.Delete("state")

		// Redirect to the dashboard
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// generateRandomState generates a random state value for CSRF protection
func generateRandomState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
