package oauth

import (
	"context"
	"time"
)

// Credentials holds OAuth token data for a provider.
// This mirrors auth.OAuthCredentials to avoid a circular import.
type Credentials struct {
	Refresh   string         `json:"refresh"`
	Access    string         `json:"access"`
	ExpiresAt time.Time      `json:"expiresAt"`
	Extra     map[string]any `json:"extra,omitempty"`
}

// OAuthProvider is implemented by OAuth-based authentication providers.
type OAuthProvider interface {
	// AuthURL builds the authorization URL the user should visit.
	AuthURL(state, codeChallenge, redirectURI string) string
	// Exchange exchanges the authorization code for tokens.
	Exchange(ctx context.Context, code, verifier, redirectURI string) (Credentials, error)
	// Refresh requests new tokens using the refresh token.
	Refresh(ctx context.Context, creds Credentials) (Credentials, error)
}
