package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// GenerateCodeVerifier generates a random PKCE code verifier.
// The verifier is a BASE64URL-encoded random value between 43 and 128 characters.
func GenerateCodeVerifier() (string, error) {
	// 32 bytes → 43 base64url chars (minimum per RFC 7636)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateCodeChallenge derives the S256 code challenge from the verifier.
// challenge = BASE64URL(SHA256(verifier))
func GenerateCodeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
