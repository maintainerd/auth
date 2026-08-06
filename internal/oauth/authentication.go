package oauth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	clientpkg "github.com/maintainerd/maintainerd-auth/internal/client"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"gorm.io/gorm"
)

func authenticateOAuthClient(db *gorm.DB, creds OAuthClientCredentials) (*Client, *apperror.OAuthError) {
	if creds.ClientID == "" {
		return nil, apperror.NewOAuthInvalidClient("client_id is required")
	}
	client, err := findActiveClientByIdentifier(db, creds.ClientID)
	if err != nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if client == nil {
		return nil, apperror.NewOAuthInvalidClient("client authentication failed")
	}
	// token_endpoint_auth_method=none means "this client presents no credential",
	// which RFC 6749 §2.1 permits ONLY for public clients — a browser or mobile app
	// that cannot keep a secret. Accepting it for a confidential or m2m client
	// makes the token endpoint unauthenticated: client_id is public (handed out by
	// GET /client and present in every authorize URL), so anyone could mint that
	// client's tokens and receive its resolved permissions.
	//
	// Public clients remain constrained by PKCE and exact redirect-URI matching;
	// an m2m client using client_credentials has neither, which is why this is
	// refused at runtime and not merely discouraged at registration time.
	// ValidateClientOAuthMatrix blocks the combination on write; this blocks it for
	// rows that predate that check.
	if client.TokenEndpointAuthMethod == TokenAuthMethodNone {
		if !clientpkg.IsPublicClientType(client.ClientType) {
			return nil, apperror.NewOAuthInvalidClient("client authentication failed")
		}
		return client, nil
	}

	// Accepted by the registry and the CHECK constraint, but there is no
	// certificate-binding implementation, so a client configured this way could
	// otherwise fall through to the generic "unsupported" error below and look
	// like a transient fault. Fail explicitly.
	if client.TokenEndpointAuthMethod == TokenAuthMethodTLSClientAuth ||
		client.TokenEndpointAuthMethod == TokenAuthMethodSelfSignedTLSClientAuth {
		return nil, apperror.NewOAuthInvalidClient("mutual-TLS client authentication is not supported by this server")
	}
	if client.TokenEndpointAuthMethod == TokenAuthMethodSecretBasic || client.TokenEndpointAuthMethod == TokenAuthMethodSecretPost {
		if !clientSecretMatches(client, creds.ClientSecret) {
			return nil, apperror.NewOAuthInvalidClient("client authentication failed")
		}
		return client, nil
	}
	if client.TokenEndpointAuthMethod == TokenAuthMethodPrivateKeyJWT {
		return authenticatePrivateKeyJWT(client, creds)
	}
	if client.TokenEndpointAuthMethod == TokenAuthMethodClientSecretJWT {
		return authenticateClientSecretJWT(client, creds)
	}
	return nil, apperror.NewOAuthInvalidClient("unsupported token_endpoint_auth_method")
}

func clientHasGrant(client *Client, grantType string) bool {
	if client.GrantTypes == nil {
		return false
	}
	for _, g := range client.GrantTypes {
		if g == grantType {
			return true
		}
	}
	return false
}

const (
	assertionTypeJWTBearer = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
	assertionMaxAge        = 5 * time.Minute
	// assertionClockSkew is the only slack allowed on an assertion's time claims.
	// It used to be assertionMaxAge, handed to jwtlib.WithLeeway, which does not
	// cap an assertion's age at all — it WIDENS the exp window by five minutes in
	// both directions, so an assertion stayed usable five minutes past the expiry
	// its own issuer chose. The age cap is now enforced explicitly in
	// validateAssertionClaims.
	assertionClockSkew = 30 * time.Second
)

var (
	// assertionRSAAlgs and assertionHMACAlgs are the signing algorithms accepted on
	// a private_key_jwt / client_secret_jwt assertion respectively. They are
	// declared here rather than inline at the jwtlib.Parse calls so the discovery
	// document's token_endpoint_auth_signing_alg_values_supported (REQUIRED by OIDC
	// Discovery 1.0 §3 once these auth methods are advertised) is derived from what
	// is actually accepted and cannot drift from it.
	assertionRSAAlgs  = []string{"RS256", "RS384", "RS512"}
	assertionHMACAlgs = []string{"HS256", "HS384", "HS512"}

	assertionSigningAlgValues = append(append([]string{}, assertionRSAAlgs...), assertionHMACAlgs...)
)

func authenticatePrivateKeyJWT(client *Client, creds OAuthClientCredentials) (*Client, *apperror.OAuthError) {
	if creds.ClientAssertionType != assertionTypeJWTBearer {
		return nil, apperror.NewOAuthInvalidClient("client_assertion_type must be " + assertionTypeJWTBearer)
	}
	if creds.ClientAssertion == "" {
		return nil, apperror.NewOAuthInvalidClient("client_assertion is required")
	}

	if !clientHasVerificationMaterial(client) {
		return nil, apperror.NewOAuthInvalidClient("client has no JWKS or jwks_uri configured")
	}

	token, err := jwtlib.Parse(creds.ClientAssertion, func(t *jwtlib.Token) (interface{}, error) {
		kid, _ := t.Header["kid"].(string)
		key, err := findClientJWK(client, kid)
		if err != nil {
			return nil, err
		}
		return key, nil
	}, jwtlib.WithLeeway(assertionClockSkew), jwtlib.WithValidMethods(assertionRSAAlgs))

	if err != nil || !token.Valid {
		return nil, apperror.NewOAuthInvalidClient("client assertion is invalid")
	}

	claims := token.Claims.(jwtlib.MapClaims)

	if err := validateAssertionClaims(claims, client); err != nil {
		return nil, apperror.NewOAuthInvalidClient(err.Error())
	}

	return client, nil
}

func authenticateClientSecretJWT(client *Client, creds OAuthClientCredentials) (*Client, *apperror.OAuthError) {
	if creds.ClientAssertionType != assertionTypeJWTBearer {
		return nil, apperror.NewOAuthInvalidClient("client_assertion_type must be " + assertionTypeJWTBearer)
	}
	if creds.ClientAssertion == "" {
		return nil, apperror.NewOAuthInvalidClient("client_assertion is required")
	}

	secrets, err := clientSecretJWTSecrets(client)
	if err != nil {
		return nil, apperror.NewOAuthInvalidClient("client assertion is invalid")
	}
	if len(secrets) == 0 {
		return nil, apperror.NewOAuthInvalidClient("client has no registered secret")
	}

	token, err := jwtlib.Parse(creds.ClientAssertion, func(t *jwtlib.Token) (interface{}, error) {
		return jwtlib.VerificationKeySet{Keys: secrets}, nil
	}, jwtlib.WithLeeway(assertionClockSkew), jwtlib.WithValidMethods(assertionHMACAlgs))

	if err != nil || !token.Valid {
		return nil, apperror.NewOAuthInvalidClient("client assertion is invalid")
	}

	claims := token.Claims.(jwtlib.MapClaims)

	if err := validateAssertionClaims(claims, client); err != nil {
		return nil, apperror.NewOAuthInvalidClient(err.Error())
	}

	return client, nil
}

func clientSecretJWTSecrets(client *Client) ([]jwtlib.VerificationKey, error) {
	secrets := make([]jwtlib.VerificationKey, 0, 2)

	if client.SecretEncrypted != nil && strings.TrimSpace(*client.SecretEncrypted) != "" {
		secret, err := crypto.DecryptAtRest(*client.SecretEncrypted)
		if err != nil {
			return nil, err
		}
		if secret != "" {
			secrets = append(secrets, []byte(secret))
		}
	}

	if client.PreviousSecretEncrypted != nil &&
		client.PreviousSecretExpiresAt != nil &&
		client.PreviousSecretExpiresAt.After(time.Now()) &&
		strings.TrimSpace(*client.PreviousSecretEncrypted) != "" {
		secret, err := crypto.DecryptAtRest(*client.PreviousSecretEncrypted)
		if err != nil {
			return nil, err
		}
		if secret != "" {
			secrets = append(secrets, []byte(secret))
		}
	}

	return secrets, nil
}

// authorizationServerAssertionAudiences is the set of `aud` values a client
// assertion may name.
//
// RFC 7523 §3 point 3 requires the audience to identify the AUTHORIZATION SERVER
// — its issuer, or the endpoint the assertion is presented to. The check used to
// be `strings.HasPrefix(aud, *client.Domain)`: the client's own domain, matched
// by prefix. That is wrong twice over. An assertion addressed to the client
// itself is not addressed to this server, so an assertion the client minted for
// any other relying party could be replayed here to authenticate it; and a
// prefix match lets any attacker-controlled host that merely starts with the
// registered domain (`https://app.example.com.evil.test`) satisfy it.
func authorizationServerAssertionAudiences() []string {
	issuer := strings.TrimRight(config.AppPublicHostname, "/")
	if issuer == "" {
		return nil
	}
	return []string{
		issuer,
		issuer + "/",
		issuer + "/api/v1/oauth/token",
		issuer + "/api/v1/oauth/par",
	}
}

// assertionAudienceValues normalises the `aud` claim, which RFC 7519 §4.1.3
// allows to be either a single string or an array of strings. Reading it only as
// a string silently dropped the array form.
func assertionAudienceValues(raw any) []string {
	switch v := raw.(type) {
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return v
	}
	return nil
}

func validateAssertionClaims(claims jwtlib.MapClaims, client *Client) error {
	iss, _ := claims["iss"].(string)
	sub, _ := claims["sub"].(string)
	jti, _ := claims["jti"].(string)
	auds := assertionAudienceValues(claims["aud"])
	exp, _ := claims["exp"].(float64)

	if iss == "" || sub == "" || len(auds) == 0 || exp == 0 {
		return fmt.Errorf("assertion missing required claims")
	}

	if iss != ptrOrEmpty(client.Identifier) {
		return fmt.Errorf("assertion issuer does not match client_id")
	}
	if sub != ptrOrEmpty(client.Identifier) {
		return fmt.Errorf("assertion subject does not match client_id")
	}

	// Fails CLOSED when APP_PUBLIC_HOSTNAME is unset: with no known issuer there
	// is no audience this server can prove was meant for it.
	accepted := authorizationServerAssertionAudiences()
	if len(accepted) == 0 {
		return fmt.Errorf("assertion audience cannot be verified")
	}
	matched := false
	for _, aud := range auds {
		for _, want := range accepted {
			if strings.TrimRight(aud, "/") == strings.TrimRight(want, "/") {
				matched = true
				break
			}
		}
		if matched {
			break
		}
	}
	if !matched {
		return fmt.Errorf("assertion audience invalid")
	}

	now := time.Now()
	expTime := time.Unix(int64(exp), 0)
	if now.After(expTime.Add(assertionClockSkew)) {
		return fmt.Errorf("assertion expired")
	}
	// Cap the lifetime the client may choose. Without this an assertion with a
	// year-long exp is a bearer credential that never expires, which is exactly
	// what private_key_jwt exists to avoid.
	if expTime.After(now.Add(assertionMaxAge + assertionClockSkew)) {
		return fmt.Errorf("assertion lifetime exceeds the maximum of %s", assertionMaxAge)
	}
	if iat, ok := claims["iat"].(float64); ok && iat > 0 {
		if now.Sub(time.Unix(int64(iat), 0)) > assertionMaxAge+assertionClockSkew {
			return fmt.Errorf("assertion is older than the maximum of %s", assertionMaxAge)
		}
	}

	// RFC 7523 §3 point 7: a jti may authenticate exactly once. Recorded LAST so a
	// malformed or expired assertion cannot burn a jti the legitimate client is
	// about to use.
	if jti == "" {
		return fmt.Errorf("assertion missing required claims")
	}
	if !assertionReplayGuard.remember(jti, now) {
		return fmt.Errorf("assertion has already been used")
	}

	return nil
}

func findClientJWK(client *Client, kid string) (interface{}, error) {
	// An inline JWKS is authoritative when present; jwks_uri is the registry's
	// other, equally valid form and has to be dereferenced or the client can
	// never authenticate (see fetchClientJWKS).
	raw := []byte(client.JWKS)
	if len(raw) == 0 && client.JWKSUri != nil {
		fetched, err := fetchClientJWKS(*client.JWKSUri)
		if err != nil {
			return nil, err
		}
		raw = fetched
	}

	if len(raw) > 0 {
		var jwks struct {
			Keys []json.RawMessage `json:"keys"`
		}
		if err := json.Unmarshal(raw, &jwks); err != nil {
			return nil, fmt.Errorf("invalid JWKS format")
		}
		for _, rawKey := range jwks.Keys {
			var key struct {
				KID string `json:"kid"`
				KTY string `json:"kty"`
				N   string `json:"n"`
				E   string `json:"e"`
				Alg string `json:"alg"`
			}
			if err := json.Unmarshal(rawKey, &key); err != nil {
				continue
			}
			if kid != "" && key.KID != kid {
				continue
			}
			if key.KTY == "RSA" {
				n := new(big.Int)
				nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
				if err != nil {
					continue
				}
				n.SetBytes(nBytes)

				eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
				if err != nil {
					continue
				}
				e := new(big.Int).SetBytes(eBytes)

				if n.BitLen() < 2048 {
					return nil, fmt.Errorf("RSA key too weak: %d bits (minimum 2048)", n.BitLen())
				}
				pubKey := &rsa.PublicKey{
					N: n,
					E: int(e.Int64()),
				}
				return pubKey, nil
			}
		}
		return nil, fmt.Errorf("no usable RSA key found in client JWKS")
	}
	return nil, fmt.Errorf("no JWKS configured for client")
}

// clientHasVerificationMaterial reports whether the client can be authenticated
// with an asymmetric assertion at all.
func clientHasVerificationMaterial(client *Client) bool {
	if len(client.JWKS) > 0 {
		return true
	}
	return client.JWKSUri != nil && strings.TrimSpace(*client.JWKSUri) != ""
}

func ptrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
