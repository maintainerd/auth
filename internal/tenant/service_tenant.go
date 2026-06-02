package tenant

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
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
	IsSystem    bool
	Metadata    datatypes.JSON
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
	GetSystem(ctx context.Context) (*TenantServiceDataResult, error)
	GetByIdentifier(ctx context.Context, identifier string) (*TenantServiceDataResult, error)
	Create(ctx context.Context, name string, displayName string, description string, status string, isPublic bool) (*TenantServiceDataResult, error)
	Update(ctx context.Context, tenantUUID uuid.UUID, name string, displayName string, description string, status string, isPublic bool) (*TenantServiceDataResult, error)
	SetStatusByUUID(ctx context.Context, tenantUUID uuid.UUID, status string) (*TenantServiceDataResult, error)
	SetActivePublicByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
	DeleteByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
}

type tenantService struct {
	tenantRepo TenantRepository
	uow        UnitOfWork
}

func NewTenantService(tenantRepo TenantRepository, uow UnitOfWork) TenantService {
	if uow == nil {
		uow = newDirectUnitOfWork(tenantRepo, nil)
	}
	return &tenantService{
		tenantRepo: tenantRepo,
		uow:        uow,
	}
}

func (s *tenantService) Get(ctx context.Context, filter TenantServiceGetFilter) (*TenantServiceGetResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenant.list")
	defer span.End()

	tenantFilter := TenantRepositoryGetFilter{
		Name:        filter.Name,
		DisplayName: filter.DisplayName,
		Description: filter.Description,
		Identifier:  filter.Identifier,
		Status:      filter.Status,
		IsPublic:    filter.IsPublic,
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

// GetSystem returns the unique system tenant (is_system = true).
// This is the root tenant used for no-client_id login and registration on port 8080.
func (s *tenantService) GetSystem(ctx context.Context) (*TenantServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenant.getSystem")
	defer span.End()

	tenant, err := s.tenantRepo.FindSystem()
	if err != nil || tenant == nil {
		if err != nil {
			span.RecordError(err)
		}
		span.SetStatus(codes.Error, "system tenant not found")
		return nil, apperror.NewNotFoundWithReason("system tenant not found")
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

func (s *tenantService) Create(ctx context.Context, name string, displayName string, description string, status string, isPublic bool) (*TenantServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenant.create")
	defer span.End()
	span.SetAttributes(attribute.String("tenant.name", name))

	var createdTenant *Tenant

	err := s.uow.Do(ctx, func(tx Transaction) error {
		txTenantRepo := tx.TenantRepository()
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
		newTenant := &Tenant{
			Name:        name,
			DisplayName: displayName,
			Description: description,
			Identifier:  identifier,
			Status:      status,
			IsPublic:    isPublic,
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

	var updatedTenant *Tenant

	err := s.uow.Do(ctx, func(tx Transaction) error {
		txTenantRepo := tx.TenantRepository()
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
	tenant.IsPublic = !tenant.IsPublic
	_, err = s.tenantRepo.CreateOrUpdate(tenant)
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

func (s *tenantService) DeleteByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenant.delete")
	defer span.End()
	span.SetAttributes(attribute.String("tenant.uuid", tenantUUID.String()))

	var result *TenantServiceDataResult

	err := s.uow.Do(ctx, func(tx Transaction) error {
		txTenantRepo := tx.TenantRepository()
		tenant, err := txTenantRepo.FindByUUID(tenantUUID)
		if err != nil {
			return err
		}
		if tenant == nil {
			return apperror.NewNotFound("tenant not found")
		}
		if tenant.IsSystem {
			return apperror.NewValidation("cannot delete system tenant")
		}

		result = toTenantServiceDataResult(tenant)
		id := tenant.TenantID

		if err := tx.DeleteTenantCascade(ctx, id); err != nil {
			return err
		}

		return txTenantRepo.DeleteByUUID(tenantUUID)
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete tenant failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

func toTenantServiceDataResult(tenant *Tenant) *TenantServiceDataResult {
	return &TenantServiceDataResult{
		TenantID:    tenant.TenantID,
		TenantUUID:  tenant.TenantUUID,
		Name:        tenant.Name,
		DisplayName: tenant.DisplayName,
		Description: tenant.Description,
		Identifier:  tenant.Identifier,
		Status:      tenant.Status,
		IsPublic:    tenant.IsPublic,
		IsSystem:    tenant.IsSystem,
		Metadata:    tenant.Metadata,
		CreatedAt:   tenant.CreatedAt,
		UpdatedAt:   tenant.UpdatedAt,
	}
}

// ValidateTenantAccess validates if an actor can access the target tenant.
// Rules:
//   - Actors with an identity in a system/default tenant can access any tenant
//   - Otherwise the actor may only access tenants they have an identity in
//   - The actor must have at least one identity
//
// The actor is supplied as a consumer-defined interface so this package does
// not depend on the user domain (see deps.go).
func ValidateTenantAccess(actor AccessActor, targetTenant *Tenant) error {
	if actor == nil {
		return apperror.NewValidation("actor user is nil")
	}
	if targetTenant == nil {
		return apperror.NewValidation("target tenant is nil")
	}
	return validateTenantAccessByID(actor, targetTenant.TenantID)
}

// ValidateTenantAccessByID validates tenant access using a tenant ID.
func ValidateTenantAccessByID(actor AccessActor, targetTenantID int64) error {
	if actor == nil {
		return apperror.NewValidation("actor user is nil")
	}
	return validateTenantAccessByID(actor, targetTenantID)
}

func validateTenantAccessByID(actor AccessActor, targetTenantID int64) error {
	identities := actor.AccessIdentities()

	// Actor must have at least one identity.
	if len(identities) == 0 {
		return apperror.NewValidation("actor user has no identities")
	}

	hasAccessToTargetTenant := false
	for _, identity := range identities {
		// An identity in a system/default tenant grants access to any tenant.
		if identity.TenantIsSystem {
			return nil
		}
		if identity.TenantID == targetTenantID {
			hasAccessToTargetTenant = true
		}
	}

	if hasAccessToTargetTenant {
		return nil
	}

	return apperror.NewForbidden("access denied: user does not have access to this tenant")
}
