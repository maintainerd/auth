package idp

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"time"

	crewsaml "github.com/crewjam/saml"
	"github.com/google/uuid"
	clientpkg "github.com/maintainerd/maintainerd-auth/internal/client"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"gorm.io/gorm"
)

const samlExchangeCodeTTL = 5 * time.Minute
const samlExchangeKeyPrefix = "saml:code:"

// samlRelayNonceKeyPrefix namespaces the single-use marker recorded for each
// RelayState nonce. Its TTL matches the RelayState validity window so a replayed
// (or duplicated) SAML Response carrying an already-seen RelayState is rejected.
const samlRelayNonceKeyPrefix = "saml:relay-nonce:"

// samlRelayNonceTTL bounds how long a used RelayState nonce is remembered. It
// matches the 15-minute RelayState expiry in verifyRelayState — an expired
// RelayState is rejected outright, so remembering it longer adds nothing.
const samlRelayNonceTTL = 15 * time.Minute

// InitiateSAMLSSO looks up the SAML IdP, builds the SP, generates an
// AuthnRequest, and returns the IdP redirect URL the browser should follow.
func (s *federationService) InitiateSAMLSSO(ctx context.Context, in SAMLInitiateInput) (*SAMLInitiateResult, error) {
	idp, err := s.idpRepo.FindByIdentifier(in.ProviderIdentifier)
	if err != nil || idp == nil {
		return nil, apperror.NewNotFound("identity provider not found")
	}
	if idp.Status != "active" {
		return nil, apperror.NewValidation("identity provider is not active")
	}

	cfg, err := parseSAMLConfig(idp)
	if err != nil {
		return nil, apperror.NewValidation(err.Error())
	}

	// Validate the requested redirect_uri against the client's registered redirect
	// URIs before we ever start the flow. This is defense-in-depth: HandleSAMLResponse
	// re-validates before appending the exchange code, but rejecting here fails fast
	// and avoids issuing an AuthnRequest for an unregistered redirect target.
	client, err := s.clientRepo.FindByClientIDAndIdentityProvider(in.ClientID, idp.Identifier)
	if err != nil || client == nil {
		return nil, apperror.NewNotFound("client not found for this provider")
	}
	if err := s.validateSAMLRedirectURI(client.ClientID, in.RedirectURI); err != nil {
		return nil, err
	}

	entityID, acsURL, metadataURL, sloURL := samlSPURLs(in.ProviderIdentifier)
	sp, err := buildSAMLServiceProvider(cfg, entityID, acsURL, metadataURL, sloURL)
	if err != nil {
		return nil, apperror.NewInternal("failed to build SAML SP", err)
	}

	// Build the AuthnRequest first so we can capture its ID and bind the eventual
	// Response to it via InResponseTo. This mirrors the crewjam
	// MakeRedirectAuthenticationRequest helper but keeps a handle on the request.
	authnReq, err := sp.MakeAuthenticationRequest(
		sp.GetSSOBindingLocation(crewsaml.HTTPRedirectBinding),
		crewsaml.HTTPRedirectBinding,
		crewsaml.HTTPPostBinding,
	)
	if err != nil {
		return nil, apperror.NewInternal("failed to generate SAML AuthnRequest", err)
	}

	relayState, err := newSAMLRelayState(in.ProviderIdentifier, in.ClientID, in.RedirectURI, in.TenantID, authnReq.ID)
	if err != nil {
		return nil, apperror.NewInternal("failed to generate relay state", err)
	}

	redirectURL, err := authnReq.Redirect(relayState, sp)
	if err != nil {
		return nil, apperror.NewInternal("failed to generate SAML AuthnRequest", err)
	}

	return &SAMLInitiateResult{RedirectURL: redirectURL.String()}, nil
}

// validateSAMLRedirectURI validates a SAML flow's redirect_uri against the
// client's registered redirect URIs using the SAME matcher the OIDC broker uses
// (exact match + dangerous-scheme block). This closes the open-redirect that
// would otherwise leak the one-time SAML exchange code to an attacker-controlled
// URL. Reuses clientpkg.MatchClientRedirectURI rather than a looser local check.
func (s *federationService) validateSAMLRedirectURI(clientID int64, redirectURI string) error {
	uris, err := s.clientRepo.FindRedirectURIs(clientID)
	if err != nil {
		return apperror.NewInternal("redirect URI lookup failed", err)
	}
	matches := make([]clientpkg.RedirectURIMatch, len(uris))
	for i, u := range uris {
		matches[i] = clientpkg.RedirectURIMatch{URI: u.URI, Type: u.Type}
	}
	if err := clientpkg.MatchClientRedirectURI(matches, redirectURI); err != nil {
		return apperror.NewValidation("redirect_uri is not registered for this client: " + err.Error())
	}
	return nil
}

// HandleSAMLResponse processes the IdP's HTTP-POST ACS response, provisions
// or authenticates the user, stores a short-lived exchange code, and returns
// the redirect URI with the code appended as a query parameter.
func (s *federationService) HandleSAMLResponse(ctx context.Context, r *http.Request, relayState string) (*SAMLCallbackResult, error) {
	rs, err := verifyRelayStateForPurpose(relayState, samlRelayPurposeSSO)
	if err != nil {
		return nil, apperror.NewUnauthorized("invalid or expired relay state")
	}

	// Enforce single-use of the RelayState so a captured (still-valid) SAML
	// Response + RelayState pair cannot be replayed within the assertion window.
	if err := s.enforceRelayStateSingleUse(ctx, rs.Nonce); err != nil {
		return nil, err
	}

	idp, err := s.idpRepo.FindByIdentifier(rs.ProviderIdentifier)
	if err != nil || idp == nil {
		return nil, apperror.NewNotFound("identity provider not found")
	}
	if idp.Status != "active" {
		return nil, apperror.NewValidation("identity provider is not active")
	}

	cfg, err := parseSAMLConfig(idp)
	if err != nil {
		return nil, apperror.NewValidation(err.Error())
	}

	entityID, acsURL, metadataURL, sloURL := samlSPURLs(rs.ProviderIdentifier)
	sp, err := buildSAMLServiceProvider(cfg, entityID, acsURL, metadataURL, sloURL)
	if err != nil {
		return nil, apperror.NewInternal("failed to build SAML SP", err)
	}

	// Parse and validate the SAML response. We bind the Response to the exact
	// AuthnRequest we issued by passing its ID as the only accepted InResponseTo
	// value. Combined with AllowIDPInitiated=false this rejects unsolicited or
	// mismatched (injected/replayed cross-session) responses. The IdP assertion
	// conditions (audience, NotBefore, NotOnOrAfter, recipient) still apply.
	assertion, err := sp.ParseResponse(r, []string{rs.RequestID})
	if err != nil {
		return nil, apperror.NewUnauthorized("SAML response validation failed: " + err.Error())
	}

	if assertion.Subject == nil || assertion.Subject.NameID == nil {
		return nil, apperror.NewUnauthorized("SAML assertion missing subject NameID")
	}
	externalSub := assertion.Subject.NameID.Value
	if externalSub == "" {
		return nil, apperror.NewUnauthorized("SAML assertion NameID is empty")
	}

	meta := extractSAMLClaims(assertion, cfg.AttributeMapping)
	email := meta.Email

	// F2: a SAML-asserted email is only treated as verified (and thus eligible to
	// merge into an existing local account) when its domain is on the provider's
	// configured email_domains allow-list. Otherwise EmailVerified stays false so
	// provisionUser can never silently merge on an unproven email; JIT creation of
	// a brand-new user still works.
	if email != "" {
		allowed, domErr := s.emailDomainRepo.FindByProviderID(idp.IdentityProviderID)
		if domErr != nil {
			return nil, apperror.NewInternal("email domain lookup failed", domErr)
		}
		meta.EmailVerified = samlEmailDomainAllowed(email, allowed)
	}

	client, err := s.clientRepo.FindByClientIDAndIdentityProvider(rs.ClientID, idp.Identifier)
	if err != nil || client == nil {
		return nil, apperror.NewNotFound("client not found for this provider")
	}

	// F1: validate the RelayState-carried redirect_uri against the client's
	// registered redirect URIs before the one-time exchange code is appended to
	// it. Without this an attacker could pass an arbitrary redirect_uri at
	// initiate time (it rides in the signed RelayState) and receive the code at a
	// URL they control — an account-takeover open redirect.
	if err := s.validateSAMLRedirectURI(client.ClientID, rs.RedirectURI); err != nil {
		return nil, err
	}

	var user *User
	var internalSub string
	var isNew bool

	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		existing, txErr := txUserIdentityRepo.FindByTenantProviderAndSub(idp.TenantID, idp.Provider, externalSub)
		if txErr != nil {
			return apperror.NewInternal("identity lookup failed", txErr)
		}

		if existing != nil {
			user, txErr = txUserRepo.FindByID(existing.UserID)
			if txErr != nil || user == nil {
				return apperror.NewInternal("user lookup failed", txErr)
			}
			_ = s.refreshMetadata(tx, existing, meta)
		} else {
			// JIT is enforced inside provisionUser (single source of truth), which
			// runs the email-collision check FIRST — so a SAML user whose email
			// matches an existing account is routed to account-linking
			// (handleEmailCollision below) instead of being refused by a premature
			// JIT check when JIT is off. provisionUser still refuses a genuinely new
			// user when JIT is disabled.
			user, isNew, txErr = s.provisionUser(ctx, tx, idp, externalSub, email, meta, &client.ClientID)
			if txErr != nil {
				return txErr
			}
		}

		identity, txErr := txUserIdentityRepo.FindByUserIDAndIdentityProviderID(user.UserID, idp.IdentityProviderID)
		if txErr != nil || identity == nil {
			return apperror.NewInternal("identity resolution failed", txErr)
		}
		internalSub = identity.Sub
		return nil
	})
	if errors.Is(err, errIdentityCreatedConcurrently) {
		user, internalSub, err = s.resolveExistingUserIdentity(idp, externalSub, false)
	}
	// F3: a verified-email collision must be surfaced for explicit confirmation,
	// never silently merged. Handle it consistently with the OIDC broker flow.
	if collisionErr := s.handleEmailCollision(ctx, err); collisionErr != nil {
		return nil, collisionErr
	}
	if err != nil {
		return nil, err
	}

	loginResp, err := s.generateTokens(ctx, internalSub, user, client)
	if err != nil {
		return nil, err
	}

	code := uuid.New().String()
	if storeErr := s.samlStore.SetSession(ctx, samlExchangeKeyPrefix+code, loginResp, samlExchangeCodeTTL); storeErr != nil {
		return nil, apperror.NewInternal("failed to store SAML exchange code", storeErr)
	}

	redirectURI := rs.RedirectURI
	sep := "?"
	if len(redirectURI) > 0 {
		for _, c := range redirectURI {
			if c == '?' {
				sep = "&"
				break
			}
		}
	}
	redirectURI = redirectURI + sep + "code=" + code

	return &SAMLCallbackResult{
		RedirectURI: redirectURI,
		Code:        code,
		IsNew:       isNew,
	}, nil
}

// enforceRelayStateSingleUse rejects a RelayState nonce that has already been
// seen, then records it so the same signed RelayState (and thus the same SAML
// Response) cannot be replayed. The HMAC-signed RelayState already guarantees
// integrity + a 15-minute TTL; this adds single-use on top. An empty nonce is
// treated as tampering and rejected (all issued RelayStates carry a nonce).
func (s *federationService) enforceRelayStateSingleUse(ctx context.Context, nonce string) error {
	if nonce == "" {
		return apperror.NewUnauthorized("invalid relay state")
	}
	seen, err := s.markSAMLMessageSeen(ctx, samlRelayNonceKeyPrefix+nonce)
	if err != nil {
		return err
	}
	if seen {
		// A value exists for this nonce — it has already been consumed.
		return apperror.NewUnauthorized("SAML relay state has already been used")
	}
	return nil
}

// ExchangeSAMLCode exchanges a short-lived SAML exchange code (issued by
// HandleSAMLResponse) for the full LoginResponseDTO. Each code is single-use.
func (s *federationService) ExchangeSAMLCode(ctx context.Context, code string) (*LoginResponseDTO, error) {
	if code == "" {
		return nil, apperror.NewValidation("code is required")
	}
	key := samlExchangeKeyPrefix + code
	var resp LoginResponseDTO
	if err := s.samlStore.GetSession(ctx, key, &resp); err != nil {
		return nil, apperror.NewUnauthorized("SAML exchange code not found or expired")
	}
	_ = s.samlStore.DeleteSession(ctx, key)
	return &resp, nil
}

// SAMLMetadata returns the SP metadata XML for the given provider identifier.
func (s *federationService) SAMLMetadata(ctx context.Context, identifier string) ([]byte, error) {
	idp, err := s.idpRepo.FindByIdentifier(identifier)
	if err != nil || idp == nil {
		return nil, apperror.NewNotFound("identity provider not found")
	}

	cfg, err := parseSAMLConfig(idp)
	if err != nil {
		return nil, apperror.NewValidation(fmt.Sprintf("invalid SAML config: %s", err))
	}

	entityID, acsURL, metadataURL, sloURL := samlSPURLs(identifier)
	sp, err := buildSAMLServiceProvider(cfg, entityID, acsURL, metadataURL, sloURL)
	if err != nil {
		return nil, apperror.NewInternal("failed to build SAML SP", err)
	}

	metadata := sp.Metadata()
	xmlBytes, err := xml.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, apperror.NewInternal("failed to marshal SP metadata", err)
	}
	return append([]byte(xml.Header), xmlBytes...), nil
}
