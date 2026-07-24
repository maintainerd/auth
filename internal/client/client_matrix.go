package client

import (
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

// The OAuth metadata on a client is only coherent as a combination. Validating
// each field against its own allowlist — which validTokenEndpointAuthMethods and
// the DB CHECK constraints already do — still permits combinations that are
// individually legal and jointly unsafe. The worst is
// `token_endpoint_auth_method=none` on a confidential or m2m client: `none` means
// "presents no credential", so paired with the client_credentials grant it turns
// the token endpoint into an unauthenticated one. client_id is public (it is
// handed out by GET /client and appears in every authorize URL), so anyone could
// mint that client's tokens and receive its resolved permissions.
//
// This file is the single place those cross-field rules live, so the HTTP
// handler, the gRPC handler and the seeder cannot drift apart.

// IsPublicClientType reports whether a client type is structurally incapable of
// keeping a secret — a browser or mobile app whose code the user can read.
// Public clients are the only ones RFC 6749 §2.1 permits to authenticate with
// method "none", and they are constrained instead by PKCE and exact
// redirect-URI matching.
func IsPublicClientType(clientType string) bool {
	return clientType == shared.ClientTypeSPA || clientType == shared.ClientTypeMobile
}

// clientAuthMethodRequiresSecret reports whether the method authenticates using
// the shared client secret, so the client must actually have one.
func clientAuthMethodRequiresSecret(method string) bool {
	switch method {
	case TokenAuthMethodSecretBasic, TokenAuthMethodSecretPost, TokenAuthMethodClientSecretJWT:
		return true
	}
	return false
}

// mTLS client authentication is accepted by the registry's allowlist and the DB
// CHECK constraint, but no certificate-binding implementation exists behind it —
// the token endpoint rejects both methods outright. Allowing one to be saved
// produces a client that can never authenticate, so refuse it at write time
// rather than at first login.
func clientAuthMethodIsUnimplemented(method string) bool {
	return method == TokenAuthMethodTLSClientAuth || method == TokenAuthMethodSelfSignedTLSClientAuth
}

// ValidateClientOAuthMatrix enforces the cross-field rules between client type,
// client authentication method and grant types. It runs on create and update,
// after the per-field allowlists, and returns a validation error naming the
// specific incompatibility so an operator can act on it.
//
// hasSecret reports whether the client will hold a usable client secret once the
// write completes. hasClientKeys reports whether it will hold a JWKS or a
// jwks_uri. allowedScopes is the client's scope allowlist, which must be
// non-empty for a client_credentials client.
func ValidateClientOAuthMatrix(clientType, authMethod string, grantTypes, allowedScopes []string, hasSecret, hasClientKeys bool) error {
	public := IsPublicClientType(clientType)

	if clientAuthMethodIsUnimplemented(authMethod) {
		return apperror.NewValidation(
			"token_endpoint_auth_method \"" + authMethod +
				"\" requires mutual-TLS client authentication, which this server does not implement yet")
	}

	// The core rule: no credential means the client must be one that cannot hold
	// a credential.
	if authMethod == TokenAuthMethodNone && !public {
		return apperror.NewValidation(
			"token_endpoint_auth_method \"none\" is only valid for public clients (spa, mobile); " +
				"a " + clientType + " client must authenticate with a secret, private_key_jwt or client_secret_jwt")
	}

	// A public client cannot keep a secret, so a secret-based method is a false
	// sense of security: the secret ships in the client and is readable.
	if public && clientAuthMethodRequiresSecret(authMethod) {
		return apperror.NewValidation(
			"a " + clientType + " client cannot keep a secret; use token_endpoint_auth_method \"none\" with PKCE")
	}

	// A method that authenticates with the shared secret needs one to exist.
	if clientAuthMethodRequiresSecret(authMethod) && !hasSecret {
		return apperror.NewValidation(
			"token_endpoint_auth_method \"" + authMethod + "\" requires a client secret")
	}

	// private_key_jwt verifies the client assertion against the client's public
	// keys. Without a JWKS or jwks_uri the token endpoint rejects every assertion
	// ("client has no JWKS or jwks_uri configured"), so saving such a client
	// produces one that can never authenticate.
	if authMethod == TokenAuthMethodPrivateKeyJWT && !hasClientKeys {
		return apperror.NewValidation(
			"token_endpoint_auth_method \"private_key_jwt\" requires the client's public keys; " +
				"set either jwks (an inline JWK Set) or jwks_uri")
	}

	for _, grant := range grantTypes {
		switch grant {
		case GrantTypeClientCredentials:
			// client_credentials authenticates the CLIENT itself with no user in
			// the loop, so an unauthenticated client would be minting tokens for
			// nobody. It also has neither PKCE nor a redirect URI to constrain it.
			if public {
				return apperror.NewValidation(
					"grant type \"client_credentials\" is not valid for a " + clientType +
						" client: it requires client authentication, which a public client cannot provide")
			}
			if authMethod == TokenAuthMethodNone {
				return apperror.NewValidation(
					"grant type \"client_credentials\" requires client authentication; " +
						"token_endpoint_auth_method \"none\" would allow anyone holding the public client_id to mint tokens")
			}
			// An empty allowed_scopes list means "all scopes permitted" — a
			// deliberate default elsewhere, but for a machine client with no user
			// in the loop it makes the credential unbounded. Require the operator
			// to state what it may request.
			if len(allowedScopes) == 0 {
				return apperror.NewValidation(
					"a client using \"client_credentials\" must declare allowed_scopes; " +
						"an empty list permits every scope, which is unbounded for a machine credential")
			}
		case GrantTypeAuthorizationCode:
			// m2m has no user and no browser, so there is no authorization code to
			// exchange.
			if clientType == shared.ClientTypeM2M {
				return apperror.NewValidation(
					"grant type \"authorization_code\" is not valid for an m2m client: there is no user to authorize")
			}
		}
	}

	return nil
}

// ─── Identity-provider connection invariants ────────────────────────────────
//
// A client's login options come entirely from its identity-provider
// connections, and the built-in (system) provider is what anchors a LOCAL user
// identity. That matters even when an external provider such as Cognito is
// configured: federated provisioning creates a user_identities row for the
// built-in provider alongside the external one, so the user exists in this
// system rather than only in the upstream directory.
//
// Consequences of losing it are silent and severe:
//   - authn connectedSystemIdentityProviderID returns nil, so password login
//     fails with a generic "authentication failed" — indistinguishable from a
//     wrong password.
//   - /oauth/connections reports password_enabled=false with an empty provider
//     list, so the hosted login page renders no sign-in option at all.
//   - the per-IdP registration gate returns "allow" with nothing to allow, so
//     self-registration fails downstream.
//
// The seeder attaches it, but nothing re-checked it afterwards — so these
// invariants are enforced here, on every connection mutation.

// isBuiltInConnection reports whether a connection points at the tenant's
// built-in (system) identity provider.
//
// It fails CLOSED: when the provider cannot be resolved it reports true, so a
// preload miss or a lookup error protects the connection rather than silently
// skipping the guard. The previous inline check tested
// `connection.IdentityProvider != nil && ...IsSystem`, which skipped protection
// entirely whenever the preload was absent.
func (s *clientService) isBuiltInConnection(tx *gorm.DB, connection *ClientIdentityProvider) (bool, error) {
	if connection.IdentityProvider != nil {
		return connection.IdentityProvider.IsSystem, nil
	}
	provider, err := s.idpRepo.WithTx(tx).FindByID(connection.IdentityProviderID)
	if err != nil {
		// Fail closed: treat an unresolvable provider as protected.
		return true, err
	}
	if provider == nil {
		return true, nil
	}
	return provider.IsSystem, nil
}

// assertConnectionMutationKeepsClientUsable rejects a connection change that
// would leave the client without a working login path.
//
// wantEnabled/wantDefault describe the connection's state AFTER the change;
// removing is true when the connection is being deleted outright.
func (s *clientService) assertConnectionMutationKeepsClientUsable(
	tx *gorm.DB,
	clientID int64,
	connection *ClientIdentityProvider,
	wantEnabled bool,
	wantDefault bool,
	removing bool,
) error {
	builtIn, err := s.isBuiltInConnection(tx, connection)
	if err != nil {
		return apperror.NewInternal("failed to resolve the identity provider for this connection", err)
	}

	if builtIn {
		if removing {
			return apperror.NewValidation("the built-in identity provider connection cannot be removed")
		}
		if !wantEnabled {
			return apperror.NewValidation(
				"the built-in identity provider connection cannot be disabled: it provides password sign-in " +
					"and the local user identity for this client")
		}
		if !wantDefault && connection.IsDefault {
			return apperror.NewValidation(
				"the built-in identity provider must remain the default for this client; " +
					"promote another provider only after connecting one that can anchor a local identity")
		}
		return nil
	}

	// Non-built-in connection: it must not be the last enabled one.
	if removing || !wantEnabled {
		connections, err := s.clientIdentityProviderRepo.WithTx(tx).FindByClientID(clientID)
		if err != nil {
			return err
		}
		remaining := 0
		for i := range connections {
			c := &connections[i]
			if c.ClientIdentityProviderID == connection.ClientIdentityProviderID {
				continue
			}
			// nil Enabled means "DB default", which is TRUE.
			if c.Enabled == nil || *c.Enabled {
				remaining++
			}
		}
		if remaining == 0 {
			return apperror.NewValidation(
				"this is the client's only enabled identity provider; disabling or removing it would leave " +
					"no way to sign in")
		}
	}

	return nil
}
