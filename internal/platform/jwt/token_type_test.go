package jwt

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every token this server issues is signed with the same key and an ID token
// carries the same sub and client_id an access token does, so token_type is the
// only thing that separates a credential a resource server may act on from a
// receipt handed to a relying party.
func TestValidateAccessTokenWithContext_EnforcesTokenType(t *testing.T) {
	initTestJWTKeys(t)
	ctx := context.Background()

	accessToken, err := GenerateAccessTokenWithContext(ctx,
		"user-1", "openid", "https://auth.example.com", "client-1", "client-1", "provider-1")
	require.NoError(t, err)

	idToken, err := GenerateIDTokenWithContext(ctx,
		"user-1", "https://auth.example.com", "client-1", "provider-1", &UserProfile{}, "", nil)
	require.NoError(t, err)

	refreshToken, err := GenerateRefreshTokenWithContext(ctx,
		"user-1", "https://auth.example.com", "client-1", "provider-1")
	require.NoError(t, err)

	t.Run("an access token is accepted", func(t *testing.T) {
		claims, err := ValidateAccessTokenWithContext(ctx, accessToken)
		require.NoError(t, err)
		assert.Equal(t, "user-1", claims["sub"])
	})

	t.Run("an ID token is rejected", func(t *testing.T) {
		// ID tokens are handed to relying parties by design, so treating one as a
		// bearer credential authenticates as the subject for the ID token's full
		// lifetime — and, carrying no sid, survives logout and revocation.
		_, err := ValidateAccessTokenWithContext(ctx, idToken)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrWrongTokenType)
	})

	t.Run("a refresh token is rejected", func(t *testing.T) {
		_, err := ValidateAccessTokenWithContext(ctx, refreshToken)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrWrongTokenType)
	})

	t.Run("a step-up challenge handle is rejected", func(t *testing.T) {
		// It carries `typ`, not `token_type`, so it must not pass an
		// access-token check by virtue of being unlabelled.
		challenge, err := GenerateStepUpChallengeToken("user-1", time.Minute)
		require.NoError(t, err)
		_, err = ValidateAccessTokenWithContext(ctx, challenge)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrWrongTokenType)
	})

	t.Run("ValidateToken itself still accepts every type", func(t *testing.T) {
		// The refresh, id_token_hint, logout-token and introspection paths all
		// need to validate their own kind of token, so the type check belongs on
		// the authorization path, not in ValidateToken.
		for name, tok := range map[string]string{
			"access":  accessToken,
			"id":      idToken,
			"refresh": refreshToken,
		} {
			_, err := ValidateTokenWithContext(ctx, tok)
			assert.NoError(t, err, name)
		}
	})

	t.Run("a DPoP-bound access token is still an access token", func(t *testing.T) {
		bound, err := GenerateAccessTokenWithOptionsContext(ctx,
			"user-1", "openid", "https://auth.example.com", "client-1", "client-1", "provider-1",
			&AccessTokenOptions{DPoPThumbprint: "thumb"})
		require.NoError(t, err)
		claims, err := ValidateAccessTokenWithContext(ctx, bound)
		require.NoError(t, err)
		assert.Equal(t, TokenTypeDPoP, claims["token_type"])
	})
}

// RFC 7519 §4.1.1: a recipient must reject a token whose issuer it does not
// recognize. That matters here because all tenants share one signing key, so
// the signature says nothing about which tenant a token came from.
func TestValidateToken_IssuerAllowlist(t *testing.T) {
	initTestJWTKeys(t)
	ctx := context.Background()
	t.Cleanup(ResetAcceptedIssuers)

	token, err := GenerateAccessTokenWithContext(ctx,
		"user-1", "openid", "https://tenant-a.example.com", "client-1", "client-1", "provider-1")
	require.NoError(t, err)

	// INVERTED. This used to assert "unconfigured allowlist accepts, so a
	// deployment is not bricked" — an empty allowlist returned nil for ANY
	// issuer. That made the tenant boundary vanish on a config failure: a
	// transient DB error in seedAcceptedIssuers, and every token from every
	// issuer validated for the life of the process. A security check may not be
	// switched off by the absence of configuration.
	t.Run("unconfigured allowlist rejects an issuer that is not the server's own", func(t *testing.T) {
		ResetAcceptedIssuers()
		origHostname := config.AppPublicHostname
		t.Cleanup(func() { config.AppPublicHostname = origHostname })
		config.AppPublicHostname = ""

		_, err := ValidateTokenWithContext(ctx, token)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a recognized issuer")
	})

	// The deployment is still not bricked: the server's own issuer is the one
	// value the process can vouch for without the database, and it is what every
	// token this server mints now carries (jwt.TokenIssuer).
	t.Run("unconfigured allowlist still accepts the server's own issuer", func(t *testing.T) {
		ResetAcceptedIssuers()
		origHostname := config.AppPublicHostname
		t.Cleanup(func() { config.AppPublicHostname = origHostname })
		config.AppPublicHostname = "https://tenant-a.example.com"

		_, err := ValidateTokenWithContext(ctx, token)
		assert.NoError(t, err)
	})

	t.Run("a recognized issuer is accepted", func(t *testing.T) {
		ResetAcceptedIssuers()
		SetAcceptedIssuers([]string{"https://tenant-a.example.com"})
		_, err := ValidateTokenWithContext(ctx, token)
		assert.NoError(t, err)
	})

	t.Run("an unrecognized issuer is rejected", func(t *testing.T) {
		ResetAcceptedIssuers()
		SetAcceptedIssuers([]string{"https://tenant-b.example.com"})
		_, err := ValidateTokenWithContext(ctx, token)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a recognized issuer")
	})

	t.Run("AddAcceptedIssuer extends the set at runtime", func(t *testing.T) {
		ResetAcceptedIssuers()
		SetAcceptedIssuers([]string{"https://tenant-b.example.com"})
		AddAcceptedIssuer("https://tenant-a.example.com")
		_, err := ValidateTokenWithContext(ctx, token)
		assert.NoError(t, err)
	})
}

// tenant_id is what middleware and the gRPC interceptor scope authorization on,
// so a caller must not be able to set it through the generic extra-claims map.
func TestAccessTokenOptions_TenantUUIDBeatsExtraClaims(t *testing.T) {
	initTestJWTKeys(t)
	ctx := context.Background()

	token, err := GenerateAccessTokenWithOptionsContext(ctx,
		"user-1", "openid", "https://auth.example.com", "client-1", "client-1", "provider-1",
		&AccessTokenOptions{
			TenantUUID:  "11111111-1111-1111-1111-111111111111",
			ExtraClaims: map[string]any{"tenant_id": "22222222-2222-2222-2222-222222222222"},
		})
	require.NoError(t, err)

	claims, err := ValidateAccessTokenWithContext(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", claims["tenant_id"])
}

func TestIDToken_AuthTimeAndAtHash(t *testing.T) {
	initTestJWTKeys(t)
	ctx := context.Background()

	accessToken, err := GenerateAccessTokenWithContext(ctx,
		"user-1", "openid", "https://auth.example.com", "client-1", "client-1", "provider-1")
	require.NoError(t, err)

	authTime := time.Now().Add(-3 * time.Hour).Truncate(time.Second)

	t.Run("auth_time reports the authentication, not the issuance", func(t *testing.T) {
		// Stamping issuance time made every token look freshly authenticated, so
		// an RP asking for max_age could never detect a stale session.
		idToken, err := GenerateIDTokenWithContext(ctx,
			"user-1", "https://auth.example.com", "client-1", "provider-1",
			&UserProfile{}, "", &IDTokenParams{AuthTime: authTime})
		require.NoError(t, err)

		claims, err := ValidateTokenWithContext(ctx, idToken)
		require.NoError(t, err)
		assert.EqualValues(t, authTime.Unix(), int64(claims["auth_time"].(float64)))
		assert.NotEqual(t, int64(claims["iat"].(float64)), int64(claims["auth_time"].(float64)))
	})

	t.Run("auth_time falls back to now when the caller has none", func(t *testing.T) {
		idToken, err := GenerateIDTokenWithContext(ctx,
			"user-1", "https://auth.example.com", "client-1", "provider-1", &UserProfile{}, "", nil)
		require.NoError(t, err)

		claims, err := ValidateTokenWithContext(ctx, idToken)
		require.NoError(t, err)
		assert.InDelta(t, time.Now().Unix(), int64(claims["auth_time"].(float64)), 5)
	})

	t.Run("at_hash is the left-most half of the access token's SHA-256", func(t *testing.T) {
		idToken, err := GenerateIDTokenWithContext(ctx,
			"user-1", "https://auth.example.com", "client-1", "provider-1",
			&UserProfile{}, "", &IDTokenParams{AccessToken: accessToken})
		require.NoError(t, err)

		claims, err := ValidateTokenWithContext(ctx, idToken)
		require.NoError(t, err)

		sum := sha256.Sum256([]byte(accessToken))
		want := base64.RawURLEncoding.EncodeToString(sum[:16])
		assert.Equal(t, want, claims["at_hash"], "OIDC Core §3.1.3.6")
	})

	t.Run("at_hash is omitted when no access token is supplied", func(t *testing.T) {
		idToken, err := GenerateIDTokenWithContext(ctx,
			"user-1", "https://auth.example.com", "client-1", "provider-1", &UserProfile{}, "", nil)
		require.NoError(t, err)

		claims, err := ValidateTokenWithContext(ctx, idToken)
		require.NoError(t, err)
		assert.NotContains(t, claims, "at_hash")
	})

	// at_hash is defined against the hash of the ID token's OWN alg, so the
	// SHA-256 assumption in computeAtHash has to hold for every algorithm the
	// key store can actually sign with. RS256 and PS256 are both SHA-256; adding
	// an RS384/RS512 branch to generateTokenWithAlgorithm without revisiting
	// computeAtHash would silently emit an at_hash no RP can verify.
	t.Run("at_hash is SHA-256 for PS256 too", func(t *testing.T) {
		idToken, err := GenerateIDTokenWithContext(ctx,
			"user-1", "https://auth.example.com", "client-1", "provider-1",
			&UserProfile{}, "", &IDTokenParams{AccessToken: accessToken, SigningAlgorithm: "PS256"})
		require.NoError(t, err)

		claims, err := ValidateTokenWithContext(ctx, idToken)
		require.NoError(t, err)

		sum := sha256.Sum256([]byte(accessToken))
		assert.Equal(t, base64.RawURLEncoding.EncodeToString(sum[:16]), claims["at_hash"])
	})
}

// ID-token ExtraClaims are merged LAST — after at_hash, auth_time and the
// registered claims — so for client-configured mappers the sanitizer at the
// read boundary is the only thing standing between an operator-supplied mapper
// and a forged authentication receipt.
func TestIDToken_SanitizedMappersCannotDisplaceAuthClaims(t *testing.T) {
	initTestJWTKeys(t)
	ctx := context.Background()

	accessToken, err := GenerateAccessTokenWithContext(ctx,
		"user-1", "openid", "https://auth.example.com", "client-1", "client-1", "provider-1")
	require.NoError(t, err)

	authTime := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	hostile := map[string]any{
		"at_hash":   "forged-at-hash",
		"auth_time": time.Now().Unix(),
		"sub":       "victim",
		"org_id":    "acme",
	}

	idToken, err := GenerateIDTokenWithContext(ctx,
		"user-1", "https://auth.example.com", "client-1", "provider-1", &UserProfile{}, "",
		&IDTokenParams{
			AccessToken: accessToken,
			AuthTime:    authTime,
			ExtraClaims: SanitizeClientClaimMappers(hostile),
		})
	require.NoError(t, err)

	claims, err := ValidateTokenWithContext(ctx, idToken)
	require.NoError(t, err)

	sum := sha256.Sum256([]byte(accessToken))
	assert.Equal(t, base64.RawURLEncoding.EncodeToString(sum[:16]), claims["at_hash"])
	assert.EqualValues(t, authTime.Unix(), int64(claims["auth_time"].(float64)))
	assert.Equal(t, "user-1", claims["sub"])
	assert.Equal(t, "acme", claims["org_id"], "the legitimate mapper claim still lands")
}
