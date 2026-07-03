package authn

import (
	"context"
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenHelper_GenerateTokenSet(t *testing.T) {
	initTestJWTKeysService(t)
	user := buildActiveUser(t, "Password123!")
	user.Fullname = "Test User"
	client := buildActiveClient()

	accessToken, idToken, refreshToken, err := generateTokenSet("sub-1", user, client)

	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, idToken)
	assert.NotEmpty(t, refreshToken)
}

func TestTokenHelper_GenerateTokenSetWithAuthContext_Defaults(t *testing.T) {
	initTestJWTKeysService(t)
	user := buildActiveUser(t, "Password123!")
	client := buildActiveClient()

	accessToken, idToken, refreshToken, err := generateTokenSetWithAuthContext(context.Background(), "sub-1", user, client, tokenAuthContext{})

	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, idToken)
	assert.NotEmpty(t, refreshToken)
}

func TestTokenHelper_GenerateTokenSetWithAuthContext_CustomContext(t *testing.T) {
	initTestJWTKeysService(t)
	user := buildActiveUser(t, "Password123!")
	client := buildActiveClient()

	accessToken, idToken, refreshToken, err := generateTokenSetWithAuthContext(context.Background(), "sub-1", user, client, tokenAuthContext{
		AMR:       []string{jwt.AMRPassword, jwt.AMROTP},
		ACR:       jwt.ACRLevel2,
		SessionID: "session-1",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, idToken)
	assert.NotEmpty(t, refreshToken)
}

func TestTokenHelper_GenerateTokenSetWithAuthContext_StampsTenantClaim(t *testing.T) {
	initTestJWTKeysService(t)
	user := buildActiveUser(t, "Password123!")
	client := buildActiveClient()
	client.TenantID = 42

	accessToken, _, _, err := generateTokenSetWithAuthContext(context.Background(), "sub-1", user, client, tokenAuthContext{
		ExtraAccessClaims: map[string]any{"tenant_id": 0},
	})
	require.NoError(t, err)

	claims, err := jwt.ValidateToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, float64(42), claims["tenant_id"])
}

func TestTokenHelper_ResponseBuilders(t *testing.T) {
	issuedAt := time.Now().Unix()
	login := buildLoginTokenResponse("access", "id", "refresh", issuedAt)
	register := buildRegisterTokenResponse("access", "id", "refresh", issuedAt)

	assert.Equal(t, "Bearer", login.TokenType)
	assert.Equal(t, issuedAt, login.IssuedAt)
	assert.Equal(t, "Bearer", register.TokenType)
	assert.Equal(t, issuedAt, register.IssuedAt)
}
