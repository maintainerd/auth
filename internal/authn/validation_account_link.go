package authn

import (
	"errors"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
)

// validateAccountLinkToken validates the confirmation token path parameter. The
// token is a server-issued base64url value; callers never craft it, so this is
// a defensive bound check, not a format contract.
func validateAccountLinkToken(token string) error {
	token = strings.TrimSpace(security.SanitizeInput(token))
	if token == "" {
		return errors.New("confirmation token is required")
	}
	if len(token) > 255 {
		return errors.New("confirmation token is invalid")
	}
	return nil
}
