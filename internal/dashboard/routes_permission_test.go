package dashboard

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/require"
)

const dashboardTestIssuer = "https://auth.dashboard.test"

type stubSummaryService struct{ called bool }

func (s *stubSummaryService) GetSummary(context.Context, int64) (*SummaryResponseDTO, error) {
	s.called = true
	return &SummaryResponseDTO{}, nil
}

// initDashboardJWT installs a throwaway key pair and issuer allowlist so the
// route's real JWTAuthMiddleware accepts the token minted below. Nothing in the
// chain is stubbed: the assertions run the middleware exactly as DashboardRoute
// composes it.
func initDashboardJWT(t *testing.T) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	prevPriv, prevPub, prevHost := config.JWTPrivateKey, config.JWTPublicKey, config.AppPublicHostname
	config.JWTPrivateKey = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	config.JWTPublicKey = pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&priv.PublicKey)})
	config.AppPublicHostname = dashboardTestIssuer
	require.NoError(t, jwt.InitJWTKeys())
	jwt.SetAcceptedIssuers([]string{dashboardTestIssuer})

	t.Cleanup(func() {
		jwt.ResetAcceptedIssuers()
		jwt.ResetJWTKeys()
		config.JWTPrivateKey, config.JWTPublicKey, config.AppPublicHostname = prevPriv, prevPub, prevHost
	})
}

// dashboardAuthContext builds an authenticated principal holding exactly the
// permissions named. It is pre-seeded on the request because
// UserContextMiddleware short-circuits when the principal is already resolved,
// which keeps the test off the database.
func dashboardAuthContext(permissions ...string) *authctx.AuthContext {
	perms := make([]authctx.AuthPermission, 0, len(permissions))
	for i, name := range permissions {
		perms = append(perms, authctx.AuthPermission{PermissionID: int64(i + 1), PermissionUUID: uuid.New(), Name: name})
	}
	return &authctx.AuthContext{
		User: &authctx.AuthUser{
			UserID:   1,
			UserUUID: uuid.New(),
			Roles:    []authctx.AuthRole{{RoleID: 1, RoleUUID: uuid.New(), Name: "member", Permissions: perms}},
		},
		Tenant: &authctx.AuthTenant{TenantID: 1, TenantUUID: uuid.New(), Name: "acme"},
	}
}

// GET /dashboard/summary used to carry JWT + user context and nothing else — no
// permission guard at all, the only management route in the internal router
// without one. Authentication is not authorization: a tenant member with a
// single self-service role could read tenant-wide user, client, identity
// provider, role and auth-event counts.
func TestDashboardSummaryRequiresTenantRead(t *testing.T) {
	initDashboardJWT(t)

	token, err := jwt.GenerateAccessToken(
		uuid.NewString(), "openid", dashboardTestIssuer,
		dashboardTestIssuer, "console", "provider-1",
	)
	require.NoError(t, err)

	serve := func(auth *authctx.AuthContext) (*httptest.ResponseRecorder, *stubSummaryService) {
		svc := &stubSummaryService{}
		r := chi.NewRouter()
		DashboardRoute(r, NewHandler(svc), nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/dashboard/summary", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req = middleware.WithAuthContext(req, auth)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr, svc
	}

	t.Run("an authenticated principal without tenant:read is refused", func(t *testing.T) {
		rr, svc := serve(dashboardAuthContext("profile:read:self"))
		require.Equal(t, http.StatusForbidden, rr.Code)
		require.False(t, svc.called, "the service must not be reached past a denial")
	})

	t.Run("a principal with no permissions at all is refused", func(t *testing.T) {
		// Fail closed: an empty permission set must not read like a wildcard.
		rr, svc := serve(dashboardAuthContext())
		require.Equal(t, http.StatusForbidden, rr.Code)
		require.False(t, svc.called)
	})

	t.Run("tenant:read is accepted", func(t *testing.T) {
		rr, svc := serve(dashboardAuthContext("tenant:read"))
		require.Equal(t, http.StatusOK, rr.Code)
		require.True(t, svc.called)
	})
}
