package idp

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// FederationHandler handles external-identity-provider flows: token exchange,
// identity link/unlink, and home-realm discovery.
type FederationHandler struct {
	federationSvc FederationService
	auditLogger   auditlog.ManagementAuditLogger
}

func NewFederationHandler(federationSvc FederationService) *FederationHandler {
	return &FederationHandler{federationSvc: federationSvc}
}

func (h *FederationHandler) SetAuditLogger(l auditlog.ManagementAuditLogger) { h.auditLogger = l }

func (h *FederationHandler) logAudit(r *http.Request, tenantID int64, actorUserID *int64, action, resourceType, resourceID string, resourceUUID *uuid.UUID, changes, outcome string) {
	if h.auditLogger == nil {
		return
	}
	_ = h.auditLogger.Log(r.Context(), auditlog.LogEntry{
		TenantID:     tenantID,
		ActorUserID:  actorUserID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceUUID: resourceUUID,
		Changes:      changes,
		Outcome:      outcome,
	})
}

// ExchangeExternalToken validates an upstream OIDC token and returns our JWT.
//
// POST /federation/token
func (h *FederationHandler) ExchangeExternalToken(w http.ResponseWriter, r *http.Request) {
	var req FederationTokenRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
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

// ExchangeOAuth2Code handles the OAuth2 authorization code callback for
// generic OAuth2 providers (not OIDC, which uses ExchangeExternalToken).
//
// POST /federation/oauth2/callback
func (h *FederationHandler) ExchangeOAuth2Code(w http.ResponseWriter, r *http.Request) {
	var req FederationOAuth2CallbackDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if req.ProviderIdentifier == "" || req.Code == "" || req.RedirectURI == "" || req.ClientID == "" {
		resp.Error(w, http.StatusBadRequest, "provider_identifier, code, redirect_uri and client_id are required")
		return
	}

	result, err := h.federationSvc.ExchangeOAuth2Code(r.Context(), req)
	if err != nil {
		resp.HandleServiceError(w, r, "OAuth2 code exchange failed", err)
		return
	}

	resp.Success(w, result, "")
}

// HomeRealmDiscovery returns the identity provider for the given email.
//
// Public surface: GET /federation/hrd?email=user@company.com&client_id=app
// The public API only accepts client_id. tenant_id is accepted solely as an
// internal-surface fallback (this route is shared with the internal router).
func (h *FederationHandler) HomeRealmDiscovery(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	clientID := r.URL.Query().Get("client_id")
	tenantIDStr := r.URL.Query().Get("tenant_id")

	if email == "" {
		resp.Error(w, http.StatusBadRequest, "email query parameter is required")
		return
	}

	if clientID != "" {
		result, svcErr := h.federationSvc.HomeRealmDiscoveryByClient(r.Context(), clientID, email)
		if svcErr != nil {
			resp.HandleServiceError(w, r, "Home realm discovery failed", svcErr)
			return
		}
		resp.Success(w, result, "")
		return
	}

	if tenantIDStr == "" {
		resp.Error(w, http.StatusBadRequest, "client_id query parameter is required")
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

	var req LinkIdentityRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
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

	changesJSON, _ := json.Marshal(map[string]any{"update": map[string]any{"provider_identifier": req.ProviderIdentifier}})
	var actorUserID *int64
	if user != nil {
		actorUserID = &user.UserID
	}
	h.logAudit(r, 0, actorUserID, "federated_identity.link", "federated_identity", req.ProviderIdentifier, nil, string(changesJSON), "success")

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

	changesJSON, _ := json.Marshal(map[string]any{"before": map[string]any{"id": identityUUID}})
	var actorUserID *int64
	if user != nil {
		actorUserID = &user.UserID
	}
	h.logAudit(r, 0, actorUserID, "federated_identity.unlink", "federated_identity", identityUUID, nil, string(changesJSON), "success")

	resp.Success(w, nil, "Identity unlinked successfully")
}

// TestConnection probes an unsaved IdP configuration. It accepts the raw
// provider fields, runs OIDC discovery and a JWKS probe via the SSRF-safe
// idpHTTPClient, and returns per-check results.
//
// POST /api/v1/identity_providers/test
func (h *FederationHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	var req TestConnectionRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}

	result, err := h.federationSvc.TestConnection(r.Context(), req)
	if err != nil {
		resp.HandleServiceError(w, r, "Test connection failed", err)
		return
	}

	resp.Success(w, result, "Test connection completed")
}
