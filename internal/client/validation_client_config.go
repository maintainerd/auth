package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"gorm.io/datatypes"
)

// The advanced client-security settings (jwks, jwks_uri, mtls_bound_cert_thumbprint,
// scope_claim_mappings, claim_mappers) travel in the free-form `config` blob and are
// mirrored into first-class columns by applyAdvancedConfigToClientColumns. That
// mapper drops anything unusable so a bad value can never reach the runtime — but
// dropping silently means an operator who pastes a malformed JWKS sees a 200 and a
// client that still cannot authenticate. These rules turn each of those cases into a
// 422 naming the offending key.
//
// mtlsThumbprintPattern is the base64url SHA-256 certificate thumbprint of RFC 8705
// §3.1 — 32 bytes, unpadded, so exactly 43 characters.
var mtlsThumbprintPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

// maxJWKSURILength matches the backchannel/frontchannel logout URI cap.
const maxJWKSURILength = 2048

// validateClientConfig rejects a config blob that is not a JSON object, then
// validates the advanced keys inside it. It is deliberately tolerant of unknown
// keys: config also carries operator metadata and settings without a column.
func validateClientConfig(value any) error {
	raw, err := clientConfigMap(value)
	if err != nil {
		return err
	}
	if raw == nil {
		return nil
	}
	return validateAdvancedClientConfig(raw)
}

// clientConfigMap decodes the config blob into a map. A nil/empty blob is treated
// as absent ("unchanged") rather than invalid, matching the service.
func clientConfigMap(value any) (map[string]any, error) {
	var encoded []byte
	switch t := value.(type) {
	case nil:
		return nil, nil
	case datatypes.JSON:
		encoded = t
	case []byte:
		encoded = t
	case string:
		encoded = []byte(t)
	default:
		return nil, fmt.Errorf("config must be a JSON object")
	}

	if len(strings.TrimSpace(string(encoded))) == 0 {
		return nil, nil
	}

	var raw map[string]any
	if err := json.Unmarshal(encoded, &raw); err != nil {
		// Without this, a malformed blob is stored verbatim and every mirrored
		// setting inside it is silently ignored.
		return nil, fmt.Errorf("config must be a JSON object")
	}
	return raw, nil
}

func validateAdvancedClientConfig(raw map[string]any) error {
	_, hasJWKS := raw["jwks"]
	jwksURI, hasJWKSURI := nonEmptyStringFromConfig(raw["jwks_uri"])

	// RFC 7591 §2: a client MUST NOT use both. Two sources of truth for the
	// verification keys means the effective one depends on lookup order.
	if hasJWKS && hasJWKSURI {
		return fmt.Errorf("config.jwks and config.jwks_uri must not both be set; choose one key source")
	}

	if hasJWKS {
		if err := validateClientJWKS(raw["jwks"]); err != nil {
			return err
		}
	}
	if hasJWKSURI {
		if err := validateJWKSURI(jwksURI); err != nil {
			return err
		}
	}

	if v, present := raw["mtls_bound_cert_thumbprint"]; present {
		s, ok := nonEmptyStringFromConfig(v)
		if !ok || !mtlsThumbprintPattern.MatchString(s) {
			return fmt.Errorf(
				"config.mtls_bound_cert_thumbprint must be the base64url-encoded SHA-256 thumbprint of the certificate (43 characters)")
		}
	}

	if v, present := raw["scope_claim_mappings"]; present {
		if _, ok := v.(map[string]any); !ok {
			return fmt.Errorf("config.scope_claim_mappings must be a JSON object mapping a scope to its claims")
		}
	}

	if v, present := raw["claim_mappers"]; present {
		mappers, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("config.claim_mappers must be a JSON object mapping a claim name to its value")
		}
		// A client-defined mapper that lands on a reserved claim would let the
		// client restate its own identity, audience or permissions in the token it
		// receives. The token issuer strips these too; rejecting here means the
		// operator learns the mapper is being ignored instead of assuming it works.
		for name := range mappers {
			if jwt.IsReservedClaim(name) {
				return fmt.Errorf(
					"config.claim_mappers must not define the reserved claim %q; it is set by the token issuer", name)
			}
		}
	}

	return nil
}

// validateClientJWKS checks the inline JWK Set that authenticatePrivateKeyJWT
// verifies client assertions against.
func validateClientJWKS(value any) error {
	set, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("config.jwks must be a JWK Set object with a \"keys\" array")
	}
	keys, ok := set["keys"].([]any)
	if !ok || len(keys) == 0 {
		return fmt.Errorf("config.jwks must contain a non-empty \"keys\" array")
	}
	for i, item := range keys {
		key, ok := item.(map[string]any)
		if !ok {
			return fmt.Errorf("config.jwks.keys[%d] must be a JWK object", i)
		}
		if kty, ok := nonEmptyStringFromConfig(key["kty"]); !ok || kty == "" {
			return fmt.Errorf("config.jwks.keys[%d] must declare a \"kty\"", i)
		}
		// A JWKS is the client's PUBLIC key set. A private component here means the
		// operator pasted the signing key into the authorization server, so refuse
		// it rather than storing someone's private key in our database.
		for _, priv := range []string{"d", "p", "q", "dp", "dq", "qi", "k"} {
			if _, present := key[priv]; present {
				return fmt.Errorf(
					"config.jwks.keys[%d] contains the private key component %q; publish only the public JWK", i, priv)
			}
		}
	}
	return nil
}

func validateJWKSURI(raw string) error {
	if len(raw) > maxJWKSURILength {
		return fmt.Errorf("config.jwks_uri must be at most %d characters", maxJWKSURILength)
	}
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" {
		return fmt.Errorf("config.jwks_uri must be an absolute URL")
	}
	// The keys it serves decide whether a client assertion is accepted, so the
	// fetch must be authenticated and tamper-proof.
	if strings.ToLower(parsed.Scheme) != "https" {
		return fmt.Errorf("config.jwks_uri must use https")
	}
	if parsed.Fragment != "" {
		return fmt.Errorf("config.jwks_uri must not contain a fragment")
	}
	return nil
}
