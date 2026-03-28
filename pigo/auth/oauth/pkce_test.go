package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestGenerateCodeVerifierLength(t *testing.T) {
	for i := 0; i < 10; i++ {
		v, err := GenerateCodeVerifier()
		if err != nil {
			t.Fatalf("GenerateCodeVerifier error: %v", err)
		}
		if len(v) < 43 || len(v) > 128 {
			t.Errorf("verifier length %d not in [43, 128]", len(v))
		}
	}
}

func TestGenerateCodeVerifierUnique(t *testing.T) {
	v1, _ := GenerateCodeVerifier()
	v2, _ := GenerateCodeVerifier()
	v3, _ := GenerateCodeVerifier()
	if v1 == v2 || v1 == v3 || v2 == v3 {
		t.Error("expected unique verifiers from multiple calls")
	}
}

func TestGenerateCodeChallengeS256(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := GenerateCodeChallenge(verifier)

	// Verify manually: BASE64URL(SHA256(verifier))
	sum := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(sum[:])

	if challenge != expected {
		t.Errorf("challenge: got %q want %q", challenge, expected)
	}
}

func TestGenerateCodeChallengeRoundTrip(t *testing.T) {
	for i := 0; i < 5; i++ {
		v, err := GenerateCodeVerifier()
		if err != nil {
			t.Fatalf("GenerateCodeVerifier: %v", err)
		}
		c := GenerateCodeChallenge(v)
		if len(c) == 0 {
			t.Errorf("empty challenge for verifier %q", v)
		}
		// Challenge should be BASE64URL-safe
		_, err = base64.RawURLEncoding.DecodeString(c)
		if err != nil {
			t.Errorf("challenge not valid base64url: %v", err)
		}
	}
}
