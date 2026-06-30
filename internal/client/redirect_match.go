package client

import (
	"fmt"

	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/shared"
)

func MatchClientRedirectURI(client *Client, candidate string) error {
	if err := security.ValidateRedirectURI(candidate); err != nil {
		return fmt.Errorf("dangerous redirect: %w", err)
	}
	if client == nil || client.ClientURIs == nil {
		return fmt.Errorf("client has no registered redirect URIs")
	}
	for _, uri := range *client.ClientURIs {
		if uri.Type == shared.ClientURITypeRedirect && uri.URI == candidate {
			return nil
		}
	}
	return fmt.Errorf("redirect URI does not match any registered redirect URI")
}
