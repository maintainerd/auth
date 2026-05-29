package client

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/gorm"
)

type APIKeyRepositoryGetFilter struct {
	TenantID    int64
	Name        *string
	Description *string
	Status      *string
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type APIKeyRepository interface {
	BaseRepositoryMethods[APIKey]
	WithTx(tx *gorm.DB) APIKeyRepository
	FindByUUIDAndTenantID(uuid string, tenantID int64) (*APIKey, error)
	FindByKeyHash(keyHash string) (*APIKey, error)
	FindByKeyPrefix(keyPrefix string) (*APIKey, error)
	DeleteByUUIDAndTenantID(uuid string, tenantID int64) error
	FindPaginated(filter APIKeyRepositoryGetFilter) (*PaginationResult[APIKey], error)
}

type apiKeyRepository struct {
	*BaseRepository[APIKey]
}

func NewAPIKeyRepository(db *gorm.DB) APIKeyRepository {
	return &apiKeyRepository{
		BaseRepository: NewBaseRepository[APIKey](db, "api_key_uuid", "api_key_id"),
	}
}

func (r *apiKeyRepository) WithTx(tx *gorm.DB) APIKeyRepository {
	return &apiKeyRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *apiKeyRepository) FindByUUIDAndTenantID(uuid string, tenantID int64) (*APIKey, error) {
	var apiKey APIKey
	err := r.DB().Where("api_key_uuid = ? AND tenant_id = ?", uuid, tenantID).First(&apiKey).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &apiKey, nil
}

func (r *apiKeyRepository) FindByKeyHash(keyHash string) (*APIKey, error) {
	var apiKey APIKey
	if err := r.DB().Where("key_hash = ?", keyHash).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &apiKey, nil
}

func (r *apiKeyRepository) DeleteByUUIDAndTenantID(uuid string, tenantID int64) error {
	result := r.DB().Where("api_key_uuid = ? AND tenant_id = ?", uuid, tenantID).Delete(&APIKey{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *apiKeyRepository) FindByKeyPrefix(keyPrefix string) (*APIKey, error) {
	var apiKey APIKey
	if err := r.DB().Where("key_prefix = ?", keyPrefix).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &apiKey, nil
}

func (r *apiKeyRepository) FindPaginated(filter APIKeyRepositoryGetFilter) (*PaginationResult[APIKey], error) {
	query := r.DB().Model(&APIKey{})

	// Always filter by tenant
	query = query.Where("tenant_id = ?", filter.TenantID)

	// Apply filters
	if filter.Name != nil {
		query = query.Where("name ILIKE ?", "%"+*filter.Name+"%")
	}
	if filter.Description != nil {
		query = query.Where("description ILIKE ?", "%"+*filter.Description+"%")
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	// Sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination guards prevent division-by-zero and negative offsets
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit
	var apiKeys []APIKey
	if err := query.Limit(filter.Limit).Offset(offset).Find(&apiKeys).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[APIKey]{
		Data:       apiKeys,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

type APIKeyAPIRepository interface {
	BaseRepositoryMethods[APIKeyAPI]
	WithTx(tx *gorm.DB) APIKeyAPIRepository
	FindByAPIKeyAndAPI(apiKeyID int64, apiID int64) (*APIKeyAPI, error)
	FindByAPIKeyUUID(apiKeyUUID uuid.UUID) ([]APIKeyAPI, error)
	FindByAPIKeyUUIDPaginated(apiKeyUUID uuid.UUID, page, limit int, sortBy, sortOrder string) (*PaginationResult[APIKeyAPI], error)
	FindByAPIKeyUUIDAndAPIUUID(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) (*APIKeyAPI, error)
	RemoveByAPIKeyAndAPI(apiKeyID int64, apiID int64) error
	RemoveByAPIKeyUUIDAndAPIUUID(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) error
}

type apiKeyAPIRepository struct {
	*BaseRepository[APIKeyAPI]
}

func NewAPIKeyAPIRepository(db *gorm.DB) APIKeyAPIRepository {
	return &apiKeyAPIRepository{
		BaseRepository: NewBaseRepository[APIKeyAPI](db, "api_key_api_uuid", "api_key_api_id"),
	}
}

func (r *apiKeyAPIRepository) WithTx(tx *gorm.DB) APIKeyAPIRepository {
	return &apiKeyAPIRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *apiKeyAPIRepository) FindByAPIKeyAndAPI(apiKeyID int64, apiID int64) (*APIKeyAPI, error) {
	var apiKeyAPI APIKeyAPI
	if err := r.DB().Where("api_key_id = ? AND api_id = ?", apiKeyID, apiID).First(&apiKeyAPI).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &apiKeyAPI, nil
}

func (r *apiKeyAPIRepository) FindByAPIKeyUUID(apiKeyUUID uuid.UUID) ([]APIKeyAPI, error) {
	var apiKeyAPIs []APIKeyAPI
	err := r.DB().Joins("JOIN api_keys ON api_keys.api_key_id = api_key_apis.api_key_id").
		Where("api_keys.api_key_uuid = ?", apiKeyUUID).
		Preload("API").
		Preload("Permissions.Permission").
		Find(&apiKeyAPIs).Error

	if err != nil {
		return nil, err
	}

	return apiKeyAPIs, nil
}

func (r *apiKeyAPIRepository) FindByAPIKeyUUIDPaginated(apiKeyUUID uuid.UUID, page, limit int, sortBy, sortOrder string) (*PaginationResult[APIKeyAPI], error) {
	var apiKeyAPIs []APIKeyAPI
	var total int64

	// Base query
	query := r.DB().Joins("JOIN api_keys ON api_keys.api_key_id = api_key_apis.api_key_id").
		Where("api_keys.api_key_uuid = ?", apiKeyUUID).
		Preload("API").
		Preload("Permissions.Permission")

	// Count total records
	if err := query.Model(&APIKeyAPI{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply sorting
	if sortBy != "" {
		orderClause := sortBy
		if sortOrder == "desc" {
			orderClause += " DESC"
		} else {
			orderClause += " ASC"
		}
		query = query.Order(orderClause)
	} else {
		query = query.Order("api_key_apis.created_at DESC") // Default sorting
	}

	// Pagination guards prevent division-by-zero and negative offsets
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit
	if err := query.Limit(limit).Offset(offset).Find(&apiKeyAPIs).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &PaginationResult[APIKeyAPI]{
		Data:       apiKeyAPIs,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *apiKeyAPIRepository) FindByAPIKeyUUIDAndAPIUUID(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) (*APIKeyAPI, error) {
	var apiKeyAPI APIKeyAPI
	err := r.DB().Joins("JOIN api_keys ON api_keys.api_key_id = api_key_apis.api_key_id").
		Joins("JOIN apis ON apis.api_id = api_key_apis.api_id").
		Where("api_keys.api_key_uuid = ? AND apis.api_uuid = ?", apiKeyUUID, apiUUID).
		Preload("API").
		Preload("Permissions.Permission").
		First(&apiKeyAPI).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &apiKeyAPI, nil
}

func (r *apiKeyAPIRepository) RemoveByAPIKeyAndAPI(apiKeyID int64, apiID int64) error {
	return r.DB().Where("api_key_id = ? AND api_id = ?", apiKeyID, apiID).Delete(&APIKeyAPI{}).Error
}

func (r *apiKeyAPIRepository) RemoveByAPIKeyUUIDAndAPIUUID(apiKeyUUID uuid.UUID, apiUUID uuid.UUID) error {
	// First find the record to get its ID
	apiKeyAPI, err := r.FindByAPIKeyUUIDAndAPIUUID(apiKeyUUID, apiUUID)
	if err != nil {
		return err
	}
	if apiKeyAPI == nil {
		return errors.New("API key API relationship not found")
	}

	// Delete by ID (more reliable than complex JOINs in DELETE)
	return r.DB().Delete(&APIKeyAPI{}, apiKeyAPI.APIKeyAPIID).Error
}

type APIKeyPermissionRepository interface {
	BaseRepositoryMethods[APIKeyPermission]
	WithTx(tx *gorm.DB) APIKeyPermissionRepository
	FindByAPIKeyAPIAndPermission(apiKeyAPIID int64, permissionID int64) (*APIKeyPermission, error)
	RemoveByAPIKeyAPIAndPermission(apiKeyAPIID int64, permissionID int64) error
	FindByAPIKeyAPIID(apiKeyAPIID int64) ([]APIKeyPermission, error)
}

type apiKeyPermissionRepository struct {
	*BaseRepository[APIKeyPermission]
}

func NewAPIKeyPermissionRepository(db *gorm.DB) APIKeyPermissionRepository {
	return &apiKeyPermissionRepository{
		BaseRepository: NewBaseRepository[APIKeyPermission](db, "api_key_permission_uuid", "api_key_permission_id"),
	}
}

func (r *apiKeyPermissionRepository) WithTx(tx *gorm.DB) APIKeyPermissionRepository {
	return &apiKeyPermissionRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *apiKeyPermissionRepository) FindByAPIKeyAPIAndPermission(apiKeyAPIID int64, permissionID int64) (*APIKeyPermission, error) {
	var apiKeyPermission APIKeyPermission
	if err := r.DB().Where("api_key_api_id = ? AND permission_id = ?", apiKeyAPIID, permissionID).First(&apiKeyPermission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &apiKeyPermission, nil
}

func (r *apiKeyPermissionRepository) RemoveByAPIKeyAPIAndPermission(apiKeyAPIID int64, permissionID int64) error {
	return r.DB().Where("api_key_api_id = ? AND permission_id = ?", apiKeyAPIID, permissionID).Delete(&APIKeyPermission{}).Error
}

func (r *apiKeyPermissionRepository) FindByAPIKeyAPIID(apiKeyAPIID int64) ([]APIKeyPermission, error) {
	var apiKeyPermissions []APIKeyPermission
	if err := r.DB().Where("api_key_api_id = ?", apiKeyAPIID).Preload("Permission").Find(&apiKeyPermissions).Error; err != nil {
		return nil, err
	}
	return apiKeyPermissions, nil
}

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
	BaseRepositoryMethods[Client]
	WithTx(tx *gorm.DB) ClientRepository
	FindByUUIDAndTenantID(clientUUID uuid.UUID, tenantID int64) (*Client, error)
	FindByNameAndIdentityProvider(name string, identityProviderID int64, tenantID int64) (*Client, error)
	FindByNameAndTenantID(name string, tenantID int64) (*Client, error)
	FindByClientID(clientID string, tenantID int64) (*Client, error)
	FindAllByTenantID(tenantID int64) ([]Client, error)
	FindSystem() (*Client, error)
	FindDefaultByTenantID(tenantID int64) (*Client, error)
	FindPaginated(filter ClientRepositoryGetFilter) (*PaginationResult[Client], error)
	SetStatusByUUID(clientUUID uuid.UUID, tenantID int64, status string) error
	FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*Client, error)
	DeleteByUUIDAndTenantID(clientUUID uuid.UUID, tenantID int64) error
}

type clientRepository struct {
	*BaseRepository[Client]
}

func NewClientRepository(db *gorm.DB) ClientRepository {
	return &clientRepository{
		BaseRepository: NewBaseRepository[Client](db, "client_uuid", "client_id"),
	}
}

func (r *clientRepository) WithTx(tx *gorm.DB) ClientRepository {
	return &clientRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *clientRepository) FindByUUIDAndTenantID(clientUUID uuid.UUID, tenantID int64) (*Client, error) {
	var client Client
	err := r.DB().
		Preload("IdentityProvider").
		Preload("ClientURIs").
		Where("client_uuid = ? AND tenant_id = ?", clientUUID, tenantID).
		First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &client, nil
}

func (r *clientRepository) FindByNameAndIdentityProvider(name string, identityProviderID int64, tenantID int64) (*Client, error) {
	var client Client
	err := r.DB().Where("name = ? AND identity_provider_id = ? AND tenant_id = ?", name, identityProviderID, tenantID).First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &client, nil
}

func (r *clientRepository) FindByClientID(clientID string, tenantID int64) (*Client, error) {
	var client Client
	err := r.DB().Where("client_id = ? AND tenant_id = ?", clientID, tenantID).First(&client).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

func (r *clientRepository) FindAllByTenantID(tenantID int64) ([]Client, error) {
	var clients []Client
	err := r.DB().
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = clients.identity_provider_id").
		Where("identity_providers.tenant_id = ?", tenantID).
		Find(&clients).Error
	return clients, err
}

// FindSystem returns the active system client belonging to the system tenant
// (is_system = true on both tenant and client). The seeded system client is the
// auth-console SPA used to bootstrap the management surface on port 8080.
func (r *clientRepository) FindSystem() (*Client, error) {
	var client Client
	err := r.DB().
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = clients.identity_provider_id").
		Joins("JOIN tenants ON tenants.tenant_id = identity_providers.tenant_id").
		Where("clients.is_system = ? AND clients.status = ?", true, shared.StatusActive).
		Where("identity_providers.status = ?", shared.StatusActive).
		Where("tenants.is_system = ?", true).
		Preload("IdentityProvider").
		Preload("IdentityProvider.Tenant").
		First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

// FindByNameAndTenantID returns the client with the given name within the
// tenant, regardless of which identity provider it is attached to.
func (r *clientRepository) FindByNameAndTenantID(name string, tenantID int64) (*Client, error) {
	var client Client
	err := r.DB().
		Where("name = ? AND tenant_id = ?", name, tenantID).
		Preload("IdentityProvider").
		First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

func (r *clientRepository) FindDefaultByTenantID(tenantID int64) (*Client, error) {
	var client Client
	err := r.DB().
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = clients.identity_provider_id").
		Where("identity_providers.tenant_id = ? AND clients.is_default = true AND clients.status = ?", tenantID, shared.StatusActive).
		First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &client, nil
}

func (r *clientRepository) FindPaginated(filter ClientRepositoryGetFilter) (*PaginationResult[Client], error) {
	query := r.DB().Model(&Client{}).Where("tenant_id = ?", filter.TenantID)

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

	// Sorting — protected against SQL injection via allowlist
	query = query.Order(sanitizeOrder(filter.SortBy, filter.SortOrder, "created_at DESC"))

	// Count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination
	filter.Page, filter.Limit = normalizePagination(filter.Page, filter.Limit)
	offset := (filter.Page - 1) * filter.Limit
	var clients []Client
	if err := query.
		Preload("IdentityProvider").
		Preload("ClientURIs").
		Limit(filter.Limit).
		Offset(offset).
		Find(&clients).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))

	return &PaginationResult[Client]{
		Data:       clients,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *clientRepository) SetStatusByUUID(clientUUID uuid.UUID, tenantID int64, status string) error {
	return r.DB().Model(&Client{}).
		Where("client_uuid = ? AND tenant_id = ?", clientUUID, tenantID).
		Update("status", status).Error
}

func (r *clientRepository) FindByClientIDAndIdentityProvider(clientID, identityProviderIdentifier string) (*Client, error) {
	var client Client

	err := r.DB().
		Joins("JOIN identity_providers ON identity_providers.identity_provider_id = clients.identity_provider_id").
		Where("clients.client_id = ? AND identity_providers.identifier = ?", clientID, identityProviderIdentifier).
		Where("clients.status = ? AND identity_providers.status = ?", shared.StatusActive, shared.StatusActive).
		Preload("IdentityProvider.Tenant").
		Preload("IdentityProvider").
		First(&client).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

func (r *clientRepository) DeleteByUUIDAndTenantID(clientUUID uuid.UUID, tenantID int64) error {
	result := r.DB().Where("client_uuid = ? AND tenant_id = ?", clientUUID, tenantID).Delete(&Client{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

type ClientAPIRepository interface {
	BaseRepositoryMethods[ClientAPI]
	WithTx(tx *gorm.DB) ClientAPIRepository
	FindByClientAndAPI(clientID int64, apiID int64) (*ClientAPI, error)
	FindByClientUUID(clientUUID uuid.UUID) ([]ClientAPI, error)
	FindByClientUUIDAndAPIUUID(clientUUID uuid.UUID, apiUUID uuid.UUID) (*ClientAPI, error)
	RemoveByClientAndAPI(clientID int64, apiID int64) error
	RemoveByClientUUIDAndAPIUUID(clientUUID uuid.UUID, apiUUID uuid.UUID) error
}

type clientAPIRepository struct {
	*BaseRepository[ClientAPI]
}

func NewClientAPIRepository(db *gorm.DB) ClientAPIRepository {
	return &clientAPIRepository{
		BaseRepository: NewBaseRepository[ClientAPI](db, "client_api_uuid", "client_api_id"),
	}
}

func (r *clientAPIRepository) WithTx(tx *gorm.DB) ClientAPIRepository {
	return &clientAPIRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *clientAPIRepository) FindByClientAndAPI(clientID int64, apiID int64) (*ClientAPI, error) {
	var clientAPI ClientAPI
	err := r.DB().Where("client_id = ? AND api_id = ?", clientID, apiID).First(&clientAPI).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &clientAPI, err
}

func (r *clientAPIRepository) FindByClientUUID(clientUUID uuid.UUID) ([]ClientAPI, error) {
	var clientAPIs []ClientAPI
	err := r.DB().Joins("JOIN clients ON clients.client_id = client_apis.client_id").
		Where("clients.client_uuid = ?", clientUUID).
		Preload("API").
		Preload("Permissions.Permission").
		Find(&clientAPIs).Error

	if err != nil {
		return nil, err
	}

	return clientAPIs, nil
}

func (r *clientAPIRepository) FindByClientUUIDAndAPIUUID(clientUUID uuid.UUID, apiUUID uuid.UUID) (*ClientAPI, error) {
	var clientAPI ClientAPI
	err := r.DB().Joins("JOIN clients ON clients.client_id = client_apis.client_id").
		Joins("JOIN apis ON apis.api_id = client_apis.api_id").
		Where("clients.client_uuid = ? AND apis.api_uuid = ?", clientUUID, apiUUID).
		Preload("API").
		Preload("Permissions.Permission").
		First(&clientAPI).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &clientAPI, nil
}

func (r *clientAPIRepository) RemoveByClientAndAPI(clientID int64, apiID int64) error {
	return r.DB().
		Where("client_id = ? AND api_id = ?", clientID, apiID).
		Unscoped().Delete(&ClientAPI{}).Error
}

func (r *clientAPIRepository) RemoveByClientUUIDAndAPIUUID(clientUUID uuid.UUID, apiUUID uuid.UUID) error {
	// First, find the client_api record to get the IDs
	var clientAPI ClientAPI
	err := r.DB().Joins("JOIN clients ON clients.client_id = client_apis.client_id").
		Joins("JOIN apis ON apis.api_id = client_apis.api_id").
		Where("clients.client_uuid = ? AND apis.api_uuid = ?", clientUUID, apiUUID).
		First(&clientAPI).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // Already deleted or doesn't exist
		}
		return err
	}

	// Now delete using the primary key
	return r.DB().Unscoped().Delete(&clientAPI).Error
}

type ClientPermissionRepository interface {
	BaseRepositoryMethods[ClientPermission]
	WithTx(tx *gorm.DB) ClientPermissionRepository
	FindByClientAPIAndPermission(clientAPIID int64, permissionID int64) (*ClientPermission, error)
	RemoveByClientAPIAndPermission(clientAPIID int64, permissionID int64) error
	FindByClientAPIID(clientAPIID int64) ([]ClientPermission, error)
}

type clientPermissionRepository struct {
	*BaseRepository[ClientPermission]
}

func NewClientPermissionRepository(db *gorm.DB) ClientPermissionRepository {
	return &clientPermissionRepository{
		BaseRepository: NewBaseRepository[ClientPermission](db, "client_permission_uuid", "client_permission_id"),
	}
}

func (r *clientPermissionRepository) WithTx(tx *gorm.DB) ClientPermissionRepository {
	return &clientPermissionRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *clientPermissionRepository) FindByClientAPIAndPermission(clientAPIID int64, permissionID int64) (*ClientPermission, error) {
	var clientPermission ClientPermission
	err := r.DB().Where("client_api_id = ? AND permission_id = ?", clientAPIID, permissionID).First(&clientPermission).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &clientPermission, err
}

func (r *clientPermissionRepository) RemoveByClientAPIAndPermission(clientAPIID int64, permissionID int64) error {
	return r.DB().
		Where("client_api_id = ? AND permission_id = ?", clientAPIID, permissionID).
		Unscoped().Delete(&ClientPermission{}).Error
}

func (r *clientPermissionRepository) FindByClientAPIID(clientAPIID int64) ([]ClientPermission, error) {
	var permissions []ClientPermission
	err := r.DB().Where("client_api_id = ?", clientAPIID).
		Preload("Permission").
		Find(&permissions).Error

	if err != nil {
		return nil, err
	}

	return permissions, nil
}

type ClientURIRepository interface {
	BaseRepositoryMethods[ClientURI]
	WithTx(tx *gorm.DB) ClientURIRepository
	FindByUUIDAndTenantID(uuid string, tenantID int64) (*ClientURI, error)
	FindByURIAndType(uri string, uriType string, clientID int64, tenantID int64) (*ClientURI, error)
	FindByClientIDAndType(clientID int64, uriType string, tenantID int64) ([]ClientURI, error)
	DeleteByUUIDAndTenantID(uuid string, tenantID int64) error
}

type clientURIRepository struct {
	*BaseRepository[ClientURI]
}

func NewClientURIRepository(db *gorm.DB) ClientURIRepository {
	return &clientURIRepository{
		BaseRepository: NewBaseRepository[ClientURI](db, "client_uri_uuid", "client_uri_id"),
	}
}

func (r *clientURIRepository) WithTx(tx *gorm.DB) ClientURIRepository {
	return &clientURIRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *clientURIRepository) FindByUUIDAndTenantID(uuid string, tenantID int64) (*ClientURI, error) {
	var clientURI ClientURI
	err := r.DB().Where("client_uri_uuid = ? AND tenant_id = ?", uuid, tenantID).First(&clientURI).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &clientURI, nil
}

func (r *clientURIRepository) FindByURIAndType(uri string, uriType string, clientID int64, tenantID int64) (*ClientURI, error) {
	var clientURI ClientURI
	err := r.DB().Where("uri = ? AND type = ? AND client_id = ? AND tenant_id = ?", uri, uriType, clientID, tenantID).First(&clientURI).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &clientURI, nil
}

func (r *clientURIRepository) FindByClientIDAndType(clientID int64, uriType string, tenantID int64) ([]ClientURI, error) {
	var clientURIs []ClientURI
	err := r.DB().Where("client_id = ? AND type = ? AND tenant_id = ?", clientID, uriType, tenantID).Find(&clientURIs).Error

	if err != nil {
		return nil, err
	}

	return clientURIs, nil
}

func (r *clientURIRepository) DeleteByUUIDAndTenantID(uuid string, tenantID int64) error {
	result := r.DB().Where("client_uri_uuid = ? AND tenant_id = ?", uuid, tenantID).Delete(&ClientURI{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
