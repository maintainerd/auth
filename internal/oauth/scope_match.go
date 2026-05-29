package oauth

import (
	"net/http"

	"github.com/maintainerd/auth/internal/dto"
)

// extractOAuthClientCredentials resolves client_id / client_secret from either
// HTTP Basic auth (RFC 6749 §2.3.1) or the request body form fields.
func extractOAuthClientCredentials(r *http.Request, clientID, clientSecret string) dto.OAuthClientCredentials {
	if id, secret, ok := r.BasicAuth(); ok {
		return dto.OAuthClientCredentials{ClientID: id, ClientSecret: secret}
	}
	return dto.OAuthClientCredentials{ClientID: clientID, ClientSecret: clientSecret}
}
