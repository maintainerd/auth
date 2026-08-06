package jwt

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"maps"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initTestJWTKeys generates a fresh RSA key pair for each test run and wires
// it into the default JWT key store used by GenerateAccessToken / ValidateToken.
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
	allowTestIssuers(t)
}

// allowTestIssuers declares the issuers this package's tests mint tokens under.
//
// validateIssuerClaim fails CLOSED on an empty allowlist: it then accepts only
// the authorization server's own issuer (APP_PUBLIC_HOSTNAME), which is unset
// in a test binary. A test that generates a token and validates it therefore
// has to register its issuer the same way a deployment does at boot, or it ends
// up asserting against the unconfigured-allowlist rejection instead of the
// behaviour it means to cover.
func allowTestIssuers(t *testing.T) {
	t.Helper()
	SetAcceptedIssuers([]string{
		"https://auth.example.com",
		"https://example.com",
		"https://tenant-a.example.com",
		// The fallback generateStepUpChallengeTokenWithContext stamps when
		// APP_PUBLIC_HOSTNAME is unset.
		"maintainerd-auth",
	})
	t.Cleanup(ResetAcceptedIssuers)
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
	tok, err := GenerateIDToken("user-uuid", "https://auth.example.com", "client-1", "provider-1", nil, "nonce123", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
}

func TestGenerateIDToken_WithProfile(t *testing.T) {
	initTestJWTKeys(t)
	profile := &UserProfile{Email: "user@example.com", EmailVerified: true, FirstName: "Test"}
	tok, err := GenerateIDToken("user-uuid", "https://auth.example.com", "client-1", "provider-1", profile, "", nil)
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
	ResetJWTKeys()
	t.Cleanup(func() { initTestJWTKeys(t) })

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

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("entropy unavailable")
}

func TestGenerateSecureID_RandomReadFailurePanics(t *testing.T) {
	oldReader := rand.Reader
	rand.Reader = failingReader{}
	t.Cleanup(func() { rand.Reader = oldReader })

	require.Panics(t, func() { GenerateSecureID() })
}

func TestGenerateSecureJTI_RandomReadFailurePanics(t *testing.T) {
	oldReader := rand.Reader
	rand.Reader = failingReader{}
	t.Cleanup(func() { rand.Reader = oldReader })

	require.Panics(t, func() { generateSecureJTI() })
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
	_, err := GenerateIDToken("", "https://auth.example.com", "client-1", "provider-1", nil, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "userUUID")
}

func TestGenerateIDToken_EmptyIssuer(t *testing.T) {
	initTestJWTKeys(t)
	_, err := GenerateIDToken("user-uuid", "", "client-1", "provider-1", nil, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "issuer")
}

func TestGenerateIDToken_EmptyClientID(t *testing.T) {
	initTestJWTKeys(t)
	_, err := GenerateIDToken("user-uuid", "https://auth.example.com", "", "provider-1", nil, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clientID")
}

func TestGenerateIDToken_EmptyProviderID(t *testing.T) {
	initTestJWTKeys(t)
	_, err := GenerateIDToken("user-uuid", "https://auth.example.com", "client-1", "", nil, "", nil)
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
	tok, err := GenerateIDToken("user-uuid", "https://auth.example.com", "client-1", "provider-1", profile, "nonce-abc", nil)
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
	initTestJWTKeys(t)
	ResetJWTKeys()
	t.Cleanup(func() { initTestJWTKeys(t) })

	claims := jwtlib.MapClaims{
		"sub": "user-uuid", "aud": "myapp", "iss": "https://auth.example.com",
		"iat": jwtlib.NewNumericDate(time.Now()), "exp": jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "test-jti",
	}
	_, err := generateToken(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private key not initialized")
}

func TestGenerateToken_MissingRequiredClaim(t *testing.T) {
	initTestJWTKeys(t)

	// "jti" is omitted — generateToken must reject it
	claims := jwtlib.MapClaims{
		"sub": "user-uuid", "aud": "myapp", "iss": "https://auth.example.com",
		"iat": jwtlib.NewNumericDate(time.Now()), "exp": jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
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

	// Craft a token signed with the real private key but carrying a KID that
	// is not registered in the active or retiring key set.
	priv, _ := activeSigningKeyForTest()

	now := time.Now()
	tok := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{
		"sub": "user-uuid", "aud": "myapp", "iss": "https://auth.example.com",
		"iat": jwtlib.NewNumericDate(now), "exp": jwtlib.NewNumericDate(now.Add(time.Hour)),
		"jti": generateSecureJTI(),
	})
	tok.Header["kid"] = "totally-unknown-key-id"
	tokenString, err := tok.SignedString(priv)
	require.NoError(t, err)

	_, err = ValidateToken(tokenString)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown key ID")
}

// ---------------------------------------------------------------------------
// ValidateToken — algorithm confusion attack (non-RSA signing method)
// ---------------------------------------------------------------------------

func TestValidateToken_AlgorithmConfusion(t *testing.T) {
	initTestJWTKeys(t)

	// HMAC-signed token must be rejected — prevent algorithm confusion attacks
	hmacTok, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{
		"sub": "attacker",
		"aud": "myapp",
		"iss": "https://auth.example.com",
		"iat": jwtlib.NewNumericDate(time.Now()),
		"exp": jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
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
	testPrivKey, _ := activeSigningKeyForTest()
	rs384Tok, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS384, jwtlib.MapClaims{
		"sub": "user-uuid",
		"aud": "myapp",
		"iss": "https://auth.example.com",
		"iat": jwtlib.NewNumericDate(time.Now()),
		"exp": jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "test-jti",
	}).SignedString(testPrivKey)
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
	base := jwtlib.MapClaims{
		"sub": "user", "aud": "app", "iss": "https://example.com",
		"iat": jwtlib.NewNumericDate(time.Now()), "exp": jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "abc123",
	}
	for _, missing := range required {
		t.Run("missing_"+missing, func(t *testing.T) {
			claims := make(jwtlib.MapClaims)
			maps.Copy(claims, base)
			delete(claims, missing)
			err := validateTokenClaims(claims)
			require.Error(t, err)
			assert.Contains(t, err.Error(), missing)
		})
	}
}

func TestValidateTokenClaims_EmptySub(t *testing.T) {
	claims := jwtlib.MapClaims{
		"sub": "", "aud": "app", "iss": "https://example.com",
		"iat": jwtlib.NewNumericDate(time.Now()), "exp": jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "abc123",
	}
	err := validateTokenClaims(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subject")
}

func TestValidateTokenClaims_EmptyAud(t *testing.T) {
	claims := jwtlib.MapClaims{
		"sub": "user", "aud": "", "iss": "https://example.com",
		"iat": jwtlib.NewNumericDate(time.Now()), "exp": jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "abc123",
	}
	err := validateTokenClaims(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audience")
}

func TestValidateTokenClaims_EmptyIss(t *testing.T) {
	claims := jwtlib.MapClaims{
		"sub": "user", "aud": "app", "iss": "",
		"iat": jwtlib.NewNumericDate(time.Now()), "exp": jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "abc123",
	}
	err := validateTokenClaims(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "issuer")
}

func TestValidateTokenClaims_EmptyJTI(t *testing.T) {
	allowTestIssuers(t)
	claims := jwtlib.MapClaims{
		"sub": "user", "aud": "app", "iss": "https://example.com",
		"iat": jwtlib.NewNumericDate(time.Now()), "exp": jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "",
	}
	err := validateTokenClaims(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JTI")
}

func TestValidateTokenClaims_Valid(t *testing.T) {
	allowTestIssuers(t)
	claims := jwtlib.MapClaims{
		"sub": "user", "aud": "app", "iss": "https://example.com",
		"iat": jwtlib.NewNumericDate(time.Now()), "exp": jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "abc123",
	}
	assert.NoError(t, validateTokenClaims(claims))
}

// ---------------------------------------------------------------------------
// InitJWTKeys — valid private key + invalid public key PEM
// ---------------------------------------------------------------------------

func TestInitJWTKeys_InvalidPublicPEM(t *testing.T) {
	saveAndRestoreJWTConfig(t)

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	config.JWTPrivateKey = privPEM
	config.JWTPublicKey = []byte("not-valid-pem")
	err = InitJWTKeys()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse public key")
}

// ---------------------------------------------------------------------------
// ValidateToken — signed token missing required claims (validateTokenClaims)
// ---------------------------------------------------------------------------

func TestValidateToken_MissingJTIInSignedToken(t *testing.T) {
	initTestJWTKeys(t)

	now := time.Now()
	claims := jwtlib.MapClaims{
		"sub": "user",
		"aud": "app",
		"iss": "https://auth.example.com",
		"iat": jwtlib.NewNumericDate(now),
		"exp": jwtlib.NewNumericDate(now.Add(time.Hour)),
		// "jti" deliberately omitted
	}
	sigPrivKey, sigKID := activeSigningKeyForTest()

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
	token.Header["kid"] = sigKID
	tokenString, err := token.SignedString(sigPrivKey)
	require.NoError(t, err)

	_, err = ValidateToken(tokenString)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jti")
}

func TestStepUpChallengeToken_IncludesRequiredClaimsAndAllowedMethods(t *testing.T) {
	initTestJWTKeys(t)
	origHostname := config.AppPublicHostname
	t.Cleanup(func() { config.AppPublicHostname = origHostname })
	config.AppPublicHostname = "https://auth.example.com"

	token, err := GenerateStepUpChallengeToken("user-uuid", time.Minute, []string{"totp", "backup_code"})
	require.NoError(t, err)

	claims, err := ValidateStepUpChallengeToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-uuid", claims["sub"])
	assert.Equal(t, "https://auth.example.com", claims["iss"])
	assert.Equal(t, "https://auth.example.com", claims["aud"])
	assert.Equal(t, "step_up_challenge", claims["typ"])
	assert.ElementsMatch(t, []any{"totp", "backup_code"}, claims["allowed_methods"])
}

func TestStepUpChallengeToken_NoAllowedMethods(t *testing.T) {
	initTestJWTKeys(t)
	origHostname := config.AppPublicHostname
	t.Cleanup(func() { config.AppPublicHostname = origHostname })
	config.AppPublicHostname = ""

	token, err := GenerateStepUpChallengeToken("user-uuid", time.Minute)
	require.NoError(t, err)

	claims, err := ValidateStepUpChallengeToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-uuid", claims["sub"])
	assert.Equal(t, "maintainerd-auth", claims["iss"])
	assert.Nil(t, claims["allowed_methods"])
}

func TestValidateStepUpChallengeToken_WrongType(t *testing.T) {
	initTestJWTKeys(t)

	tok, err := GenerateAccessToken("user-uuid", "read", "https://auth.example.com", "myapp", "client-1", "provider-1")
	require.NoError(t, err)

	_, err = ValidateStepUpChallengeToken(tok)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a step-up challenge token")
}

func TestValidateStepUpChallengeToken_InvalidToken(t *testing.T) {
	_, err := ValidateStepUpChallengeToken("not-a-token")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Key store tests
// ---------------------------------------------------------------------------

func TestKeyStore_PublicKey_And_AllPublicKeys(t *testing.T) {
	initTestJWTKeys(t)

	pub := GetPublicKey()
	require.NotNil(t, pub)

	all := GetAllPublicKeys()
	require.Len(t, all, 1)
	assert.NotEmpty(t, all[0].KID)
}

func TestKeyStore_Rotate(t *testing.T) {
	initTestJWTKeys(t)

	origKID := GetAllPublicKeys()[0].KID
	err := RotateKeys()
	require.NoError(t, err)

	all := GetAllPublicKeys()
	require.Len(t, all, 2)
	assert.NotEqual(t, origKID, all[0].KID)
}

func TestKeyStore_VerificationKey_EmptyKID(t *testing.T) {
	initTestJWTKeys(t)

	priv, kid := activeSigningKeyForTest()
	require.NotNil(t, priv)

	pub, err := defaultKeyStore.verificationKey("")
	require.NoError(t, err)
	assert.NotNil(t, pub)

	pub2, err := defaultKeyStore.verificationKey(kid)
	require.NoError(t, err)
	assert.NotNil(t, pub2)
}

// ---------------------------------------------------------------------------
// GenerateAccessTokenWithOptions
// ---------------------------------------------------------------------------

func TestGenerateAccessTokenWithOptions(t *testing.T) {
	initTestJWTKeys(t)

	opts := &AccessTokenOptions{
		DPoPThumbprint: "dpop-thumb",
		AccessTokenTTL: 30 * time.Second,
		AMR:            []string{"pwd"},
		ACR:            "1",
		SessionID:      "session-123",
	}
	tok, err := GenerateAccessTokenWithOptions("user-uuid", "read", "https://auth.example.com", "myapp", "client-1", "provider-1", opts)
	require.NoError(t, err)
	assert.NotEmpty(t, tok)

	claims, err := ValidateToken(tok)
	require.NoError(t, err)
	assert.Equal(t, "DPoP", claims["token_type"])
	cnf, ok := claims["cnf"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "dpop-thumb", cnf["jkt"])
	assert.Equal(t, []any{"pwd"}, claims["amr"])
	assert.Equal(t, "1", claims["acr"])
	assert.Equal(t, "session-123", claims["sid"])
}

func TestGenerateAccessTokenWithOptions_NoOptions(t *testing.T) {
	initTestJWTKeys(t)

	tok, err := GenerateAccessTokenWithOptions("user-uuid", "read", "https://auth.example.com", "myapp", "client-1", "provider-1", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
}

// ---------------------------------------------------------------------------
// ID token — scope filtering and extra claims
// ---------------------------------------------------------------------------

func TestGenerateIDToken_ScopeFiltering(t *testing.T) {
	initTestJWTKeys(t)
	profile := &UserProfile{
		Email: "user@example.com", EmailVerified: true,
		Phone: "+1234567890", PhoneVerified: true,
		FirstName: "Jane", MiddleName: "A", LastName: "Doe",
		Name: "Jane Doe",
	}

	params := &IDTokenParams{
		RequestedScopes: []string{"email"},
	}
	tok, err := GenerateIDToken("user-uuid", "https://auth.example.com", "client-1", "provider-1", profile, "", params)
	require.NoError(t, err)

	claims, err := ValidateToken(tok)
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", claims["email"])
	assert.True(t, claims["email_verified"].(bool))
	assert.Nil(t, claims["first_name"])
}

func TestGenerateIDToken_CustomScopeClaimMappings(t *testing.T) {
	initTestJWTKeys(t)
	profile := &UserProfile{FirstName: "Jane", Email: "user@example.com"}

	params := &IDTokenParams{
		RequestedScopes:    []string{"custom"},
		ScopeClaimMappings: map[string][]string{"custom": {"first_name"}},
	}
	tok, err := GenerateIDToken("user-uuid", "https://auth.example.com", "client-1", "provider-1", profile, "", params)
	require.NoError(t, err)

	claims, err := ValidateToken(tok)
	require.NoError(t, err)
	assert.Equal(t, "Jane", claims["first_name"])
	assert.Nil(t, claims["email"])
}

func TestGenerateIDToken_ExtraClaims(t *testing.T) {
	initTestJWTKeys(t)

	params := &IDTokenParams{
		ExtraClaims: map[string]any{"custom_key": "custom_value"},
		AMR:         []string{"pwd"},
		ACR:         "2",
	}
	tok, err := GenerateIDToken("user-uuid", "https://auth.example.com", "client-1", "provider-1", nil, "", params)
	require.NoError(t, err)

	claims, err := ValidateToken(tok)
	require.NoError(t, err)
	assert.Equal(t, "custom_value", claims["custom_key"])
	assert.Equal(t, []any{"pwd"}, claims["amr"])
	assert.Equal(t, "2", claims["acr"])
}

func TestGenerateIDToken_AMRAndACR(t *testing.T) {
	initTestJWTKeys(t)

	params := &IDTokenParams{AMR: []string{"pwd", "otp"}, ACR: "2"}
	tok, err := GenerateIDToken("user-uuid", "https://auth.example.com", "client-1", "provider-1", nil, "", params)
	require.NoError(t, err)

	claims, err := ValidateToken(tok)
	require.NoError(t, err)
	assert.Equal(t, []any{"pwd", "otp"}, claims["amr"])
	assert.Equal(t, "2", claims["acr"])
}

func TestGenerateIDTokenWithContext_Success(t *testing.T) {
	initTestJWTKeys(t)
	tok, err := GenerateIDTokenWithContext(context.Background(), "user-uuid", "https://auth.example.com", "client-1", "provider-1", nil, "", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
}

// ---------------------------------------------------------------------------
// Refresh token context wrappers
// ---------------------------------------------------------------------------

func TestGenerateRefreshTokenWithContext_Success(t *testing.T) {
	initTestJWTKeys(t)
	tok, err := GenerateRefreshTokenWithContext(context.Background(), "user-uuid", "https://auth.example.com", "client-1", "provider-1")
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
}

// ---------------------------------------------------------------------------
// JTI denylist checker
// ---------------------------------------------------------------------------

func TestSetAndResetJTIChecker(t *testing.T) {
	called := false
	SetJTIChecker(func(ctx context.Context, jti string) (bool, error) {
		called = true
		return false, nil
	})

	checker := getJTIChecker()
	require.NotNil(t, checker)
	denied, err := checker(context.Background(), "test-jti")
	require.NoError(t, err)
	assert.False(t, denied)
	assert.True(t, called)

	ResetJTIChecker()
	assert.Nil(t, getJTIChecker())
}

func TestValidateToken_JTIDenylistDenied(t *testing.T) {
	initTestJWTKeys(t)

	SetJTIChecker(func(ctx context.Context, jti string) (bool, error) {
		return true, nil
	})
	t.Cleanup(ResetJTIChecker)

	tok, err := GenerateAccessToken("user-uuid", "read", "https://auth.example.com", "myapp", "client-1", "provider-1")
	require.NoError(t, err)

	_, err = ValidateToken(tok)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "revoked")
}

func TestValidateToken_JTIDenylistError(t *testing.T) {
	initTestJWTKeys(t)

	SetJTIChecker(func(ctx context.Context, jti string) (bool, error) {
		return false, errors.New("redis down")
	})
	t.Cleanup(ResetJTIChecker)

	tok, err := GenerateAccessToken("user-uuid", "read", "https://auth.example.com", "myapp", "client-1", "provider-1")
	require.NoError(t, err)

	_, err = ValidateToken(tok)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "revocation check failed")
}

func TestValidateToken_NoJTIChecker(t *testing.T) {
	initTestJWTKeys(t)
	ResetJTIChecker()

	tok, err := GenerateAccessToken("user-uuid", "read", "https://auth.example.com", "myapp", "client-1", "provider-1")
	require.NoError(t, err)

	claims, err := ValidateToken(tok)
	require.NoError(t, err)
	assert.NotNil(t, claims)
}

// ---------------------------------------------------------------------------
// ValidateTokenWithContext
// ---------------------------------------------------------------------------

func TestValidateTokenWithContext_Success(t *testing.T) {
	initTestJWTKeys(t)
	tok, err := GenerateAccessToken("user-uuid", "read", "https://auth.example.com", "myapp", "client-1", "provider-1")
	require.NoError(t, err)

	claims, err := ValidateTokenWithContext(context.Background(), tok)
	require.NoError(t, err)
	assert.Equal(t, "user-uuid", claims["sub"])
}

func TestValidateTokenWithContext_TODOContext(t *testing.T) {
	initTestJWTKeys(t)
	tok, err := GenerateAccessToken("user-uuid", "read", "https://auth.example.com", "myapp", "client-1", "provider-1")
	require.NoError(t, err)

	claims, err := ValidateTokenWithContext(context.TODO(), tok)
	require.NoError(t, err)
	assert.NotNil(t, claims)
}

func TestGenerateAccessTokenWithOptionsContext_TODOContext(t *testing.T) {
	initTestJWTKeys(t)
	tok, err := GenerateAccessTokenWithOptionsContext(context.TODO(), "user-uuid", "read", "https://auth.example.com", "myapp", "client-1", "provider-1", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
}

// ---------------------------------------------------------------------------
// generateToken — missing required claims (each one)
// ---------------------------------------------------------------------------

func TestGenerateToken_MissingEachRequiredClaim(t *testing.T) {
	initTestJWTKeys(t)

	required := []string{"sub", "aud", "iss", "iat", "exp", "jti"}
	base := jwtlib.MapClaims{
		"sub": "user-uuid", "aud": "myapp", "iss": "https://auth.example.com",
		"iat": jwtlib.NewNumericDate(time.Now()), "exp": jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "test-jti",
	}
	for _, claim := range required {
		t.Run(claim, func(t *testing.T) {
			claims := make(jwtlib.MapClaims)
			maps.Copy(claims, base)
			delete(claims, claim)
			_, err := generateToken(claims)
			require.Error(t, err)
			assert.Contains(t, err.Error(), claim)
		})
	}
}

// ---------------------------------------------------------------------------
// buildAllowedClaimsSet
// ---------------------------------------------------------------------------

func TestBuildAllowedClaimsSet(t *testing.T) {
	assert.Nil(t, buildAllowedClaimsSet(nil))

	params := &IDTokenParams{}
	assert.Nil(t, buildAllowedClaimsSet(params))

	params.RequestedScopes = []string{"profile"}
	allowed := buildAllowedClaimsSet(params)
	require.NotNil(t, allowed)
	assert.Contains(t, allowed, "name")
	assert.NotContains(t, allowed, "email")

	params.RequestedScopes = []string{"custom"}
	params.ScopeClaimMappings = map[string][]string{"custom": {"foo", "bar"}}
	allowed = buildAllowedClaimsSet(params)
	assert.NotNil(t, allowed)
	assert.Contains(t, allowed, "foo")
	assert.Contains(t, allowed, "bar")
}

// ---------------------------------------------------------------------------
// Export wrapper tests
// ---------------------------------------------------------------------------

func TestGetPublicKey(t *testing.T) {
	ResetJWTKeys()
	t.Cleanup(func() { initTestJWTKeys(t) })

	pub := GetPublicKey()
	assert.Nil(t, pub)

	initTestJWTKeys(t)
	pub = GetPublicKey()
	assert.NotNil(t, pub)
}

func TestGetAllPublicKeys(t *testing.T) {
	ResetJWTKeys()
	t.Cleanup(func() { initTestJWTKeys(t) })

	all := GetAllPublicKeys()
	assert.Empty(t, all)

	initTestJWTKeys(t)
	all = GetAllPublicKeys()
	assert.NotEmpty(t, all)
}

func TestValidateTokenClaims_SubNotString(t *testing.T) {
	claims := jwtlib.MapClaims{
		"sub": 123, "aud": "app", "iss": "https://example.com",
		"iat": jwtlib.NewNumericDate(time.Now()), "exp": jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "abc123",
	}
	err := validateTokenClaims(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subject")
}

func TestValidateTokenClaims_AudNotString(t *testing.T) {
	claims := jwtlib.MapClaims{
		"sub": "user", "aud": 456, "iss": "https://example.com",
		"iat": jwtlib.NewNumericDate(time.Now()), "exp": jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "abc123",
	}
	err := validateTokenClaims(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audience")
}

func TestValidateTokenClaims_IssNotString(t *testing.T) {
	claims := jwtlib.MapClaims{
		"sub": "user", "aud": "app", "iss": 789,
		"iat": jwtlib.NewNumericDate(time.Now()), "exp": jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "abc123",
	}
	err := validateTokenClaims(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "issuer")
}

func TestValidateTokenClaims_JTINotString(t *testing.T) {
	allowTestIssuers(t)
	claims := jwtlib.MapClaims{
		"sub": "user", "aud": "app", "iss": "https://example.com",
		"iat": jwtlib.NewNumericDate(time.Now()), "exp": jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": 321,
	}
	err := validateTokenClaims(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JTI")
}

// ---------------------------------------------------------------------------
// ValidateToken — signed token with empty sub/aud/iss/jti
// ---------------------------------------------------------------------------

func TestValidateToken_EmptySubInSignedToken(t *testing.T) {
	initTestJWTKeys(t)
	now := time.Now()
	claims := jwtlib.MapClaims{
		"sub": "", "aud": "app", "iss": "https://auth.example.com",
		"iat": jwtlib.NewNumericDate(now), "exp": jwtlib.NewNumericDate(now.Add(time.Hour)),
		"jti": "abc123",
	}
	sigPrivKey, sigKID := activeSigningKeyForTest()
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
	token.Header["kid"] = sigKID
	tokenString, err := token.SignedString(sigPrivKey)
	require.NoError(t, err)
	_, err = ValidateToken(tokenString)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subject")
}

func TestValidateToken_EmptyAudInSignedToken(t *testing.T) {
	initTestJWTKeys(t)
	now := time.Now()
	claims := jwtlib.MapClaims{
		"sub": "user", "aud": "", "iss": "https://auth.example.com",
		"iat": jwtlib.NewNumericDate(now), "exp": jwtlib.NewNumericDate(now.Add(time.Hour)),
		"jti": "abc123",
	}
	sigPrivKey, sigKID := activeSigningKeyForTest()
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
	token.Header["kid"] = sigKID
	tokenString, err := token.SignedString(sigPrivKey)
	require.NoError(t, err)
	_, err = ValidateToken(tokenString)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "audience")
}

func TestValidateToken_EmptyIssInSignedToken(t *testing.T) {
	initTestJWTKeys(t)
	now := time.Now()
	claims := jwtlib.MapClaims{
		"sub": "user", "aud": "app", "iss": "",
		"iat": jwtlib.NewNumericDate(now), "exp": jwtlib.NewNumericDate(now.Add(time.Hour)),
		"jti": "abc123",
	}
	sigPrivKey, sigKID := activeSigningKeyForTest()
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
	token.Header["kid"] = sigKID
	tokenString, err := token.SignedString(sigPrivKey)
	require.NoError(t, err)
	_, err = ValidateToken(tokenString)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "issuer")
}

func TestValidateToken_EmptyJTIInSignedToken(t *testing.T) {
	initTestJWTKeys(t)
	now := time.Now()
	claims := jwtlib.MapClaims{
		"sub": "user", "aud": "app", "iss": "https://auth.example.com",
		"iat": jwtlib.NewNumericDate(now), "exp": jwtlib.NewNumericDate(now.Add(time.Hour)),
		"jti": "",
	}
	sigPrivKey, sigKID := activeSigningKeyForTest()
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
	token.Header["kid"] = sigKID
	tokenString, err := token.SignedString(sigPrivKey)
	require.NoError(t, err)
	_, err = ValidateToken(tokenString)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JTI")
}

// ---------------------------------------------------------------------------
// ValidateTokenWithContext — no public key
// ---------------------------------------------------------------------------

func TestValidateTokenWithContext_NoPublicKey(t *testing.T) {
	ResetJWTKeys()

	_, err := ValidateTokenWithContext(context.Background(), "any.token.string")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "public key not initialized")
}

// ---------------------------------------------------------------------------
// Verification key — retired key lookup
// ---------------------------------------------------------------------------

func TestVerificationKey_RetiredKey(t *testing.T) {
	initTestJWTKeys(t)

	err := RotateKeys()
	require.NoError(t, err)

	all := GetAllPublicKeys()
	require.Len(t, all, 2)

	oldKID := all[1].KID
	pub, err := defaultKeyStore.verificationKey(oldKID)
	require.NoError(t, err)
	assert.NotNil(t, pub)
}

// ---------------------------------------------------------------------------
// GenerateToken — missing sub claim
// ---------------------------------------------------------------------------

func TestGenerateToken_MissingSub(t *testing.T) {
	initTestJWTKeys(t)
	claims := jwtlib.MapClaims{
		"aud": "myapp", "iss": "https://auth.example.com",
		"iat": jwtlib.NewNumericDate(time.Now()), "exp": jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		"jti": "test-jti",
	}
	_, err := generateToken(claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sub")
}

// ---------------------------------------------------------------------------
// GenerateAccessTokenWithOptionsContext — nil context
// ---------------------------------------------------------------------------

func TestGenerateAccessTokenWithOptionsContext_Success(t *testing.T) {
	initTestJWTKeys(t)
	tok, err := GenerateAccessTokenWithOptionsContext(context.Background(), "user-uuid", "read", "https://auth.example.com", "myapp", "client-1", "provider-1", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
}

// ---------------------------------------------------------------------------
// GenerateIDToken — nonce
// ---------------------------------------------------------------------------

func TestGenerateIDToken_Nonce(t *testing.T) {
	initTestJWTKeys(t)
	tok, err := GenerateIDToken("user-uuid", "https://auth.example.com", "client-1", "provider-1", nil, "mynonce", nil)
	require.NoError(t, err)

	claims, err := ValidateToken(tok)
	require.NoError(t, err)
	assert.Equal(t, "mynonce", claims["nonce"])
}
