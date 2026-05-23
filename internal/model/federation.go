package model

// OIDCProviderConfig is stored as JSONB in IdentityProvider.Config for
// providers with provider_type = "social" (external OIDC/OAuth2 upstreams).
type OIDCProviderConfig struct {
	// Issuer is the OIDC discovery base URL (e.g. "https://accounts.google.com").
	// The library will fetch /<issuer>/.well-known/openid-configuration automatically.
	Issuer string `json:"issuer"`

	// ClientID is the OAuth2 client ID registered with the upstream provider.
	// Required for audience ("aud") validation of incoming ID tokens.
	ClientID string `json:"client_id"`

	// ClientSecret is the OAuth2 client secret. Only needed when our backend
	// needs to exchange an authorization code (not required for token presentation).
	// TODO: encrypt at rest before storing in production.
	ClientSecret string `json:"client_secret,omitempty"`

	// Scopes requested during the authorization flow (e.g. ["openid","email","profile"]).
	Scopes []string `json:"scopes,omitempty"`

	// AllowJITProvisioning enables just-in-time user creation when a valid
	// external token belongs to an unknown user.
	AllowJITProvisioning bool `json:"allow_jit_provisioning"`

	// AttributeMapping maps upstream OIDC claim names to our local field names.
	// Key = our field ("email", "name", "picture"), value = claim name ("email", "name").
	// If empty, the default OIDC standard claim names are used.
	AttributeMapping map[string]string `json:"attribute_mapping,omitempty"`

	// EmailDomains lists the email domains that should be routed to this
	// provider via Home Realm Discovery (e.g. ["company.com"]).
	EmailDomains []string `json:"email_domains,omitempty"`

	// UserinfoEndpoint overrides the userinfo URL discovered from the OIDC
	// document. Useful for non-standard providers.
	UserinfoEndpoint string `json:"userinfo_endpoint,omitempty"`
}

// IdentityMetadata is stored as JSONB in UserIdentity.Metadata for external
// provider identities. It captures the profile attributes returned by the
// upstream provider at link/login time.
type IdentityMetadata struct {
	Email      string `json:"email,omitempty"`
	Name       string `json:"name,omitempty"`
	GivenName  string `json:"given_name,omitempty"`
	FamilyName string `json:"family_name,omitempty"`
	Picture    string `json:"picture,omitempty"`
	Locale     string `json:"locale,omitempty"`
}
