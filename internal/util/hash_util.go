package util

import "golang.org/x/crypto/bcrypt"

// HashPassword hashes a password using bcrypt with the default cost.
// Exposed as a function variable so tests can inject errors.
var HashPassword = func(password []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
}
