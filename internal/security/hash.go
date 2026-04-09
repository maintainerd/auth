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
