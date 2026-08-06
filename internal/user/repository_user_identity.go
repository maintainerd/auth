package user

import (
	"errors"

	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

type GetUserIdentitiesFilter struct {
	UserID    int64
	Provider  *string
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

type UserIdentityRepository interface {
	BaseRepositoryMethods[UserIdentity]
	FindAll(preloads ...string) ([]UserIdentity, error)
	FindByUUID(uuid any, preloads ...string) (*UserIdentity, error)
	FindByUUIDs(uuids []string, preloads ...string) ([]UserIdentity, error)
	FindByID(id any, preloads ...string) (*UserIdentity, error)
	UpdateByUUID(uuid any, updatedData any) (*UserIdentity, error)
	UpdateByID(id any, updatedData any) (*UserIdentity, error)
	DeleteByUUID(uuid any) error
	DeleteByID(id any) error
	Paginate(conditions map[string]any, page int, limit int, preloads ...string) (*PaginationResult[UserIdentity], error)
	WithTx(tx *gorm.DB) UserIdentityRepository
	FindByUserID(userID int64) ([]UserIdentity, error)
	FindUserIdentitiesPaginated(filter GetUserIdentitiesFilter) (*PaginationResult[UserIdentity], error)
	// FindByUserIDAndClientReachable returns the identity this user may present
	// to the given client. Identities belong to an identity provider, never to a
	// client (migration 030), so reachability is resolved through
	// client_identity_providers: the client must have an ENABLED connection to
	// the identity's provider. Disabling that connection immediately stops the
	// client authenticating the user, which a client_id column on the identity
	// could not express.
	FindByUserIDAndClientReachable(userID int64, clientID int64) (*UserIdentity, error)
	// FindByUserIDAndProvider returns the first identity for a user with the given provider slug.
	FindByUserIDAndProvider(userID int64, provider string) (*UserIdentity, error)
	// FindByUserIDAndIdentityProviderID returns the user's identity linked to a
	// specific configured IdP. This disambiguates the case where a user holds two
	// identities sharing the same provider slug (e.g. the built-in system
	// "maintainerd" and an external federated "maintainerd") — they differ only by
	// identity_provider_id / sub.
	FindByUserIDAndIdentityProviderID(userID int64, idpID int64) (*UserIdentity, error)
	// FindByIdentityProviderID lists all identities linked to a configured IDP.
	FindByIdentityProviderID(idpID int64) ([]UserIdentity, error)
	// FindByTenantProviderAndSub resolves an external identity by its
	// (tenant, provider, sub) triple. Returns nil when unlinked.
	FindByTenantProviderAndSub(tenantID int64, provider, sub string) (*UserIdentity, error)
	DeleteByUserID(userID int64) error
}

type userIdentityRepository struct {
	*BaseRepository[UserIdentity]
}

func NewUserIdentityRepository(db *gorm.DB) UserIdentityRepository {
	return &userIdentityRepository{
		BaseRepository: database.NewBaseRepository[UserIdentity](db, "user_identity_uuid", "user_identity_id"),
	}
}

func (r *userIdentityRepository) WithTx(tx *gorm.DB) UserIdentityRepository {
	return &userIdentityRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userIdentityRepository) FindByUserID(userID int64) ([]UserIdentity, error) {
	var identities []UserIdentity
	err := r.DB().Where("user_id = ?", userID).Find(&identities).Error
	return identities, err
}

func (r *userIdentityRepository) FindUserIdentitiesPaginated(filter GetUserIdentitiesFilter) (*PaginationResult[UserIdentity], error) {
	query := r.DB().Model(&UserIdentity{}).Where("user_id = ?", filter.UserID)

	query = database.ApplyILike(query, "provider", filter.Provider)

	query = query.Order(database.SanitizeOrderPrefixed("user_identities.", filter.SortBy, filter.SortOrder, "user_identities.created_at DESC"))

	return database.PaginateQuery[UserIdentity](query, filter.Page, filter.Limit)
}

func (r *userIdentityRepository) FindByUserIDAndClientReachable(userID int64, clientID int64) (*UserIdentity, error) {
	var identity UserIdentity
	// Ordering is deterministic so a user holding identities under two providers
	// both connected to this client always gets the same `sub`: an unstable sub
	// would break every relying party. Built-in (system) providers sort first —
	// these callers are the credential paths (password, SMS OTP, magic link,
	// backup code), which authenticate against the built-in provider.
	err := r.DB().
		Joins(`JOIN client_identity_providers cip
			ON cip.identity_provider_id = user_identities.identity_provider_id
			AND cip.tenant_id = user_identities.tenant_id`).
		// A soft-deleted or deactivated provider must stop authenticating, not
		// just stop accepting new logins. IdP deletion is soft, so the FK's
		// ON DELETE RESTRICT never fires and the connection rows survive.
		Joins(`JOIN identity_providers idp
			ON idp.identity_provider_id = user_identities.identity_provider_id
			AND idp.deleted_at IS NULL
			AND idp.status = ?`, shared.StatusActive).
		Where("user_identities.user_id = ?", userID).
		Where("cip.client_id = ? AND cip.enabled = TRUE AND cip.deleted_at IS NULL", clientID).
		Order("idp.is_system DESC, user_identities.user_identity_id ASC").
		First(&identity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &identity, nil
}

func (r *userIdentityRepository) FindByTenantProviderAndSub(tenantID int64, provider, sub string) (*UserIdentity, error) {
	var identity UserIdentity
	err := r.DB().Where("tenant_id = ? AND provider = ? AND sub = ?", tenantID, provider, sub).First(&identity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &identity, nil
}

func (r *userIdentityRepository) FindByUserIDAndProvider(userID int64, provider string) (*UserIdentity, error) {
	var identity UserIdentity
	err := r.DB().Where("user_id = ? AND provider = ?", userID, provider).First(&identity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &identity, nil
}

func (r *userIdentityRepository) FindByUserIDAndIdentityProviderID(userID int64, idpID int64) (*UserIdentity, error) {
	var identity UserIdentity
	err := r.DB().Where("user_id = ? AND identity_provider_id = ?", userID, idpID).First(&identity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &identity, nil
}

func (r *userIdentityRepository) FindByIdentityProviderID(idpID int64) ([]UserIdentity, error) {
	var identities []UserIdentity
	err := r.DB().Where("identity_provider_id = ?", idpID).Find(&identities).Error
	return identities, err
}

func (r *userIdentityRepository) DeleteByUserID(userID int64) error {
	return r.DB().Where("user_id = ?", userID).Delete(&UserIdentity{}).Error
}
