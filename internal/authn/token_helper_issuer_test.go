package authn

import (
	"context"
	"testing"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unverifiedClaims decodes a token's payload without checking the signature.
// These tests assert on what was STAMPED, so verification is beside the point.
func unverifiedClaims(t *testing.T, token string) jwtlib.MapClaims {
	t.Helper()
	claims := jwtlib.MapClaims{}
	_, _, err := jwtlib.NewParser().ParseUnverified(token, claims)
	require.NoError(t, err)
	return claims
}

// Every token in one set must name the SAME issuer, and that issuer is the
// authorization server (APP_PUBLIC_HOSTNAME), never the client's own domain.
//
// The access token was moved onto jwt.TokenIssuer but the ID and refresh tokens
// were left stamping *client.Domain, so one login returned a set whose members
// disagreed about who issued them. The ID token is the one a relying party runs
// the OIDC Core §3.1.3.7 step-2 issuer comparison against, so a compliant RP
// rejected every first-party login and registration token.
func TestTokenSetIssuerIsTheAuthorizationServerNotTheClientDomain(t *testing.T) {
	initTestJWTKeysService(t)

	const asIssuer = "https://as.example.test"
	prev := config.AppPublicHostname
	config.AppPublicHostname = asIssuer
	t.Cleanup(func() { config.AppPublicHostname = prev })

	user := buildActiveUser(t, "Password123!")
	client := buildActiveClient()
	require.NotNil(t, client.Domain)
	require.NotEqual(t, asIssuer, *client.Domain, "fixture must differ from the AS issuer or this proves nothing")

	access, id, refresh, err := generateTokenSetWithAuthContext(
		context.Background(), "sub-1", user, client, tokenAuthContext{SessionID: "session-abc"},
	)
	require.NoError(t, err)

	for name, token := range map[string]string{"access": access, "id": id, "refresh": refresh} {
		claims := unverifiedClaims(t, token)
		assert.Equal(t, asIssuer, claims["iss"], "%s token must carry the authorization server's issuer", name)
		assert.NotEqual(t, *client.Domain, claims["iss"], "%s token must not carry the client domain", name)
	}
}

// sid is what lets a relying party act on a back-channel logout token:
// GenerateLogoutToken stamps sid, so the ID token has to carry the same value
// for the RP to know WHICH of its sessions to end (OIDC Back-Channel Logout 1.0
// §2.1). The ID token previously carried no sid at all.
func TestIDTokenCarriesTheSessionID(t *testing.T) {
	initTestJWTKeysService(t)

	user := buildActiveUser(t, "Password123!")
	client := buildActiveClient()

	const sessionID = "9e4d11a8-1757-45f1-9493-4fd33ae069e3"
	access, id, _, err := generateTokenSetWithAuthContext(
		context.Background(), "sub-1", user, client, tokenAuthContext{SessionID: sessionID},
	)
	require.NoError(t, err)

	assert.Equal(t, sessionID, unverifiedClaims(t, id)["sid"], "ID token must carry sid")
	assert.Equal(t, sessionID, unverifiedClaims(t, access)["sid"], "access token must carry sid")
}

// A token set minted outside any session (no SessionID) must simply omit sid
// rather than stamping an empty one, which an RP would read as a real session.
func TestIDTokenOmitsSessionIDWhenThereIsNoSession(t *testing.T) {
	initTestJWTKeysService(t)

	user := buildActiveUser(t, "Password123!")
	client := buildActiveClient()

	_, id, _, err := generateTokenSetWithAuthContext(
		context.Background(), "sub-1", user, client, tokenAuthContext{},
	)
	require.NoError(t, err)

	_, present := unverifiedClaims(t, id)["sid"]
	assert.False(t, present, "ID token must omit sid when there is no session")
}
