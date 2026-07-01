package client

import (
	"fmt"

	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/shared"
)

type RedirectURIMatch struct {
	URI  string
	Type string
}

func MatchClientRedirectURI(uris []RedirectURIMatch, candidate string) error {
	if err := security.ValidateRedirectURI(candidate); err != nil {
		return fmt.Errorf("forbidden scheme: %w", err)
	}
	if len(uris) == 0 {
		return fmt.Errorf("no redirect URIs registered for this client")
	}
	for _, uri := range uris {
		if uri.Type == shared.ClientURITypeRedirect && uri.URI == candidate {
			return nil
		}
	}
	return fmt.Errorf("redirect_uri does not match any registered redirect URIs")
}
