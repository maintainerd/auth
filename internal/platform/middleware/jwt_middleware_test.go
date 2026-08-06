package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initTestJWTKeys generates a fresh RSA-2048 key pair and wires it into the
// package-level config variables that GenerateAccessToken / ValidateToken read from.
func initTestJWTKeys(t *testing.T) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&priv.PublicKey)})
	config.JWTPrivateKey = privPEM
	config.JWTPublicKey = pubPEM
	require.NoError(t, jwt.InitJWTKeys())

	// jwt.validateIssuerClaim fails CLOSED on an empty allowlist — it then
	// accepts only APP_PUBLIC_HOSTNAME, which is unset in a test binary — so
	// every issuer these tests mint tokens under has to be registered the way a
	// deployment registers its clients at boot.
	jwt.SetAcceptedIssuers([]string{
		"https://auth.example.com",
		"https://api.example.com",
		"https://issuer",
	})
	t.Cleanup(jwt.ResetAcceptedIssuers)
}

// okHandler is a minimal next-handler that always responds 200 OK.
func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestJWTAuthMiddleware(t *testing.T) {
	initTestJWTKeys(t)

	validUserUUID := uuid.New().String()
	validToken, err := jwt.GenerateAccessToken(
		validUserUUID, "read", "https://auth.example.com",
		"https://api.example.com", "my-client", "provider-1",
	)
	require.NoError(t, err)

	cases := []struct {
		name         string
		setupRequest func(r *http.Request)
		wantStatus   int
	}{
		{
			name:         "no token → 401",
			setupRequest: func(_ *http.Request) {},
			wantStatus:   http.StatusUnauthorized,
		},
		{
			name: "invalid bearer token → 401",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer invalid.token.here")
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "wrong auth scheme → 401",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "valid bearer token → 200",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+validToken)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "valid cookie token → 200",
			setupRequest: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: "access_token", Value: validToken})
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tc.setupRequest(req)
			rr := httptest.NewRecorder()
			JWTAuthMiddleware(okHandler()).ServeHTTP(rr, req)
			assert.Equal(t, tc.wantStatus, rr.Code)
		})
	}
}

func TestJWTAuthMiddleware_ContextValues(t *testing.T) {
	initTestJWTKeys(t)

	userUUID := uuid.New().String()
	token, err := jwt.GenerateAccessTokenWithOptions(
		userUUID, "read write", "https://auth.example.com",
		"https://api.example.com", "my-client", "provider-1",
		&jwt.AccessTokenOptions{AMR: []string{jwt.AMRPassword, jwt.AMROTP}, ACR: jwt.ACRLevel2},
	)
	require.NoError(t, err)

	var capturedClientID, capturedProviderID string
	var capturedClaims *JWTClaims
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedClientID = GetClientIDFromContext(r)
		capturedProviderID = GetProviderIDFromContext(r)
		capturedClaims = JWTClaimsFromRequest(r)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	JWTAuthMiddleware(next).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "my-client", capturedClientID)
	assert.Equal(t, "provider-1", capturedProviderID)
	require.NotNil(t, capturedClaims)
	assert.Equal(t, jwt.ACRLevel2, capturedClaims.ACR)
	assert.ElementsMatch(t, []string{jwt.AMRPassword, jwt.AMROTP}, capturedClaims.AMR)
}

func TestJWTAuthMiddleware_ServiceClaims(t *testing.T) {
	initTestJWTKeys(t)

	token, err := jwt.GenerateAccessTokenWithOptions(
		"serviceA", "", "https://auth.example.com",
		"https://api.example.com", "service-client", "provider-1",
		&jwt.AccessTokenOptions{Service: "serviceA", SubjectType: "service"},
	)
	require.NoError(t, err)

	var capturedClaims *JWTClaims
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedClaims = JWTClaimsFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	JWTAuthMiddleware(next).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.NotNil(t, capturedClaims)
	assert.Equal(t, "serviceA", capturedClaims.Sub)
	assert.Equal(t, "serviceA", capturedClaims.Service)
	assert.Equal(t, "service", capturedClaims.SubjectType)
}

func TestGetClientIDFromContext(t *testing.T) {
	t.Run("present → returns value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = WithJWTClaims(req, &JWTClaims{ClientID: "test-client"})
		assert.Equal(t, "test-client", GetClientIDFromContext(req))
	})

	t.Run("absent → empty string", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		assert.Equal(t, "", GetClientIDFromContext(req))
	})
}

func TestGetProviderIDFromContext(t *testing.T) {
	t.Run("present → returns value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = WithJWTClaims(req, &JWTClaims{ProviderID: "test-provider"})
		assert.Equal(t, "test-provider", GetProviderIDFromContext(req))
	})

	t.Run("absent → empty string", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		assert.Equal(t, "", GetProviderIDFromContext(req))
	})
}

// makeTokenWithClaims signs a JWT with the RSA key loaded by initTestJWTKeys.
func makeTokenWithClaims(t *testing.T, claims jwtlib.MapClaims) string {
	t.Helper()
	block, _ := pem.Decode(config.JWTPrivateKey)
	require.NotNil(t, block, "JWTPrivateKey not initialised – call initTestJWTKeys first")
	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)
	tok := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
	s, err := tok.SignedString(privKey)
	require.NoError(t, err)
	return s
}

func TestJWTAuthMiddleware_MissingOrInvalidSub(t *testing.T) {
	initTestJWTKeys(t)

	exp := jwtlib.NewNumericDate(time.Now().Add(time.Hour))

	iat := jwtlib.NewNumericDate(time.Now().Add(-time.Second))
	baseClaims := jwtlib.MapClaims{
		"exp": exp, "iat": iat, "iss": "https://issuer", "aud": "https://api",
		"jti": uuid.New().String(),
	}

	t.Run("token missing sub → 401", func(t *testing.T) {
		tok := makeTokenWithClaims(t, baseClaims)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		rr := httptest.NewRecorder()
		JWTAuthMiddleware(okHandler()).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("sub is not a valid UUID passes through as pairwise subject", func(t *testing.T) {
		claims := jwtlib.MapClaims{
			"sub": "not-a-uuid", "exp": exp, "iat": iat,
			"iss": "https://issuer", "aud": "https://api",
			"jti": uuid.New().String(),
			// token_type is now required on the authorization path; this case is
			// about the shape of `sub`, so it has to present a real access token.
			"token_type": jwt.TokenTypeAccess,
		}
		tok := makeTokenWithClaims(t, claims)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		rr := httptest.NewRecorder()
		JWTAuthMiddleware(okHandler()).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

// TestJWTAuthMiddleware_RejectsNonAccessTokens covers the core of the ID-token
// confusion: every token this server signs shares one key and carries the same
// sub/client_id, so only the token_type claim separates a credential the
// resource server may act on from a receipt handed to a relying party.
func TestJWTAuthMiddleware_RejectsNonAccessTokens(t *testing.T) {
	initTestJWTKeys(t)

	exp := jwtlib.NewNumericDate(time.Now().Add(time.Hour))
	iat := jwtlib.NewNumericDate(time.Now().Add(-time.Second))
	sub := uuid.New().String()

	tokenWithType := func(tokenType string) string {
		claims := jwtlib.MapClaims{
			"sub": sub, "exp": exp, "iat": iat,
			"iss": "https://issuer", "aud": "https://api",
			"jti":       uuid.New().String(),
			"client_id": "console",
		}
		if tokenType != "" {
			claims["token_type"] = tokenType
		}
		return makeTokenWithClaims(t, claims)
	}

	rejected := []struct {
		name      string
		tokenType string
	}{
		{"id token", jwt.TokenTypeID},
		{"refresh token", jwt.TokenTypeRefresh},
		{"unlabelled token", ""},
	}
	for _, tc := range rejected {
		t.Run(tc.name+" is rejected", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+tokenWithType(tc.tokenType))
			rr := httptest.NewRecorder()
			JWTAuthMiddleware(okHandler()).ServeHTTP(rr, req)
			assert.Equal(t, http.StatusUnauthorized, rr.Code)
		})
	}

	accepted := []struct {
		name      string
		tokenType string
	}{
		{"access token", jwt.TokenTypeAccess},
		// A DPoP-bound access token carries token_type "DPoP" (RFC 9449 §6.1); it
		// is still an access token. Without a cnf claim there is no binding to
		// enforce, so it reaches the handler.
		{"DPoP access token", jwt.TokenTypeDPoP},
	}
	for _, tc := range accepted {
		t.Run(tc.name+" is accepted", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+tokenWithType(tc.tokenType))
			rr := httptest.NewRecorder()
			JWTAuthMiddleware(okHandler()).ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestRequireStepUp(t *testing.T) {
	t.Run("missing claims returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/sensitive", nil)
		rr := httptest.NewRecorder()

		RequireStepUp(okHandler()).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("acr below level 2 returns 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/sensitive", nil)
		req = WithJWTClaims(req, &JWTClaims{ACR: jwt.ACRLevel1})
		rr := httptest.NewRecorder()

		RequireStepUp(okHandler()).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
		// Clients branch on this stable code to trigger the step-up handshake.
		assert.Contains(t, rr.Body.String(), `"code":"step_up_required"`)
	})

	t.Run("acr level 2 passes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/sensitive", nil)
		req = WithJWTClaims(req, &JWTClaims{ACR: jwt.ACRLevel2})
		rr := httptest.NewRecorder()

		RequireStepUp(okHandler()).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
