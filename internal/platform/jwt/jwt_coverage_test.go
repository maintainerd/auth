package jwt

import (
	"context"
	"crypto/rsa"
	"errors"
	"io"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var nilTestContext context.Context

func TestRotateKeys_KeyGenerationFailure(t *testing.T) {
	initTestJWTKeys(t)

	oldGenerateRSAKey := generateRSAKey
	generateRSAKey = func(io.Reader, int) (*rsa.PrivateKey, error) {
		return nil, errors.New("entropy unavailable")
	}
	t.Cleanup(func() {
		generateRSAKey = oldGenerateRSAKey
		initTestJWTKeys(t)
	})

	err := RotateKeys()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "generate rotation key")
}

func TestGenerateAccessTokenWithOptionsContext_AllClaimsWithNilContext(t *testing.T) {
	initTestJWTKeys(t)

	opts := &AccessTokenOptions{
		DPoPThumbprint: "thumbprint",
		AccessTokenTTL: time.Minute,
		AMR:            []string{"pwd", "otp"},
		ACR:            "2",
		SessionID:      "session-1",
		Service:        "billing-api",
		SubjectType:    "service",
	}
	tok, err := GenerateAccessTokenWithOptionsContext(nilTestContext, "billing-api", "read", "https://auth.example.com", "api", "client-1", "provider-1", opts)
	require.NoError(t, err)

	claims, err := ValidateToken(tok)
	require.NoError(t, err)
	assert.Equal(t, "DPoP", claims["token_type"])
	assert.Equal(t, []any{"pwd", "otp"}, claims["amr"])
	assert.Equal(t, "2", claims["acr"])
	assert.Equal(t, "session-1", claims["sid"])
	assert.Equal(t, "billing-api", claims["svc"])
	assert.Equal(t, "service", claims["sub_type"])
}

func TestGenerateAccessTokenWithOptionsContext_SigningKeyError(t *testing.T) {
	initTestJWTKeys(t)
	ResetJWTKeys()
	t.Cleanup(func() { initTestJWTKeys(t) })

	_, err := GenerateAccessTokenWithOptionsContext(context.Background(), "user-uuid", "read", "https://auth.example.com", "api", "client-1", "provider-1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private key not initialized")
}

func TestGenerateIDTokenWithContext_NilContextAndNameClaim(t *testing.T) {
	initTestJWTKeys(t)

	profile := &UserProfile{Name: "Jane Doe"}
	tok, err := GenerateIDTokenWithContext(nilTestContext, "user-uuid", "https://auth.example.com", "client-1", "provider-1", profile, "", nil)
	require.NoError(t, err)

	claims, err := ValidateToken(tok)
	require.NoError(t, err)
	assert.Equal(t, "Jane Doe", claims["name"])
}

func TestGenerateIDTokenWithContext_SigningKeyError(t *testing.T) {
	initTestJWTKeys(t)
	ResetJWTKeys()
	t.Cleanup(func() { initTestJWTKeys(t) })

	_, err := GenerateIDTokenWithContext(context.Background(), "user-uuid", "https://auth.example.com", "client-1", "provider-1", nil, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private key not initialized")
}

func TestGenerateRefreshTokenWithContext_NilContextAndSigningKeyError(t *testing.T) {
	initTestJWTKeys(t)

	tok, err := GenerateRefreshTokenWithContext(nilTestContext, "user-uuid", "https://auth.example.com", "client-1", "provider-1")
	require.NoError(t, err)
	assert.NotEmpty(t, tok)

	ResetJWTKeys()
	t.Cleanup(func() { initTestJWTKeys(t) })

	_, err = GenerateRefreshTokenWithContext(context.Background(), "user-uuid", "https://auth.example.com", "client-1", "provider-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private key not initialized")
}

func TestGenerateStepUpChallengeTokenWithContext_NilContextAllowedMethodsAndSigningKeyError(t *testing.T) {
	initTestJWTKeys(t)
	oldHostname := config.AppPublicHostname
	config.AppPublicHostname = ""
	t.Cleanup(func() { config.AppPublicHostname = oldHostname })

	tok, err := GenerateStepUpChallengeTokenWithContext(nilTestContext, "user-uuid", time.Minute, []string{"totp"})
	require.NoError(t, err)

	claims, err := ValidateStepUpChallengeToken(tok)
	require.NoError(t, err)
	assert.Equal(t, "maintainerd-auth", claims["iss"])
	assert.Equal(t, []any{"totp"}, claims["allowed_methods"])

	ResetJWTKeys()
	t.Cleanup(func() { initTestJWTKeys(t) })

	_, err = GenerateStepUpChallengeTokenWithContext(context.Background(), "user-uuid", time.Minute)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "step-up challenge token")
}

func TestValidateTokenWithContext_NilContextAndJTICheckError(t *testing.T) {
	initTestJWTKeys(t)

	tok, err := GenerateAccessToken("user-uuid", "read", "https://auth.example.com", "api", "client-1", "provider-1")
	require.NoError(t, err)

	checkErr := errors.New("store down")
	SetJTIChecker(func(ctx context.Context, jti string) (bool, error) {
		require.NotNil(t, ctx)
		return false, checkErr
	})
	t.Cleanup(ResetJTIChecker)

	_, err = ValidateTokenWithContext(nilTestContext, tok)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token revocation check failed")
}

func TestValidateTokenWithContext_JTICheckSkippedWithoutJTIString(t *testing.T) {
	initTestJWTKeys(t)

	now := time.Now()
	claims := jwtlib.MapClaims{
		"sub": "user-uuid",
		"aud": "api",
		"iss": "https://auth.example.com",
		"iat": jwtlib.NewNumericDate(now),
		"exp": jwtlib.NewNumericDate(now.Add(time.Hour)),
		"jti": "jti-1",
	}
	sigPrivKey, sigKID := activeSigningKeyForTest()
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
	token.Header["kid"] = sigKID
	tokenString, err := token.SignedString(sigPrivKey)
	require.NoError(t, err)

	SetJTIChecker(func(ctx context.Context, jti string) (bool, error) {
		return false, nil
	})
	t.Cleanup(ResetJTIChecker)

	validated, err := ValidateTokenWithContext(context.Background(), tokenString)
	require.NoError(t, err)
	assert.Equal(t, "jti-1", validated["jti"])
}
