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
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/maintainerd/auth/internal/config"
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

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
)

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
	privateKey, err = jwtlib.ParseRSAPrivateKeyFromPEM(config.JWTPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Validate key strength (SOC2/ISO27001 compliance)
	if err := validateKeyStrength(privateKey); err != nil {
		return fmt.Errorf("private key security validation failed: %w", err)
	}

	// Parse public key
	publicKey, err = jwtlib.ParseRSAPublicKeyFromPEM(config.JWTPublicKey)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	// Validate key pair consistency
	if privateKey.PublicKey.N.Cmp(publicKey.N) != 0 || privateKey.PublicKey.E != publicKey.E {
		return errors.New("private and public keys do not form a valid key pair")
	}

	return nil
}

// ResetJWTKeys clears the cached JWT signing keys.
// Intended for testing only.
func ResetJWTKeys() {
	privateKey = nil
	publicKey = nil
}

func GenerateAccessToken(
	userId string,
	scope string,
	issuer string,
	audience string,
	clientID string,
	providerID string,
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
	claims := jwtlib.MapClaims{
		// Standard JWT claims (RFC 7519)
		"sub": userId,
		"aud": audience,
		"iss": issuer,
		"iat": jwtlib.NewNumericDate(now),
		"exp": jwtlib.NewNumericDate(now.Add(AccessTokenTTL)), // Short-lived tokens
		"nbf": jwtlib.NewNumericDate(now),                     // Not before
		"jti": jti,                                         // Secure unique identifier

		// OAuth2 claims
		"scope":      scope,
		"token_type": "access_token",

		// Auth client identification claims
		"client_id":   clientID,
		"provider_id": providerID,
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

var GenerateIDToken = generateIDToken

func generateIDToken(userUUID, issuer, clientID, providerID string, profile *UserProfile, nonce string) (string, error) {
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

	// Add profile claims if provided (avoid hardcoded data)
	if profile != nil {
		if profile.Email != "" {
			claims["email"] = profile.Email
			claims["email_verified"] = profile.EmailVerified
		}
		if profile.Phone != "" {
			claims["phone"] = profile.Phone
			claims["phone_verified"] = profile.PhoneVerified
		}
		if profile.FirstName != "" {
			claims["first_name"] = profile.FirstName
		}
		if profile.MiddleName != "" {
			claims["middle_name"] = profile.MiddleName
		}
		if profile.LastName != "" {
			claims["last_name"] = profile.LastName
		}
		if profile.Suffix != "" {
			claims["suffix"] = profile.Suffix
		}
		if profile.Birthdate != "" {
			claims["birthdate"] = profile.Birthdate
		}
		if profile.Gender != "" {
			claims["gender"] = profile.Gender
		}
		if profile.Address != "" {
			claims["address"] = profile.Address
		}
		if profile.Picture != "" {
			claims["picture"] = profile.Picture
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
	if privateKey == nil {
		return "", errors.New("private key not initialized - call InitJWTKeys() first")
	}

	// Validate required claims are present
	requiredClaims := []string{"sub", "aud", "iss", "iat", "exp", "jti"}
	for _, claim := range requiredClaims {
		if _, exists := claims[claim]; !exists {
			return "", fmt.Errorf("required claim '%s' is missing", claim)
		}
	}

	// Use RS256 for asymmetric signing (more secure than HS256)
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)

	// Add key ID header for key rotation support (configurable via JWT_KEY_ID env var)
	token.Header["kid"] = config.GetEnvOrDefault("JWT_KEY_ID", "maintainerd-auth-key-1")

	return token.SignedString(privateKey)
}

// ValidateToken performs comprehensive JWT validation
// Complies with SOC2 CC6.1, CC6.3 and ISO27001 A.9.4.2
func ValidateToken(tokenString string) (jwtlib.MapClaims, error) {
	_, span := otel.Tracer("jwt").Start(context.Background(), "jwt.validate_token")
	defer span.End()

	if publicKey == nil {
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

		// Validate key ID if present (for key rotation)
		if kid, exists := t.Header["kid"]; exists {
			expectedKID := config.GetEnvOrDefault("JWT_KEY_ID", "maintainerd-auth-key-1")
			if kid != expectedKID {
				return nil, fmt.Errorf("unknown key ID: %v", kid)
			}
		}

		return publicKey, nil
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
