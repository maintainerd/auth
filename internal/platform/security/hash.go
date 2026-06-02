package security

import (
	"context"
	"crypto/sha256"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/bcrypt"
)

// BcryptCost is the work factor used for all bcrypt hashes. 12 meets the
// minimum recommended by OWASP (2023) and satisfies SOC2/ISO27001 requirements.
const BcryptCost = 12

// HashPassword hashes a password using bcrypt. The caller's ctx is propagated
// into the tracing span.
func HashPassword(ctx context.Context, password []byte) ([]byte, error) {
	_, span := otel.Tracer("security").Start(ctx, "security.hash_password")
	defer span.End()

	hash, err := bcrypt.GenerateFromPassword(bcryptInput(password), BcryptCost)
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
// the database is compromised.
func HashClientSecret(ctx context.Context, secret string) (string, error) {
	_, span := otel.Tracer("security").Start(ctx, "security.hash_client_secret")
	defer span.End()

	hash, err := bcrypt.GenerateFromPassword(bcryptInput([]byte(secret)), BcryptCost)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "hash client secret failed")
		return "", err
	}
	span.SetStatus(codes.Ok, "")
	return string(hash), nil
}

// ComparePassword validates passwords hashed by HashPassword. It also accepts
// legacy raw bcrypt hashes so existing users are not locked out during rollout.
func ComparePassword(hash []byte, password []byte) bool {
	if bcrypt.CompareHashAndPassword(hash, bcryptInput(password)) == nil {
		return true
	}
	return bcrypt.CompareHashAndPassword(hash, password) == nil
}

// CompareClientSecret performs a constant-time bcrypt comparison between a
// plaintext client secret and its stored hash, preventing timing attacks.
func CompareClientSecret(plaintext, hash string) bool {
	if bcrypt.CompareHashAndPassword([]byte(hash), bcryptInput([]byte(plaintext))) == nil {
		return true
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext)) == nil
}

func bcryptInput(input []byte) []byte {
	sum := sha256.Sum256(input)
	return sum[:]
}
