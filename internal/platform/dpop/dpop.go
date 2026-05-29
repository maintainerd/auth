// Package dpop implements Demonstrating Proof of Possession (DPoP) for OAuth 2.0
// per RFC 9449. DPoP binds access and refresh tokens to a client's ephemeral
// key pair, preventing bearer-token theft from being useful without the
// corresponding private key.
//
// Flow:
//  1. Client generates an ephemeral EC/RSA key pair.
//  2. On every token request, the client creates a DPoP proof JWT signed by
//     the private key and sends it in the DPoP header.
//  3. This server validates the proof, extracts the public key thumbprint
//     (JWK SHA-256 thumbprint, RFC 7638), and embeds it as cnf.jkt in the
//     issued access token.
//  4. On every resource request the client sends a fresh DPoP proof. Resource
//     servers verify both the access token cnf.jkt and the proof signature.
package dpop

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	// proofMaxAge is the maximum age of a DPoP proof JWT (RFC 9449 §11.1).
	proofMaxAge = 5 * time.Minute

	// replayWindowTTL is how long a used DPoP JTI is kept in the denylist to
	// prevent replay attacks. Must be >= proofMaxAge.
	replayWindowTTL = proofMaxAge
)

// JTIStore is the interface used to check and record DPoP proof JTIs for
// replay prevention. Implemented by the Redis cache.
type JTIStore interface {
	DenyJTI(ctx context.Context, jti string, ttl time.Duration) error
	IsJTIDenied(ctx context.Context, jti string) (bool, error)
}

// Claims holds the validated contents of a DPoP proof JWT.
type Claims struct {
	// JTI is the unique identifier of this proof (replay prevention).
	JTI string
	// HTTPMethod is the HTTP method the proof was created for.
	HTTPMethod string
	// HTTPURL is the full HTTP URL the proof was created for.
	HTTPURL string
	// IssuedAt is when the proof was signed.
	IssuedAt time.Time
	// Thumbprint is the SHA-256 JWK thumbprint of the proof's public key.
	Thumbprint string
}

// ValidateProof validates a DPoP proof JWT and returns the extracted claims.
//
// Parameters:
//   - proofHeader: the raw value of the DPoP HTTP header.
//   - method: expected HTTP method (e.g. "POST").
//   - requestURL: expected full request URL (e.g. "https://auth.example.com/oauth/token").
//   - accessTokenHash: SHA-256 hash of the access token encoded as base64url
//     (ath claim). Pass "" to skip ath validation (during token issuance there
//     is no access token yet).
//   - store: JTI denylist for replay prevention. May be nil (disables replay
//     prevention — only safe in tests).
func ValidateProof(
	ctx context.Context,
	proofHeader string,
	method string,
	requestURL string,
	accessTokenHash string,
	store JTIStore,
) (*Claims, error) {
	_, span := otel.Tracer("dpop").Start(ctx, "dpop.validate_proof")
	defer span.End()
	span.SetAttributes(
		attribute.String("http.method", method),
		attribute.String("http.url", requestURL),
	)

	if strings.TrimSpace(proofHeader) == "" {
		span.SetStatus(codes.Error, "missing DPoP proof")
		return nil, errors.New("DPoP proof header is missing")
	}

	// Parse the JWT without verifying the signature yet — we need the header
	// to extract the public key.
	unverified, _, err := new(jwtlib.Parser).ParseUnverified(proofHeader, jwtlib.MapClaims{})
	if err != nil {
		span.SetStatus(codes.Error, "dpop parse failed")
		return nil, fmt.Errorf("DPoP proof is not a valid JWT: %w", err)
	}

	// Confirm typ header is "dpop+jwt" (RFC 9449 §4.2).
	if typ, _ := unverified.Header["typ"].(string); !strings.EqualFold(typ, "dpop+jwt") {
		span.SetStatus(codes.Error, "wrong typ")
		return nil, fmt.Errorf("DPoP proof typ must be 'dpop+jwt', got %q", typ)
	}

	// Extract the embedded JWK from the proof header.
	jwkRaw, ok := unverified.Header["jwk"]
	if !ok {
		span.SetStatus(codes.Error, "missing jwk header")
		return nil, errors.New("DPoP proof header missing 'jwk'")
	}

	pubKey, thumbprint, err := extractPublicKeyAndThumbprint(jwkRaw)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "jwk extraction failed")
		return nil, fmt.Errorf("DPoP proof JWK invalid: %w", err)
	}
	span.SetAttributes(attribute.String("dpop.thumbprint", thumbprint))

	// Now verify the signature using the extracted public key.
	verified, err := jwtlib.Parse(proofHeader, func(t *jwtlib.Token) (interface{}, error) {
		switch t.Method.(type) {
		case *jwtlib.SigningMethodECDSA, *jwtlib.SigningMethodRSA:
			return pubKey, nil
		default:
			return nil, fmt.Errorf("unsupported DPoP signing algorithm: %v", t.Header["alg"])
		}
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "signature verification failed")
		return nil, fmt.Errorf("DPoP proof signature invalid: %w", err)
	}

	mc, ok := verified.Claims.(jwtlib.MapClaims)
	if !ok {
		return nil, errors.New("DPoP proof claims are malformed")
	}

	// Validate required claims.
	jti, _ := mc["jti"].(string)
	if strings.TrimSpace(jti) == "" {
		return nil, errors.New("DPoP proof missing jti claim")
	}

	htm, _ := mc["htm"].(string)
	if !strings.EqualFold(htm, method) {
		return nil, fmt.Errorf("DPoP proof htm %q does not match request method %q", htm, method)
	}

	htu, _ := mc["htu"].(string)
	if !strings.EqualFold(stripQuery(htu), stripQuery(requestURL)) {
		return nil, fmt.Errorf("DPoP proof htu %q does not match request URL %q", htu, requestURL)
	}

	// Validate iat freshness (RFC 9449 §11.1).
	iat, err := extractNumericDate(mc, "iat")
	if err != nil {
		return nil, fmt.Errorf("DPoP proof iat invalid: %w", err)
	}
	age := time.Since(iat)
	if age > proofMaxAge || age < -30*time.Second {
		return nil, fmt.Errorf("DPoP proof is too old or in the future (age=%s)", age.Round(time.Second))
	}

	// Validate ath when an access token is bound (RFC 9449 §4.2).
	if accessTokenHash != "" {
		ath, _ := mc["ath"].(string)
		if ath != accessTokenHash {
			return nil, errors.New("DPoP proof ath does not match access token hash")
		}
	}

	// Replay prevention: check the JTI has not been used within the replay window.
	if store != nil {
		used, checkErr := store.IsJTIDenied(ctx, "dpop:"+jti)
		if checkErr != nil {
			span.RecordError(checkErr)
			// On cache error, fail closed to prevent replay on degraded state.
			return nil, fmt.Errorf("DPoP replay check failed: %w", checkErr)
		}
		if used {
			return nil, errors.New("DPoP proof JTI has already been used (replay detected)")
		}
		// Record the JTI as used.
		_ = store.DenyJTI(ctx, "dpop:"+jti, replayWindowTTL)
	}

	span.SetStatus(codes.Ok, "")
	return &Claims{
		JTI:        jti,
		HTTPMethod: htm,
		HTTPURL:    htu,
		IssuedAt:   iat,
		Thumbprint: thumbprint,
	}, nil
}

// AccessTokenHash computes the base64url-encoded SHA-256 hash of a raw access
// token string, as required by the DPoP ath claim (RFC 9449 §4.2).
func AccessTokenHash(accessToken string) string {
	sum := sha256.Sum256([]byte(accessToken))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// ──────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ──────────────────────────────────────────────────────────────────────────────

// extractPublicKeyAndThumbprint parses a JWK object (from the JWT header) and
// returns the Go public key together with its RFC 7638 SHA-256 thumbprint.
func extractPublicKeyAndThumbprint(jwkRaw any) (crypto.PublicKey, string, error) {
	// Marshal then unmarshal to a flat map for reliable access.
	jwkBytes, err := json.Marshal(jwkRaw)
	if err != nil {
		return nil, "", fmt.Errorf("cannot marshal JWK: %w", err)
	}

	var jwkMap map[string]any
	if err := json.Unmarshal(jwkBytes, &jwkMap); err != nil {
		return nil, "", fmt.Errorf("cannot unmarshal JWK: %w", err)
	}

	kty, _ := jwkMap["kty"].(string)
	switch kty {
	case "EC":
		pub, thumb, err := parseECJWK(jwkMap, jwkBytes)
		return pub, thumb, err
	case "RSA":
		pub, thumb, err := parseRSAJWK(jwkMap, jwkBytes)
		return pub, thumb, err
	default:
		return nil, "", fmt.Errorf("unsupported JWK kty: %q", kty)
	}
}

// jwkThumbprint computes the RFC 7638 SHA-256 thumbprint for a JWK given its
// required members in lexicographic order, already JSON-encoded.
func jwkThumbprint(requiredMembersJSON []byte) string {
	sum := sha256.Sum256(requiredMembersJSON)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// parseECJWK parses an EC public key from a JWK map and computes its thumbprint.
func parseECJWK(jwkMap map[string]any, _ []byte) (*ecdsa.PublicKey, string, error) {
	crv, _ := jwkMap["crv"].(string)
	xb64, _ := jwkMap["x"].(string)
	yb64, _ := jwkMap["y"].(string)

	if crv == "" || xb64 == "" || yb64 == "" {
		return nil, "", errors.New("EC JWK missing crv, x, or y")
	}

	x, err := base64.RawURLEncoding.DecodeString(xb64)
	if err != nil {
		return nil, "", fmt.Errorf("EC JWK x decode: %w", err)
	}
	y, err := base64.RawURLEncoding.DecodeString(yb64)
	if err != nil {
		return nil, "", fmt.Errorf("EC JWK y decode: %w", err)
	}

	pub, err := buildECPublicKey(crv, x, y)
	if err != nil {
		return nil, "", err
	}

	// RFC 7638 §3.2 — required members in lexicographic order for EC.
	thumbJSON, _ := json.Marshal(map[string]string{
		"crv": crv,
		"kty": "EC",
		"x":   xb64,
		"y":   yb64,
	})
	return pub, jwkThumbprint(thumbJSON), nil
}

// parseRSAJWK parses an RSA public key from a JWK map and computes its thumbprint.
func parseRSAJWK(jwkMap map[string]any, _ []byte) (*rsa.PublicKey, string, error) {
	nb64, _ := jwkMap["n"].(string)
	eb64, _ := jwkMap["e"].(string)

	if nb64 == "" || eb64 == "" {
		return nil, "", errors.New("RSA JWK missing n or e")
	}

	pub, err := buildRSAPublicKey(nb64, eb64)
	if err != nil {
		return nil, "", err
	}

	// RFC 7638 §3.2 — required members in lexicographic order for RSA.
	thumbJSON, _ := json.Marshal(map[string]string{
		"e":   eb64,
		"kty": "RSA",
		"n":   nb64,
	})
	return pub, jwkThumbprint(thumbJSON), nil
}

// stripQuery removes the query string from a URL for htu comparison.
func stripQuery(u string) string {
	if idx := strings.IndexByte(u, '?'); idx >= 0 {
		return u[:idx]
	}
	return u
}

// extractNumericDate extracts a time.Time from a JWT claim that may be stored
// as a float64 (seconds since epoch) or a *jwtlib.NumericDate.
func extractNumericDate(mc jwtlib.MapClaims, key string) (time.Time, error) {
	v, ok := mc[key]
	if !ok {
		return time.Time{}, fmt.Errorf("claim %q is missing", key)
	}
	switch t := v.(type) {
	case float64:
		return time.Unix(int64(t), 0), nil
	case json.Number:
		n, err := t.Int64()
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(n, 0), nil
	case *jwtlib.NumericDate:
		return t.Time, nil
	default:
		return time.Time{}, fmt.Errorf("claim %q has unexpected type %T", key, v)
	}
}
