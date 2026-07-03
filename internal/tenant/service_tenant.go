package tenant

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/event"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
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
	IsSystem    bool
	Metadata    datatypes.JSON
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TenantServiceGetFilter struct {
	Name        *string
	DisplayName *string
	Description *string
	Identifier  *string
	Status      []string
	IsSystem    *bool
	// TenantIDs, when non-empty, restricts results to these tenants. Used to
	// scope a regular user's listing to the tenant(s) they belong to.
	TenantIDs []int64
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
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
	Create(ctx context.Context, name string, displayName string, description string, status string) (*TenantServiceDataResult, error)
	Update(ctx context.Context, tenantUUID uuid.UUID, name string, displayName string, description string, status string) (*TenantServiceDataResult, error)
	SetStatusByUUID(ctx context.Context, tenantUUID uuid.UUID, status string) (*TenantServiceDataResult, error)
	DeleteByUUID(ctx context.Context, tenantUUID uuid.UUID, actorUserID int64) (*TenantServiceDataResult, error)
}

type tenantService struct {
	tenantRepo   TenantRepository
	uow          UnitOfWork
	eventService event.EventService
	seeder       TenantSeeder
}

// NewTenantService builds the tenant service. The optional seeder, when
// provided, runs the per-tenant baseline seed inside the create transaction so
// a newly created tenant comes up fully provisioned (roles, permissions,
// client, idp, branding, etc.). It is variadic so existing callers/tests that
// do not need seeding remain unchanged.
func NewTenantService(tenantRepo TenantRepository, uow UnitOfWork, eventService event.EventService, seeder ...TenantSeeder) TenantService {
	if uow == nil {
		uow = newDirectUnitOfWork(tenantRepo, nil)
	}
	s := &tenantService{
		tenantRepo:   tenantRepo,
		uow:          uow,
		eventService: eventService,
	}
	if len(seeder) > 0 {
		s.seeder = seeder[0]
	}
	return s
}

func (s *tenantService) Get(ctx context.Context, filter TenantServiceGetFilter) (*TenantServiceGetResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "tenant.list")
	defer span.End()

	tenantFilter := TenantRepositoryGetFilter(filter)

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

func (s *tenantService) Create(ctx context.Context, name string, displayName string, description string, status string) (*TenantServiceDataResult, error) {
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

		// A tenant becomes complete only after its first owner is assigned.
		// The baseline is seeded here, but ownership is a separate privileged
		// transition performed by tenant member management.
		newTenant := &Tenant{
			Name:        name,
			DisplayName: displayName,
			Description: description,
			Identifier:  identifier,
			Status:      status,
			IsCompleted: false,
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

		// Seed the per-tenant baseline (roles, permissions, client, idp,
		// branding, etc.) inside the same transaction so the new tenant is
		// fully provisioned or not created at all. Requires a transactional
		// unit of work; the direct (non-tx) UoW cannot seed safely.
		if s.seeder != nil {
			tx := tx.Tx()
			if tx == nil {
				return apperror.NewInternal("tenant seeding requires a transactional unit of work", nil)
			}
			if err := s.seeder.SeedTenant(ctx, tx, createdTenant.TenantID); err != nil {
				return err
			}
		}

		// Emit tenant.created inside the transaction
		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx.Tx(), event.NewIntegrationEvent(
				event.EventTypeTenantCreated, 1, createdTenant.TenantID,
			).SetSubject(&createdTenant.TenantUUID, "tenant")); emitErr != nil {
				return emitErr
			}
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

func (s *tenantService) Update(ctx context.Context, tenantUUID uuid.UUID, name string, displayName string, description string, status string) (*TenantServiceDataResult, error) {
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

		// Track changed fields
		var changed []string
		if tenant.Name != name {
			changed = append(changed, "name")
		}
		if tenant.DisplayName != displayName {
			changed = append(changed, "display_name")
		}
		if tenant.Description != description {
			changed = append(changed, "description")
		}
		if tenant.Status != status {
			changed = append(changed, "status")
		}

		// Update tenant
		tenant.Name = name
		tenant.DisplayName = displayName
		tenant.Description = description
		tenant.Status = status

		updatedTenant, err = txTenantRepo.CreateOrUpdate(tenant)
		if err != nil {
			return err
		}

		// Emit tenant.updated inside the transaction
		if s.eventService != nil && len(changed) > 0 {
			if _, emitErr := s.eventService.Emit(ctx, tx.Tx(), event.NewIntegrationEvent(
				event.EventTypeTenantUpdated, 1, updatedTenant.TenantID,
			).SetSubject(&updatedTenant.TenantUUID, "tenant").
				SetChangedFields(changed...)); emitErr != nil {
				return emitErr
			}
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

	var updatedTenant *Tenant
	err := s.uow.Do(ctx, func(tx Transaction) error {
		txTenantRepo := tx.TenantRepository()

		tenant, err := txTenantRepo.FindByUUID(tenantUUID)
		if err != nil {
			return err
		}
		if tenant == nil {
			return apperror.NewNotFound("tenant not found")
		}

		if err := txTenantRepo.SetStatusByUUID(tenantUUID, status); err != nil {
			return err
		}

		updatedTenant, err = txTenantRepo.FindByUUID(tenantUUID)
		if err != nil {
			return err
		}

		// Emit tenant.status_changed inside the transaction.
		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx.Tx(), event.NewIntegrationEvent(
				event.EventTypeTenantStatusChanged, 1, updatedTenant.TenantID,
			).SetSubject(&updatedTenant.TenantUUID, "tenant").
				SetChangedFields("status")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "set tenant status failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toTenantServiceDataResult(updatedTenant), nil
}

func (s *tenantService) DeleteByUUID(ctx context.Context, tenantUUID uuid.UUID, actorUserID int64) (*TenantServiceDataResult, error) {
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
		systemTenant, err := txTenantRepo.FindSystem()
		if err != nil {
			return apperror.NewInternal("failed to resolve system tenant", err)
		}
		if systemTenant == nil {
			return apperror.NewForbidden("system tenant membership is required to delete a tenant")
		}
		actorMembership, err := tx.TenantMemberRepository().FindByTenantAndUser(systemTenant.TenantID, actorUserID)
		if err != nil {
			return apperror.NewInternal("failed to verify system tenant membership", err)
		}
		if actorMembership == nil {
			return apperror.NewForbidden("only system tenant administrators can delete a tenant")
		}

		result = toTenantServiceDataResult(tenant)
		id := tenant.TenantID

		if err := tx.DeleteTenantCascade(ctx, id); err != nil {
			return err
		}

		if err := txTenantRepo.DeleteByUUID(tenantUUID); err != nil {
			return err
		}

		// Emit tenant.deleted inside the transaction
		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx.Tx(), event.NewIntegrationEvent(
				event.EventTypeTenantDeleted, 1, tenant.TenantID,
			).SetSubject(&tenant.TenantUUID, "tenant")); emitErr != nil {
				return emitErr
			}
		}

		return nil
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
