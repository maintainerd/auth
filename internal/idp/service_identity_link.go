package idp

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"go.opentelemetry.io/otel"
)

// identityLinkTTL bounds how long a started link may sit unfinished. Short,
// because the user is redirected to the provider and straight back — anything
// longer only widens the window in which a pending request could be misused.
const identityLinkTTL = 10 * time.Minute

// IdentityLinkRequest is the pending state of an in-flight account link.
//
// Declared here rather than importing oauth.OAuthBrokerSession because oauth
// already depends on this package's neighbours; internal/app supplies the
// adapter. It maps onto the same table the login broker uses, discriminated by
// purpose.
type IdentityLinkRequest struct {
	ID                 int64
	UserID             int64
	TenantID           int64
	ClientID           int64
	IdentityProviderID int64
	ProviderIdentifier string
	State              string
	PKCEVerifier       string
	Nonce              string
	ExpiresAt          time.Time
	Consumed           bool
}

// IdentityLinkRequestStore persists in-flight link requests.
//
// Consume MUST be atomic and single-use: it returns an error if the row was
// already consumed. That is the replay defence — two callbacks racing the same
// state must not both succeed.
type IdentityLinkRequestStore interface {
	Create(ctx context.Context, req *IdentityLinkRequest) error
	FindByState(ctx context.Context, state string) (*IdentityLinkRequest, error)
	Consume(ctx context.Context, id int64) error
}

// StartIdentityLinkResult carries the provider redirect the caller must follow.
type StartIdentityLinkResult struct {
	AuthorizationURL string `json:"authorization_url"`
	State            string `json:"state"`
}

// StartIdentityLink begins attaching a provider identity to an ALREADY
// signed-in account.
//
// The request is bound to userID at this point, and the callback can only ever
// attach to the user recorded here. That binding is the account-linking CSRF
// defence: without it an attacker could start a link with their own provider
// account, induce the victim to complete the callback, and end up with a
// provider identity that signs into the victim's account.
func (s *federationService) StartIdentityLink(
	ctx context.Context,
	userID, tenantID, clientID int64,
	providerIdentifier, redirectURI string,
) (*StartIdentityLinkResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "federation.start_identity_link")
	defer span.End()

	if s.linkStore == nil {
		return nil, apperror.NewInternal("identity link store not configured", nil)
	}

	idp, err := s.idpRepo.FindByIdentifier(providerIdentifier)
	if err != nil || idp == nil {
		return nil, apperror.NewNotFound("identity provider not found")
	}
	// Only providers belonging to the caller's tenant, and only active ones.
	// Without the tenant check a user could enumerate and link a provider from
	// another tenant.
	if idp.TenantID != tenantID {
		return nil, apperror.NewNotFound("identity provider not found")
	}
	if idp.Status != "active" {
		return nil, apperror.NewValidation("identity provider is not active")
	}

	// The provider will redirect back here with an authorization code, so an
	// unregistered URI would be an open redirect that leaks that code. Reuses
	// the same exact-match validator the SAML and OIDC broker paths use.
	if err := s.validateSAMLRedirectURI(clientID, redirectURI); err != nil {
		return nil, err
	}

	info, err := s.ResolveBrokerProvider(ctx, providerIdentifier)
	if err != nil {
		return nil, err
	}

	state, err := randomURLToken(32)
	if err != nil {
		return nil, apperror.NewInternal("failed to generate link state", err)
	}
	nonce, err := randomURLToken(32)
	if err != nil {
		return nil, apperror.NewInternal("failed to generate link nonce", err)
	}
	verifier, err := randomURLToken(48)
	if err != nil {
		return nil, apperror.NewInternal("failed to generate PKCE verifier", err)
	}

	if err := s.linkStore.Create(ctx, &IdentityLinkRequest{
		UserID:             userID,
		TenantID:           tenantID,
		ClientID:           clientID,
		IdentityProviderID: idp.IdentityProviderID,
		ProviderIdentifier: providerIdentifier,
		State:              state,
		PKCEVerifier:       verifier,
		Nonce:              nonce,
		ExpiresAt:          time.Now().Add(identityLinkTTL),
	}); err != nil {
		return nil, err
	}

	authorizeURL, err := buildProviderAuthorizeURL(info, redirectURI, state, nonce, verifier)
	if err != nil {
		return nil, err
	}

	return &StartIdentityLinkResult{AuthorizationURL: authorizeURL, State: state}, nil
}

// CompleteIdentityLink finishes the flow: verify the state belongs to this
// caller, consume it, exchange the code with the provider server-side, and
// attach the resulting identity to the caller's account.
//
// It never issues a session. Linking is not a sign-in — the caller is already
// authenticated, and minting one here would let this endpoint be used to
// switch accounts.
func (s *federationService) CompleteIdentityLink(
	ctx context.Context,
	userID int64,
	state, code, redirectURI string,
) (*IdentityDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "federation.complete_identity_link")
	defer span.End()

	if s.linkStore == nil {
		return nil, apperror.NewInternal("identity link store not configured", nil)
	}
	if strings.TrimSpace(state) == "" || strings.TrimSpace(code) == "" {
		return nil, apperror.NewValidation("state and code are required")
	}

	req, err := s.linkStore.FindByState(ctx, state)
	if err != nil {
		return nil, err
	}
	// One error for every rejection: an attacker probing states must not be able
	// to tell "no such state" from "belongs to someone else" from "expired".
	if req == nil || req.Consumed || time.Now().After(req.ExpiresAt) || req.UserID != userID {
		return nil, apperror.NewUnauthorized("this link request is no longer valid")
	}

	// Consume BEFORE the upstream exchange. Single-use is enforced by the store,
	// so a replayed callback fails here rather than performing a second exchange.
	if err := s.linkStore.Consume(ctx, req.ID); err != nil {
		return nil, apperror.NewUnauthorized("this link request is no longer valid")
	}

	if err := s.validateSAMLRedirectURI(req.ClientID, redirectURI); err != nil {
		return nil, err
	}

	idp, err := s.idpRepo.FindByID(req.IdentityProviderID)
	if err != nil || idp == nil {
		return nil, apperror.NewNotFound("identity provider not found")
	}
	cfg, err := buildOIDCConfig(idp)
	if err != nil {
		return nil, apperror.NewValidation("identity provider configuration is invalid")
	}
	clientSecret, err := idp.DecryptedProviderClientSecretStrict()
	if err != nil {
		return nil, apperror.NewInternal("provider client secret unavailable", err)
	}

	// Server-side exchange: the client secret never reaches the browser, and the
	// id_token is validated (issuer, audience, nonce) before we trust its sub.
	rawIDToken, _, err := s.exchangeUpstreamCode(ctx, idp, cfg, clientSecret, code, req.PKCEVerifier, req.Nonce, redirectURI)
	if err != nil {
		return nil, err
	}

	// Reuse the existing link path, which re-validates the token and refuses an
	// identity already attached to a different account.
	return s.LinkIdentity(ctx, userID, LinkIdentityRequestDTO{
		ProviderIdentifier: req.ProviderIdentifier,
		ExternalToken:      rawIDToken,
	})
}

// buildProviderAuthorizeURL assembles the upstream authorization request.
func buildProviderAuthorizeURL(info *BrokerProviderInfo, redirectURI, state, nonce, verifier string) (string, error) {
	u, err := url.Parse(info.AuthorizationEndpoint)
	if err != nil {
		return "", apperror.NewValidation("identity provider has an invalid authorization endpoint")
	}

	scopes := info.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", info.ClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", strings.Join(scopes, " "))
	q.Set("state", state)
	q.Set("nonce", nonce)
	q.Set("code_challenge", crypto.GeneratePKCEChallenge(verifier))
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// randomURLToken returns n bytes of CSPRNG output, URL-safe encoded.
func randomURLToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
