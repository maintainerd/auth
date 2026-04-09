package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/crypto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

type TenantServiceDataResult struct {
	TenantID    int64
	TenantUUID  uuid.UUID
	Name        string
	DisplayName string
	Description string
	Identifier  string
	Status      string
	IsPublic    bool
	IsDefault   bool
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TenantServiceGetFilter struct {
	Name        *string
	DisplayName *string
	Description *string
	APIType     *string
	Identifier  *string
	Status      []string
	IsPublic    *bool
	IsDefault   *bool
	IsSystem    *bool
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
	Get(ctx context.Context, filter TenantServiceGetFilter) (*TenantServiceGetResult, error)
	GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
	GetDefault(ctx context.Context) (*TenantServiceDataResult, error)
	GetByIdentifier(ctx context.Context, identifier string) (*TenantServiceDataResult, error)
	Create(ctx context.Context, name string, displayName string, description string, status string, isPublic bool, isDefault bool) (*TenantServiceDataResult, error)
	Update(ctx context.Context, tenantUUID uuid.UUID, name string, displayName string, description string, status string, isPublic bool) (*TenantServiceDataResult, error)
	SetStatusByUUID(ctx context.Context, tenantUUID uuid.UUID, status string) (*TenantServiceDataResult, error)
	SetActivePublicByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
	SetDefaultStatusByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
	DeleteByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
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

func (s *tenantService) Get(ctx context.Context, filter TenantServiceGetFilter) (*TenantServiceGetResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenant.list")
	defer span.End()

	tenantFilter := repository.TenantRepositoryGetFilter{
		Name:        filter.Name,
		DisplayName: filter.DisplayName,
		Description: filter.Description,
		Identifier:  filter.Identifier,
		Status:      filter.Status,
		IsPublic:    filter.IsPublic,
		IsDefault:   filter.IsDefault,
		IsSystem:    filter.IsSystem,
		Page:        filter.Page,
		Limit:       filter.Limit,
		SortBy:      filter.SortBy,
		SortOrder:   filter.SortOrder,
	}

	result, err := s.tenantRepo.FindPaginated(tenantFilter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list tenants failed")
		return nil, err
	}

	resData := make([]TenantServiceDataResult, len(result.Data))
	for i, r := range result.Data {
		resData[i] = *toTenantServiceDataResult(&r)
	}

	span.SetStatus(codes.Ok, "")
	return &TenantServiceGetResult{
		Data:       resData,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *tenantService) GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenant.getByUUID")
	defer span.End()
	span.SetAttributes(attribute.String("tenant.uuid", tenantUUID.String()))

	tenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil || tenant == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "tenant not found")
		return nil, apperror.NewNotFound("tenant not found")
	}

	span.SetStatus(codes.Ok, "")
	return toTenantServiceDataResult(tenant), nil
}

func (s *tenantService) GetDefault(ctx context.Context) (*TenantServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenant.getDefault")
	defer span.End()

	tenant, err := s.tenantRepo.FindDefault()
	if err != nil || tenant == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "default tenant not found")
		return nil, apperror.NewNotFoundWithReason("default tenant not found")
	}

	span.SetStatus(codes.Ok, "")
	return toTenantServiceDataResult(tenant), nil
}

func (s *tenantService) GetByIdentifier(ctx context.Context, identifier string) (*TenantServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenant.getByIdentifier")
	defer span.End()
	span.SetAttributes(attribute.String("tenant.identifier", identifier))

	tenant, err := s.tenantRepo.FindByIdentifier(identifier)
	if err != nil || tenant == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "tenant not found")
		return nil, apperror.NewNotFound("tenant not found")
	}

	span.SetStatus(codes.Ok, "")
	return toTenantServiceDataResult(tenant), nil
}

func (s *tenantService) Create(ctx context.Context, name string, displayName string, description string, status string, isPublic bool, isDefault bool) (*TenantServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenant.create")
	defer span.End()
	span.SetAttributes(attribute.String("tenant.name", name))

	var createdTenant *model.Tenant

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txTenantRepo := s.tenantRepo.WithTx(tx)

		// Check if tenant already exists
		existingTenant, err := txTenantRepo.FindByName(name)
		if err != nil {
			return err
		}
		if existingTenant != nil {
			return apperror.NewConflict(name + " tenant already exists")
		}

		// Generate identifier
		identifier, err := crypto.GenerateIdentifier(12)
		if err != nil {
			return err
		}

		// Create tenant
		newTenant := &model.Tenant{
			Name:        name,
			DisplayName: displayName,
			Description: description,
			Identifier:  identifier,
			Status:      status,
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "create tenant failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toTenantServiceDataResult(createdTenant), nil
}

func (s *tenantService) Update(ctx context.Context, tenantUUID uuid.UUID, name string, displayName string, description string, status string, isPublic bool) (*TenantServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenant.update")
	defer span.End()
	span.SetAttributes(attribute.String("tenant.uuid", tenantUUID.String()))

	var updatedTenant *model.Tenant

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txTenantRepo := s.tenantRepo.WithTx(tx)

		// Find existing tenant
		tenant, err := txTenantRepo.FindByUUID(tenantUUID)
		if err != nil {
			return err
		}
		if tenant == nil {
			return apperror.NewNotFound("tenant not found")
		}

		// Check if tenant name is taken by another tenant
		if tenant.Name != name {
			existingTenant, err := txTenantRepo.FindByName(name)
			if err != nil {
				return err
			}
			if existingTenant != nil {
				return apperror.NewConflict(name + " tenant already exists")
			}
		}

		// Update tenant
		tenant.Name = name
		tenant.DisplayName = displayName
		tenant.Description = description
		tenant.Status = status
		tenant.IsPublic = isPublic

		updatedTenant, err = txTenantRepo.CreateOrUpdate(tenant)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update tenant failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toTenantServiceDataResult(updatedTenant), nil
}

func (s *tenantService) SetStatusByUUID(ctx context.Context, tenantUUID uuid.UUID, status string) (*TenantServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenant.setStatus")
	defer span.End()
	span.SetAttributes(attribute.String("tenant.uuid", tenantUUID.String()), attribute.String("tenant.status", status))

	tenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil || tenant == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "tenant not found")
		return nil, apperror.NewNotFound("tenant not found")
	}

	err = s.tenantRepo.SetStatusByUUID(tenantUUID, status)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "set tenant status failed")
		return nil, err
	}

	// Fetch updated tenant
	updatedTenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get updated tenant failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toTenantServiceDataResult(updatedTenant), nil
}

func (s *tenantService) SetActivePublicByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenant.setActivePublic")
	defer span.End()
	span.SetAttributes(attribute.String("tenant.uuid", tenantUUID.String()))

	tenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil || tenant == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "tenant not found")
		return nil, apperror.NewNotFound("tenant not found")
	}

	// Toggle public status
	err = s.db.Model(&model.Tenant{}).Where("tenant_uuid = ?", tenantUUID).Update("is_public", !tenant.IsPublic).Error
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "set active public failed")
		return nil, err
	}

	// Fetch updated tenant
	updatedTenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get updated tenant failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toTenantServiceDataResult(updatedTenant), nil
}

func (s *tenantService) SetDefaultStatusByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenant.setDefaultStatus")
	defer span.End()
	span.SetAttributes(attribute.String("tenant.uuid", tenantUUID.String()))

	tenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil || tenant == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "tenant not found")
		return nil, apperror.NewNotFound("tenant not found")
	}

	err = s.tenantRepo.SetDefaultStatusByUUID(tenantUUID, !tenant.IsDefault)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "set default status failed")
		return nil, err
	}

	// Fetch updated tenant
	updatedTenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get updated tenant failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toTenantServiceDataResult(updatedTenant), nil
}

func (s *tenantService) DeleteByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenant.delete")
	defer span.End()
	span.SetAttributes(attribute.String("tenant.uuid", tenantUUID.String()))

	tenant, err := s.tenantRepo.FindByUUID(tenantUUID)
	if err != nil || tenant == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "tenant not found")
		return nil, apperror.NewNotFound("tenant not found")
	}

	// Prevent deletion of system tenants
	if tenant.IsSystem {
		span.SetStatus(codes.Error, "cannot delete system tenant")
		return nil, apperror.NewValidation("cannot delete system tenant")
	}

	// Prevent deletion of default tenants
	if tenant.IsDefault {
		span.SetStatus(codes.Error, "cannot delete default tenant")
		return nil, apperror.NewValidation("cannot delete default tenant")
	}

	result := toTenantServiceDataResult(tenant)

	err = s.tenantRepo.DeleteByUUID(tenantUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete tenant failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

func toTenantServiceDataResult(tenant *model.Tenant) *TenantServiceDataResult {
	return &TenantServiceDataResult{
		TenantID:    tenant.TenantID,
		TenantUUID:  tenant.TenantUUID,
		Name:        tenant.Name,
		DisplayName: tenant.DisplayName,
		Description: tenant.Description,
		Identifier:  tenant.Identifier,
		Status:      tenant.Status,
		IsPublic:    tenant.IsPublic,
		IsDefault:   tenant.IsDefault,
		IsSystem:    tenant.IsSystem,
		CreatedAt:   tenant.CreatedAt,
		UpdatedAt:   tenant.UpdatedAt,
	}
}
