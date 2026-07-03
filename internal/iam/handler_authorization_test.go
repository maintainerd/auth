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

func TestAuthorizationHandler_Authorize(t *testing.T) {
	h := NewAuthorizationHandler(&mockAuthorizationService{
		authorizeFn: func(req AuthzRequest) Decision {
			if req.Principal != "serviceA" || req.Action != "serviceB:invoke" || req.Resource != "serviceB:grpc" {
				t.Fatalf("req = %+v", req)
			}
			return Decision{Allowed: true, Reason: "matched allow"}
		},
	})
	w := httptest.NewRecorder()
	h.Authorize(w, httptest.NewRequest(http.MethodPost, "/authorize", strings.NewReader(`{"principal":"serviceA","action":"serviceB:invoke","resource":"serviceB:grpc"}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	h.Authorize(w, httptest.NewRequest(http.MethodPost, "/authorize", strings.NewReader(`{bad`)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad json status = %d", w.Code)
	}
}
