package util

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("your-secret-key")

func GenerateToken(userUUID string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"user_uuid": userUUID,
		"exp":       jwt.NewNumericDate(now.Add(24 * time.Hour)),
		"iat":       jwt.NewNumericDate(now),
		"type":      "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func GenerateRefreshToken(userUUID string) (string, error) {
	claims := jwt.MapClaims{
		"user_uuid": userUUID,
		"exp":       jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
		"type":      "refresh",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ValidateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	if exp, ok := claims["exp"].(float64); ok {
		if int64(exp) < time.Now().Unix() {
			return nil, errors.New("token expired")
		}
	}

	return claims, nil
}
