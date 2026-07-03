package idp

import (
	"context"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

type PrincipalResolver interface {
	ResolvePrincipal(ctx context.Context, rawToken string, idp *IdentityProvider, audAllowed func(aud string) bool) (*FederatedPrincipal, error)
}

type FederatedPrincipal struct {
	UserID   int64
	UserUUID string
	TenantID int64
	Sub      string
	IsNew    bool
}

func (s *federationService) ResolvePrincipal(
	ctx context.Context,
	rawToken string,
	idp *IdentityProvider,
	audAllowed func(aud string) bool,
) (*FederatedPrincipal, error) {
	return s.resolveFederatedPrincipal(ctx, rawToken, idp, audAllowed, 0)
}

func (s *federationService) ResolveFederatedPrincipal(
	ctx context.Context,
	rawToken string,
	idp *IdentityProvider,
	audAllowed func(aud string) bool,
) (*FederatedPrincipal, error) {
	return s.resolveFederatedPrincipal(ctx, rawToken, idp, audAllowed, 0)
}

func (s *federationService) resolveFederatedPrincipal(
	ctx context.Context,
	rawToken string,
	idp *IdentityProvider,
	audAllowed func(aud string) bool,
	clientID int64,
) (*FederatedPrincipal, error) {
	claims, err := idpValidateOIDCToken(s, ctx, idp.IssuerOrEmpty(), idp.ProviderClientIDOrEmpty(), rawToken)
	if err != nil {
		return nil, apperror.NewUnauthorized("external token validation failed")
	}

	externalSub := stringClaim(claims, "sub")
	if externalSub == "" {
		return nil, apperror.NewValidation("external token missing 'sub' claim")
	}

	if tokenUse, ok := claims["token_use"].(string); ok && tokenUse != "id" {
		return nil, apperror.NewUnauthorized("token_use must be 'id'")
	}

	if audAllowed != nil && !validateAudience(claims, audAllowed) {
		return nil, apperror.NewUnauthorized("invalid token audience")
	}

	cfg, _ := buildOIDCConfig(idp)
	meta := extractMetadata(claims, cfg.AttributeMapping)
	email := meta.Email
	if email == "" {
		email = stringClaim(claims, "email")
	}

	var user *User
	var isNew bool
	var userSub string

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
				return apperror.NewInternal("user not found for existing identity", txErr)
			}
			_ = s.refreshMetadata(tx, existing, meta)
		} else {
			if !idp.AllowJITProvisioning {
				return apperror.NewUnauthorized("user not provisioned and JIT provisioning is disabled")
			}
			var provisionErr error
			user, isNew, provisionErr = s.provisionUser(ctx, tx, idp, externalSub, email, meta, clientID)
			if provisionErr != nil {
				return provisionErr
			}
		}

		defaultIdentity, txErr := txUserIdentityRepo.FindByUserIDAndProvider(user.UserID, shared.ProviderMaintainerd)
		if txErr != nil {
			return apperror.NewInternal("default identity lookup failed", txErr)
		}
		if defaultIdentity == nil {
			return apperror.NewInternal("user has no default identity", nil)
		}
		userSub = defaultIdentity.Sub
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &FederatedPrincipal{
		UserID:   user.UserID,
		UserUUID: user.UserUUID.String(),
		TenantID: idp.TenantID,
		Sub:      userSub,
		IsNew:    isNew,
	}, nil
}

func validateAudience(claims map[string]interface{}, audAllowed func(aud string) bool) bool {
	switch a := claims["aud"].(type) {
	case string:
		return a != "" && audAllowed(a)
	case []interface{}:
		for _, v := range a {
			if s, ok := v.(string); ok && s != "" && audAllowed(s) {
				return true
			}
		}
	}
	return false
}
