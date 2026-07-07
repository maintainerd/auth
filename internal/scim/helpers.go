package scim

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/lib/pq"
)

func hashBearerToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}
