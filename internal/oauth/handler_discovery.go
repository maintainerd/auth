package oauth

import (
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// OAuthDiscoveryHandler serves the OpenID Connect discovery document and
// the JSON Web Key Set (JWKS).
type OAuthDiscoveryHandler struct {
	keySvc KeyRotationService
}

// NewOAuthDiscoveryHandler creates a new OAuthDiscoveryHandler.
// keySvc is used to serve JWKS from the DB-backed key rotation service;
// when no DB keys exist it falls back to the in-memory JWT key store.
func NewOAuthDiscoveryHandler(keySvc KeyRotationService) *OAuthDiscoveryHandler {
	return &OAuthDiscoveryHandler{keySvc: keySvc}
}

// Discovery handles GET /.well-known/openid-configuration (OIDC Discovery 1.0).
func (h *OAuthDiscoveryHandler) Discovery(w http.ResponseWriter, r *http.Request) {
	issuer := config.AppPublicHostname

	doc := OAuthDiscoveryResponseDTO{
		Issuer:                issuer,
		AuthorizationEndpoint: issuer + "/api/v1/oauth/authorize",
		TokenEndpoint:         issuer + "/api/v1/oauth/token",
		UserinfoEndpoint:      issuer + "/api/v1/oauth/userinfo",
		JwksURI:               issuer + "/.well-known/jwks.json",
		RevocationEndpoint:    issuer + "/api/v1/oauth/revoke",
		IntrospectionEndpoint: issuer + "/api/v1/oauth/introspect",
		ScopesSupported:       []string{"openid", "profile", "email", "offline_access"},
		ResponseTypesSupp:     []string{"code"},
		GrantTypesSupported: []string{
			"authorization_code", "refresh_token", "client_credentials",
			"urn:ietf:params:oauth:grant-type:device_code",
			"urn:ietf:params:oauth:grant-type:token-exchange",
			"urn:openid:params:grant-type:ciba",
		},
		SubjectTypesSupported: []string{"public"},
		IDTokenSignAlgValues:  []string{"RS256"},
		TokenEndpointAuth:     []string{"client_secret_basic", "client_secret_post", "none", "private_key_jwt", "client_secret_jwt"},
		CodeChallengeMethods:  []string{"S256"},
		DPoPSigningAlgValues:  []string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", shared.DefaultDiscoveryCacheMaxAge)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(doc)
}

// AuthorizationServerMetadata handles GET /.well-known/oauth-authorization-server (RFC 8414).
func (h *OAuthDiscoveryHandler) AuthorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	issuer := config.AppPublicHostname

	doc := OAuthAuthorizationServerMetadataDTO{
		Issuer:                      issuer,
		AuthorizationEndpoint:       issuer + "/api/v1/oauth/authorize",
		TokenEndpoint:               issuer + "/api/v1/oauth/token",
		JwksURI:                     issuer + "/.well-known/jwks.json",
		RevocationEndpoint:          issuer + "/api/v1/oauth/revoke",
		IntrospectionEndpoint:       issuer + "/api/v1/oauth/introspect",
		PAREndpoint:                 issuer + "/api/v1/oauth/par",
		DeviceAuthorizationEndpoint: issuer + "/api/v1/oauth/device_authorization",
		// registration_endpoint is deliberately omitted: Dynamic Client
		// Registration is unmounted (see oauth/routes.go). Advertising an endpoint
		// that does not exist makes conformant clients attempt it and fail.
		// Restore this line together with the route.
		EndSessionEndpoint:     issuer + "/api/v1/oauth/end_session",
		BCAuthorizeEndpoint:    issuer + "/api/v1/oauth/ciba",
		ScopesSupported:        []string{"openid", "profile", "email", "offline_access"},
		ResponseTypesSupported: []string{"code"},
		GrantTypesSupported: []string{
			"authorization_code", "refresh_token", "client_credentials",
			"urn:ietf:params:oauth:grant-type:device_code",
			"urn:ietf:params:oauth:grant-type:token-exchange",
			"urn:openid:params:grant-type:ciba",
		},
		TokenEndpointAuthMethods:      []string{"client_secret_basic", "client_secret_post", "none", "private_key_jwt", "client_secret_jwt"},
		CodeChallengeMethods:          []string{"S256"},
		BackchannelTokenDeliveryModes: []string{"poll"},
		DPoPSigningAlgValues:          []string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", shared.DefaultDiscoveryCacheMaxAge)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(doc)
}

// JWKS handles GET /.well-known/jwks.json (RFC 7517). Serves keys from the
// DB-backed key rotation service (tenantID=0 → global scope). Falls back to
// the in-memory JWT key store when no DB keys exist yet.
func (h *OAuthDiscoveryHandler) JWKS(w http.ResponseWriter, r *http.Request) {
	// Primary: DB-backed JWKS (tenantID=0 matches global keys with tenant_id IS NULL).
	if h.keySvc != nil {
		jwks, err := h.keySvc.ListJWKS(r.Context(), 0)
		if err == nil && len(jwks) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", shared.DefaultDiscoveryCacheMaxAge)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(JWKSResponseDTO{Keys: jwks})
			return
		}
	}

	// Fallback: in-memory keys loaded from JWT_PRIVATE_KEY env var.
	entries := jwt.GetAllPublicKeys()
	if len(entries) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "keys not initialised"})
		return
	}

	keys := make([]JWKKeyDTO, 0, len(entries))
	for _, e := range entries {
		keys = append(keys, JWKKeyDTO{
			Kty: "RSA",
			Use: "sig",
			Kid: e.KID,
			Alg: "RS256",
			N:   base64URLEncodeUint(e.PubKey.N),
			E:   base64URLEncodeUint(big.NewInt(int64(e.PubKey.E))),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", shared.DefaultDiscoveryCacheMaxAge)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(JWKSResponseDTO{Keys: keys})
}

// base64URLEncodeUint encodes a big.Int as a base64url string without padding
// per JWK RFC 7517 §4.
func base64URLEncodeUint(v *big.Int) string {
	return base64.RawURLEncoding.EncodeToString(v.Bytes())
}
