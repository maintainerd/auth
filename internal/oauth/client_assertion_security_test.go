package oauth

import (
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These cases could not be written before: the audience check was
// `strings.HasPrefix(aud, *client.Domain)` and there was no replay store, so an
// assertion addressed to the CLIENT (or to any host merely starting with the
// registered domain) authenticated it, over and over.
func TestValidateAssertionClaims_Security(t *testing.T) {
	clientID := "my-client"
	domain := "https://app.example.com"
	client := &Client{ClientID: 1, Identifier: &clientID, Domain: ptr.Ptr(domain)}

	base := func() jwtlib.MapClaims {
		return jwtlib.MapClaims{
			"iss": clientID,
			"sub": clientID,
			"aud": config.AppPublicHostname,
			"jti": newTestAssertionJTI(),
			"iat": float64(time.Now().Unix()),
			"exp": float64(time.Now().Add(2 * time.Minute).Unix()),
		}
	}

	t.Run("an assertion addressed to the client's own domain is refused", func(t *testing.T) {
		claims := base()
		claims["aud"] = domain
		err := validateAssertionClaims(claims, client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "audience")
	})

	t.Run("a host that merely shares the domain prefix is refused", func(t *testing.T) {
		claims := base()
		claims["aud"] = domain + ".evil.test"
		err := validateAssertionClaims(claims, client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "audience")
	})

	t.Run("the token endpoint URL is an accepted audience", func(t *testing.T) {
		claims := base()
		claims["aud"] = config.AppPublicHostname + "/api/v1/oauth/token"
		require.NoError(t, validateAssertionClaims(claims, client))
	})

	t.Run("aud may be an array containing the authorization server", func(t *testing.T) {
		claims := base()
		claims["aud"] = []any{"https://elsewhere.example", config.AppPublicHostname}
		require.NoError(t, validateAssertionClaims(claims, client))
	})

	t.Run("a jti may authenticate only once", func(t *testing.T) {
		claims := base()
		require.NoError(t, validateAssertionClaims(claims, client))

		replay := base()
		replay["jti"] = claims["jti"]
		err := validateAssertionClaims(replay, client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already been used")
	})

	t.Run("an assertion with no jti is refused", func(t *testing.T) {
		claims := base()
		delete(claims, "jti")
		require.Error(t, validateAssertionClaims(claims, client))
	})

	// jwtlib.WithLeeway(assertionMaxAge) widened the exp window instead of
	// capping the lifetime, so a client could mint an assertion that never
	// practically expires.
	t.Run("an assertion whose lifetime exceeds the maximum is refused", func(t *testing.T) {
		claims := base()
		claims["exp"] = float64(time.Now().Add(24 * time.Hour).Unix())
		err := validateAssertionClaims(claims, client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lifetime")
	})

	t.Run("an assertion older than the maximum is refused", func(t *testing.T) {
		claims := base()
		claims["iat"] = float64(time.Now().Add(-time.Hour).Unix())
		err := validateAssertionClaims(claims, client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "older")
	})

	// Fails CLOSED: with no APP_PUBLIC_HOSTNAME there is no audience this server
	// can prove was meant for it.
	t.Run("no configured issuer means no assertion can be verified", func(t *testing.T) {
		orig := config.AppPublicHostname
		config.AppPublicHostname = ""
		defer func() { config.AppPublicHostname = orig }()

		claims := base()
		claims["aud"] = orig
		err := validateAssertionClaims(claims, client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be verified")
	})
}

func TestValidateClientJWKSURI(t *testing.T) {
	t.Run("https is accepted", func(t *testing.T) {
		require.NoError(t, validateClientJWKSURI("https://rp.example.com/jwks.json"))
	})

	t.Run("http is only accepted for loopback", func(t *testing.T) {
		require.NoError(t, validateClientJWKSURI("http://localhost:9000/jwks.json"))
		require.Error(t, validateClientJWKSURI("http://rp.example.com/jwks.json"))
	})

	// The token endpoint is unauthenticated up to the point findClientJWK runs, so
	// a jwks_uri pointing at an internal address would make this server an SSRF
	// probe whose result leaks through the authentication error.
	t.Run("private and loopback literals are refused", func(t *testing.T) {
		for _, uri := range []string{
			"https://127.0.0.1/jwks.json",
			"https://10.0.0.5/jwks.json",
			"https://169.254.169.254/latest/meta-data",
			"https://[::1]/jwks.json",
		} {
			require.Error(t, validateClientJWKSURI(uri), uri)
		}
	})

	t.Run("non-http schemes are refused", func(t *testing.T) {
		require.Error(t, validateClientJWKSURI("file:///etc/passwd"))
		require.Error(t, validateClientJWKSURI(""))
	})
}

func TestClientHasVerificationMaterial(t *testing.T) {
	t.Run("a jwks_uri counts as verification material", func(t *testing.T) {
		assert.True(t, clientHasVerificationMaterial(&Client{JWKSUri: ptr.Ptr("https://rp.example.com/jwks.json")}))
	})

	t.Run("neither inline JWKS nor jwks_uri means none", func(t *testing.T) {
		assert.False(t, clientHasVerificationMaterial(&Client{}))
		assert.False(t, clientHasVerificationMaterial(&Client{JWKSUri: ptr.Ptr("   ")}))
	})
}
