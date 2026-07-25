package iam

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
)

func TestAuthorizationHandler_PolicyBundle_ServiceTokenRequired(t *testing.T) {
	h := NewAuthorizationHandler(&mockAuthorizationService{})
	w := httptest.NewRecorder()
	h.PolicyBundle(w, httptest.NewRequest(http.MethodGet, "/services/me/policy-bundle", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestAuthorizationHandler_PolicyBundle_UserTokenRejected(t *testing.T) {
	h := NewAuthorizationHandler(&mockAuthorizationService{})
	r := middleware.WithJWTClaims(httptest.NewRequest(http.MethodGet, "/services/me/policy-bundle", nil), &middleware.JWTClaims{Sub: "user", SubjectType: "user"})
	w := httptest.NewRecorder()
	h.PolicyBundle(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestAuthorizationHandler_PolicyBundle_ServiceError(t *testing.T) {
	h := NewAuthorizationHandler(&mockAuthorizationService{
		policyBundleFn: func(ServiceIdentity) (*PolicyBundle, string, error) {
			return nil, "", assert.AnError
		},
	})
	r := middleware.WithJWTClaims(httptest.NewRequest(http.MethodGet, "/services/me/policy-bundle", nil), &middleware.JWTClaims{Service: "serviceA"})
	w := httptest.NewRecorder()
	h.PolicyBundle(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestAuthorizationHandler_PolicyBundle_SuccessAndNotModified(t *testing.T) {
	h := NewAuthorizationHandler(&mockAuthorizationService{
		policyBundleFn: func(identity ServiceIdentity) (*PolicyBundle, string, error) {
			if identity.ServiceName != "serviceA" {
				t.Fatalf("identity = %+v", identity)
			}
			return &PolicyBundle{Service: "serviceA", Version: "v1"}, `"v1"`, nil
		},
	})

	r := middleware.WithJWTClaims(httptest.NewRequest(http.MethodGet, "/services/me/policy-bundle", nil), &middleware.JWTClaims{Service: "serviceA", ClientID: "clientA"})
	w := httptest.NewRecorder()
	h.PolicyBundle(w, r)
	if w.Code != http.StatusOK || w.Header().Get("ETag") != `"v1"` || w.Header().Get("Cache-Control") != "max-age=30" {
		t.Fatalf("status=%d etag=%q cache=%q", w.Code, w.Header().Get("ETag"), w.Header().Get("Cache-Control"))
	}

	r = middleware.WithJWTClaims(httptest.NewRequest(http.MethodGet, "/services/me/policy-bundle", nil), &middleware.JWTClaims{SubjectType: "service", Sub: "serviceA"})
	r.Header.Set("If-None-Match", `"v1"`)
	w = httptest.NewRecorder()
	h.PolicyBundle(w, r)
	if w.Code != http.StatusNotModified || strings.TrimSpace(w.Body.String()) != "" {
		t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
	}
}

// The caller supplies only the QUESTION (action + resource). Principal and tenant
// come from the signed token. This route runs JWTAuthMiddleware WITHOUT
// UserContextMiddleware, so the old conditional override never fired and both
// fields were mass-assignable from the request body — any valid token could probe
// allow/deny for any principal in any tenant.
func TestAuthorizationHandler_Authorize(t *testing.T) {
	authorizeReq := func(body string, claims *middleware.JWTClaims) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/authorize", strings.NewReader(body))
		if claims == nil {
			return r
		}
		return middleware.WithJWTClaims(r, claims)
	}

	t.Run("uses the token's principal and tenant", func(t *testing.T) {
		var got AuthzRequest
		h := NewAuthorizationHandler(&mockAuthorizationService{
			authorizeFn: func(req AuthzRequest) Decision {
				got = req
				return Decision{Allowed: true, Reason: "matched allow"}
			},
		})
		w := httptest.NewRecorder()
		h.Authorize(w, authorizeReq(
			`{"action":"serviceB:invoke","resource":"serviceB:grpc"}`,
			&middleware.JWTClaims{Service: "serviceA", TenantID: 7},
		))

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
		}
		if got.Principal != "serviceA" || got.TenantID != 7 {
			t.Fatalf("principal/tenant not taken from the token: %+v", got)
		}
		if got.Action != "serviceB:invoke" || got.Resource != "serviceB:grpc" {
			t.Fatalf("action/resource must come from the body: %+v", got)
		}
	})

	t.Run("ignores a principal and tenant supplied in the body", func(t *testing.T) {
		var got AuthzRequest
		h := NewAuthorizationHandler(&mockAuthorizationService{
			authorizeFn: func(req AuthzRequest) Decision {
				got = req
				return Decision{Allowed: true}
			},
		})
		w := httptest.NewRecorder()
		h.Authorize(w, authorizeReq(
			`{"principal":"victim-service","tenant_id":99,"action":"a","resource":"r"}`,
			&middleware.JWTClaims{Service: "serviceA", TenantID: 7},
		))

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d", w.Code)
		}
		if got.Principal != "serviceA" {
			t.Fatalf("body overrode the principal: %q", got.Principal)
		}
		if got.TenantID != 7 {
			t.Fatalf("body overrode the tenant: %d", got.TenantID)
		}
	})

	t.Run("refuses a token with no principal or no tenant", func(t *testing.T) {
		h := NewAuthorizationHandler(&mockAuthorizationService{
			authorizeFn: func(AuthzRequest) Decision {
				t.Fatal("must not evaluate a decision for an unbound token")
				return Decision{}
			},
		})

		w := httptest.NewRecorder()
		h.Authorize(w, authorizeReq(`{"action":"a","resource":"r"}`, nil))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("no claims status = %d", w.Code)
		}

		w = httptest.NewRecorder()
		h.Authorize(w, authorizeReq(`{"action":"a","resource":"r"}`,
			&middleware.JWTClaims{TenantID: 7}))
		if w.Code != http.StatusForbidden {
			t.Fatalf("no principal status = %d", w.Code)
		}

		// A tenant-less decision would be evaluated against whichever tenant's
		// policies were found first.
		w = httptest.NewRecorder()
		h.Authorize(w, authorizeReq(`{"action":"a","resource":"r"}`,
			&middleware.JWTClaims{Service: "serviceA"}))
		if w.Code != http.StatusForbidden {
			t.Fatalf("no tenant status = %d", w.Code)
		}
	})

	t.Run("rejects a malformed body", func(t *testing.T) {
		h := NewAuthorizationHandler(&mockAuthorizationService{})
		w := httptest.NewRecorder()
		h.Authorize(w, authorizeReq(`{bad`, &middleware.JWTClaims{Service: "serviceA", TenantID: 7}))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("bad json status = %d", w.Code)
		}
	})
}
