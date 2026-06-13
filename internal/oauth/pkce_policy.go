package oauth

import (
	"strings"

	"github.com/maintainerd/auth/internal/platform/apperror"
)

func validateOAuthPKCE(codeChallenge, method string, required bool) *apperror.OAuthError {
	codeChallenge = strings.TrimSpace(codeChallenge)
	method = strings.TrimSpace(method)
	if required && codeChallenge == "" {
		return apperror.NewOAuthInvalidRequest("code_challenge is required")
	}
	if required && method == "" {
		return apperror.NewOAuthInvalidRequest("code_challenge_method is required")
	}
	if codeChallenge == "" && method == "" {
		return nil
	}
	if codeChallenge == "" || len(codeChallenge) < 43 || len(codeChallenge) > 128 {
		return apperror.NewOAuthInvalidRequest("code_challenge must be between 43 and 128 characters")
	}
	if method != "S256" {
		return apperror.NewOAuthInvalidRequest("code_challenge_method must be 'S256'")
	}
	return nil
}
