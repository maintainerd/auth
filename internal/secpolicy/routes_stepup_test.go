package secpolicy

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

const secpolicyStepUpTestIssuer = "https://auth.secpolicy-stepup.test"

// initSecPolicyStepUpJWT installs a throwaway RSA key pair and an issuer
// allowlist so the router's real JWTAuthMiddleware accepts the tokens minted
// below. Nothing is stubbed: the assertions run the production middleware chain
// exactly as SecuritySettingRoute composes it, which is the only way a missing
// per-route guard shows up.
func initSecPolicyStepUpJWT(t *testing.T) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	prevPriv, prevPub, prevHost := config.JWTPrivateKey, config.JWTPublicKey, config.AppPublicHostname
	config.JWTPrivateKey = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	config.JWTPublicKey = pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&priv.PublicKey)})
	config.AppPublicHostname = secpolicyStepUpTestIssuer
	require.NoError(t, jwt.InitJWTKeys())
	jwt.SetAcceptedIssuers([]string{secpolicyStepUpTestIssuer})

	t.Cleanup(func() {
		jwt.ResetAcceptedIssuers()
		jwt.ResetJWTKeys()
		config.JWTPrivateKey, config.JWTPublicKey, config.AppPublicHostname = prevPriv, prevPub, prevHost
	})
}

func secpolicyStepUpToken(t *testing.T, acr string) string {
	t.Helper()
	token, err := jwt.GenerateAccessTokenWithOptions(
		uuid.NewString(), "openid", secpolicyStepUpTestIssuer,
		secpolicyStepUpTestIssuer, "secpolicy-console", "provider-1",
		&jwt.AccessTokenOptions{ACR: acr},
	)
	require.NoError(t, err)
	return token
}

// secpolicyStepUpAuthContext gives the caller both security-setting permissions,
// so a rejection can only come from the step-up gate and never from
// PermissionMiddleware. It is pre-seeded on the request because
// UserContextMiddleware short-circuits when the principal is already resolved,
// which keeps the test off the database.
func secpolicyStepUpAuthContext() *authctx.AuthContext {
	names := []string{"security-setting:read", "security-setting:update"}
	perms := make([]authctx.AuthPermission, 0, len(names))
	for i, name := range names {
		perms = append(perms, authctx.AuthPermission{PermissionID: int64(i + 1), PermissionUUID: uuid.New(), Name: name})
	}
	return &authctx.AuthContext{
		User: &authctx.AuthUser{
			UserID:   1,
			UserUUID: uuid.New(),
			Roles:    []authctx.AuthRole{{RoleID: 1, RoleUUID: uuid.New(), Name: "security-admin", Permissions: perms}},
		},
		Tenant: &authctx.AuthTenant{TenantID: 1, TenantUUID: uuid.New(), Name: "acme"},
	}
}

// secpolicyStepUpCode returns the machine-readable error code from the response
// body, or "" when the response carries none.
func secpolicyStepUpCode(t *testing.T, body []byte) string {
	t.Helper()
	var decoded struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return ""
	}
	return decoded.Code
}

func serveSecPolicyStepUp(t *testing.T, router chi.Router, method, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req = middleware.WithAuthContext(req, secpolicyStepUpAuthContext())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// securitySettingMutations is every write on /security-settings. The table form
// is deliberate: PUT /security-settings/mfa was the one row of seven that
// shipped without middleware.RequireStepUp, and only an exhaustive table makes
// the next omission fail a test instead of passing review.
var securitySettingMutations = []string{
	"/security-settings/mfa",
	"/security-settings/password",
	"/security-settings/session",
	"/security-settings/threat",
	"/security-settings/lockout",
	"/security-settings/registration",
	"/security-settings/token",
}

// PUT /security-settings/mfa used to be reachable on a plain acr=1 session while
// its six siblings all carried middleware.RequireStepUp. That let a hijacked
// console session switch off enforced MFA and clear
// require_mfa_for_sensitive_actions tenant-wide, which in turn degrades
// MFAHandler.RequirePolicyStepUp into a pass-through on the email-change,
// username-change and password-change routes — one unstepped-up request
// dismantling every step-up gate the tenant relies on.
func TestSecuritySettingMutationsRequireStepUp(t *testing.T) {
	initSecPolicyStepUpJWT(t)

	router := chi.NewRouter()
	SecuritySettingRoute(router, NewSecuritySettingHandler(&mockSecuritySettingService{}), nil, nil)

	acr1 := secpolicyStepUpToken(t, jwt.ACRLevel1)
	acr2 := secpolicyStepUpToken(t, jwt.ACRLevel2)

	for _, path := range securitySettingMutations {
		t.Run("acr=1 rejected: PUT "+path, func(t *testing.T) {
			rr := serveSecPolicyStepUp(t, router, http.MethodPut, path, acr1)
			require.Equal(t, http.StatusForbidden, rr.Code, "body=%s", rr.Body.String())
			require.Equal(t, "step_up_required", secpolicyStepUpCode(t, rr.Body.Bytes()))
		})

		t.Run("acr=2 admitted: PUT "+path, func(t *testing.T) {
			rr := serveSecPolicyStepUp(t, router, http.MethodPut, path, acr2)
			require.NotEqual(t, "step_up_required", secpolicyStepUpCode(t, rr.Body.Bytes()),
				"a stepped-up caller must get past the gate; body=%s", rr.Body.String())
		})
	}

	// Reads stay outside the gate: they expose no lever an attacker can pull.
	// Asserted so the fix is not quietly widened into a blanket acr=2
	// requirement that would lock the console's settings screens out of a
	// normal session.
	for _, path := range securitySettingMutations {
		t.Run("acr=1 allowed past the gate: GET "+path, func(t *testing.T) {
			rr := serveSecPolicyStepUp(t, router, http.MethodGet, path, acr1)
			require.NotEqual(t, "step_up_required", secpolicyStepUpCode(t, rr.Body.Bytes()),
				"body=%s", rr.Body.String())
		})
	}
}
