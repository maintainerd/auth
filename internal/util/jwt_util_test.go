package util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"maps"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

func TestValidateToken_NilPublicKey(t *testing.T) {
	initTestJWTKeys(t)
	saved := publicKey
	publicKey = nil
	t.Cleanup(func() { publicKey = saved })

	_, err := ValidateToken("any.token.string")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "public key not initialized")
}

// ---------------------------------------------------------------------------
// GenerateSecureID
// ---------------------------------------------------------------------------

func TestGenerateSecureID_Format(t *testing.T) {
	id := GenerateSecureID()
	assert.Len(t, id, 32, "hex-encoded 16 bytes = 32 chars")
	for _, c := range id {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'), "must be lowercase hex")
	}
}

func TestGenerateSecureID_Unique(t *testing.T) {
	a, b := GenerateSecureID(), GenerateSecureID()
	assert.NotEqual(t, a, b, "two consecutive IDs must differ")
}

// ---------------------------------------------------------------------------
// validateKeyStrength (unexported — accessible within package)
// ---------------------------------------------------------------------------

func TestValidateKeyStrength_Valid(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	assert.NoError(t, validateKeyStrength(priv))
}

func TestValidateKeyStrength_TooSmall(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 1024)
	require.NoError(t, err)
	err = validateKeyStrength(priv)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "below minimum required")
}

// ---------------------------------------------------------------------------
// InitJWTKeys error paths
// ---------------------------------------------------------------------------

func saveAndRestoreJWTConfig(t *testing.T) {
	t.Helper()
	savedPriv := config.JWTPrivateKey
	savedPub := config.JWTPublicKey
	t.Cleanup(func() {
		config.JWTPrivateKey = savedPriv
		config.JWTPublicKey = savedPub
		_ = InitJWTKeys()
	})
}

func TestInitJWTKeys_EmptyPrivateKey(t *testing.T) {
	saveAndRestoreJWTConfig(t)
	config.JWTPrivateKey = nil
	config.JWTPublicKey = []byte("dummy")
	err := InitJWTKeys()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_PRIVATE_KEY")
}

func TestInitJWTKeys_EmptyPublicKey(t *testing.T) {
	saveAndRestoreJWTConfig(t)
	config.JWTPrivateKey = []byte("dummy")
	config.JWTPublicKey = nil
	err := InitJWTKeys()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_PUBLIC_KEY")
}

func TestInitJWTKeys_InvalidPrivatePEM(t *testing.T) {
	saveAndRestoreJWTConfig(t)
	config.JWTPrivateKey = []byte("not-valid-pem")
	config.JWTPublicKey = []byte("also-not-valid-pem")
	err := InitJWTKeys()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse private key")
}

func TestInitJWTKeys_MismatchedKeys(t *testing.T) {
	saveAndRestoreJWTConfig(t)

	priv1, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	priv2, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv1)})
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&priv2.PublicKey)})

	config.JWTPrivateKey = privPEM
	config.JWTPublicKey = pubPEM
	err = InitJWTKeys()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "do not form a valid key pair")
}

// ---------------------------------------------------------------------------
// GenerateAccessToken — additional validation branches
// ---------------------------------------------------------------------------

func TestGenerateAccessToken_EmptyClientID(t *testing.T) {
	initTestJWTKeys(t)
	_, err := GenerateAccessToken("user-uuid", "read", "https://auth.example.com", "myapp", "", "provider-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clientID")
}

func TestGenerateAccessToken_EmptyProviderID(t *testing.T) {
	initTestJWTKeys(t)
	_, err := GenerateAccessToken("user-uuid", "read", "https://auth.example.com", "myapp", "client-1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "providerID")
}

// ---------------------------------------------------------------------------
// GenerateIDToken — validation branches
// ---------------------------------------------------------------------------

func TestGenerateIDToken_EmptyUserUUID(t *testing.T) {
	initTestJWTKeys(t)
	_, err := GenerateIDToken("", "https://auth.example.com", "client-1", "provider-1", nil, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "userUUID")
}

func TestGenerateIDToken_EmptyIssuer(t *testing.T) {
	initTestJWTKeys(t)
	_, err := GenerateIDToken("user-uuid", "", "client-1", "provider-1", nil, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "issuer")
}

func TestGenerateIDToken_EmptyClientID(t *testing.T) {
	initTestJWTKeys(t)
	_, err := GenerateIDToken("user-uuid", "https://auth.example.com", "", "provider-1", nil, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clientID")
}

func TestGenerateIDToken_EmptyProviderID(t *testing.T) {
	initTestJWTKeys(t)
	_, err := GenerateIDToken("user-uuid", "https://auth.example.com", "client-1", "", nil, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "providerID")
}

func TestGenerateIDToken_FullProfile(t *testing.T) {
	initTestJWTKeys(t)
	profile := &UserProfile{
		Email: "user@example.com", EmailVerified: true,
		Phone: "+1234567890", PhoneVerified: true,
		FirstName: "Jane", MiddleName: "A", LastName: "Doe",
		Suffix: "Jr", Birthdate: "1990-01-01", Gender: "F",
		Address: "123 Main St", Picture: "https://example.com/pic.jpg",
	}
	tok, err := GenerateIDToken("user-uuid", "https://auth.example.com", "client-1", "provider-1", profile, "nonce-abc")
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
}

// ---------------------------------------------------------------------------
// GenerateRefreshToken — validation branches
// ---------------------------------------------------------------------------

func TestGenerateRefreshToken_EmptyUserUUID(t *testing.T) {
	initTestJWTKeys(t)
	_, err := GenerateRefreshToken("", "https://auth.example.com", "client-1", "provider-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "userUUID")
}

func TestGenerateRefreshToken_EmptyIssuer(t *testing.T) {
	initTestJWTKeys(t)
	_, err := GenerateRefreshToken("user-uuid", "", "client-1", "provider-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "issuer")
}

func TestGenerateRefreshToken_EmptyClientID(t *testing.T) {
	initTestJWTKeys(t)
	_, err := GenerateRefreshToken("user-uuid", "https://auth.example.com", "", "provider-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clientID")
}

func TestGenerateRefreshToken_EmptyProviderID(t *testing.T) {
	initTestJWTKeys(t)
	_, err := GenerateRefreshToken("user-uuid", "https://auth.example.com", "client-1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "providerID")
}

// ---------------------------------------------------------------------------
// InitJWTKeys — weak private key rejected
// ---------------------------------------------------------------------------

func TestInitJWTKeys_WeakPrivateKey(t *testing.T) {
	saveAndRestoreJWTConfig(t)

	// Generate a valid 1024-bit key — parses OK but fails strength check
	weakPriv, err := rsa.GenerateKey(rand.Reader, 1024)
	require.NoError(t, err)

	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(weakPriv)})
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&weakPriv.PublicKey)})

	config.JWTPrivateKey = privPEM
	config.JWTPublicKey = pubPEM
	err = InitJWTKeys()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private key security validation failed")
}

// ---------------------------------------------------------------------------
// generateToken — nil private key (unexported, accessible within package)
// ---------------------------------------------------------------------------

func TestGenerateToken_NilPrivateKey(t *testing.T) {
	saved := privateKey
	privateKey = nil
	t.Cleanup(func() { privateKey = saved })

	claims := jwt.MapClaims{
		"sub": "user-uuid", "aud": "myapp", "iss": "https://auth.example.com",
		"iat": jwt.NewNumericDate(time.Now()), "exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "test-jti",
	}
	_, err := generateToken(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private key not initialized")
}

func TestGenerateToken_MissingRequiredClaim(t *testing.T) {
	initTestJWTKeys(t)

	// "jti" is omitted — generateToken must reject it
	claims := jwt.MapClaims{
		"sub": "user-uuid", "aud": "myapp", "iss": "https://auth.example.com",
		"iat": jwt.NewNumericDate(time.Now()), "exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	_, err := generateToken(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jti")
}

// ---------------------------------------------------------------------------
// ValidateToken — KID mismatch path
// ---------------------------------------------------------------------------

func TestValidateToken_KIDMismatch(t *testing.T) {
	initTestJWTKeys(t)
	// Generate token with default KID "maintainerd-auth-key-1"
	tok, err := GenerateAccessToken("user-uuid", "read", "https://auth.example.com", "myapp", "client-1", "provider-1")
	require.NoError(t, err)

	// Now tell ValidateToken to expect a different KID
	t.Setenv("JWT_KEY_ID", "rotated-key-2")
	_, err = ValidateToken(tok)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown key ID")
}

// ---------------------------------------------------------------------------
// ValidateToken — algorithm confusion attack (non-RSA signing method)
// ---------------------------------------------------------------------------

func TestValidateToken_AlgorithmConfusion(t *testing.T) {
	initTestJWTKeys(t)

	// HMAC-signed token must be rejected — prevent algorithm confusion attacks
	hmacTok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "attacker",
		"aud": "myapp",
		"iss": "https://auth.example.com",
		"iat": jwt.NewNumericDate(time.Now()),
		"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "test-jti",
	}).SignedString([]byte("hmac-secret"))
	require.NoError(t, err)

	_, err = ValidateToken(hmacTok)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected signing method")
}

// ---------------------------------------------------------------------------
// ValidateToken — wrong RSA variant (RS384 instead of RS256)
// ---------------------------------------------------------------------------

func TestValidateToken_WrongRSAVariant(t *testing.T) {
	initTestJWTKeys(t)

	// RS384-signed token — only RS256 is allowed
	rs384Tok, err := jwt.NewWithClaims(jwt.SigningMethodRS384, jwt.MapClaims{
		"sub": "user-uuid",
		"aud": "myapp",
		"iss": "https://auth.example.com",
		"iat": jwt.NewNumericDate(time.Now()),
		"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "test-jti",
	}).SignedString(privateKey)
	require.NoError(t, err)

	_, err = ValidateToken(rs384Tok)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected RSA signing method")
}

// ---------------------------------------------------------------------------
// validateTokenClaims — direct tests of each missing/empty claim
// ---------------------------------------------------------------------------

func TestValidateTokenClaims_MissingClaim(t *testing.T) {
	required := []string{"sub", "aud", "iss", "iat", "exp", "jti"}
	base := jwt.MapClaims{
		"sub": "user", "aud": "app", "iss": "https://example.com",
		"iat": jwt.NewNumericDate(time.Now()), "exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "abc123",
	}
	for _, missing := range required {
		t.Run("missing_"+missing, func(t *testing.T) {
			claims := make(jwt.MapClaims)
			maps.Copy(claims, base)
			delete(claims, missing)
			err := validateTokenClaims(claims)
			require.Error(t, err)
			assert.Contains(t, err.Error(), missing)
		})
	}
}

func TestValidateTokenClaims_EmptySub(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": "", "aud": "app", "iss": "https://example.com",
		"iat": jwt.NewNumericDate(time.Now()), "exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "abc123",
	}
	err := validateTokenClaims(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subject")
}

func TestValidateTokenClaims_EmptyAud(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": "user", "aud": "", "iss": "https://example.com",
		"iat": jwt.NewNumericDate(time.Now()), "exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "abc123",
	}
	err := validateTokenClaims(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audience")
}

func TestValidateTokenClaims_EmptyIss(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": "user", "aud": "app", "iss": "",
		"iat": jwt.NewNumericDate(time.Now()), "exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "abc123",
	}
	err := validateTokenClaims(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "issuer")
}

func TestValidateTokenClaims_EmptyJTI(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": "user", "aud": "app", "iss": "https://example.com",
		"iat": jwt.NewNumericDate(time.Now()), "exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "",
	}
	err := validateTokenClaims(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JTI")
}

func TestValidateTokenClaims_Valid(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": "user", "aud": "app", "iss": "https://example.com",
		"iat": jwt.NewNumericDate(time.Now()), "exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "abc123",
	}
	assert.NoError(t, validateTokenClaims(claims))
}
