package app

import (
	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/user"
	"gorm.io/datatypes"
)

// accountLinkIdentityLinker adapts the user_identities repository to the narrow
// authn.AccountIdentityLinker interface used by the account-link confirmation
// flow. It creates identity links against the provider that issued them, since an
// identity link is user data, not client data.
type accountLinkIdentityLinker struct {
	repo user.UserIdentityRepository
}

func newAccountLinkIdentityLinker(repo user.UserIdentityRepository) authn.AccountIdentityLinker {
	return &accountLinkIdentityLinker{repo: repo}
}

func (a *accountLinkIdentityLinker) FindLinkedUserID(tenantID int64, provider, sub string) (int64, bool, error) {
	identity, err := a.repo.FindByTenantProviderAndSub(tenantID, provider, sub)
	if err != nil {
		return 0, false, err
	}
	if identity == nil {
		return 0, false, nil
	}
	return identity.UserID, true, nil
}

func (a *accountLinkIdentityLinker) LinkIdentity(tenantID, userID, identityProviderID int64, provider, sub string, claims []byte) error {
	if len(claims) == 0 {
		claims = []byte("{}")
	}
	_, err := a.repo.Create(&user.UserIdentity{
		TenantID:           tenantID,
		UserID:             userID,
		IdentityProviderID: identityProviderID,
		Provider:           provider,
		Sub:                sub,
		Metadata:           datatypes.JSON(claims),
	})
	return err
}
