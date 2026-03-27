package auth

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ai-gateway/pi-go/auth/oauth"
)

// envKeyMap maps provider IDs to the environment variable holding the API key.
var envKeyMap = map[string]string{
	"anthropic": "ANTHROPIC_API_KEY",
	"openai":    "OPENAI_API_KEY",
	"gemini":    "GEMINI_API_KEY",
	"google":    "GOOGLE_API_KEY",
}

// AuthResolver resolves API keys using a priority chain:
// runtime key > stored API key > environment variable > stored OAuth token > keychain.
type AuthResolver struct {
	storage        *AuthStorage
	oauthProviders map[string]oauth.OAuthProvider
}

// NewAuthResolver creates an AuthResolver backed by the given storage.
func NewAuthResolver(storage *AuthStorage) *AuthResolver {
	return &AuthResolver{
		storage:        storage,
		oauthProviders: make(map[string]oauth.OAuthProvider),
	}
}

// RegisterOAuthProvider registers an OAuth provider for automatic token refresh.
func (r *AuthResolver) RegisterOAuthProvider(id string, p oauth.OAuthProvider) {
	r.oauthProviders[id] = p
}

// ResolveAPIKey resolves the API key for the given provider.
// Priority: runtime key > stored API key > environment variable > OAuth access token > keychain.
func (r *AuthResolver) ResolveAPIKey(ctx context.Context, provider string) (string, error) {
	// 1. Runtime key (also returns stored API key if no runtime key present)
	if key, ok := r.storage.GetAPIKey(provider); ok {
		return key, nil
	}

	// 2. Environment variable
	if envVar, ok := envKeyMap[provider]; ok {
		if val := os.Getenv(envVar); val != "" {
			return val, nil
		}
	}

	// 3. Stored OAuth access token (with auto-refresh)
	if creds, ok := r.storage.GetOAuth(provider); ok && creds.Access != "" {
		// If expired and we have an OAuth provider, refresh
		if !creds.ExpiresAt.IsZero() && time.Now().After(creds.ExpiresAt) {
			if oap, hasOAP := r.oauthProviders[provider]; hasOAP {
				oauthCreds := oauth.Credentials{
					Refresh:   creds.Refresh,
					Access:    creds.Access,
					ExpiresAt: creds.ExpiresAt,
					Extra:     creds.Extra,
				}
				if fresh, err := oap.Refresh(ctx, oauthCreds); err == nil {
					newCreds := OAuthCredentials{
						Refresh:   fresh.Refresh,
						Access:    fresh.Access,
						ExpiresAt: fresh.ExpiresAt,
						Extra:     fresh.Extra,
					}
					_ = r.storage.SetOAuth(provider, newCreds)
					return fresh.Access, nil
				}
			}
		}
		return creds.Access, nil
	}

	// 4. Keychain
	if key, err := KeychainGet("pi-go", provider); err == nil && key != "" {
		return key, nil
	}

	return "", fmt.Errorf("no API key found for provider %q", provider)
}
