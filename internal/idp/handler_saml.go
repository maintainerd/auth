package idp

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// InitiateSAML begins a SAML SSO flow by redirecting the browser to the IdP.
//
// GET /federation/saml/initiate?provider_identifier=...&client_id=...&redirect_uri=...&tenant_id=...
func (h *FederationHandler) InitiateSAML(w http.ResponseWriter, r *http.Request) {
	providerIdentifier := r.URL.Query().Get("provider_identifier")
	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	tenantIDStr := r.URL.Query().Get("tenant_id")

	if providerIdentifier == "" || clientID == "" || redirectURI == "" {
		resp.Error(w, http.StatusBadRequest, "provider_identifier, client_id and redirect_uri are required")
		return
	}

	var tenantID int64
	if tenantIDStr != "" {
		var err error
		tenantID, err = strconv.ParseInt(tenantIDStr, 10, 64)
		if err != nil {
			resp.Error(w, http.StatusBadRequest, "invalid tenant_id")
			return
		}
	}

	result, err := h.federationSvc.InitiateSAMLSSO(r.Context(), SAMLInitiateInput{
		ProviderIdentifier: providerIdentifier,
		ClientID:           clientID,
		RedirectURI:        redirectURI,
		TenantID:           tenantID,
	})
	if err != nil {
		resp.HandleServiceError(w, r, "SAML initiation failed", err)
		return
	}

	http.Redirect(w, r, result.RedirectURL, http.StatusFound)
}

// SAMLCallback is the Assertion Consumer Service (ACS) endpoint. The IdP
// HTTP-POSTs the SAMLResponse here after the user authenticates.
//
// POST /federation/saml/acs/{provider_identifier}
func (h *FederationHandler) SAMLCallback(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		resp.Error(w, http.StatusBadRequest, "failed to parse form")
		return
	}

	relayState := r.FormValue("RelayState")
	if relayState == "" {
		resp.Error(w, http.StatusBadRequest, "missing RelayState")
		return
	}

	result, err := h.federationSvc.HandleSAMLResponse(r.Context(), r, relayState)
	if err != nil {
		resp.HandleServiceError(w, r, "SAML callback failed", err)
		return
	}

	http.Redirect(w, r, result.RedirectURI, http.StatusFound)
}

// ExchangeSAMLCode exchanges a short-lived SAML code for access/ID/refresh tokens.
//
// POST /federation/saml/exchange
func (h *FederationHandler) ExchangeSAMLCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.BadRequestBody(w)
		return
	}
	if req.Code == "" {
		resp.Error(w, http.StatusBadRequest, "code is required")
		return
	}

	result, err := h.federationSvc.ExchangeSAMLCode(r.Context(), req.Code)
	if err != nil {
		resp.HandleServiceError(w, r, "SAML code exchange failed", err)
		return
	}

	resp.Success(w, result, "")
}

// SAMLMetadata serves the SP metadata XML for the given provider.
//
// GET /federation/saml/metadata/{provider_identifier}
func (h *FederationHandler) SAMLMetadata(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "provider_identifier")
	if identifier == "" {
		resp.Error(w, http.StatusBadRequest, "provider_identifier is required")
		return
	}

	xmlBytes, err := h.federationSvc.SAMLMetadata(r.Context(), identifier)
	if err != nil {
		resp.HandleServiceError(w, r, "SAML metadata failed", err)
		return
	}

	w.Header().Set("Content-Type", "application/samlmetadata+xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(xmlBytes)
}
