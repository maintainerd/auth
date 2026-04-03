package util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/maintainerd/auth/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initTestJWTKeys generates a fresh RSA key pair for each test run and wires
// it into the package-level variables used by GenerateAccessToken / ValidateToken.
func initTestJWTKeys(t *testing.T) {
	t.Helper()

	// Generate a 2048-bit key (minimum allowed by validateKeyStrength)
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "RSA key generation failed")

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&priv.PublicKey),
	})

	config.JWTPrivateKey = privPEM
	config.JWTPublicKey = pubPEM

	require.NoError(t, InitJWTKeys())
}

// ---------------------------------------------------------------------------
// GenerateAccessToken
// ---------------------------------------------------------------------------

func TestGenerateAccessToken_ValidInputs(t *testing.T) {
	initTestJWTKeys(t)
	tok, err := GenerateAccessToken("user-uuid", "read write", "https://auth.example.com", "myapp", "client-1", "provider-1")
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
}

func TestGenerateAccessToken_EmptyUserID(t *testing.T) {
	initTestJWTKeys(t)
	_, err := GenerateAccessToken("", "read", "https://auth.example.com", "myapp", "client-1", "provider-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "userId")
}

func TestGenerateAccessToken_EmptyIssuer(t *testing.T) {
	initTestJWTKeys(t)
	_, err := GenerateAccessToken("user-uuid", "read", "", "myapp", "client-1", "provider-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "issuer")
}

func TestGenerateAccessToken_EmptyAudience(t *testing.T) {
	initTestJWTKeys(t)
	_, err := GenerateAccessToken("user-uuid", "read", "https://auth.example.com", "", "client-1", "provider-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audience")
}

// ---------------------------------------------------------------------------
// GenerateIDToken
// ---------------------------------------------------------------------------

func TestGenerateIDToken_ValidInputs(t *testing.T) {
	initTestJWTKeys(t)
	tok, err := GenerateIDToken("user-uuid", "https://auth.example.com", "client-1", "provider-1", nil, "nonce123")
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
}

func TestGenerateIDToken_WithProfile(t *testing.T) {
	initTestJWTKeys(t)
	profile := &UserProfile{Email: "user@example.com", EmailVerified: true, FirstName: "Test"}
	tok, err := GenerateIDToken("user-uuid", "https://auth.example.com", "client-1", "provider-1", profile, "")
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
}

// ---------------------------------------------------------------------------
// GenerateRefreshToken
// ---------------------------------------------------------------------------

func TestGenerateRefreshToken_ValidInputs(t *testing.T) {
	initTestJWTKeys(t)
	tok, err := GenerateRefreshToken("user-uuid", "https://auth.example.com", "client-1", "provider-1")
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
}

// ---------------------------------------------------------------------------
// ValidateToken (round-trip)
// ---------------------------------------------------------------------------

func TestValidateToken_RoundTrip(t *testing.T) {
	initTestJWTKeys(t)
	tok, err := GenerateAccessToken("user-uuid", "read", "https://auth.example.com", "myapp", "client-1", "provider-1")
	require.NoError(t, err)

	claims, err := ValidateToken(tok)
	require.NoError(t, err)
	assert.Equal(t, "user-uuid", claims["sub"])
	assert.Equal(t, "access_token", claims["token_type"])
}

func TestValidateToken_EmptyString(t *testing.T) {
	initTestJWTKeys(t)
	_, err := ValidateToken("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestValidateToken_TamperedToken(t *testing.T) {
	initTestJWTKeys(t)
	tok, err := GenerateAccessToken("user-uuid", "read", "https://auth.example.com", "myapp", "client-1", "provider-1")
	require.NoError(t, err)

	// Flip a byte in the signature
	tampered := tok[:len(tok)-5] + "XXXXX"
	_, err = ValidateToken(tampered)
	require.Error(t, err)
}

