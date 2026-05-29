package oauth

import (
	"net/http"
)

// extractOAuthClientCredentials resolves client_id / client_secret from either
// HTTP Basic auth (RFC 6749 §2.3.1) or the request body form fields.
func extractOAuthClientCredentials(r *http.Request, clientID, clientSecret string) OAuthClientCredentials {
	if id, secret, ok := r.BasicAuth(); ok {
		return OAuthClientCredentials{ClientID: id, ClientSecret: secret}
	}
	return OAuthClientCredentials{ClientID: clientID, ClientSecret: clientSecret}
}
