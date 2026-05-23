package dto

// ──────────────────────────────────────────────────────────────────────────────
// Token exchange (external token → our JWT)
// ──────────────────────────────────────────────────────────────────────────────

// FederationTokenRequestDTO is the body for POST /federation/token.
// The caller presents an ID token from an upstream provider and our client ID;
// we validate it and return our own access + ID + refresh tokens.
type FederationTokenRequestDTO struct {
	// ProviderIdentifier is the identifier of the configured IdentityProvider
	// record (e.g. "idp-abc123xyz").
	ProviderIdentifier string `json:"provider_identifier"`
	// ExternalToken is the raw OIDC ID token (JWT) from the upstream provider.
	ExternalToken string `json:"external_token"`
	// ClientID is our OAuth2 client identifier used to scope the issued tokens.
	ClientID string `json:"client_id"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Identity link / unlink
// ──────────────────────────────────────────────────────────────────────────────

// LinkIdentityRequestDTO is the body for POST /account/identities/link.
type LinkIdentityRequestDTO struct {
	ProviderIdentifier string `json:"provider_identifier"`
	ExternalToken      string `json:"external_token"`
}

// IdentityDTO is the public view of a UserIdentity record.
type IdentityDTO struct {
	IdentityUUID string  `json:"identity_uuid"`
	Provider     string  `json:"provider"`
	Sub          string  `json:"sub"`
	IsDefault    bool    `json:"is_default"`
	CreatedAt    string  `json:"created_at"`
	Email        *string `json:"email,omitempty"`
	Name         *string `json:"name,omitempty"`
	Picture      *string `json:"picture,omitempty"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Home Realm Discovery
// ──────────────────────────────────────────────────────────────────────────────

// HRDResponseDTO tells the frontend which provider handles the given email.
type HRDResponseDTO struct {
	ProviderIdentifier string `json:"provider_identifier"`
	Provider           string `json:"provider"`
	DisplayName        string `json:"display_name"`
}
