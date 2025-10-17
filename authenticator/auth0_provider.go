package authenticator

import (
	"context"
	"errors"
	"os"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Auth0Provider implements the Provider interface for Auth0
type Auth0Provider struct {
	provider *oidc.Provider
	config   oauth2.Config
}

// NewAuth0Provider creates a new Auth0 provider
func NewAuth0Provider() (Provider, error) {
	ctx := context.Background()

	provider, err := oidc.NewProvider(
		ctx,
		"https://"+os.Getenv("AUTH0_DOMAIN")+"/",
	)
	if err != nil {
		return nil, err
	}

	conf := oauth2.Config{
		ClientID:     os.Getenv("AUTH0_CLIENT_ID"), // TODO: required env var
		ClientSecret: os.Getenv("AUTH0_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("AUTH0_CALLBACK_URL"),
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile"},
	}

	return &Auth0Provider{
		provider: provider,
		config:   conf,
	}, nil
}

// GetAuthURL returns the authorization URL for Auth0
func (p *Auth0Provider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state)
}

// ExchangeCode exchanges an authorization code for tokens
func (p *Auth0Provider) ExchangeCode(ctx context.Context, code string) (*Token, error) {
	oauth2Token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	// Convert oauth2.Token to our Token type
	token := &Token{
		AccessToken:  oauth2Token.AccessToken,
		RefreshToken: oauth2Token.RefreshToken,
		Expiry:       oauth2Token.Expiry.Unix(),
	}

	// Extract ID token if present
	if idToken, ok := oauth2Token.Extra("id_token").(string); ok {
		token.IDToken = idToken
	}

	return token, nil
}

// GetClaims extracts user claims from the ID token
func (p *Auth0Provider) GetClaims(ctx context.Context, token *Token) (Claims, error) {
	if token.IDToken == "" {
		return nil, errors.New("no id_token in token")
	}

	oidcConfig := &oidc.Config{
		ClientID: p.config.ClientID,
	}

	idToken, err := p.provider.Verifier(oidcConfig).Verify(ctx, token.IDToken)
	if err != nil {
		return nil, err
	}

	var claims Claims
	if err := idToken.Claims(&claims); err != nil {
		return nil, err
	}

	return claims, nil
}
