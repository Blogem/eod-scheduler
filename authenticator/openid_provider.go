package authenticator

import (
	"context"
	"errors"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OpenIDProvider implements the Provider interface for OpenID Connect
type OpenIDProvider struct {
	provider *oidc.Provider
	config   oauth2.Config
}

// OpenIDConfig holds OpenID Connect configuration
type OpenIDConfig struct {
	Domain       string
	ClientID     string
	ClientSecret string
	CallbackURL  string
}

// NewOpenIDProvider creates a new OpenID Connect provider with the given configuration
func NewOpenIDProvider(cfg OpenIDConfig) (Provider, error) {
	ctx := context.Background()

	// Validate required configuration
	if cfg.Domain == "" {
		return nil, errors.New("domain is required")
	}
	if cfg.ClientID == "" {
		return nil, errors.New("client ID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, errors.New("client secret is required")
	}
	if cfg.CallbackURL == "" {
		return nil, errors.New("callback URL is required")
	}

	provider, err := oidc.NewProvider(
		ctx,
		"https://"+cfg.Domain+"/",
	)
	if err != nil {
		return nil, err
	}

	conf := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.CallbackURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile"},
	}

	return &OpenIDProvider{
		provider: provider,
		config:   conf,
	}, nil
}

// GetAuthURL returns the authorization URL for OpenID Connect
func (p *OpenIDProvider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state)
}

// ExchangeCode exchanges an authorization code for tokens
func (p *OpenIDProvider) ExchangeCode(ctx context.Context, code string) (*Token, error) {
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
func (p *OpenIDProvider) GetClaims(ctx context.Context, token *Token) (Claims, error) {
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
