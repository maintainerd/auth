package idp

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"gorm.io/gorm"
)

const samlExchangeCodeTTL = 5 * time.Minute
const samlExchangeKeyPrefix = "saml:code:"

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

	entityID, acsURL, metadataURL := samlSPURLs(in.ProviderIdentifier)
	sp, err := buildSAMLServiceProvider(cfg, entityID, acsURL, metadataURL)
	if err != nil {
		return nil, apperror.NewInternal("failed to build SAML SP", err)
	}

	relayState, err := newSAMLRelayState(in.ProviderIdentifier, in.ClientID, in.RedirectURI, in.TenantID)
	if err != nil {
		return nil, apperror.NewInternal("failed to generate relay state", err)
	}

	redirectURL, err := sp.MakeRedirectAuthenticationRequest(relayState)
	if err != nil {
		return nil, apperror.NewInternal("failed to generate SAML AuthnRequest", err)
	}

	return &SAMLInitiateResult{RedirectURL: redirectURL.String()}, nil
}

// HandleSAMLResponse processes the IdP's HTTP-POST ACS response, provisions
// or authenticates the user, stores a short-lived exchange code, and returns
// the redirect URI with the code appended as a query parameter.
func (s *federationService) HandleSAMLResponse(ctx context.Context, r *http.Request, relayState string) (*SAMLCallbackResult, error) {
	rs, err := verifyRelayState(relayState)
	if err != nil {
		return nil, apperror.NewUnauthorized("invalid or expired relay state")
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

	entityID, acsURL, metadataURL := samlSPURLs(rs.ProviderIdentifier)
	sp, err := buildSAMLServiceProvider(cfg, entityID, acsURL, metadataURL)
	if err != nil {
		return nil, apperror.NewInternal("failed to build SAML SP", err)
	}

	// Parse and validate the SAML response. We pass an empty requestIDs slice
	// because relay state is stateless — the IdP assertion conditions (audience,
	// NotBefore, NotOnOrAfter, recipient) still enforce freshness and binding.
	assertion, err := sp.ParseResponse(r, []string{})
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

	client, err := s.clientRepo.FindByClientIDAndIdentityProvider(rs.ClientID, idp.Identifier)
	if err != nil || client == nil {
		return nil, apperror.NewNotFound("client not found for this provider")
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
			if !idp.AllowJITProvisioning {
				return apperror.NewUnauthorized("user not found and JIT provisioning is disabled for this provider")
			}
			user, isNew, txErr = s.provisionUser(ctx, tx, idp, externalSub, email, meta, &client.ClientID)
			if txErr != nil {
				return txErr
			}
		}

		identity, txErr := txUserIdentityRepo.FindByUserIDAndProvider(user.UserID, idp.Provider)
		if txErr != nil || identity == nil {
			return apperror.NewInternal("identity resolution failed", txErr)
		}
		internalSub = identity.Sub
		return nil
	})
	if errors.Is(err, errIdentityCreatedConcurrently) {
		user, internalSub, err = s.resolveExistingUserIdentity(idp.TenantID, idp.Provider, externalSub, idp.Provider)
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

	entityID, acsURL, metadataURL := samlSPURLs(identifier)
	sp, err := buildSAMLServiceProvider(cfg, entityID, acsURL, metadataURL)
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
