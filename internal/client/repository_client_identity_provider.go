package client

import (
	"errors"

	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type ClientIdentityProviderRepository interface {
	BaseRepositoryMethods[ClientIdentityProvider]
	WithTx(tx *gorm.DB) ClientIdentityProviderRepository
	FindByUUIDAndTenantID(uuid string, tenantID int64) (*ClientIdentityProvider, error)
	FindByClientAndProvider(clientID int64, identityProviderID int64) (*ClientIdentityProvider, error)
	FindByClientID(clientID int64) ([]ClientIdentityProvider, error)
	UnsetDefaultForClient(clientID int64, exceptID int64) error
	DeleteByUUIDAndTenantID(uuid string, tenantID int64) error
}

type clientIdentityProviderRepository struct {
	*BaseRepository[ClientIdentityProvider]
}

func NewClientIdentityProviderRepository(db *gorm.DB) ClientIdentityProviderRepository {
	return &clientIdentityProviderRepository{
		BaseRepository: database.NewBaseRepository[ClientIdentityProvider](db, "client_identity_provider_uuid", "client_identity_provider_id"),
	}
}

func (r *clientIdentityProviderRepository) WithTx(tx *gorm.DB) ClientIdentityProviderRepository {
	return &clientIdentityProviderRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *clientIdentityProviderRepository) FindByUUIDAndTenantID(uuid string, tenantID int64) (*ClientIdentityProvider, error) {
	var connection ClientIdentityProvider
	err := r.DB().
		Preload("IdentityProvider").
		Where("client_identity_provider_uuid = ? AND tenant_id = ?", uuid, tenantID).
		First(&connection).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &connection, nil
}

func (r *clientIdentityProviderRepository) FindByClientAndProvider(clientID int64, identityProviderID int64) (*ClientIdentityProvider, error) {
	var connection ClientIdentityProvider
	err := r.DB().
		Where("client_id = ? AND identity_provider_id = ?", clientID, identityProviderID).
		First(&connection).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &connection, nil
}

func (r *clientIdentityProviderRepository) FindByClientID(clientID int64) ([]ClientIdentityProvider, error) {
	var connections []ClientIdentityProvider
	err := r.DB().
		Preload("IdentityProvider").
		Where("client_id = ?", clientID).
		Order("display_order ASC").
		Find(&connections).Error

	if err != nil {
		return nil, err
	}

	return connections, nil
}

// UnsetDefaultForClient clears the default flag on every connection of a client
// except the one identified by exceptID. The partial unique index on
// client_identity_providers enforces at most one default per client, so callers
// must run this inside the same transaction before promoting a new default.
func (r *clientIdentityProviderRepository) UnsetDefaultForClient(clientID int64, exceptID int64) error {
	return r.DB().
		Model(&ClientIdentityProvider{}).
		Where("client_id = ? AND client_identity_provider_id <> ? AND is_default = ?", clientID, exceptID, true).
		Update("is_default", false).Error
}

func (r *clientIdentityProviderRepository) DeleteByUUIDAndTenantID(uuid string, tenantID int64) error {
	result := r.DB().Where("client_identity_provider_uuid = ? AND tenant_id = ?", uuid, tenantID).Delete(&ClientIdentityProvider{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
