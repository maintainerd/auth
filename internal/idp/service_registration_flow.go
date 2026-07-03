package idp

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

type RegistrationFlowServiceDataResult struct {
	RegistrationFlowUUID uuid.UUID
	Name                 string
	Description          string
	Identifier           string
	Status               string
	ClientUUID           uuid.UUID
	VerificationRequired bool
	RequiredFields       string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type RegistrationFlowServiceListResult struct {
	Data       []RegistrationFlowServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type RegistrationFlowRoleServiceDataResult struct {
	RegistrationFlowRoleUUID uuid.UUID
	RegistrationFlowUUID     uuid.UUID
	RoleUUID                 uuid.UUID
	RoleName                 string
	RoleDescription          string
	RoleStatus               string
	RoleIsDefault            bool
	RoleIsSystem             bool
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type RegistrationFlowRoleServiceListResult struct {
	Data       []RegistrationFlowRoleServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type RegistrationFlowService interface {
	GetAll(ctx context.Context, tenantID int64, name, identifier *string, status []string, ClientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*RegistrationFlowServiceListResult, error)
	GetByUUID(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64) (*RegistrationFlowServiceDataResult, error)
	Create(ctx context.Context, tenantID int64, name, description, status string, ClientUUID uuid.UUID, identifier *string, roleUUIDs []uuid.UUID, verificationRequired bool, requiredFields string) (*RegistrationFlowServiceDataResult, error)
	Update(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, name, description, status string, roleUUIDs []uuid.UUID, verificationRequired bool, requiredFields string) (*RegistrationFlowServiceDataResult, error)
	UpdateStatus(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, status string) (*RegistrationFlowServiceDataResult, error)
	Delete(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64) (*RegistrationFlowServiceDataResult, error)
	AssignRoles(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, roleUUIDs []uuid.UUID) ([]RegistrationFlowRoleServiceDataResult, error)
	GetRoles(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, page, limit int) (*RegistrationFlowRoleServiceListResult, error)
	RemoveRole(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, roleUUID uuid.UUID) error
}

type registrationFlowService struct {
	db                       *gorm.DB
	registrationFlowRepo     RegistrationFlowRepository
	registrationFlowRoleRepo RegistrationFlowRoleRepository
	roleRepo                 RoleRepository
	clientRepo               ClientRepository
}

func NewRegistrationFlowService(
	db *gorm.DB,
	registrationFlowRepo RegistrationFlowRepository,
	registrationFlowRoleRepo RegistrationFlowRoleRepository,
	roleRepo RoleRepository,
	clientRepo ClientRepository,
) RegistrationFlowService {
	return &registrationFlowService{
		db:                       db,
		registrationFlowRepo:     registrationFlowRepo,
		registrationFlowRoleRepo: registrationFlowRoleRepo,
		roleRepo:                 roleRepo,
		clientRepo:               clientRepo,
	}
}

// syncRoles replaces the flow's role membership to exactly roleUUIDs (adds
// missing, removes extra). Runs inside the supplied transaction.
func (s *registrationFlowService) syncRoles(tx *gorm.DB, registrationFlow *RegistrationFlow, roleUUIDs []uuid.UUID) error {
	txRoleRepo := s.roleRepo.WithTx(tx)
	txAFRRepo := s.registrationFlowRoleRepo.WithTx(tx)

	desired := make(map[int64]bool, len(roleUUIDs))
	for _, ru := range roleUUIDs {
		role, err := txRoleRepo.FindByUUID(ru)
		if err != nil || role == nil {
			return apperror.NewNotFoundWithReason("role not found: " + ru.String())
		}
		if role.TenantID != registrationFlow.TenantID {
			return apperror.NewForbidden("role does not belong to the same tenant as the registration flow")
		}
		desired[role.RoleID] = true
	}

	existing, err := txAFRRepo.FindByRegistrationFlowID(registrationFlow.RegistrationFlowID)
	if err != nil {
		return err
	}
	existingSet := make(map[int64]bool, len(existing))
	for _, e := range existing {
		existingSet[e.RoleID] = true
		if !desired[e.RoleID] {
			if err := txAFRRepo.DeleteByRegistrationFlowIDAndRoleID(registrationFlow.RegistrationFlowID, e.RoleID); err != nil {
				return err
			}
		}
	}
	for roleID := range desired {
		if !existingSet[roleID] {
			if _, err := txAFRRepo.Create(&RegistrationFlowRole{RegistrationFlowID: registrationFlow.RegistrationFlowID, RoleID: roleID}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *registrationFlowService) GetAll(ctx context.Context, tenantID int64, name, identifier *string, status []string, ClientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*RegistrationFlowServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	var ClientID *int64
	if ClientUUID != nil {
		Client, err := s.clientRepo.FindByUUID(*ClientUUID)
		if err != nil || Client == nil {
			return nil, apperror.NewNotFoundWithReason("auth client not found")
		}
		if Client.TenantID != tenantID {
			return nil, apperror.NewForbidden("client does not belong to your tenant")
		}
		ClientID = &Client.ClientID
	}

	filter := RegistrationFlowRepositoryGetFilter{
		Name:       name,
		Identifier: identifier,
		Status:     status,
		TenantID:   &tenantID,
		ClientID:   ClientID,
		Page:       page,
		Limit:      limit,
		SortBy:     sortBy,
		SortOrder:  sortOrder,
	}

	result, err := s.registrationFlowRepo.FindPaginated(filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list registration flows failed")
		return nil, err
	}

	data := make([]RegistrationFlowServiceDataResult, len(result.Data))
	for i, sf := range result.Data {
		data[i] = *toRegistrationFlowServiceDataResult(&sf)
	}

	span.SetStatus(codes.Ok, "")
	return &RegistrationFlowServiceListResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *registrationFlowService) GetByUUID(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64) (*RegistrationFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.getByUUID")
	defer span.End()
	span.SetAttributes(attribute.String("registrationFlow.uuid", registrationFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	registrationFlow, err := s.registrationFlowRepo.FindByUUIDAndTenantID(registrationFlowUUID, tenantID, "Client")
	if err != nil || registrationFlow == nil {
		span.SetStatus(codes.Error, "registration flow not found or access denied")
		return nil, apperror.NewNotFoundWithReason("registration flow not found or access denied")
	}

	span.SetStatus(codes.Ok, "")
	return toRegistrationFlowServiceDataResult(registrationFlow), nil
}

func (s *registrationFlowService) Create(ctx context.Context, tenantID int64, name, description, status string, ClientUUID uuid.UUID, identifier *string, roleUUIDs []uuid.UUID, verificationRequired bool, requiredFields string) (*RegistrationFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	var createdRegistrationFlow *RegistrationFlow

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRegistrationFlowRepo := s.registrationFlowRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)

		// Find auth client
		Client, err := txClientRepo.FindByUUID(ClientUUID)
		if err != nil || Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
		}
		if Client.TenantID != tenantID {
			return apperror.NewForbidden("client does not belong to your tenant")
		}
		if Client.Status != shared.StatusActive {
			return apperror.NewValidation("auth client is inactive or deleted")
		}

		// Check if name already exists
		existingName, err := txRegistrationFlowRepo.FindByNameAndTenantID(name, tenantID)
		if err != nil {
			return err
		}
		if existingName != nil {
			return apperror.NewConflict("registration flow with this name already exists")
		}

		// Use admin-supplied identifier or auto-generate one
		var flowIdentifier string
		if identifier != nil && strings.TrimSpace(*identifier) != "" {
			flowIdentifier = strings.TrimSpace(*identifier)
			existing, err := txRegistrationFlowRepo.FindByIdentifierAndClientID(flowIdentifier, Client.ClientID)
			if err != nil {
				return err
			}
			if existing != nil {
				return apperror.NewConflict("registration flow with this identifier already exists for this client")
			}
		} else {
			for {
				flowIdentifier, err = crypto.GenerateIdentifier(16)
				if err != nil {
					return err
				}
				existing, err := txRegistrationFlowRepo.FindByIdentifierAndClientID(flowIdentifier, Client.ClientID)
				if err != nil {
					return err
				}
				if existing == nil {
					break
				}
			}
		}

		// Create registration flow
		registrationFlow := &RegistrationFlow{
			TenantID:             tenantID,
			Name:                 name,
			Description:          description,
			Identifier:           flowIdentifier,
			Status:               status,
			ClientID:             Client.ClientID,
			VerificationRequired: verificationRequired,
			RequiredFields:       requiredFields,
		}

		created, err := txRegistrationFlowRepo.Create(registrationFlow)
		if err != nil {
			return err
		}

		// Attach roles + callback URIs in the same transaction.
		if len(roleUUIDs) > 0 {
			if err := s.syncRoles(tx, created, roleUUIDs); err != nil {
				return err
			}
		}

		createdRegistrationFlow = created
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create registration flow failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.GetByUUID(ctx, createdRegistrationFlow.RegistrationFlowUUID, tenantID)
}

func (s *registrationFlowService) Update(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, name, description, status string, roleUUIDs []uuid.UUID, verificationRequired bool, requiredFields string) (*RegistrationFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.update")
	defer span.End()
	span.SetAttributes(attribute.String("registrationFlow.uuid", registrationFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	var updatedRegistrationFlow *RegistrationFlow

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRegistrationFlowRepo := s.registrationFlowRepo.WithTx(tx)

		// Find existing registration flow and validate tenant ownership
		registrationFlow, err := txRegistrationFlowRepo.FindByUUIDAndTenantID(registrationFlowUUID, tenantID)
		if err != nil || registrationFlow == nil {
			return apperror.NewNotFoundWithReason("registration flow not found or access denied")
		}

		// Check if name is being changed and if it conflicts
		if name != registrationFlow.Name {
			existingName, err := txRegistrationFlowRepo.FindByNameAndTenantID(name, tenantID)
			if err != nil {
				return err
			}
			if existingName != nil && existingName.RegistrationFlowID != registrationFlow.RegistrationFlowID {
				return apperror.NewConflict("registration flow with this name already exists")
			}
		}

		// Update fields (identifier remains unchanged)
		registrationFlow.Name = name
		registrationFlow.Description = description
		registrationFlow.Status = status
		registrationFlow.VerificationRequired = verificationRequired
		registrationFlow.RequiredFields = requiredFields

		updated, err := txRegistrationFlowRepo.CreateOrUpdate(registrationFlow)
		if err != nil {
			return err
		}

		// nil = leave membership untouched; non-nil (incl. empty) = replace it.
		if roleUUIDs != nil {
			if err := s.syncRoles(tx, updated, roleUUIDs); err != nil {
				return err
			}
		}

		updatedRegistrationFlow = updated
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update registration flow failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.GetByUUID(ctx, updatedRegistrationFlow.RegistrationFlowUUID, tenantID)
}

func (s *registrationFlowService) UpdateStatus(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, status string) (*RegistrationFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.updateStatus")
	defer span.End()
	span.SetAttributes(attribute.String("registrationFlow.uuid", registrationFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	var updatedRegistrationFlow *RegistrationFlow

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRegistrationFlowRepo := s.registrationFlowRepo.WithTx(tx)

		registrationFlow, err := txRegistrationFlowRepo.FindByUUIDAndTenantID(registrationFlowUUID, tenantID)
		if err != nil || registrationFlow == nil {
			return apperror.NewNotFoundWithReason("registration flow not found or access denied")
		}

		registrationFlow.Status = status

		updated, err := txRegistrationFlowRepo.CreateOrUpdate(registrationFlow)
		if err != nil {
			return err
		}

		updatedRegistrationFlow = updated
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update registration flow status failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.GetByUUID(ctx, updatedRegistrationFlow.RegistrationFlowUUID, tenantID)
}

func (s *registrationFlowService) Delete(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64) (*RegistrationFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.delete")
	defer span.End()
	span.SetAttributes(attribute.String("registrationFlow.uuid", registrationFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	registrationFlow, err := s.registrationFlowRepo.FindByUUIDAndTenantID(registrationFlowUUID, tenantID, "Client")
	if err != nil || registrationFlow == nil {
		span.SetStatus(codes.Error, "registration flow not found or access denied")
		return nil, apperror.NewNotFoundWithReason("registration flow not found or access denied")
	}

	// Guard: prevent deletion if the flow is referenced by any pending invites.
	var pendingCount int64
	if err := s.db.
		Table("invites").
		Where("registration_flow_id = ? AND status = ?", registrationFlow.RegistrationFlowID, "pending").
		Count(&pendingCount).Error; err != nil {
		span.RecordError(err)
		return nil, err
	}
	if pendingCount > 0 {
		return nil, apperror.NewConflict("cannot delete registration flow that is referenced by pending invites")
	}

	if registrationFlow.IsSystem {
		return nil, apperror.NewValidation("cannot delete system registration flow")
	}

	result := toRegistrationFlowServiceDataResult(registrationFlow)

	err = s.registrationFlowRepo.DeleteByUUID(registrationFlowUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete registration flow failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

func toRegistrationFlowServiceDataResult(sf *RegistrationFlow) *RegistrationFlowServiceDataResult {
	if sf == nil {
		return nil
	}

	var ClientUUID uuid.UUID
	if sf.Client != nil {
		ClientUUID = sf.Client.ClientUUID
	}

	return &RegistrationFlowServiceDataResult{
		RegistrationFlowUUID: sf.RegistrationFlowUUID,
		Name:                 sf.Name,
		Description:          sf.Description,
		Identifier:           sf.Identifier,
		Status:               sf.Status,
		ClientUUID:           ClientUUID,
		VerificationRequired: sf.VerificationRequired,
		RequiredFields:       sf.RequiredFields,
		CreatedAt:            sf.CreatedAt,
		UpdatedAt:            sf.UpdatedAt,
	}
}

func (s *registrationFlowService) AssignRoles(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, roleUUIDs []uuid.UUID) ([]RegistrationFlowRoleServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.assignRoles")
	defer span.End()
	span.SetAttributes(attribute.String("registrationFlow.uuid", registrationFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	var assignedRoles []RegistrationFlowRoleServiceDataResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRegistrationFlowRepo := s.registrationFlowRepo.WithTx(tx)
		txRegistrationFlowRoleRepo := s.registrationFlowRoleRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)

		// Verify registration flow exists and belongs to tenant
		registrationFlow, err := txRegistrationFlowRepo.FindByUUIDAndTenantID(registrationFlowUUID, tenantID)
		if err != nil || registrationFlow == nil {
			return apperror.NewNotFoundWithReason("registration flow not found or access denied")
		}

		// Assign each role
		for _, roleUUID := range roleUUIDs {
			role, err := txRoleRepo.FindByUUID(roleUUID)
			if err != nil || role == nil {
				return apperror.NewNotFoundWithReason("role not found: " + roleUUID.String())
			}
			if role.TenantID != registrationFlow.TenantID {
				return apperror.NewForbidden("role does not belong to the same tenant as the registration flow")
			}

			// Check if already assigned
			existing, err := txRegistrationFlowRoleRepo.FindByRegistrationFlowIDAndRoleID(registrationFlow.RegistrationFlowID, role.RoleID)
			if err != nil {
				return err
			}
			if existing != nil {
				continue // Skip if already assigned
			}

			// Create registration flow role
			registrationFlowRole := &RegistrationFlowRole{
				RegistrationFlowID: registrationFlow.RegistrationFlowID,
				RoleID:             role.RoleID,
			}

			created, err := txRegistrationFlowRoleRepo.Create(registrationFlowRole)
			if err != nil {
				return err
			}

			assignedRoles = append(assignedRoles, RegistrationFlowRoleServiceDataResult{
				RegistrationFlowRoleUUID: created.RegistrationFlowRoleUUID,
				RegistrationFlowUUID:     registrationFlow.RegistrationFlowUUID,
				RoleUUID:                 role.RoleUUID,
				RoleName:                 role.Name,
				RoleDescription:          role.Description,
				RoleStatus:               role.Status,
				RoleIsDefault:            role.IsDefault,
				RoleIsSystem:             role.IsSystem,
				CreatedAt:                created.CreatedAt,
				UpdatedAt:                role.UpdatedAt,
			})
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "assign roles failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return assignedRoles, nil
}

func (s *registrationFlowService) GetRoles(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, page, limit int) (*RegistrationFlowRoleServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.getRoles")
	defer span.End()
	span.SetAttributes(attribute.String("registrationFlow.uuid", registrationFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	// Verify registration flow exists and belongs to tenant
	registrationFlow, err := s.registrationFlowRepo.FindByUUIDAndTenantID(registrationFlowUUID, tenantID)
	if err != nil || registrationFlow == nil {
		span.SetStatus(codes.Error, "registration flow not found or access denied")
		return nil, apperror.NewNotFoundWithReason("registration flow not found or access denied")
	}

	// Get paginated registration flow roles
	registrationFlowRoles, total, err := s.registrationFlowRoleRepo.FindByRegistrationFlowIDPaginated(registrationFlow.RegistrationFlowID, page, limit)
	if err != nil {
		return nil, err
	}

	roles := make([]RegistrationFlowRoleServiceDataResult, len(registrationFlowRoles))
	for i, sfr := range registrationFlowRoles {
		if sfr.Role != nil {
			roles[i] = RegistrationFlowRoleServiceDataResult{
				RegistrationFlowRoleUUID: sfr.RegistrationFlowRoleUUID,
				RegistrationFlowUUID:     registrationFlow.RegistrationFlowUUID,
				RoleUUID:                 sfr.Role.RoleUUID,
				RoleName:                 sfr.Role.Name,
				RoleDescription:          sfr.Role.Description,
				RoleStatus:               sfr.Role.Status,
				RoleIsDefault:            sfr.Role.IsDefault,
				RoleIsSystem:             sfr.Role.IsSystem,
				CreatedAt:                sfr.CreatedAt,
				UpdatedAt:                sfr.Role.UpdatedAt,
			}
		}
	}

	// Calculate total pages
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	span.SetStatus(codes.Ok, "")
	return &RegistrationFlowRoleServiceListResult{
		Data:       roles,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (s *registrationFlowService) RemoveRole(ctx context.Context, registrationFlowUUID uuid.UUID, tenantID int64, roleUUID uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "registrationFlow.removeRole")
	defer span.End()
	span.SetAttributes(attribute.String("registrationFlow.uuid", registrationFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRegistrationFlowRepo := s.registrationFlowRepo.WithTx(tx)
		txRegistrationFlowRoleRepo := s.registrationFlowRoleRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)

		// Verify registration flow exists and belongs to tenant
		registrationFlow, err := txRegistrationFlowRepo.FindByUUIDAndTenantID(registrationFlowUUID, tenantID)
		if err != nil || registrationFlow == nil {
			return apperror.NewNotFoundWithReason("registration flow not found or access denied")
		}

		// Verify role exists
		role, err := txRoleRepo.FindByUUID(roleUUID)
		if err != nil || role == nil {
			return apperror.NewNotFound("role not found")
		}
		if role.TenantID != registrationFlow.TenantID {
			return apperror.NewNotFoundWithReason("role not found: " + roleUUID.String())
		}

		// Delete registration flow role
		return txRegistrationFlowRoleRepo.DeleteByRegistrationFlowIDAndRoleID(registrationFlow.RegistrationFlowID, role.RoleID)
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "remove role failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}
