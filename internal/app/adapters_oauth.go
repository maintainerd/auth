package app

import (
	"github.com/maintainerd/auth/internal/oauth"
	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type oauthClientRepo struct {
	*database.BaseRepository[oauth.Client]
}

func newOAuthClientRepo(db *gorm.DB) oauth.ClientRepository {
	return &oauthClientRepo{database.NewBaseRepository[oauth.Client](db, "client_uuid", "client_id")}
}

func (r *oauthClientRepo) WithTx(tx *gorm.DB) oauth.ClientRepository {
	return &oauthClientRepo{r.BaseRepository.WithTx(tx)}
}

func (r *oauthClientRepo) FindSystem() (*oauth.Client, error) {
	var c oauth.Client
	err := r.DB().Where("is_system = ?", true).First(&c).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &c, nil
}

func (r *oauthClientRepo) FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*oauth.Client, error) {
	query := r.DB().Model(&oauth.Client{}).Where("clients.identifier = ?", clientID)
	if identityProviderIdentifier != "" {
		query = query.Joins("JOIN identity_providers ON identity_providers.identity_provider_id = clients.identity_provider_id").
			Where("identity_providers.identifier = ?", identityProviderIdentifier)
	}
	var c oauth.Client
	err := query.First(&c).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &c, nil
}

type oauthClientURIRepo struct {
	*database.BaseRepository[oauth.ClientURI]
}

func newOAuthClientURIRepo(db *gorm.DB) oauth.ClientURIRepository {
	return &oauthClientURIRepo{database.NewBaseRepository[oauth.ClientURI](db, "client_uri_uuid", "client_uri_id")}
}

func (r *oauthClientURIRepo) WithTx(tx *gorm.DB) oauth.ClientURIRepository {
	return &oauthClientURIRepo{r.BaseRepository.WithTx(tx)}
}

type oauthTenantRepo struct {
	*database.BaseRepository[oauth.Tenant]
}

func newOAuthTenantRepo(db *gorm.DB) oauth.TenantRepository {
	return &oauthTenantRepo{database.NewBaseRepository[oauth.Tenant](db, "tenant_uuid", "tenant_id")}
}

func (r *oauthTenantRepo) WithTx(tx *gorm.DB) oauth.TenantRepository {
	return &oauthTenantRepo{r.BaseRepository.WithTx(tx)}
}

func (r *oauthTenantRepo) FindSystem() (*oauth.Tenant, error) {
	var t oauth.Tenant
	err := r.DB().Where("is_system = ?", true).First(&t).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &t, nil
}

type oauthUserRepo struct {
	*database.BaseRepository[oauth.User]
}

func newOAuthUserRepo(db *gorm.DB) oauth.UserRepository {
	return &oauthUserRepo{database.NewBaseRepository[oauth.User](db, "user_uuid", "user_id")}
}

func (r *oauthUserRepo) WithTx(tx *gorm.DB) oauth.UserRepository {
	return &oauthUserRepo{r.BaseRepository.WithTx(tx)}
}

func (r *oauthUserRepo) FindByEmail(email string) (*oauth.User, error) {
	var u oauth.User
	err := r.DB().Where("email = ?", email).First(&u).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &u, nil
}

func (r *oauthUserRepo) FindByEmailAndTenantID(email string, tenantID int64) (*oauth.User, error) {
	var u oauth.User
	err := r.DB().Where("email = ? AND tenant_id = ?", email, tenantID).First(&u).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &u, nil
}

func (r *oauthUserRepo) FindBySubAndClientID(sub, clientID string) (*oauth.User, error) {
	var u oauth.User
	err := r.DB().Joins("JOIN user_identities ON user_identities.user_id = users.user_id").
		Joins("JOIN clients ON clients.client_id = user_identities.client_id").
		Where("user_identities.sub = ? AND clients.identifier = ?", sub, clientID).
		First(&u).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &u, nil
}

type oauthUserIdentityRepo struct {
	*database.BaseRepository[oauth.UserIdentity]
}

func newOAuthUserIdentityRepo(db *gorm.DB) oauth.UserIdentityRepository {
	return &oauthUserIdentityRepo{database.NewBaseRepository[oauth.UserIdentity](db, "user_identity_uuid", "user_identity_id")}
}

func (r *oauthUserIdentityRepo) WithTx(tx *gorm.DB) oauth.UserIdentityRepository {
	return &oauthUserIdentityRepo{r.BaseRepository.WithTx(tx)}
}

func (r *oauthUserIdentityRepo) FindByUserIDAndClientID(userID, clientID int64) (*oauth.UserIdentity, error) {
	var identity oauth.UserIdentity
	err := r.DB().Where("user_id = ? AND client_id = ?", userID, clientID).First(&identity).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &identity, nil
}
