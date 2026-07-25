package iam

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// AuthorizationHandler exposes service-to-service bundle distribution and PDP checks.
type AuthorizationHandler struct {
	service ServiceAuthorizationService
}

func NewAuthorizationHandler(service ServiceAuthorizationService) *AuthorizationHandler {
	return &AuthorizationHandler{service: service}
}

func (h *AuthorizationHandler) PolicyBundle(w http.ResponseWriter, r *http.Request) {
	identity, ok := serviceIdentityFromRequest(r)
	if !ok {
		resp.Error(w, http.StatusUnauthorized, "Service token required")
		return
	}

	bundle, etag, err := h.service.PolicyBundle(r.Context(), identity)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to resolve policy bundle", err)
		return
	}
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "max-age=30")
	resp.Success(w, bundle, "Policy bundle fetched successfully")
}

func (h *AuthorizationHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	var req AuthzRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	// The caller supplies only the QUESTION (action + resource). Principal and
	// tenant are taken from the signed token and overwrite whatever the body said.
	//
	// This route runs JWTAuthMiddleware only — no UserContextMiddleware — so
	// AuthFromRequest().Tenant was always nil and the previous conditional override
	// never fired. Both fields were therefore mass-assignable from the request body,
	// letting any valid token probe allow/deny against any principal in any tenant.
	claims := middleware.JWTClaimsFromRequest(r)
	if claims == nil {
		resp.Error(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	req.Principal = authorizationPrincipal(claims)
	req.TenantID = claims.TenantID
	if auth := middleware.AuthFromRequest(r); auth != nil && auth.Tenant != nil {
		req.TenantID = auth.Tenant.TenantID
	}
	if req.Principal == "" {
		resp.Error(w, http.StatusForbidden, "This token has no principal to authorize")
		return
	}
	if req.TenantID == 0 {
		// A tenant-less decision would be evaluated against whichever tenant's
		// policies happened to be found first.
		resp.Error(w, http.StatusForbidden, "This token is not bound to a tenant")
		return
	}

	decision := h.service.Authorize(r.Context(), req)
	resp.Success(w, decision, "Authorization decision evaluated")
}

// authorizationPrincipal derives the principal name from the token: the service
// claim, or the subject when the token represents a service.
func authorizationPrincipal(claims *middleware.JWTClaims) string {
	if claims.Service != "" {
		return claims.Service
	}
	if claims.SubjectType == "service" {
		return claims.Sub
	}
	return ""
}

func serviceIdentityFromRequest(r *http.Request) (ServiceIdentity, bool) {
	claims := middleware.JWTClaimsFromRequest(r)
	if claims == nil {
		return ServiceIdentity{}, false
	}
	serviceName := authorizationPrincipal(claims)
	if serviceName == "" {
		return ServiceIdentity{}, false
	}
	identity := ServiceIdentity{ServiceName: serviceName, ClientID: claims.ClientID}
	// Tenant comes from the signed `tenant_id` claim. This route does not run
	// UserContextMiddleware, so AuthFromRequest().Tenant is nil here and relying on
	// it left the tenant at 0 — which selected the unscoped principal lookup.
	identity.TenantID = claims.TenantID
	if auth := middleware.AuthFromRequest(r); auth != nil && auth.Tenant != nil {
		identity.TenantID = auth.Tenant.TenantID
	}
	return identity, true
}
