package jwt

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/maintainerd/auth/internal/platform/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Security constants for SOC2/ISO27001 compliance
const (
	// Token expiration times (SOC2 CC6.3 - Logical Access Controls)
	AccessTokenTTL  = 15 * time.Minute   // Short-lived access tokens
	IDTokenTTL      = 1 * time.Hour      // ID tokens for user info
	RefreshTokenTTL = 7 * 24 * time.Hour // 7 days max for refresh tokens

	// Security parameters
	MinKeySize = 2048 // Minimum RSA key size (ISO27001 A.10.1.1)
	JTILength  = 32   // JTI entropy length
)

// GenerateSecureID generates a cryptographically secure random ID
// Complies with SOC2 CC6.1 and ISO27001 A.10.1.1
func GenerateSecureID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// keyMu guards all fields below. Use RLock for reads, Lock for writes.
var keyMu sync.RWMutex

// activePrivKey / activePubKey are the current signing key pair.
var (
	activePrivKey *rsa.PrivateKey
	activePubKey  *rsa.PublicKey
	activeKID     string
)

// retiredKey holds a public key that has been rotated out but may still be
// needed to verify tokens whose refresh window has not yet expired.
type retiredKey struct {
	kid       string
	pubKey    *rsa.PublicKey
	retiredAt time.Time
}

// retiringKeys is the ordered list of recently retired keys (oldest first).
var retiringKeys []retiredKey

// PublicKeyEntry is a KID-tagged public key returned by GetAllPublicKeys.
type PublicKeyEntry struct {
	KID    string
	PubKey *rsa.PublicKey
}

// JTIChecker is called during ValidateToken to check whether a token's JTI
// has been added to the denylist (e.g. by an explicit revocation call).
// Set this at startup: jwt.JTIChecker = cache.IsJTIDenied
// When nil, no denylist check is performed (tokens are validated purely by signature + expiry).
var JTIChecker func(ctx context.Context, jti string) (bool, error)

// GetPublicKey returns the active RSA public key used for JWT verification.
// Returns nil if InitJWTKeys has not been called.
func GetPublicKey() *rsa.PublicKey {
	keyMu.RLock()
	defer keyMu.RUnlock()
	return activePubKey
}

// GetAllPublicKeys returns the active key followed by any retiring keys that
// are still within the refresh-token validity window. The JWKS endpoint calls
// this so that clients can verify tokens signed by recently rotated keys.
func GetAllPublicKeys() []PublicKeyEntry {
	keyMu.RLock()
	defer keyMu.RUnlock()

	out := make([]PublicKeyEntry, 0, 1+len(retiringKeys))
	if activePubKey != nil {
		out = append(out, PublicKeyEntry{KID: activeKID, PubKey: activePubKey})
	}
	for _, rk := range retiringKeys {
		out = append(out, PublicKeyEntry{KID: rk.kid, PubKey: rk.pubKey})
	}
	return out
}

// RotateKeys generates a fresh RSA-2048 key pair, promotes the current active
// key to the retiring list, and prunes any retiring keys older than
// RefreshTokenTTL (tokens signed with them cannot be valid any more).
func RotateKeys() error {
	newKey, err := rsa.GenerateKey(rand.Reader, MinKeySize)
	if err != nil {
		return fmt.Errorf("jwt: generate rotation key: %w", err)
	}
	newKID := GenerateSecureID()

	keyMu.Lock()
	defer keyMu.Unlock()

	// Promote current active key to retiring list.
	if activePubKey != nil {
		retiringKeys = append(retiringKeys, retiredKey{
			kid:       activeKID,
			pubKey:    activePubKey,
			retiredAt: time.Now(),
		})
	}

	// Prune keys that are older than RefreshTokenTTL — no valid token can
	// still carry one of those KIDs.
	cutoff := time.Now().Add(-RefreshTokenTTL)
	live := retiringKeys[:0]
	for _, rk := range retiringKeys {
		if rk.retiredAt.After(cutoff) {
			live = append(live, rk)
		}
	}
	retiringKeys = live

	activePrivKey = newKey
	activePubKey = &newKey.PublicKey
	activeKID = newKID
	return nil
}

// generateSecureJTI creates a cryptographically secure unique token identifier
// Complies with SOC2 CC6.1 and ISO27001 A.10.1.1
func generateSecureJTI() string {
	bytes := make([]byte, JTILength)
	_, _ = rand.Read(bytes)

	// Create deterministic hash for uniqueness validation
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:16]) // 32 character hex string
}

// validateKeyStrength ensures RSA keys meet minimum security requirements
// Complies with ISO27001 A.10.1.1 (Key management policy)
func validateKeyStrength(key *rsa.PrivateKey) error {
	if key.Size()*8 < MinKeySize {
		return fmt.Errorf("RSA key size %d bits is below minimum required %d bits", key.Size()*8, MinKeySize)
	}
	return nil
}

func InitJWTKeys() error {
	var err error

	// Validate environment variables are not empty
	if len(config.JWTPrivateKey) == 0 {
		return errors.New("JWT_PRIVATE_KEY environment variable is required")
	}
	if len(config.JWTPublicKey) == 0 {
		return errors.New("JWT_PUBLIC_KEY environment variable is required")
	}

	// Parse private key with security validation
	privKey, err := jwtlib.ParseRSAPrivateKeyFromPEM(config.JWTPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Validate key strength (SOC2/ISO27001 compliance)
	if err := validateKeyStrength(privKey); err != nil {
		return fmt.Errorf("private key security validation failed: %w", err)
	}

	// Parse public key
	pubKey, err := jwtlib.ParseRSAPublicKeyFromPEM(config.JWTPublicKey)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	// Validate key pair consistency
	if privKey.PublicKey.N.Cmp(pubKey.N) != 0 || privKey.PublicKey.E != pubKey.E {
		return errors.New("private and public keys do not form a valid key pair")
	}

	keyMu.Lock()
	activePrivKey = privKey
	activePubKey = pubKey
	activeKID = config.GetEnvOrDefault("JWT_KEY_ID", "maintainerd-auth-key-1")
	keyMu.Unlock()

	return nil
}

// ResetJWTKeys clears the cached JWT signing keys.
// Intended for testing only.
func ResetJWTKeys() {
	keyMu.Lock()
	activePrivKey = nil
	activePubKey = nil
	activeKID = ""
	retiringKeys = nil
	keyMu.Unlock()
}

// AccessTokenOptions carries optional per-issuance parameters for access tokens.
type AccessTokenOptions struct {
	// DPoPThumbprint is the RFC 7638 JWK thumbprint of the client's DPoP key.
	// When non-empty, the token will contain a cnf.jkt claim and must be
	// presented with a matching DPoP proof on each resource request (RFC 9449).
	DPoPThumbprint string

	// AccessTokenTTL overrides the default AccessTokenTTL when > 0.
	// Used for per-client token lifetime configuration.
	AccessTokenTTL time.Duration
}

// GenerateAccessToken is the standard (Bearer) entry point for access token
// issuance. Use GenerateAccessTokenWithOptions when DPoP binding is needed.
func GenerateAccessToken(
	userId string,
	scope string,
	issuer string,
	audience string,
	clientID string,
	providerID string,
) (string, error) {
	return GenerateAccessTokenWithOptions(userId, scope, issuer, audience, clientID, providerID, nil)
}

// GenerateAccessTokenWithOptions issues an access token and optionally binds
// it to a DPoP key via the cnf.jkt claim (RFC 9449 §6.1).
func GenerateAccessTokenWithOptions(
	userId string,
	scope string,
	issuer string,
	audience string,
	clientID string,
	providerID string,
	opts *AccessTokenOptions,
) (string, error) {
	_, span := otel.Tracer("jwt").Start(context.Background(), "jwt.generate_access_token")
	defer span.End()
	span.SetAttributes(
		attribute.String("user_id", userId),
		attribute.String("issuer", issuer),
		attribute.String("audience", audience),
		attribute.String("client_id", clientID),
	)
	// Input validation (SOC2 CC6.1 - Logical Access Controls)
	if strings.TrimSpace(userId) == "" {
		span.SetStatus(codes.Error, "invalid input")
		return "", errors.New("userId cannot be empty")
	}
	if strings.TrimSpace(issuer) == "" {
		span.SetStatus(codes.Error, "invalid input")
		return "", errors.New("issuer cannot be empty")
	}
	if strings.TrimSpace(audience) == "" {
		span.SetStatus(codes.Error, "invalid input")
		return "", errors.New("audience cannot be empty")
	}
	if strings.TrimSpace(clientID) == "" {
		span.SetStatus(codes.Error, "invalid input")
		return "", errors.New("clientID cannot be empty")
	}
	if strings.TrimSpace(providerID) == "" {
		span.SetStatus(codes.Error, "invalid input")
		return "", errors.New("providerID cannot be empty")
	}

	// Generate secure JTI (ISO27001 A.10.1.1)
	jti := generateSecureJTI()

	now := time.Now()
	ttl := AccessTokenTTL
	if opts != nil && opts.AccessTokenTTL > 0 {
		ttl = opts.AccessTokenTTL
	}
	claims := jwtlib.MapClaims{
		// Standard JWT claims (RFC 7519)
		"sub": userId,
		"aud": audience,
		"iss": issuer,
		"iat": jwtlib.NewNumericDate(now),
		"exp": jwtlib.NewNumericDate(now.Add(ttl)),
		"nbf": jwtlib.NewNumericDate(now),
		"jti": jti,                                            // Secure unique identifier

		// OAuth2 claims
		"scope":      scope,
		"token_type": "access_token",

		// Auth client identification claims
		"client_id":   clientID,
		"provider_id": providerID,
	}

	// Bind to the client's DPoP key when a thumbprint was provided (RFC 9449 §6.1).
	if opts != nil && opts.DPoPThumbprint != "" {
		claims["cnf"] = map[string]string{"jkt": opts.DPoPThumbprint}
		claims["token_type"] = "DPoP"
	}

	tok, err := generateToken(claims)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "generate access token failed")
		return "", err
	}
	span.SetStatus(codes.Ok, "")
	return tok, nil
}

// UserProfile represents user profile data for ID tokens
type UserProfile struct {
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified"`
	Phone         string `json:"phone,omitempty"`
	PhoneVerified bool   `json:"phone_verified"`
	FirstName     string `json:"first_name,omitempty"`
	MiddleName    string `json:"middle_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
	Suffix        string `json:"suffix,omitempty"`
	Birthdate     string `json:"birthdate,omitempty"`
	Gender        string `json:"gender,omitempty"`
	Address       string `json:"address,omitempty"`
	Picture       string `json:"picture,omitempty"`
}

// defaultScopeClaimMap is the standard OIDC scope → profile claim mapping
// (OpenID Connect Core 1.0 §5.4). Clients may override this via ScopeClaimMappings.
var defaultScopeClaimMap = map[string][]string{
	"profile": {"first_name", "middle_name", "last_name", "suffix", "birthdate", "gender", "picture"},
	"email":   {"email", "email_verified"},
	"phone":   {"phone", "phone_verified"},
	"address": {"address"},
}

// IDTokenParams carries optional scope-filtering and custom claim data for
// ID token generation. Pass nil to get the legacy behaviour (all claims).
type IDTokenParams struct {
	// RequestedScopes is the set of scopes the client requested.
	// When non-nil, only claims mapped to these scopes are included.
	RequestedScopes []string
	// ScopeClaimMappings overrides defaultScopeClaimMap for this client.
	ScopeClaimMappings map[string][]string
	// ExtraClaims are static custom claims merged into the token last.
	ExtraClaims map[string]any
	// AMR is the list of Authentication Methods References (RFC 8176).
	// e.g. ["pwd"], ["pwd", "otp"], ["webauthn"]
	AMR []string
	// ACR is the Authentication Context Class Reference.
	// "1" = single-factor, "2" = multi-factor.
	ACR string
}

// buildAllowedClaimsSet returns the set of profile claim names that should be
// included based on the requested scopes and the client's mapping config.
// Returns nil when all claims should be included (params == nil).
func buildAllowedClaimsSet(params *IDTokenParams) map[string]struct{} {
	if params == nil || len(params.RequestedScopes) == 0 {
		return nil // include everything
	}
	mapping := defaultScopeClaimMap
	if params.ScopeClaimMappings != nil {
		mapping = params.ScopeClaimMappings
	}
	allowed := make(map[string]struct{})
	for _, scope := range params.RequestedScopes {
		for _, claim := range mapping[scope] {
			allowed[claim] = struct{}{}
		}
	}
	return allowed
}

// GenerateIDToken is the public entry point for ID token generation.
// It is assigned to the private implementation so tests can swap it out.
var GenerateIDToken = generateIDToken

func generateIDToken(userUUID, issuer, clientID, providerID string, profile *UserProfile, nonce string, params *IDTokenParams) (string, error) {
	_, span := otel.Tracer("jwt").Start(context.Background(), "jwt.generate_id_token")
	defer span.End()
	span.SetAttributes(
		attribute.String("user_uuid", userUUID),
		attribute.String("issuer", issuer),
		attribute.String("client_id", clientID),
	)
	// Input validation (SOC2 CC6.1 - Logical Access Controls)
	if strings.TrimSpace(userUUID) == "" {
		span.SetStatus(codes.Error, "invalid input")
		return "", errors.New("userUUID cannot be empty")
	}
	if strings.TrimSpace(issuer) == "" {
		span.SetStatus(codes.Error, "invalid input")
		return "", errors.New("issuer cannot be empty")
	}
	if strings.TrimSpace(clientID) == "" {
		span.SetStatus(codes.Error, "invalid input")
		return "", errors.New("clientID cannot be empty")
	}
	if strings.TrimSpace(providerID) == "" {
		span.SetStatus(codes.Error, "invalid input")
		return "", errors.New("providerID cannot be empty")
	}

	// Generate secure JTI
	jti := generateSecureJTI()

	now := time.Now()
	claims := jwtlib.MapClaims{
		// Standard OIDC claims (OpenID Connect Core 1.0)
		"sub":        userUUID,
		"aud":        clientID,
		"iss":        issuer,
		"iat":        jwtlib.NewNumericDate(now),
		"exp":        jwtlib.NewNumericDate(now.Add(IDTokenTTL)),
		"nbf":        jwtlib.NewNumericDate(now),
		"jti":        jti,
		"auth_time":  jwtlib.NewNumericDate(now),
		"token_type": "id_token",

		// Auth client identification claims
		"client_id":   clientID,
		"provider_id": providerID,
	}

	// Add nonce if provided (OIDC security requirement)
	if strings.TrimSpace(nonce) != "" {
		claims["nonce"] = nonce
	}

	// Build the set of allowed profile claims based on requested scopes.
	allowedClaims := buildAllowedClaimsSet(params)

	// addClaim adds a claim only when it passes the scope-filter.
	addClaim := func(name string, value any) {
		if allowedClaims == nil {
			claims[name] = value
			return
		}
		if _, ok := allowedClaims[name]; ok {
			claims[name] = value
		}
	}

	// Add profile claims filtered by the requested scopes.
	if profile != nil {
		if profile.Email != "" {
			addClaim("email", profile.Email)
			addClaim("email_verified", profile.EmailVerified)
		}
		if profile.Phone != "" {
			addClaim("phone", profile.Phone)
			addClaim("phone_verified", profile.PhoneVerified)
		}
		if profile.FirstName != "" {
			addClaim("first_name", profile.FirstName)
		}
		if profile.MiddleName != "" {
			addClaim("middle_name", profile.MiddleName)
		}
		if profile.LastName != "" {
			addClaim("last_name", profile.LastName)
		}
		if profile.Suffix != "" {
			addClaim("suffix", profile.Suffix)
		}
		if profile.Birthdate != "" {
			addClaim("birthdate", profile.Birthdate)
		}
		if profile.Gender != "" {
			addClaim("gender", profile.Gender)
		}
		if profile.Address != "" {
			addClaim("address", profile.Address)
		}
		if profile.Picture != "" {
			addClaim("picture", profile.Picture)
		}
	}

	// Merge custom claim mappers last (they may override profile claims).
	if params != nil {
		for k, v := range params.ExtraClaims {
			claims[k] = v
		}
		// Authentication context claims (OIDC Core §2 + RFC 8176).
		if len(params.AMR) > 0 {
			claims["amr"] = params.AMR
		}
		if params.ACR != "" {
			claims["acr"] = params.ACR
		}
	}

	tok, err := generateToken(claims)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "generate id token failed")
		return "", err
	}
	span.SetStatus(codes.Ok, "")
	return tok, nil
}

// GenerateRefreshToken is the public entry point for refresh token generation.
// It is assigned to the private implementation so tests can swap it out.
var GenerateRefreshToken = generateRefreshToken

func generateRefreshToken(userUUID, issuer, clientID, providerID string) (string, error) {
	_, span := otel.Tracer("jwt").Start(context.Background(), "jwt.generate_refresh_token")
	defer span.End()
	span.SetAttributes(
		attribute.String("user_uuid", userUUID),
		attribute.String("issuer", issuer),
		attribute.String("client_id", clientID),
	)
	// Input validation (SOC2 CC6.1 - Logical Access Controls)
	if strings.TrimSpace(userUUID) == "" {
		span.SetStatus(codes.Error, "invalid input")
		return "", errors.New("userUUID cannot be empty")
	}
	if strings.TrimSpace(issuer) == "" {
		span.SetStatus(codes.Error, "invalid input")
		return "", errors.New("issuer cannot be empty")
	}
	if strings.TrimSpace(clientID) == "" {
		span.SetStatus(codes.Error, "invalid input")
		return "", errors.New("clientID cannot be empty")
	}
	if strings.TrimSpace(providerID) == "" {
		span.SetStatus(codes.Error, "invalid input")
		return "", errors.New("providerID cannot be empty")
	}

	// Generate secure JTI
	jti := generateSecureJTI()

	now := time.Now()
	claims := jwtlib.MapClaims{
		// Standard JWT claims
		"sub":        userUUID,
		"aud":        clientID,
		"iss":        issuer,
		"iat":        jwtlib.NewNumericDate(now),
		"exp":        jwtlib.NewNumericDate(now.Add(RefreshTokenTTL)), // Configurable TTL
		"nbf":        jwtlib.NewNumericDate(now),
		"jti":        jti, // Secure unique identifier
		"token_type": "refresh_token",

		// Auth client identification claims
		"client_id":   clientID,
		"provider_id": providerID,
	}

	tok, err := generateToken(claims)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "generate refresh token failed")
		return "", err
	}
	span.SetStatus(codes.Ok, "")
	return tok, nil
}

// generateToken creates a JWT with enhanced security validation
// Complies with SOC2 CC6.1 and ISO27001 A.10.1.1
func generateToken(claims jwtlib.MapClaims) (string, error) {
	keyMu.RLock()
	priv := activePrivKey
	kid := activeKID
	keyMu.RUnlock()

	if priv == nil {
		return "", errors.New("private key not initialized - call InitJWTKeys() first")
	}

	// Validate required claims are present
	requiredClaims := []string{"sub", "aud", "iss", "iat", "exp", "jti"}
	for _, claim := range requiredClaims {
		if _, exists := claims[claim]; !exists {
			return "", fmt.Errorf("required claim '%s' is missing", claim)
		}
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	return token.SignedString(priv)
}

// ValidateToken performs comprehensive JWT validation
// Complies with SOC2 CC6.1, CC6.3 and ISO27001 A.9.4.2
// ──────────────────────────────────────────────────────────────────────────────
// Step-up challenge tokens (short-lived, MFA re-auth)
// ──────────────────────────────────────────────────────────────────────────────

// GenerateStepUpChallengeToken issues a short-lived signed JWT used as a
// step-up challenge handle. The token encodes the user's UUID so the MFA
// endpoint can cross-check it against the authenticated session.
func GenerateStepUpChallengeToken(userUUID string, ttl time.Duration) (string, error) {
	jti := generateSecureJTI()
	now := time.Now()
	claims := jwtlib.MapClaims{
		"sub": userUUID,
		"jti": jti,
		"typ": "step_up_challenge",
		"iat": jwtlib.NewNumericDate(now),
		"exp": jwtlib.NewNumericDate(now.Add(ttl)),
	}
	tok, err := generateToken(claims)
	if err != nil {
		return "", fmt.Errorf("step-up challenge token: %w", err)
	}
	return tok, nil
}

// ValidateStepUpChallengeToken validates a step-up challenge token and returns
// its claims. Returns an error when expired, wrong type, or signature invalid.
func ValidateStepUpChallengeToken(tokenString string) (jwtlib.MapClaims, error) {
	claims, err := ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	if typ, _ := claims["typ"].(string); typ != "step_up_challenge" {
		return nil, errors.New("token is not a step-up challenge token")
	}
	return claims, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// acr / amr helpers
// ──────────────────────────────────────────────────────────────────────────────

// AMRValues contains the standard Authentication Methods References.
const (
	AMRPassword = "pwd"
	AMROTP      = "otp"
	AMRMFA      = "mfa"
	AMRWebAuthn = "webauthn"
	AMRSMS      = "sms"
)

// ACRLevel1 is the ACR value for single-factor authentication (password only).
const ACRLevel1 = "1"

// ACRLevel2 is the ACR value for multi-factor authentication.
const ACRLevel2 = "2"

func ValidateToken(tokenString string) (jwtlib.MapClaims, error) {
	_, span := otel.Tracer("jwt").Start(context.Background(), "jwt.validate_token")
	defer span.End()

	keyMu.RLock()
	noKeys := activePubKey == nil
	keyMu.RUnlock()

	if noKeys {
		err := errors.New("public key not initialized - call InitJWTKeys() first")
		span.RecordError(err)
		span.SetStatus(codes.Error, "validate token failed")
		return nil, err
	}

	// Input validation
	if strings.TrimSpace(tokenString) == "" {
		span.SetStatus(codes.Error, "invalid input")
		return nil, errors.New("token cannot be empty")
	}

	// Parse and validate token
	token, err := jwtlib.Parse(tokenString, func(t *jwtlib.Token) (interface{}, error) {
		// Validate signing method (prevent algorithm confusion attacks)
		if method, ok := t.Method.(*jwtlib.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		} else if method != jwtlib.SigningMethodRS256 {
			return nil, fmt.Errorf("unexpected RSA signing method: %v", method.Alg())
		}

		// Look up the public key by KID. This supports both the active key and
		// any recently retired keys (within RefreshTokenTTL) so tokens signed
		// before a rotation remain valid until they naturally expire.
		// Tokens without a KID header (pre-rotation or test-crafted) are
		// verified with the active key for backward compatibility.
		kidVal, _ := t.Header["kid"].(string)

		keyMu.RLock()
		defer keyMu.RUnlock()

		if kidVal == "" || kidVal == activeKID {
			return activePubKey, nil
		}
		for _, rk := range retiringKeys {
			if rk.kid == kidVal {
				return rk.pubKey, nil
			}
		}
		return nil, fmt.Errorf("unknown key ID: %v", kidVal)
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "token parsing failed")
		return nil, fmt.Errorf("token parsing failed: %w", err)
	}

	// Extract and validate claims
	// jwtlib.Parse with no error guarantees MapClaims and token.Valid == true
	claims := token.Claims.(jwtlib.MapClaims)

	// Additional security validations
	if err := validateTokenClaims(claims); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "token claims validation failed")
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// JTI denylist check — rejects explicitly revoked access tokens.
	if JTIChecker != nil {
		if jti, ok := claims["jti"].(string); ok && jti != "" {
			denied, checkErr := JTIChecker(context.Background(), jti)
			if checkErr != nil {
				span.RecordError(checkErr)
				span.SetStatus(codes.Error, "jti denylist check failed")
				return nil, fmt.Errorf("token revocation check failed: %w", checkErr)
			}
			if denied {
				span.SetStatus(codes.Error, "token revoked")
				return nil, errors.New("token has been revoked")
			}
		}
	}

	span.SetStatus(codes.Ok, "")
	return claims, nil
}

// validateTokenClaims performs additional security validations on JWT claims
func validateTokenClaims(claims jwtlib.MapClaims) error {
	// Validate required claims exist
	requiredClaims := []string{"sub", "aud", "iss", "iat", "exp", "jti"}
	for _, claim := range requiredClaims {
		if _, exists := claims[claim]; !exists {
			return fmt.Errorf("required claim '%s' is missing", claim)
		}
	}

	// Validate subject is not empty
	if sub, ok := claims["sub"].(string); !ok || strings.TrimSpace(sub) == "" {
		return errors.New("subject (sub) claim is invalid or empty")
	}

	// Validate audience is not empty
	if aud, ok := claims["aud"].(string); !ok || strings.TrimSpace(aud) == "" {
		return errors.New("audience (aud) claim is invalid or empty")
	}

	// Validate issuer is not empty
	if iss, ok := claims["iss"].(string); !ok || strings.TrimSpace(iss) == "" {
		return errors.New("issuer (iss) claim is invalid or empty")
	}

	// Validate JTI is not empty (prevents token reuse)
	if jti, ok := claims["jti"].(string); !ok || strings.TrimSpace(jti) == "" {
		return errors.New("JTI (jti) claim is invalid or empty")
	}

	// Additional time-based validations are handled by jwt library
	// but we could add custom business logic here

	return nil
}
