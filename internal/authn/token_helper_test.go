package authn

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
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

	// The tenant_id claim VALUE is the tenant's opaque UUID (resolved from the PK
	// via the injected resolver), NOT the internal PK.
	tenantUUID := uuid.New()
	shared.SetTenantRefResolver(staticTenantRef{id: 42, id2uuid: tenantUUID})
	t.Cleanup(func() { shared.SetTenantRefResolver(nil) })

	accessToken, _, _, err := generateTokenSetWithAuthContext(context.Background(), "sub-1", user, client, tokenAuthContext{
		ExtraAccessClaims: map[string]any{"tenant_id": 0},
	})
	require.NoError(t, err)

	claims, err := jwt.ValidateToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, tenantUUID.String(), claims["tenant_id"], "tenant_id claim must carry the opaque UUID, not the PK")
}

// staticTenantRef is a test double for shared.TenantRefResolver mapping one
// tenant's internal id to its uuid.
type staticTenantRef struct {
	id      int64
	id2uuid uuid.UUID
}

func (s staticTenantRef) TenantUUIDByID(_ context.Context, id int64) (uuid.UUID, bool) {
	if id == s.id {
		return s.id2uuid, true
	}
	return uuid.Nil, false
}

func (s staticTenantRef) TenantIDByUUID(_ context.Context, u uuid.UUID) (int64, bool) {
	if u == s.id2uuid {
		return s.id, true
	}
	return 0, false
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
