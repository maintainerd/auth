package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"gorm.io/gorm"
)

type TenantServiceDataResult struct {
	TenantUUID  uuid.UUID
	Name        string
	Description string
	Identifier  string
	IsActive    bool
	IsPublic    bool
	IsDefault   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TenantServiceGetFilter struct {
	Name        *string
	Description *string
	APIType     *string
	Identifier  *string
	IsActive    *bool
	IsPublic    *bool
	IsDefault   *bool
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type TenantServiceGetResult struct {
	Data       []TenantServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type TenantService interface {
	Get(filter TenantServiceGetFilter) (*TenantServiceGetResult, error)
	GetByUUID(tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
	GetDefault() (*TenantServiceDataResult, error)
	GetByIdentifier(identifier string) (*TenantServiceDataResult, error)
	Create(name string, description string, isActive bool, isPublic bool, isDefault bool) (*TenantServiceDataResult, error)
	Update(tenantUUID uuid.UUID, name string, description string, isActive bool, isPublic bool, isDefault bool) (*TenantServiceDataResult, error)
	SetActiveStatusByUUID(tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
	SetActivePublicByUUID(tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
	SetDefaultStatusByUUID(tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
	DeleteByUUID(tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
}

type tenantService struct {
	db         *gorm.DB
	tenantRepo repository.TenantRepository
}

func NewTenantService(db *gorm.DB, tenantRepo repository.TenantRepository) TenantService {
	return &tenantService{
		db:         db,
		tenantRepo: tenantRepo,
	}
}

func (s *tenantService) Get(filter TenantServiceGetFilter) (*TenantServiceGetResult, error) {
	tenantFilter := repository.TenantRepositoryGetFilter{
		Name:        filter.Name,
		Description: filter.Description,
		Identifier:  filter.Identifier,
		IsActive:    filter.IsActive,
		IsPublic:    filter.IsPublic,
		IsDefault:   filter.IsDefault,
		Page:        filter.Page,
		Limit:       filter.Limit,
		SortBy:      filter.SortBy,
		SortOrder:   filter.SortOrder,
	}

	result, err := s.tenantRepo.FindPaginated(tenantFilter)
	if err != nil {
		return nil, err
	}

	resData := make([]TenantServiceDataResult, len(result.Data))
	for i, r := range result.Data {
		resData[i] = *toTenantServiceDataResult(&r)
	}

	return &TenantServiceGetResult{
		Data:       resData,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *tenantService) GetByUUID(tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
	tenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil || tenant == nil {
		return nil, errors.New("tenant not found")
	}

	return toTenantServiceDataResult(tenant), nil
}

func (s *tenantService) GetDefault() (*TenantServiceDataResult, error) {
	tenant, err := s.tenantRepo.FindDefault()
	if err != nil || tenant == nil {
		return nil, errors.New("default tenant not found")
	}

	return toTenantServiceDataResult(tenant), nil
}

func (s *tenantService) GetByIdentifier(identifier string) (*TenantServiceDataResult, error) {
	tenant, err := s.tenantRepo.FindByIdentifier(identifier)
	if err != nil || tenant == nil {
		return nil, errors.New("tenant not found")
	}

	return toTenantServiceDataResult(tenant), nil
}

func (s *tenantService) Create(name string, description string, isActive bool, isPublic bool, isDefault bool) (*TenantServiceDataResult, error) {
	var createdTenant *model.Tenant

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txTenantRepo := s.tenantRepo.WithTx(tx)

		// Check if tenant already exists
		existingTenant, err := txTenantRepo.FindByName(name)
		if err != nil {
			return err
		}
		if existingTenant != nil {
			return errors.New(name + " tenant already exists")
		}

		// Generate identifier
		identifier := util.GenerateIdentifier(12)

		// Create tenant
		newTenant := &model.Tenant{
			Name:        name,
			Description: description,
			Identifier:  identifier,
			IsActive:    isActive,
			IsPublic:    isPublic,
			IsDefault:   isDefault,
		}

		_, err = txTenantRepo.CreateOrUpdate(newTenant)
		if err != nil {
			return err
		}

		// Fetch Tenant with relationships preloaded
		createdTenant, err = txTenantRepo.FindByUUID(newTenant.TenantUUID)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toTenantServiceDataResult(createdTenant), nil
}

func (s *tenantService) Update(tenantUUID uuid.UUID, name string, description string, isActive bool, isPublic bool, isDefault bool) (*TenantServiceDataResult, error) {
	var updatedTenant *model.Tenant

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txTenantRepo := s.tenantRepo.WithTx(tx)

		// Find existing tenant
		tenant, err := txTenantRepo.FindByUUID(tenantUUID)
		if err != nil {
			return err
		}
		if tenant == nil {
			return errors.New("tenant not found")
		}

		// Check if tenant name is taken by another tenant
		if tenant.Name != name {
			existingTenant, err := txTenantRepo.FindByName(name)
			if err != nil {
				return err
			}
			if existingTenant != nil {
				return errors.New(name + " tenant already exists")
			}
		}

		// Update tenant
		tenant.Name = name
		tenant.Description = description
		tenant.IsActive = isActive
		tenant.IsPublic = isPublic
		tenant.IsDefault = isDefault

		updatedTenant, err = txTenantRepo.CreateOrUpdate(tenant)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toTenantServiceDataResult(updatedTenant), nil
}

func (s *tenantService) SetActiveStatusByUUID(tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
	tenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil || tenant == nil {
		return nil, errors.New("tenant not found")
	}

	err = s.tenantRepo.SetActiveStatusByUUID(tenantUUID, !tenant.IsActive)
	if err != nil {
		return nil, err
	}

	// Fetch updated tenant
	updatedTenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil {
		return nil, err
	}

	return toTenantServiceDataResult(updatedTenant), nil
}

func (s *tenantService) SetActivePublicByUUID(tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
	tenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil || tenant == nil {
		return nil, errors.New("tenant not found")
	}

	// Toggle public status
	err = s.db.Model(&model.Tenant{}).Where("tenant_uuid = ?", tenantUUID).Update("is_public", !tenant.IsPublic).Error
	if err != nil {
		return nil, err
	}

	// Fetch updated tenant
	updatedTenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil {
		return nil, err
	}

	return toTenantServiceDataResult(updatedTenant), nil
}

func (s *tenantService) SetDefaultStatusByUUID(tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
	tenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil || tenant == nil {
		return nil, errors.New("tenant not found")
	}

	err = s.tenantRepo.SetDefaultStatusByUUID(tenantUUID, !tenant.IsDefault)
	if err != nil {
		return nil, err
	}

	// Fetch updated tenant
	updatedTenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil {
		return nil, err
	}

	return toTenantServiceDataResult(updatedTenant), nil
}

func (s *tenantService) DeleteByUUID(tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
	tenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil || tenant == nil {
		return nil, errors.New("tenant not found")
	}

	result := toTenantServiceDataResult(tenant)

	err = s.tenantRepo.DeleteByUUID(tenantUUID)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func toTenantServiceDataResult(tenant *model.Tenant) *TenantServiceDataResult {
	return &TenantServiceDataResult{
		TenantUUID:  tenant.TenantUUID,
		Name:        tenant.Name,
		Description: tenant.Description,
		Identifier:  tenant.Identifier,
		IsActive:    tenant.IsActive,
		IsPublic:    tenant.IsPublic,
		IsDefault:   tenant.IsDefault,
		CreatedAt:   tenant.CreatedAt,
		UpdatedAt:   tenant.UpdatedAt,
	}
}
