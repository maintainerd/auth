package handler

import (
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/jwt"
)

// OAuthDiscoveryHandler serves the OpenID Connect discovery document and
// the JSON Web Key Set (JWKS).
type OAuthDiscoveryHandler struct{}

// NewOAuthDiscoveryHandler creates a new OAuthDiscoveryHandler.
func NewOAuthDiscoveryHandler() *OAuthDiscoveryHandler {
	return &OAuthDiscoveryHandler{}
}

// Discovery handles GET /.well-known/openid-configuration (OIDC Discovery 1.0).
func (h *OAuthDiscoveryHandler) Discovery(w http.ResponseWriter, r *http.Request) {
	issuer := config.AppPublicHostname

	doc := dto.OAuthDiscoveryResponseDTO{
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
		TokenEndpointAuth:     []string{"client_secret_basic", "client_secret_post", "none"},
		CodeChallengeMethods:  []string{"S256"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(doc)
}

// AuthorizationServerMetadata handles GET /.well-known/oauth-authorization-server (RFC 8414).
func (h *OAuthDiscoveryHandler) AuthorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	issuer := config.AppPublicHostname

	doc := dto.OAuthAuthorizationServerMetadataDTO{
		Issuer:                      issuer,
		AuthorizationEndpoint:       issuer + "/api/v1/oauth/authorize",
		TokenEndpoint:               issuer + "/api/v1/oauth/token",
		JwksURI:                     issuer + "/.well-known/jwks.json",
		RevocationEndpoint:          issuer + "/api/v1/oauth/revoke",
		IntrospectionEndpoint:       issuer + "/api/v1/oauth/introspect",
		PAREndpoint:                 issuer + "/api/v1/oauth/par",
		DeviceAuthorizationEndpoint: issuer + "/api/v1/oauth/device_authorization",
		RegistrationEndpoint:        issuer + "/api/v1/oauth/register",
		EndSessionEndpoint:          issuer + "/api/v1/oauth/end_session",
		BCAuthorizeEndpoint:         issuer + "/api/v1/oauth/ciba",
		ScopesSupported:             []string{"openid", "profile", "email", "offline_access"},
		ResponseTypesSupported:      []string{"code"},
		GrantTypesSupported: []string{
			"authorization_code", "refresh_token", "client_credentials",
			"urn:ietf:params:oauth:grant-type:device_code",
			"urn:ietf:params:oauth:grant-type:token-exchange",
			"urn:openid:params:grant-type:ciba",
		},
		TokenEndpointAuthMethods:      []string{"client_secret_basic", "client_secret_post", "none"},
		CodeChallengeMethods:          []string{"S256"},
		BackchannelTokenDeliveryModes: []string{"poll"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(doc)
}

// JWKS handles GET /.well-known/jwks.json (RFC 7517). Returns the active public
// RSA key plus any recently retired keys so that tokens signed before the last
// rotation continue to verify until they expire naturally.
func (h *OAuthDiscoveryHandler) JWKS(w http.ResponseWriter, r *http.Request) {
	entries := jwt.GetAllPublicKeys()
	if len(entries) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "keys not initialised"})
		return
	}

	keys := make([]dto.JWKKeyDTO, 0, len(entries))
	for _, e := range entries {
		keys = append(keys, dto.JWKKeyDTO{
			Kty: "RSA",
			Use: "sig",
			Kid: e.KID,
			Alg: "RS256",
			N:   base64URLEncodeUint(e.PubKey.N),
			E:   base64URLEncodeUint(big.NewInt(int64(e.PubKey.E))),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(dto.JWKSResponseDTO{Keys: keys})
}

// base64URLEncodeUint encodes a big.Int as a base64url string without padding
// per JWK RFC 7517 §4.
func base64URLEncodeUint(v *big.Int) string {
	return base64.RawURLEncoding.EncodeToString(v.Bytes())
}
