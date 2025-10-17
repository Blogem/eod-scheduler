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
func (ac *AuthController) Login(auth *authenticator.Authenticator) http.HandlerFunc {
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

		// Redirect to Auth0 login page
		http.Redirect(w, r, auth.AuthCodeURL(state), http.StatusTemporaryRedirect)
	}
}

// Callback handles the callback from Auth0
func (ac *AuthController) Callback(auth *authenticator.Authenticator) http.HandlerFunc {
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
		token, err := auth.Exchange(r.Context(), r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "Failed to exchange authorization code for a token: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Verify the ID token
		idToken, err := auth.VerifyIDToken(r.Context(), token)
		if err != nil {
			http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Extract profile information
		var profile map[string]interface{}
		if err := idToken.Claims(&profile); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Store the user session with nickname
		sess.Set("user_id", profile["sub"].(string))

		// Try to get nickname, fallback to name, then email, then sub
		var displayName string
		if nickname, ok := profile["nickname"].(string); ok && nickname != "" {
			displayName = nickname
		} else if name, ok := profile["name"].(string); ok && name != "" {
			displayName = name
		} else if email, ok := profile["email"].(string); ok && email != "" {
			displayName = email
		} else {
			displayName = profile["sub"].(string)
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
