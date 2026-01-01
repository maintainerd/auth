package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type AuthClientRepositoryGetFilter struct {
	TenantID           int64
	Name               *string
	DisplayName        *string
	ClientType         []string
	Status             []string
	IsDefault          *bool
	IsSystem           *bool
	IdentityProviderID *int64
	Page               int
	Limit              int
	SortBy             string
	SortOrder          string
}

type AuthClientRepository interface {
	BaseRepositoryMethods[model.AuthClient]
	WithTx(tx *gorm.DB) AuthClientRepository
	FindByUUIDAndTenantID(authClientUUID uuid.UUID, tenantID int64) (*model.AuthClient, error)
	FindByNameAndIdentityProvider(name string, identityProviderID int64, tenantID int64) (*model.AuthClient, error)
	FindByClientID(clientID string, tenantID int64) (*model.AuthClient, error)
	FindAllByTenantID(tenantID int64) ([]model.AuthClient, error)
	FindDefault() (*model.AuthClient, error)
	FindDefaultByTenantID(tenantID int64) (*model.AuthClient, error)
	FindPaginated(filter AuthClientRepositoryGetFilter) (*PaginationResult[model.AuthClient], error)
	SetStatusByUUID(authClientUUID uuid.UUID, tenantID int64, status string) error
	FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*model.AuthClient, error)
	DeleteByUUIDAndTenantID(authClientUUID uuid.UUID, tenantID int64) error
}

type authClientRepository struct {
	*BaseRepository[model.AuthClient]
	db *gorm.DB
}

func NewAuthClientRepository(db *gorm.DB) AuthClientRepository {
	return &authClientRepository{
		BaseRepository: NewBaseRepository[model.AuthClient](db, "auth_client_uuid", "auth_client_id"),
		db:             db,
	}
}

func (r *authClientRepository) WithTx(tx *gorm.DB) AuthClientRepository {
	return &authClientRepository{
		BaseRepository: NewBaseRepository[model.AuthClient](tx, "auth_client_uuid", "auth_client_id"),
		db:             tx,
	}
}

func (r *authClientRepository) FindByUUIDAndTenantID(authClientUUID uuid.UUID, tenantID int64) (*model.AuthClient, error) {
	var client model.AuthClient
	err := r.db.
		Preload("IdentityProvider").
		Preload("AuthClientURIs").
		Where("auth_client_uuid = ? AND tenant_id = ?", authClientUUID, tenantID).
		First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &client, nil
}

func (r *authClientRepository) FindByNameAndIdentityProvider(name string, identityProviderID int64, tenantID int64) (*model.AuthClient, error) {
	var client model.AuthClient
	err := r.db.Where("name = ? AND identity_provider_id = ? AND tenant_id = ?", name, identityProviderID, tenantID).First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &client, nil
}

func (r *authClientRepository) FindByClientID(clientID string, tenantID int64) (*model.AuthClient, error) {
	var client model.AuthClient
	err := r.db.Where("client_id = ? AND tenant_id = ?", clientID, tenantID).First(&client).Error
	return &client, err
}

func (r *authClientRepository) FindAllByTenantID(tenantID int64) ([]model.AuthClient, error) {
	var clients []model.AuthClient
	err := r.db.
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = auth_clients.identity_provider_id").
		Where("identity_providers.tenant_id = ?", tenantID).
		Find(&clients).Error
	return clients, err
}

func (r *authClientRepository) FindDefault() (*model.AuthClient, error) {
	var client model.AuthClient
	err := r.db.
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = auth_clients.identity_provider_id").
		Where("auth_clients.is_default = ? AND auth_clients.status = ?", true, "active").
		Where("auth_clients.client_type = ?", "traditional").
		Where("identity_providers.status = ?", "active").
		Preload("IdentityProvider").
		Preload("IdentityProvider.Tenant").
		First(&client).Error

	return &client, err
}

func (r *authClientRepository) FindDefaultByTenantID(tenantID int64) (*model.AuthClient, error) {
	var client model.AuthClient
	err := r.db.
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = auth_clients.identity_provider_id").
		Where("identity_providers.tenant_id = ? AND auth_clients.is_default = true AND auth_clients.status = ?", tenantID, "active").
		First(&client).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &client, nil
}

func (r *authClientRepository) FindPaginated(filter AuthClientRepositoryGetFilter) (*PaginationResult[model.AuthClient], error) {
	query := r.db.Model(&model.AuthClient{}).Where("tenant_id = ?", filter.TenantID)

	// Filters with LIKE
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.DisplayName != nil {
		query = query.Where("display_name ILIKE ?", "%"+*filter.DisplayName+"%")
	}

	// Filters with exact match
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
	}
	if len(filter.ClientType) > 0 {
		query = query.Where("client_type IN ?", filter.ClientType)
	}
	if filter.IdentityProviderID != nil {
		query = query.Where("identity_provider_id = ?", *filter.IdentityProviderID)
	}

	// Sorting
	orderBy := filter.SortBy + " " + filter.SortOrder
	query = query.Order(orderBy)

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination
	offset := (filter.Page - 1) * filter.Limit
	var clients []model.AuthClient
	if err := query.
		Preload("IdentityProvider").
		Preload("AuthClientURIs").
		Limit(filter.Limit).
		Offset(offset).
		Find(&clients).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[model.AuthClient]{
		Data:       clients,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *authClientRepository) SetStatusByUUID(authClientUUID uuid.UUID, tenantID int64, status string) error {
	return r.db.Model(&model.AuthClient{}).
		Where("auth_client_uuid = ? AND tenant_id = ?", authClientUUID, tenantID).
		Update("status", status).Error
}

func (r *authClientRepository) FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*model.AuthClient, error) {
	var client model.AuthClient

	err := r.db.
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = auth_clients.identity_provider_id").
		Where("auth_clients.client_id = ? AND identity_providers.identifier = ?", clientID, identityProviderIdentifier).
		Where("auth_clients.status = ? AND identity_providers.status = ?", "active", "active").
		Preload("IdentityProvider.Tenant").
		Preload("IdentityProvider").
		First(&client).Error

	return &client, err
}

func (r *authClientRepository) DeleteByUUIDAndTenantID(authClientUUID uuid.UUID, tenantID int64) error {
	result := r.db.Where("auth_client_uuid = ? AND tenant_id = ?", authClientUUID, tenantID).Delete(&model.AuthClient{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
