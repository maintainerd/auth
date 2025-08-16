package util

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/config"
)

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
)

func InitJWTKeys() error {
	var err error

	privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(config.JWTPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	publicKey, err = jwt.ParseRSAPublicKeyFromPEM(config.JWTPublicKey)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	return nil
}

func GenerateAccessToken(
	userId string,
	scope string,
	issuer string,
	audience string,
	authContainerId uuid.UUID,
	authClientID string,
	identityProviderId uuid.UUID,
) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   userId,
		"scope": scope,
		"exp":   jwt.NewNumericDate(now.Add(1 * time.Hour)),
		"iat":   jwt.NewNumericDate(now),
		"iss":   issuer,
		"aud":   audience,
		// Custom claims
		"m9d_auth_container_id":    authContainerId,
		"m9d_auth_client_id":       authClientID,
		"m9d_identity_provider_id": identityProviderId,
	}
	return generateToken(claims)
}

func GenerateIDToken(userUUID, issuer, clientID string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":            userUUID,
		"email_verified": true,
		"exp":            jwt.NewNumericDate(now.Add(1 * time.Hour)),
		"iat":            jwt.NewNumericDate(now),
		"iss":            issuer,
		"aud":            clientID,
	}
	return generateToken(claims)
}

func GenerateRefreshToken(userUUID, issuer, clientID string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":        userUUID,
		"token_type": "refresh_token",
		"exp":        jwt.NewNumericDate(now.Add(30 * 24 * time.Hour)),
		"iat":        jwt.NewNumericDate(now),
		"iss":        issuer,
		"aud":        clientID,
	}
	return generateToken(claims)
}

func generateToken(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

func ValidateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("token parsing failed: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims or token invalid")
	}

	return claims, nil
}
