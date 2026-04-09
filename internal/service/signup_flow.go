package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/crypto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SignupFlowServiceDataResult struct {
	SignupFlowUUID uuid.UUID
	Name           string
	Description    string
	Identifier     string
	Config         map[string]any
	Status         string
	ClientUUID     uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type SignupFlowServiceListResult struct {
	Data       []SignupFlowServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type SignupFlowRoleServiceDataResult struct {
	SignupFlowRoleUUID uuid.UUID
	SignupFlowUUID     uuid.UUID
	RoleUUID           uuid.UUID
	RoleName           string
	RoleDescription    string
	RoleStatus         string
	RoleIsDefault      bool
	RoleIsSystem       bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type SignupFlowRoleServiceListResult struct {
	Data       []SignupFlowRoleServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type SignupFlowService interface {
	GetAll(ctx context.Context, tenantID int64, name, identifier *string, status []string, ClientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*SignupFlowServiceListResult, error)
	GetByUUID(ctx context.Context, signupFlowUUID uuid.UUID, tenantID int64) (*SignupFlowServiceDataResult, error)
	Create(ctx context.Context, tenantID int64, name, description string, config map[string]any, status string, ClientUUID uuid.UUID) (*SignupFlowServiceDataResult, error)
	Update(ctx context.Context, signupFlowUUID uuid.UUID, tenantID int64, name, description string, config map[string]any, status string) (*SignupFlowServiceDataResult, error)
	UpdateStatus(ctx context.Context, signupFlowUUID uuid.UUID, tenantID int64, status string) (*SignupFlowServiceDataResult, error)
	Delete(ctx context.Context, signupFlowUUID uuid.UUID, tenantID int64) (*SignupFlowServiceDataResult, error)
	AssignRoles(ctx context.Context, signupFlowUUID uuid.UUID, tenantID int64, roleUUIDs []uuid.UUID) ([]SignupFlowRoleServiceDataResult, error)
	GetRoles(ctx context.Context, signupFlowUUID uuid.UUID, tenantID int64, page, limit int) (*SignupFlowRoleServiceListResult, error)
	RemoveRole(ctx context.Context, signupFlowUUID uuid.UUID, tenantID int64, roleUUID uuid.UUID) error
}

type signupFlowService struct {
	db                 *gorm.DB
	signupFlowRepo     repository.SignupFlowRepository
	signupFlowRoleRepo repository.SignupFlowRoleRepository
	roleRepo           repository.RoleRepository
	clientRepo         repository.ClientRepository
}

func NewSignupFlowService(
	db *gorm.DB,
	signupFlowRepo repository.SignupFlowRepository,
	signupFlowRoleRepo repository.SignupFlowRoleRepository,
	roleRepo repository.RoleRepository,
	clientRepo repository.ClientRepository,
) SignupFlowService {
	return &signupFlowService{
		db:                 db,
		signupFlowRepo:     signupFlowRepo,
		signupFlowRoleRepo: signupFlowRoleRepo,
		roleRepo:           roleRepo,
		clientRepo:         clientRepo,
	}
}

func (s *signupFlowService) GetAll(ctx context.Context, tenantID int64, name, identifier *string, status []string, ClientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*SignupFlowServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "signupFlow.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	var ClientID *int64
	if ClientUUID != nil {
		Client, err := s.clientRepo.FindByUUID(*ClientUUID)
		if err != nil || Client == nil {
			return nil, apperror.NewNotFoundWithReason("auth client not found")
		}
		ClientID = &Client.ClientID
	}

	filter := repository.SignupFlowRepositoryGetFilter{
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

	result, err := s.signupFlowRepo.FindPaginated(filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list signup flows failed")
		return nil, err
	}

	data := make([]SignupFlowServiceDataResult, len(result.Data))
	for i, sf := range result.Data {
		data[i] = *toSignupFlowServiceDataResult(&sf)
	}

	span.SetStatus(codes.Ok, "")
	return &SignupFlowServiceListResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *signupFlowService) GetByUUID(ctx context.Context, signupFlowUUID uuid.UUID, tenantID int64) (*SignupFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "signupFlow.getByUUID")
	defer span.End()
	span.SetAttributes(attribute.String("signupFlow.uuid", signupFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	signupFlow, err := s.signupFlowRepo.FindByUUIDAndTenantID(signupFlowUUID, tenantID, "Client")
	if err != nil || signupFlow == nil {
		span.SetStatus(codes.Error, "signup flow not found or access denied")
		return nil, apperror.NewNotFoundWithReason("signup flow not found or access denied")
	}

	span.SetStatus(codes.Ok, "")
	return toSignupFlowServiceDataResult(signupFlow), nil
}

func (s *signupFlowService) Create(ctx context.Context, tenantID int64, name, description string, config map[string]any, status string, ClientUUID uuid.UUID) (*SignupFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "signupFlow.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	var createdSignupFlow *model.SignupFlow

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txSignupFlowRepo := s.signupFlowRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)

		// Find auth client
		Client, err := txClientRepo.FindByUUID(ClientUUID)
		if err != nil || Client == nil {
			return apperror.NewNotFoundWithReason("auth client not found")
		}

		// Check if name already exists
		existingName, err := txSignupFlowRepo.FindByName(name)
		if err != nil {
			return err
		}
		if existingName != nil {
			return apperror.NewConflict("signup flow with this name already exists")
		}

		// Generate unique identifier
		var identifier string
		for {
			identifier = crypto.GenerateIdentifier(16)
			existing, err := txSignupFlowRepo.FindByIdentifierAndClientID(identifier, Client.ClientID)
			if err != nil {
				return err
			}
			if existing == nil {
				break
			}
		}

		// Convert config to JSONB
		var configJSON datatypes.JSON
		if config != nil {
			configBytes, err := json.Marshal(config)
			if err != nil {
				return err
			}
			configJSON = datatypes.JSON(configBytes)
		} else {
			configJSON = datatypes.JSON([]byte("{}"))
		}

		// Create signup flow
		signupFlow := &model.SignupFlow{
			TenantID:    tenantID,
			Name:        name,
			Description: description,
			Identifier:  identifier,
			Config:      configJSON,
			Status:      status,
			ClientID:    Client.ClientID,
		}

		created, err := txSignupFlowRepo.Create(signupFlow)
		if err != nil {
			return err
		}

		createdSignupFlow = created
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create signup flow failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.GetByUUID(ctx, createdSignupFlow.SignupFlowUUID, tenantID)
}

func (s *signupFlowService) Update(ctx context.Context, signupFlowUUID uuid.UUID, tenantID int64, name, description string, config map[string]any, status string) (*SignupFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "signupFlow.update")
	defer span.End()
	span.SetAttributes(attribute.String("signupFlow.uuid", signupFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	var updatedSignupFlow *model.SignupFlow

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txSignupFlowRepo := s.signupFlowRepo.WithTx(tx)

		// Find existing signup flow and validate tenant ownership
		signupFlow, err := txSignupFlowRepo.FindByUUIDAndTenantID(signupFlowUUID, tenantID)
		if err != nil || signupFlow == nil {
			return apperror.NewNotFoundWithReason("signup flow not found or access denied")
		}

		// Check if name is being changed and if it conflicts
		if name != signupFlow.Name {
			existingName, err := txSignupFlowRepo.FindByName(name)
			if err != nil {
				return err
			}
			if existingName != nil && existingName.SignupFlowID != signupFlow.SignupFlowID {
				return apperror.NewConflict("signup flow with this name already exists")
			}
		}

		// Convert config to JSONB
		var configJSON datatypes.JSON
		if config != nil {
			configBytes, err := json.Marshal(config)
			if err != nil {
				return err
			}
			configJSON = datatypes.JSON(configBytes)
		} else {
			configJSON = datatypes.JSON([]byte("{}"))
		}

		// Update fields (identifier remains unchanged)
		signupFlow.Name = name
		signupFlow.Description = description
		signupFlow.Config = configJSON
		signupFlow.Status = status

		updated, err := txSignupFlowRepo.CreateOrUpdate(signupFlow)
		if err != nil {
			return err
		}

		updatedSignupFlow = updated
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update signup flow failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.GetByUUID(ctx, updatedSignupFlow.SignupFlowUUID, tenantID)
}

func (s *signupFlowService) UpdateStatus(ctx context.Context, signupFlowUUID uuid.UUID, tenantID int64, status string) (*SignupFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "signupFlow.updateStatus")
	defer span.End()
	span.SetAttributes(attribute.String("signupFlow.uuid", signupFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	var updatedSignupFlow *model.SignupFlow

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txSignupFlowRepo := s.signupFlowRepo.WithTx(tx)

		signupFlow, err := txSignupFlowRepo.FindByUUIDAndTenantID(signupFlowUUID, tenantID)
		if err != nil || signupFlow == nil {
			return apperror.NewNotFoundWithReason("signup flow not found or access denied")
		}

		signupFlow.Status = status

		updated, err := txSignupFlowRepo.CreateOrUpdate(signupFlow)
		if err != nil {
			return err
		}

		updatedSignupFlow = updated
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update signup flow status failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return s.GetByUUID(ctx, updatedSignupFlow.SignupFlowUUID, tenantID)
}

func (s *signupFlowService) Delete(ctx context.Context, signupFlowUUID uuid.UUID, tenantID int64) (*SignupFlowServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "signupFlow.delete")
	defer span.End()
	span.SetAttributes(attribute.String("signupFlow.uuid", signupFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	signupFlow, err := s.signupFlowRepo.FindByUUIDAndTenantID(signupFlowUUID, tenantID, "Client")
	if err != nil || signupFlow == nil {
		span.SetStatus(codes.Error, "signup flow not found or access denied")
		return nil, apperror.NewNotFoundWithReason("signup flow not found or access denied")
	}

	result := toSignupFlowServiceDataResult(signupFlow)

	err = s.signupFlowRepo.DeleteByUUID(signupFlowUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete signup flow failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

func toSignupFlowServiceDataResult(sf *model.SignupFlow) *SignupFlowServiceDataResult {
	if sf == nil {
		return nil
	}

	var config map[string]any
	if len(sf.Config) > 0 {
		if err := json.Unmarshal(sf.Config, &config); err != nil {
			config = nil
		}
	}

	var ClientUUID uuid.UUID
	if sf.Client != nil {
		ClientUUID = sf.Client.ClientUUID
	}

	return &SignupFlowServiceDataResult{
		SignupFlowUUID: sf.SignupFlowUUID,
		Name:           sf.Name,
		Description:    sf.Description,
		Identifier:     sf.Identifier,
		Config:         config,
		Status:         sf.Status,
		ClientUUID:     ClientUUID,
		CreatedAt:      sf.CreatedAt,
		UpdatedAt:      sf.UpdatedAt,
	}
}

func (s *signupFlowService) AssignRoles(ctx context.Context, signupFlowUUID uuid.UUID, tenantID int64, roleUUIDs []uuid.UUID) ([]SignupFlowRoleServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "signupFlow.assignRoles")
	defer span.End()
	span.SetAttributes(attribute.String("signupFlow.uuid", signupFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	var assignedRoles []SignupFlowRoleServiceDataResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txSignupFlowRepo := s.signupFlowRepo.WithTx(tx)
		txSignupFlowRoleRepo := s.signupFlowRoleRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)

		// Verify signup flow exists and belongs to tenant
		signupFlow, err := txSignupFlowRepo.FindByUUIDAndTenantID(signupFlowUUID, tenantID)
		if err != nil || signupFlow == nil {
			return apperror.NewNotFoundWithReason("signup flow not found or access denied")
		}

		// Assign each role
		for _, roleUUID := range roleUUIDs {
			role, err := txRoleRepo.FindByUUID(roleUUID)
			if err != nil || role == nil {
				return apperror.NewNotFoundWithReason("role not found: " + roleUUID.String())
			}

			// Check if already assigned
			existing, err := txSignupFlowRoleRepo.FindBySignupFlowIDAndRoleID(signupFlow.SignupFlowID, role.RoleID)
			if err != nil {
				return err
			}
			if existing != nil {
				continue // Skip if already assigned
			}

			// Create signup flow role
			signupFlowRole := &model.SignupFlowRole{
				SignupFlowID: signupFlow.SignupFlowID,
				RoleID:       role.RoleID,
			}

			created, err := txSignupFlowRoleRepo.Create(signupFlowRole)
			if err != nil {
				return err
			}

			assignedRoles = append(assignedRoles, SignupFlowRoleServiceDataResult{
				SignupFlowRoleUUID: created.SignupFlowRoleUUID,
				SignupFlowUUID:     signupFlow.SignupFlowUUID,
				RoleUUID:           role.RoleUUID,
				RoleName:           role.Name,
				RoleDescription:    role.Description,
				RoleStatus:         role.Status,
				RoleIsDefault:      role.IsDefault,
				RoleIsSystem:       role.IsSystem,
				CreatedAt:          created.CreatedAt,
				UpdatedAt:          role.UpdatedAt,
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

func (s *signupFlowService) GetRoles(ctx context.Context, signupFlowUUID uuid.UUID, tenantID int64, page, limit int) (*SignupFlowRoleServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "signupFlow.getRoles")
	defer span.End()
	span.SetAttributes(attribute.String("signupFlow.uuid", signupFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	// Verify signup flow exists and belongs to tenant
	signupFlow, err := s.signupFlowRepo.FindByUUIDAndTenantID(signupFlowUUID, tenantID)
	if err != nil || signupFlow == nil {
		span.SetStatus(codes.Error, "signup flow not found or access denied")
		return nil, apperror.NewNotFoundWithReason("signup flow not found or access denied")
	}

	// Get paginated signup flow roles
	signupFlowRoles, total, err := s.signupFlowRoleRepo.FindBySignupFlowIDPaginated(signupFlow.SignupFlowID, page, limit)
	if err != nil {
		return nil, err
	}

	roles := make([]SignupFlowRoleServiceDataResult, len(signupFlowRoles))
	for i, sfr := range signupFlowRoles {
		if sfr.Role != nil {
			roles[i] = SignupFlowRoleServiceDataResult{
				SignupFlowRoleUUID: sfr.SignupFlowRoleUUID,
				SignupFlowUUID:     signupFlow.SignupFlowUUID,
				RoleUUID:           sfr.Role.RoleUUID,
				RoleName:           sfr.Role.Name,
				RoleDescription:    sfr.Role.Description,
				RoleStatus:         sfr.Role.Status,
				RoleIsDefault:      sfr.Role.IsDefault,
				RoleIsSystem:       sfr.Role.IsSystem,
				CreatedAt:          sfr.CreatedAt,
				UpdatedAt:          sfr.Role.UpdatedAt,
			}
		}
	}

	// Calculate total pages
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	span.SetStatus(codes.Ok, "")
	return &SignupFlowRoleServiceListResult{
		Data:       roles,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (s *signupFlowService) RemoveRole(ctx context.Context, signupFlowUUID uuid.UUID, tenantID int64, roleUUID uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "signupFlow.removeRole")
	defer span.End()
	span.SetAttributes(attribute.String("signupFlow.uuid", signupFlowUUID.String()), attribute.Int64("tenant.id", tenantID))

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txSignupFlowRepo := s.signupFlowRepo.WithTx(tx)
		txSignupFlowRoleRepo := s.signupFlowRoleRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)

		// Verify signup flow exists and belongs to tenant
		signupFlow, err := txSignupFlowRepo.FindByUUIDAndTenantID(signupFlowUUID, tenantID)
		if err != nil || signupFlow == nil {
			return apperror.NewNotFoundWithReason("signup flow not found or access denied")
		}

		// Verify role exists
		role, err := txRoleRepo.FindByUUID(roleUUID)
		if err != nil || role == nil {
			return apperror.NewNotFound("role not found")
		}

		// Delete signup flow role
		return txSignupFlowRoleRepo.DeleteBySignupFlowIDAndRoleID(signupFlow.SignupFlowID, role.RoleID)
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "remove role failed")
		return err
	}
	span.SetStatus(codes.Ok, "")
	return nil
}
