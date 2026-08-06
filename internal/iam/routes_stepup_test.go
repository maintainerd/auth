package iam

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

const stepUpTestIssuer = "https://auth.iam-stepup.test"

// initIAMStepUpJWT installs a throwaway RSA key pair and an issuer allowlist so
// the routes' real JWTAuthMiddleware accepts the tokens minted below. Nothing is
// stubbed: the assertions below run the production middleware chain exactly as
// APIRoute/PermissionRoute/PolicyRoute/ServiceRoute compose it.
func initIAMStepUpJWT(t *testing.T) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	prevPriv, prevPub, prevHost := config.JWTPrivateKey, config.JWTPublicKey, config.AppPublicHostname
	config.JWTPrivateKey = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	config.JWTPublicKey = pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&priv.PublicKey)})
	config.AppPublicHostname = stepUpTestIssuer
	require.NoError(t, jwt.InitJWTKeys())
	jwt.SetAcceptedIssuers([]string{stepUpTestIssuer})

	t.Cleanup(func() {
		jwt.ResetAcceptedIssuers()
		jwt.ResetJWTKeys()
		config.JWTPrivateKey, config.JWTPublicKey, config.AppPublicHostname = prevPriv, prevPub, prevHost
	})
}

func iamStepUpToken(t *testing.T, acr string) string {
	t.Helper()
	token, err := jwt.GenerateAccessTokenWithOptions(
		uuid.NewString(), "openid", stepUpTestIssuer,
		stepUpTestIssuer, "iam-console", "provider-1",
		&jwt.AccessTokenOptions{ACR: acr},
	)
	require.NoError(t, err)
	return token
}

// iamStepUpAuthContext gives the caller every IAM permission, so a rejection can
// only come from the step-up gate and never from PermissionMiddleware. It is
// pre-seeded on the request because UserContextMiddleware short-circuits when the
// principal is already resolved, which keeps the test off the database.
func iamStepUpAuthContext() *authctx.AuthContext {
	names := []string{
		"api:read", "api:create", "api:update", "api:delete",
		"permission:read", "permission:create", "permission:update", "permission:delete",
		"policy:read", "policy:create", "policy:update", "policy:delete",
		"service:read", "service:create", "service:update", "service:delete",
		"service:policy:assign", "service:policy:remove",
	}
	perms := make([]authctx.AuthPermission, 0, len(names))
	for i, name := range names {
		perms = append(perms, authctx.AuthPermission{PermissionID: int64(i + 1), PermissionUUID: uuid.New(), Name: name})
	}
	return &authctx.AuthContext{
		User: &authctx.AuthUser{
			UserID:   1,
			UserUUID: uuid.New(),
			Roles:    []authctx.AuthRole{{RoleID: 1, RoleUUID: uuid.New(), Name: "iam-admin", Permissions: perms}},
		},
		Tenant: &authctx.AuthTenant{TenantID: 1, TenantUUID: uuid.New(), Name: "acme"},
	}
}

func iamStepUpRouter() chi.Router {
	r := chi.NewRouter()
	APIRoute(r, NewAPIHandler(&mockAPIService{}), nil, nil)
	PermissionRoute(r, NewPermissionHandler(&mockPermissionService{}), nil, nil)
	PolicyRoute(r, NewPolicyHandler(&mockPolicyService{}), NewPolicyHistoryHandler(&mockPolicyService{}), nil, nil)
	ServiceRoute(r, NewServiceHandler(&mockServiceService{}), NewAuthorizationHandler(&mockAuthorizationService{}), nil, nil)
	return r
}

// iamStepUpCode returns the machine-readable error code from the response body,
// or "" when the response carries none.
func iamStepUpCode(t *testing.T, body []byte) string {
	t.Helper()
	var decoded struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return ""
	}
	return decoded.Code
}

func serveIAMStepUp(t *testing.T, router chi.Router, method, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req = middleware.WithAuthContext(req, iamStepUpAuthContext())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// Service, API, permission and policy mutations used to be reachable with a
// plain acr=1 session: only the role routes carried middleware.RequireStepUp.
// That let a hijacked console session rewrite the authorization substrate
// itself — rename a permission and every role grant pointing at that row starts
// satisfying a different route guard; edit or detach a policy and the decisions
// a running service enforces change underneath it — all without the fresh proof
// of presence the equivalent role edit demands.
func TestIAMMutationRoutesRequireStepUp(t *testing.T) {
	initIAMStepUpJWT(t)
	router := iamStepUpRouter()
	acr1 := iamStepUpToken(t, jwt.ACRLevel1)
	acr2 := iamStepUpToken(t, jwt.ACRLevel2)

	apiUUID := uuid.NewString()
	permUUID := uuid.NewString()
	policyUUID := uuid.NewString()
	serviceUUID := uuid.NewString()

	gated := []struct {
		method string
		path   string
	}{
		{http.MethodPut, "/apis/" + apiUUID},
		{http.MethodPut, "/apis/" + apiUUID + "/status"},
		{http.MethodDelete, "/apis/" + apiUUID},

		{http.MethodPut, "/permissions/" + permUUID},
		{http.MethodPut, "/permissions/" + permUUID + "/status"},
		{http.MethodDelete, "/permissions/" + permUUID},

		{http.MethodPut, "/policies/" + policyUUID},
		{http.MethodPut, "/policies/" + policyUUID + "/status"},
		{http.MethodDelete, "/policies/" + policyUUID},

		{http.MethodPut, "/services/" + serviceUUID},
		{http.MethodPut, "/services/" + serviceUUID + "/status"},
		{http.MethodDelete, "/services/" + serviceUUID},
		{http.MethodPost, "/services/" + serviceUUID + "/policies/" + policyUUID},
		{http.MethodDelete, "/services/" + serviceUUID + "/policies/" + policyUUID},
	}

	for _, tc := range gated {
		t.Run("acr=1 rejected: "+tc.method+" "+tc.path, func(t *testing.T) {
			rr := serveIAMStepUp(t, router, tc.method, tc.path, acr1)
			require.Equal(t, http.StatusForbidden, rr.Code, "body=%s", rr.Body.String())
			require.Equal(t, "step_up_required", iamStepUpCode(t, rr.Body.Bytes()))
		})

		t.Run("acr=2 admitted: "+tc.method+" "+tc.path, func(t *testing.T) {
			rr := serveIAMStepUp(t, router, tc.method, tc.path, acr2)
			require.NotEqual(t, "step_up_required", iamStepUpCode(t, rr.Body.Bytes()),
				"a stepped-up caller must get past the gate; body=%s", rr.Body.String())
		})
	}

	// Reads and creates deliberately stay outside the gate: a brand-new row is
	// inert until it is attached to a role or a service, and those edges are the
	// ones that carry step-up. Asserted so the gate is not quietly widened into a
	// blanket acr=2 requirement for the whole surface.
	ungated := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/apis/"},
		{http.MethodPost, "/apis/"},
		{http.MethodGet, "/permissions/"},
		{http.MethodPost, "/permissions/"},
		{http.MethodGet, "/policies/"},
		{http.MethodPost, "/policies/"},
		{http.MethodGet, "/services/"},
		{http.MethodPost, "/services/"},
	}

	for _, tc := range ungated {
		t.Run("acr=1 allowed past the gate: "+tc.method+" "+tc.path, func(t *testing.T) {
			rr := serveIAMStepUp(t, router, tc.method, tc.path, acr1)
			require.NotEqual(t, "step_up_required", iamStepUpCode(t, rr.Body.Bytes()),
				"body=%s", rr.Body.String())
		})
	}
}
