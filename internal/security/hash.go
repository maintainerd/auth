package security

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a password using bcrypt with the default cost.
// Exposed as a function variable so tests can inject errors.
var HashPassword = func(password []byte) ([]byte, error) {
	_, span := otel.Tracer("security").Start(context.Background(), "security.hash_password")
	defer span.End()

	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "hash password failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return hash, nil
}

// HashClientSecret hashes a client secret with bcrypt. Client secrets are
// high-entropy random strings, but bcrypt ensures the hash is useless even if
// the database is compromised. Exposed as a function variable for test injection.
var HashClientSecret = func(secret string) (string, error) {
	_, span := otel.Tracer("security").Start(context.Background(), "security.hash_client_secret")
	defer span.End()

	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "hash client secret failed")
		return "", err
	}
	span.SetStatus(codes.Ok, "")
	return string(hash), nil
}

// CompareClientSecret performs a constant-time bcrypt comparison between a
// plaintext client secret and its stored hash, preventing timing attacks.
func CompareClientSecret(plaintext, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext)) == nil
}
