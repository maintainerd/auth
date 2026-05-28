package security

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/bcrypt"
)

// BcryptCost is the work factor used for all bcrypt hashes. 12 meets the
// minimum recommended by OWASP (2023) and satisfies SOC2/ISO27001 requirements.
const BcryptCost = 12

// HashPassword hashes a password using bcrypt. The caller's ctx is propagated
// into the tracing span. Exposed as a var so tests can inject errors.
var HashPassword = func(ctx context.Context, password []byte) ([]byte, error) {
	_, span := otel.Tracer("security").Start(ctx, "security.hash_password")
	defer span.End()

	hash, err := bcrypt.GenerateFromPassword(password, BcryptCost)
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
// the database is compromised. Exposed as a var so tests can inject errors.
var HashClientSecret = func(ctx context.Context, secret string) (string, error) {
	_, span := otel.Tracer("security").Start(ctx, "security.hash_client_secret")
	defer span.End()

	hash, err := bcrypt.GenerateFromPassword([]byte(secret), BcryptCost)
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
