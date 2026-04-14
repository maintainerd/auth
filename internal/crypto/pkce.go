package crypto

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"regexp"
)

// PKCE constants following RFC 7636.
const (
	// PKCEMinVerifierLength is the minimum allowed code_verifier length (43 chars).
	PKCEMinVerifierLength = 43
	// PKCEMaxVerifierLength is the maximum allowed code_verifier length (128 chars).
	PKCEMaxVerifierLength = 128
	// PKCEChallengeMethodS256 is the only supported challenge method.
	PKCEChallengeMethodS256 = "S256"
)

// pkceVerifierPattern matches the allowed characters per RFC 7636 §4.1:
// [A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~"
var pkceVerifierPattern = regexp.MustCompile(`^[A-Za-z0-9\-._~]+$`)

// ValidatePKCEChallenge verifies that the code_challenge_method is S256 and
// that the code_verifier produces the expected code_challenge. Returns an error
// describing the failure if validation fails.
func ValidatePKCEChallenge(codeVerifier, codeChallenge, codeChallengeMethod string) error {
	if codeChallengeMethod != PKCEChallengeMethodS256 {
		return fmt.Errorf("unsupported code_challenge_method: %s", codeChallengeMethod)
	}

	if len(codeVerifier) < PKCEMinVerifierLength || len(codeVerifier) > PKCEMaxVerifierLength {
		return fmt.Errorf("code_verifier length must be between %d and %d", PKCEMinVerifierLength, PKCEMaxVerifierLength)
	}

	if !pkceVerifierPattern.MatchString(codeVerifier) {
		return fmt.Errorf("code_verifier contains invalid characters")
	}

	computed := ComputeS256Challenge(codeVerifier)
	if computed != codeChallenge {
		return fmt.Errorf("code_verifier does not match code_challenge")
	}

	return nil
}

// ComputeS256Challenge computes the S256 code challenge for a given verifier:
// BASE64URL(SHA256(code_verifier)) per RFC 7636 §4.2.
func ComputeS256Challenge(codeVerifier string) string {
	sum := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// HashAuthorizationCode produces a SHA-256 hash of the raw authorization code
// for storage. The raw code is never persisted.
func HashAuthorizationCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// HashRefreshToken produces a SHA-256 hash of the raw refresh token for
// storage. The raw token is never persisted.
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
