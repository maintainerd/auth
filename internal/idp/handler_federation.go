package idp

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/platform/middleware"
	resp "github.com/maintainerd/auth/internal/platform/response"
	"github.com/maintainerd/auth/internal/service"
)

// FederationHandler handles external-identity-provider flows: token exchange,
// identity link/unlink, and home-realm discovery.
type FederationHandler struct {
	federationSvc service.FederationService
}

func NewFederationHandler(federationSvc service.FederationService) *FederationHandler {
	return &FederationHandler{federationSvc: federationSvc}
}

// ExchangeExternalToken validates an upstream OIDC token and returns our JWT.
//
// POST /federation/token
func (h *FederationHandler) ExchangeExternalToken(w http.ResponseWriter, r *http.Request) {
	var req dto.FederationTokenRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}
	if req.ProviderIdentifier == "" || req.ExternalToken == "" || req.ClientID == "" {
		resp.Error(w, http.StatusBadRequest, "provider_identifier, external_token and client_id are required")
		return
	}

	result, err := h.federationSvc.ExchangeExternalToken(r.Context(), req)
	if err != nil {
		resp.HandleServiceError(w, r, "Token exchange failed", err)
		return
	}

	resp.Success(w, result, "")
}

// HomeRealmDiscovery returns the identity provider for the given email.
//
// GET /federation/hrd?email=user@company.com&tenant_id=123
func (h *FederationHandler) HomeRealmDiscovery(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	tenantIDStr := r.URL.Query().Get("tenant_id")

	if email == "" || tenantIDStr == "" {
		resp.Error(w, http.StatusBadRequest, "email and tenant_id query parameters are required")
		return
	}

	tenantID, err := strconv.ParseInt(tenantIDStr, 10, 64)
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	result, svcErr := h.federationSvc.HomeRealmDiscovery(r.Context(), tenantID, email)
	if svcErr != nil {
		resp.HandleServiceError(w, r, "Home realm discovery failed", svcErr)
		return
	}

	resp.Success(w, result, "")
}

// GetIdentities returns all provider identities linked to the authenticated user.
//
// GET /account/identities
func (h *FederationHandler) GetIdentities(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	identities, err := h.federationSvc.GetUserIdentities(r.Context(), user.UserID)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to retrieve identities", err)
		return
	}

	resp.Success(w, identities, "")
}

// LinkIdentity attaches an external provider identity to the authenticated user.
//
// POST /account/identities/link
func (h *FederationHandler) LinkIdentity(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req dto.LinkIdentityRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}
	if req.ProviderIdentifier == "" || req.ExternalToken == "" {
		resp.Error(w, http.StatusBadRequest, "provider_identifier and external_token are required")
		return
	}

	identity, err := h.federationSvc.LinkIdentity(r.Context(), user.UserID, req)
	if err != nil {
		resp.HandleServiceError(w, r, "Failed to link identity", err)
		return
	}

	resp.Success(w, identity, "Identity linked successfully")
}

// UnlinkIdentity removes an external provider identity from the authenticated user.
//
// DELETE /account/identities/{identity_uuid}
func (h *FederationHandler) UnlinkIdentity(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	identityUUID := chi.URLParam(r, "identity_uuid")

	if err := h.federationSvc.UnlinkIdentity(r.Context(), user.UserID, identityUUID); err != nil {
		resp.HandleServiceError(w, r, "Failed to unlink identity", err)
		return
	}

	resp.Success(w, nil, "Identity unlinked successfully")
}
