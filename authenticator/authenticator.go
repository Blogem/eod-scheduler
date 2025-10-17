package authenticator

import (
	"context"
)

// Config holds OAuth provider configuration
type Config struct {
	ProviderURL  string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// Token represents an authentication token
type Token struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
	Expiry       int64
}

// Claims represents user claims from the ID token
type Claims map[string]interface{}

// Provider interface abstracts OAuth provider operations
type Provider interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*Token, error)
	GetClaims(ctx context.Context, token *Token) (Claims, error)
}
