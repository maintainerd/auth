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
		// introspection_endpoint is deliberately omitted: POST /oauth/introspect is
		// mounted only on the internal control-plane router (see
		// OAuthInternalRouteWithRegistration), so the URL this document would
		// advertise on the public host 404s. Restore it together with a publicly
		// mounted, client-authenticated endpoint.
		EndSessionEndpoint:          issuer + "/api/v1/oauth/end_session",
		PAREndpoint:                 issuer + "/api/v1/oauth/par",
		DeviceAuthorizationEndpoint: issuer + "/api/v1/oauth/device_authorization",
		BCAuthorizeEndpoint:         issuer + "/api/v1/oauth/ciba",
		ScopesSupported:             []string{"openid", "profile", "email", "offline_access"},
		ResponseTypesSupp:           []string{"code"},
		ResponseModesSupported:      []string{"query"},
		GrantTypesSupported: []string{
			"authorization_code", "refresh_token", "client_credentials",
			"urn:ietf:params:oauth:grant-type:device_code",
			"urn:ietf:params:oauth:grant-type:token-exchange",
			"urn:openid:params:grant-type:ciba",
		},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSignAlgValues:              []string{"RS256"},
		TokenEndpointAuth:                 []string{"client_secret_basic", "client_secret_post", "none", "private_key_jwt", "client_secret_jwt"},
		TokenEndpointAuthSigningAlgValues: assertionSigningAlgValues,
		// acr_values_supported is what tells an RP it may ask for step-up at the
		// protocol level. Without it there is no discoverable way to request MFA.
		ACRValuesSupported: []string{jwt.ACRLevel1, jwt.ACRLevel2},
		ClaimsSupported: []string{
			"sub", "iss", "aud", "exp", "iat", "nbf", "jti", "auth_time",
			"nonce", "acr", "amr", "sid", "scope", "client_id",
			"email", "email_verified", "phone_number", "phone_number_verified",
			"name", "picture", "updated_at",
		},
		CodeChallengeMethods: []string{"S256"},
		// The JAR `request` parameter is not implemented; `request_uri` is, through
		// PAR (RFC 9126). Advertising both accurately is what stops an RP silently
		// having its signed request object ignored.
		RequestParameterSupported:    false,
		RequestURIParameterSupported: true,
		DPoPSigningAlgValues:         []string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512"},
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
		Issuer:                issuer,
		AuthorizationEndpoint: issuer + "/api/v1/oauth/authorize",
		TokenEndpoint:         issuer + "/api/v1/oauth/token",
		JwksURI:               issuer + "/.well-known/jwks.json",
		RevocationEndpoint:    issuer + "/api/v1/oauth/revoke",
		// introspection_endpoint is deliberately omitted — see Discovery above; the
		// route is control-plane only, so advertising it on the public host 404s.
		PAREndpoint:                 issuer + "/api/v1/oauth/par",
		DeviceAuthorizationEndpoint: issuer + "/api/v1/oauth/device_authorization",
		// registration_endpoint is deliberately omitted: Dynamic Client
		// Registration is mounted on the CONTROL plane only (see
		// OAuthInternalRouteWithRegistration), so the URL this document would
		// advertise on the public host 404s. Advertising an endpoint that is not
		// reachable here makes conformant clients attempt it and fail — and
		// advertising it at all invites callers to treat client creation as a
		// public, token-gated operation, which is exactly what keeping it on the
		// control plane refuses. Restore this line only if the route is ever
		// mounted on the public plane.
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
		TokenEndpointAuthMethods:          []string{"client_secret_basic", "client_secret_post", "none", "private_key_jwt", "client_secret_jwt"},
		TokenEndpointAuthSigningAlgValues: assertionSigningAlgValues,
		CodeChallengeMethods:              []string{"S256"},
		BackchannelTokenDeliveryModes:     []string{"poll"},
		DPoPSigningAlgValues:              []string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", shared.DefaultDiscoveryCacheMaxAge)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(doc)
}

// JWKS handles GET /.well-known/jwks.json (RFC 7517).
//
// It publishes the UNION of the DB-backed global keys and the keys held in the
// process's in-memory key store, de-duplicated by kid.
//
// The DB rows used to WIN outright, with the in-memory store consulted only when
// the table was empty. That published a key set that did not contain the key
// actually signing tokens: the key-rotation runner rotates the in-memory key at
// boot and every 24h without writing the new key to signing_keys, so from the
// first rotation onward every token carried a kid that JWKS did not list and
// every external relying party's verification failed. A JWKS must contain every
// key that can verify a token this server issued; anything narrower is a
// verification outage, and anything published here is a public key, so the union
// discloses nothing.
func (h *OAuthDiscoveryHandler) JWKS(w http.ResponseWriter, r *http.Request) {
	keys := make([]JWKKeyDTO, 0, 4)
	seen := map[string]struct{}{}

	// DB-backed keys (tenantID=0 matches global keys with tenant_id IS NULL).
	if h.keySvc != nil {
		if dbKeys, err := h.keySvc.ListJWKS(r.Context(), 0); err == nil {
			for _, k := range dbKeys {
				if _, dup := seen[k.Kid]; dup {
					continue
				}
				seen[k.Kid] = struct{}{}
				keys = append(keys, k)
			}
		}
	}

	// In-memory keys: the active signing key plus any still inside the refresh
	// window (jwt.GetAllPublicKeys).
	for _, e := range jwt.GetAllPublicKeys() {
		if _, dup := seen[e.KID]; dup {
			continue
		}
		seen[e.KID] = struct{}{}
		keys = append(keys, JWKKeyDTO{
			Kty: "RSA",
			Use: "sig",
			Kid: e.KID,
			Alg: "RS256",
			N:   base64URLEncodeUint(e.PubKey.N),
			E:   base64URLEncodeUint(big.NewInt(int64(e.PubKey.E))),
		})
	}

	if len(keys) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "keys not initialised"})
		return
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
