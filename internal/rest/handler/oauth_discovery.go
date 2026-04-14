package handler

import (
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"

	"github.com/maintainerd/auth/internal/config"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/jwt"
)

// OAuthDiscoveryHandler serves the OpenID Connect discovery document and
// the JSON Web Key Set (JWKS).
type OAuthDiscoveryHandler struct{}

// NewOAuthDiscoveryHandler creates a new OAuthDiscoveryHandler.
func NewOAuthDiscoveryHandler() *OAuthDiscoveryHandler {
	return &OAuthDiscoveryHandler{}
}

// Discovery handles GET /.well-known/openid-configuration (RFC 8414).
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
		GrantTypesSupported:   []string{"authorization_code", "refresh_token", "client_credentials"},
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

// JWKS handles GET /.well-known/jwks.json (RFC 7517). Returns the public RSA
// key used to verify JWTs.
func (h *OAuthDiscoveryHandler) JWKS(w http.ResponseWriter, r *http.Request) {
	pubKey := jwt.GetPublicKey()
	if pubKey == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "keys not initialised"})
		return
	}

	kid := config.GetEnvOrDefault("JWT_KEY_ID", "maintainerd-auth-key-1")

	jwk := dto.JWKKeyDTO{
		Kty: "RSA",
		Use: "sig",
		Kid: kid,
		Alg: "RS256",
		N:   base64URLEncodeUint(pubKey.N),
		E:   base64URLEncodeUint(big.NewInt(int64(pubKey.E))),
	}

	result := dto.JWKSResponseDTO{
		Keys: []dto.JWKKeyDTO{jwk},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

// base64URLEncodeUint encodes a big.Int as a base64url string without padding
// per JWK RFC 7517 §4.
func base64URLEncodeUint(v *big.Int) string {
	return base64.RawURLEncoding.EncodeToString(v.Bytes())
}
