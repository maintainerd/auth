package idp

import (
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
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/require"
)

const idpStepUpTestIssuer = "https://auth.idp-stepup.test"

// initIDPStepUpJWT installs a throwaway RSA key pair and an issuer allowlist so
// the routes' real JWTAuthMiddleware accepts the tokens minted below. Nothing is
// stubbed: the assertions below run the production middleware chain exactly as
// IdentityProviderRoute composes it.
func initIDPStepUpJWT(t *testing.T) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	prevPriv, prevPub, prevHost := config.JWTPrivateKey, config.JWTPublicKey, config.AppPublicHostname
	config.JWTPrivateKey = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	config.JWTPublicKey = pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&priv.PublicKey)})
	config.AppPublicHostname = idpStepUpTestIssuer
	require.NoError(t, jwt.InitJWTKeys())
	jwt.SetAcceptedIssuers([]string{idpStepUpTestIssuer})

	t.Cleanup(func() {
		jwt.ResetAcceptedIssuers()
		jwt.ResetJWTKeys()
		config.JWTPrivateKey, config.JWTPublicKey, config.AppPublicHostname = prevPriv, prevPub, prevHost
	})
}

func idpStepUpToken(t *testing.T, acr string) string {
	t.Helper()
	token, err := jwt.GenerateAccessTokenWithOptions(
		uuid.NewString(), "openid", idpStepUpTestIssuer,
		idpStepUpTestIssuer, "idp-console", "provider-1",
		&jwt.AccessTokenOptions{ACR: acr},
	)
	require.NoError(t, err)
	return token
}

// idpStepUpAuthContext gives the caller every identity-provider permission, so a
// rejection can only come from the step-up gate and never from
// PermissionMiddleware. It is pre-seeded on the request because
// UserContextMiddleware short-circuits when the principal is already resolved,
// which keeps the test off the database.
func idpStepUpAuthContext() *authctx.AuthContext {
	names := []string{"idp:read", "idp:create", "idp:update", "idp:delete"}
	perms := make([]authctx.AuthPermission, 0, len(names))
	for i, name := range names {
		perms = append(perms, authctx.AuthPermission{PermissionID: int64(i + 1), PermissionUUID: uuid.New(), Name: name})
	}
	return &authctx.AuthContext{
		User: &authctx.AuthUser{
			UserID:   1,
			UserUUID: uuid.New(),
			Roles:    []authctx.AuthRole{{RoleID: 1, RoleUUID: uuid.New(), Name: "idp-admin", Permissions: perms}},
		},
		Tenant: &authctx.AuthTenant{TenantID: 1, TenantUUID: uuid.New(), Name: "acme"},
	}
}

func idpStepUpCode(t *testing.T, body []byte) string {
	t.Helper()
	var decoded struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return ""
	}
	return decoded.Code
}

func serveIDPStepUp(t *testing.T, router chi.Router, method, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req = middleware.WithAuthContext(req, idpStepUpAuthContext())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// Identity-provider mutations used to be reachable with a plain acr=1 session:
// every one of them carried a permission check and nothing else, while the
// comparable client and role mutations already demanded step-up.
//
// An identity_providers row IS an authentication trust anchor. Its issuer, JWKS
// certificate, allowed audiences and allow_jit_provisioning flag decide whose
// assertions are believed here and whether believing one mints a brand-new user.
// Re-pointing a JIT-enabled provider at an issuer the attacker controls — or
// flipping a dormant provider to active — turns a hijacked console session into
// "sign in as anybody", with no fresh proof of presence anywhere in the path.
func TestIdentityProviderMutationRoutesRequireStepUp(t *testing.T) {
	initIDPStepUpJWT(t)

	// The mutation hooks return a non-nil result because the handlers dereference
	// it on the happy path; the acr=2 leg of this test has to reach them.
	svc := &mockIdentityProviderService{
		getByUUIDFn: func(uuid.UUID, int64) (*IdentityProviderServiceDataResult, error) {
			return &IdentityProviderServiceDataResult{}, nil
		},
		createFn: func(IdentityProviderCreateInput) (*IdentityProviderServiceDataResult, error) {
			return &IdentityProviderServiceDataResult{}, nil
		},
		updateFn: func(IdentityProviderUpdateInput) (*IdentityProviderServiceDataResult, error) {
			return &IdentityProviderServiceDataResult{}, nil
		},
		setStatusByUUIDFn: func(uuid.UUID, string, int64, uuid.UUID) (*IdentityProviderServiceDataResult, error) {
			return &IdentityProviderServiceDataResult{}, nil
		},
		deleteByUUIDFn: func(uuid.UUID, int64, uuid.UUID) (*IdentityProviderServiceDataResult, error) {
			return &IdentityProviderServiceDataResult{}, nil
		},
	}

	router := chi.NewRouter()
	IdentityProviderRoute(router, NewIdentityProviderHandler(svc), nil, nil, nil)

	acr1 := idpStepUpToken(t, jwt.ACRLevel1)
	acr2 := idpStepUpToken(t, jwt.ACRLevel2)
	idpUUID := uuid.NewString()

	gated := []struct {
		method string
		path   string
	}{
		// Create is gated too, unlike the iam create routes. A new IdP is not
		// inert: it is matched by issuer for token federation the moment it is
		// active, so creating one is itself a way to add a trusted issuer.
		{http.MethodPost, "/identity_providers/"},
		{http.MethodPut, "/identity_providers/" + idpUUID},
		{http.MethodPut, "/identity_providers/" + idpUUID + "/status"},
		{http.MethodDelete, "/identity_providers/" + idpUUID},
	}

	for _, tc := range gated {
		t.Run("acr=1 rejected: "+tc.method+" "+tc.path, func(t *testing.T) {
			rr := serveIDPStepUp(t, router, tc.method, tc.path, acr1)
			require.Equal(t, http.StatusForbidden, rr.Code, "body=%s", rr.Body.String())
			require.Equal(t, "step_up_required", idpStepUpCode(t, rr.Body.Bytes()))
		})

		t.Run("acr=2 admitted: "+tc.method+" "+tc.path, func(t *testing.T) {
			rr := serveIDPStepUp(t, router, tc.method, tc.path, acr2)
			require.NotEqual(t, "step_up_required", idpStepUpCode(t, rr.Body.Bytes()),
				"a stepped-up caller must get past the gate; body=%s", rr.Body.String())
		})
	}

	// Reads stay outside the gate. Asserted so the gate is not quietly widened
	// into a blanket acr=2 requirement for the whole surface.
	ungated := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/identity_providers/"},
		{http.MethodGet, "/identity_providers/" + idpUUID},
	}

	for _, tc := range ungated {
		t.Run("acr=1 allowed past the gate: "+tc.method+" "+tc.path, func(t *testing.T) {
			rr := serveIDPStepUp(t, router, tc.method, tc.path, acr1)
			require.NotEqual(t, "step_up_required", idpStepUpCode(t, rr.Body.Bytes()),
				"body=%s", rr.Body.String())
		})
	}
}
