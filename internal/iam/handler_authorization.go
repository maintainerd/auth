package iam

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/auth/internal/platform/middleware"
	resp "github.com/maintainerd/auth/internal/platform/response"
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
	auth := middleware.AuthFromRequest(r)
	if auth != nil && auth.Tenant != nil {
		req.TenantID = auth.Tenant.TenantID
	}
	decision := h.service.Authorize(r.Context(), req)
	resp.Success(w, decision, "Authorization decision evaluated")
}

func serviceIdentityFromRequest(r *http.Request) (ServiceIdentity, bool) {
	claims := middleware.JWTClaimsFromRequest(r)
	if claims == nil {
		return ServiceIdentity{}, false
	}
	serviceName := claims.Service
	if serviceName == "" && claims.SubjectType == "service" {
		serviceName = claims.Sub
	}
	if serviceName == "" {
		return ServiceIdentity{}, false
	}
	identity := ServiceIdentity{ServiceName: serviceName, ClientID: claims.ClientID}
	auth := middleware.AuthFromRequest(r)
	if auth != nil && auth.Tenant != nil {
		identity.TenantID = auth.Tenant.TenantID
	}
	return identity, true
}
