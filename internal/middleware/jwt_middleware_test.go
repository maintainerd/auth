package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/config"
	"github.com/maintainerd/auth/internal/util"
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
	require.NoError(t, util.InitJWTKeys())
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
	validToken, err := util.GenerateAccessToken(
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
	token, err := util.GenerateAccessToken(
		userUUID, "read write", "https://auth.example.com",
		"https://api.example.com", "my-client", "provider-1",
	)
	require.NoError(t, err)

	var capturedClientID, capturedProviderID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedClientID = GetClientIDFromContext(r)
		capturedProviderID = GetProviderIDFromContext(r)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	JWTAuthMiddleware(next).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "my-client", capturedClientID)
	assert.Equal(t, "provider-1", capturedProviderID)
}

func TestGetClientIDFromContext(t *testing.T) {
	t.Run("present → returns value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := context.WithValue(req.Context(), ClientIDKey, "test-client")
		assert.Equal(t, "test-client", GetClientIDFromContext(req.WithContext(ctx)))
	})

	t.Run("absent → empty string", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		assert.Equal(t, "", GetClientIDFromContext(req))
	})
}

func TestGetProviderIDFromContext(t *testing.T) {
	t.Run("present → returns value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := context.WithValue(req.Context(), ProviderIDKey, "test-provider")
		assert.Equal(t, "test-provider", GetProviderIDFromContext(req.WithContext(ctx)))
	})

	t.Run("absent → empty string", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		assert.Equal(t, "", GetProviderIDFromContext(req))
	})
}

// makeTokenWithClaims signs a JWT with the RSA key loaded by initTestJWTKeys.
func makeTokenWithClaims(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	block, _ := pem.Decode(config.JWTPrivateKey)
	require.NotNil(t, block, "JWTPrivateKey not initialised – call initTestJWTKeys first")
	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	s, err := tok.SignedString(privKey)
	require.NoError(t, err)
	return s
}

func TestJWTAuthMiddleware_MissingOrInvalidSub(t *testing.T) {
	initTestJWTKeys(t)

	exp := jwt.NewNumericDate(time.Now().Add(time.Hour))

	iat := jwt.NewNumericDate(time.Now().Add(-time.Second))
	baseClaims := jwt.MapClaims{
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

	t.Run("sub is not a valid UUID → 400", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub": "not-a-uuid", "exp": exp, "iat": iat,
			"iss": "https://issuer", "aud": "https://api",
			"jti": uuid.New().String(),
		}
		tok := makeTokenWithClaims(t, claims)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		rr := httptest.NewRecorder()
		JWTAuthMiddleware(okHandler()).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
