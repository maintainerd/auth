package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type ClientRepositoryGetFilter struct {
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

type ClientRepository interface {
	BaseRepositoryMethods[model.Client]
	WithTx(tx *gorm.DB) ClientRepository
	FindByUUIDAndTenantID(ClientUUID uuid.UUID, tenantID int64) (*model.Client, error)
	FindByNameAndIdentityProvider(name string, identityProviderID int64, tenantID int64) (*model.Client, error)
	FindByClientID(clientID string, tenantID int64) (*model.Client, error)
	FindAllByTenantID(tenantID int64) ([]model.Client, error)
	FindDefault() (*model.Client, error)
	FindDefaultByTenantID(tenantID int64) (*model.Client, error)
	FindPaginated(filter ClientRepositoryGetFilter) (*PaginationResult[model.Client], error)
	SetStatusByUUID(ClientUUID uuid.UUID, tenantID int64, status string) error
	FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*model.Client, error)
	DeleteByUUIDAndTenantID(ClientUUID uuid.UUID, tenantID int64) error
}

type clientRepository struct {
	*BaseRepository[model.Client]
	db *gorm.DB
}

func NewClientRepository(db *gorm.DB) ClientRepository {
	return &clientRepository{
		BaseRepository: NewBaseRepository[model.Client](db, "client_uuid", "client_id"),
		db:             db,
	}
}

func (r *clientRepository) WithTx(tx *gorm.DB) ClientRepository {
	return &clientRepository{
		BaseRepository: NewBaseRepository[model.Client](tx, "client_uuid", "client_id"),
		db:             tx,
	}
}

func (r *clientRepository) FindByUUIDAndTenantID(ClientUUID uuid.UUID, tenantID int64) (*model.Client, error) {
	var client model.Client
	err := r.db.
		Preload("IdentityProvider").
		Preload("ClientURIs").
		Where("client_uuid = ? AND tenant_id = ?", ClientUUID, tenantID).
		First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &client, nil
}

func (r *clientRepository) FindByNameAndIdentityProvider(name string, identityProviderID int64, tenantID int64) (*model.Client, error) {
	var client model.Client
	err := r.db.Where("name = ? AND identity_provider_id = ? AND tenant_id = ?", name, identityProviderID, tenantID).First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &client, nil
}

func (r *clientRepository) FindByClientID(clientID string, tenantID int64) (*model.Client, error) {
	var client model.Client
	err := r.db.Where("client_id = ? AND tenant_id = ?", clientID, tenantID).First(&client).Error
	return &client, err
}

func (r *clientRepository) FindAllByTenantID(tenantID int64) ([]model.Client, error) {
	var clients []model.Client
	err := r.db.
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = clients.identity_provider_id").
		Where("identity_providers.tenant_id = ?", tenantID).
		Find(&clients).Error
	return clients, err
}

func (r *clientRepository) FindDefault() (*model.Client, error) {
	var client model.Client
	err := r.db.
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = clients.identity_provider_id").
		Where("clients.is_default = ? AND clients.status = ?", true, "active").
		Where("clients.client_type = ?", "traditional").
		Where("identity_providers.status = ?", "active").
		Preload("IdentityProvider").
		Preload("IdentityProvider.Tenant").
		First(&client).Error

	return &client, err
}

func (r *clientRepository) FindDefaultByTenantID(tenantID int64) (*model.Client, error) {
	var client model.Client
	err := r.db.
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = clients.identity_provider_id").
		Where("identity_providers.tenant_id = ? AND clients.is_default = true AND clients.status = ?", tenantID, "active").
		First(&client).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &client, nil
}

func (r *clientRepository) FindPaginated(filter ClientRepositoryGetFilter) (*PaginationResult[model.Client], error) {
	query := r.db.Model(&model.Client{}).Where("tenant_id = ?", filter.TenantID)

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
	var clients []model.Client
	if err := query.
		Preload("IdentityProvider").
		Preload("ClientURIs").
		Limit(filter.Limit).
		Offset(offset).
		Find(&clients).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[model.Client]{
		Data:       clients,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *clientRepository) SetStatusByUUID(ClientUUID uuid.UUID, tenantID int64, status string) error {
	return r.db.Model(&model.Client{}).
		Where("client_uuid = ? AND tenant_id = ?", ClientUUID, tenantID).
		Update("status", status).Error
}

func (r *clientRepository) FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*model.Client, error) {
	var client model.Client

	err := r.db.
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = clients.identity_provider_id").
		Where("clients.client_id = ? AND identity_providers.identifier = ?", clientID, identityProviderIdentifier).
		Where("clients.status = ? AND identity_providers.status = ?", "active", "active").
		Preload("IdentityProvider.Tenant").
		Preload("IdentityProvider").
		First(&client).Error

	return &client, err
}

func (r *clientRepository) DeleteByUUIDAndTenantID(ClientUUID uuid.UUID, tenantID int64) error {
	result := r.db.Where("client_uuid = ? AND tenant_id = ?", ClientUUID, tenantID).Delete(&model.Client{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
