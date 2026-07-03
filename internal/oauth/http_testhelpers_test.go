package oauth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/require"
)

var (
	testUserUUID     = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	testResourceUUID = uuid.MustParse("00000000-0000-0000-0000-000000000099")
	testTenantUUID   = uuid.MustParse("00000000-0000-0000-0000-000000000001")

	errNotFound  = apperror.NewNotFoundWithReason("not found")
	errForbidden = apperror.NewForbidden("access denied")
)

func jsonReq(t *testing.T, method, url string, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	r := httptest.NewRequest(method, url, &buf)
	r.Header.Set("Content-Type", "application/json")
	return r
}

func badJSONReq(t *testing.T, method, url string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(method, url, strings.NewReader("{invalid"))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func pemEncodeRSAPrivateKey(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

func pemEncodeRSAPublicKey(pub *rsa.PublicKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(pub),
	})
}

// initTestJWTKeysService generates a fresh RSA key pair and wires it into the
// package-level config so the JWT helpers can sign/verify tokens in tests.
func initTestJWTKeysService(t *testing.T) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	config.JWTPrivateKey = pemEncodeRSAPrivateKey(key)
	config.JWTPublicKey = pemEncodeRSAPublicKey(&key.PublicKey)
	require.NoError(t, jwt.InitJWTKeys())
}

// withUser injects an authenticated user into the request context.
func withUser(r *http.Request) *http.Request {
	user := &authctx.AuthUser{UserID: 1, UserUUID: testUserUUID}
	tenant := &authctx.AuthTenant{TenantID: 1, TenantUUID: testTenantUUID}
	return middleware.WithAuthContext(r, &authctx.AuthContext{User: user, Tenant: tenant})
}

// withChiParam injects a chi URL parameter into the request context.
func withChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		rctx = chi.NewRouteContext()
	}
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}
