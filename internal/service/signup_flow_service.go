package service

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SignupFlowServiceDataResult struct {
	SignupFlowUUID uuid.UUID
	Name           string
	Description    string
	Identifier     string
	Config         map[string]interface{}
	Status         string
	AuthClientUUID uuid.UUID
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
	GetAll(name, identifier *string, status []string, authClientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*SignupFlowServiceListResult, error)
	GetByUUID(signupFlowUUID uuid.UUID) (*SignupFlowServiceDataResult, error)
	Create(name, description string, config map[string]interface{}, status string, authClientUUID uuid.UUID) (*SignupFlowServiceDataResult, error)
	Update(signupFlowUUID uuid.UUID, name, description string, config map[string]interface{}, status string) (*SignupFlowServiceDataResult, error)
	UpdateStatus(signupFlowUUID uuid.UUID, status string) (*SignupFlowServiceDataResult, error)
	Delete(signupFlowUUID uuid.UUID) (*SignupFlowServiceDataResult, error)
	AssignRoles(signupFlowUUID uuid.UUID, roleUUIDs []uuid.UUID) ([]SignupFlowRoleServiceDataResult, error)
	GetRoles(signupFlowUUID uuid.UUID, page, limit int) (*SignupFlowRoleServiceListResult, error)
	RemoveRole(signupFlowUUID uuid.UUID, roleUUID uuid.UUID) error
}

type signupFlowService struct {
	db                 *gorm.DB
	signupFlowRepo     repository.SignupFlowRepository
	signupFlowRoleRepo repository.SignupFlowRoleRepository
	roleRepo           repository.RoleRepository
	authClientRepo     repository.AuthClientRepository
}

func NewSignupFlowService(
	db *gorm.DB,
	signupFlowRepo repository.SignupFlowRepository,
	signupFlowRoleRepo repository.SignupFlowRoleRepository,
	roleRepo repository.RoleRepository,
	authClientRepo repository.AuthClientRepository,
) SignupFlowService {
	return &signupFlowService{
		db:                 db,
		signupFlowRepo:     signupFlowRepo,
		signupFlowRoleRepo: signupFlowRoleRepo,
		roleRepo:           roleRepo,
		authClientRepo:     authClientRepo,
	}
}

func (s *signupFlowService) GetAll(name, identifier *string, status []string, authClientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*SignupFlowServiceListResult, error) {
	var authClientID *int64
	if authClientUUID != nil {
		authClient, err := s.authClientRepo.FindByUUID(*authClientUUID)
		if err != nil || authClient == nil {
			return nil, errors.New("auth client not found")
		}
		authClientID = &authClient.AuthClientID
	}

	filter := repository.SignupFlowRepositoryGetFilter{
		Name:         name,
		Identifier:   identifier,
		Status:       status,
		AuthClientID: authClientID,
		Page:         page,
		Limit:        limit,
		SortBy:       sortBy,
		SortOrder:    sortOrder,
	}

	result, err := s.signupFlowRepo.FindPaginated(filter)
	if err != nil {
		return nil, err
	}

	data := make([]SignupFlowServiceDataResult, len(result.Data))
	for i, sf := range result.Data {
		data[i] = *toSignupFlowServiceDataResult(&sf)
	}

	return &SignupFlowServiceListResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *signupFlowService) GetByUUID(signupFlowUUID uuid.UUID) (*SignupFlowServiceDataResult, error) {
	signupFlow, err := s.signupFlowRepo.FindByUUID(signupFlowUUID, "AuthClient")
	if err != nil || signupFlow == nil {
		return nil, errors.New("signup flow not found")
	}

	return toSignupFlowServiceDataResult(signupFlow), nil
}

func (s *signupFlowService) Create(name, description string, config map[string]interface{}, status string, authClientUUID uuid.UUID) (*SignupFlowServiceDataResult, error) {
	var createdSignupFlow *model.SignupFlow

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txSignupFlowRepo := s.signupFlowRepo.WithTx(tx)
		txAuthClientRepo := s.authClientRepo.WithTx(tx)

		// Find auth client
		authClient, err := txAuthClientRepo.FindByUUID(authClientUUID)
		if err != nil || authClient == nil {
			return errors.New("auth client not found")
		}

		// Check if name already exists
		existingName, err := txSignupFlowRepo.FindByName(name)
		if err != nil {
			return err
		}
		if existingName != nil {
			return errors.New("signup flow with this name already exists")
		}

		// Generate unique identifier
		var identifier string
		for {
			identifier = util.GenerateIdentifier(16)
			existing, err := txSignupFlowRepo.FindByIdentifierAndAuthClientID(identifier, authClient.AuthClientID)
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
			Name:         name,
			Description:  description,
			Identifier:   identifier,
			Config:       configJSON,
			Status:       status,
			AuthClientID: authClient.AuthClientID,
		}

		created, err := txSignupFlowRepo.Create(signupFlow)
		if err != nil {
			return err
		}

		createdSignupFlow = created
		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.GetByUUID(createdSignupFlow.SignupFlowUUID)
}

func (s *signupFlowService) Update(signupFlowUUID uuid.UUID, name, description string, config map[string]interface{}, status string) (*SignupFlowServiceDataResult, error) {
	var updatedSignupFlow *model.SignupFlow

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txSignupFlowRepo := s.signupFlowRepo.WithTx(tx)

		// Find existing signup flow
		signupFlow, err := txSignupFlowRepo.FindByUUID(signupFlowUUID)
		if err != nil || signupFlow == nil {
			return errors.New("signup flow not found")
		}

		// Check if name is being changed and if it conflicts
		if name != signupFlow.Name {
			existingName, err := txSignupFlowRepo.FindByName(name)
			if err != nil {
				return err
			}
			if existingName != nil && existingName.SignupFlowID != signupFlow.SignupFlowID {
				return errors.New("signup flow with this name already exists")
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
		return nil, err
	}

	return s.GetByUUID(updatedSignupFlow.SignupFlowUUID)
}

func (s *signupFlowService) UpdateStatus(signupFlowUUID uuid.UUID, status string) (*SignupFlowServiceDataResult, error) {
	var updatedSignupFlow *model.SignupFlow

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txSignupFlowRepo := s.signupFlowRepo.WithTx(tx)

		signupFlow, err := txSignupFlowRepo.FindByUUID(signupFlowUUID)
		if err != nil || signupFlow == nil {
			return errors.New("signup flow not found")
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
		return nil, err
	}

	return s.GetByUUID(updatedSignupFlow.SignupFlowUUID)
}

func (s *signupFlowService) Delete(signupFlowUUID uuid.UUID) (*SignupFlowServiceDataResult, error) {
	signupFlow, err := s.signupFlowRepo.FindByUUID(signupFlowUUID, "AuthClient")
	if err != nil || signupFlow == nil {
		return nil, errors.New("signup flow not found")
	}

	result := toSignupFlowServiceDataResult(signupFlow)

	err = s.signupFlowRepo.DeleteByUUID(signupFlowUUID)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func toSignupFlowServiceDataResult(sf *model.SignupFlow) *SignupFlowServiceDataResult {
	if sf == nil {
		return nil
	}

	var config map[string]interface{}
	if len(sf.Config) > 0 {
		if err := json.Unmarshal(sf.Config, &config); err != nil {
			config = nil
		}
	}

	var authClientUUID uuid.UUID
	if sf.AuthClient != nil {
		authClientUUID = sf.AuthClient.AuthClientUUID
	}

	return &SignupFlowServiceDataResult{
		SignupFlowUUID: sf.SignupFlowUUID,
		Name:           sf.Name,
		Description:    sf.Description,
		Identifier:     sf.Identifier,
		Config:         config,
		Status:         sf.Status,
		AuthClientUUID: authClientUUID,
		CreatedAt:      sf.CreatedAt,
		UpdatedAt:      sf.UpdatedAt,
	}
}

func (s *signupFlowService) AssignRoles(signupFlowUUID uuid.UUID, roleUUIDs []uuid.UUID) ([]SignupFlowRoleServiceDataResult, error) {
	var assignedRoles []SignupFlowRoleServiceDataResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txSignupFlowRepo := s.signupFlowRepo.WithTx(tx)
		txSignupFlowRoleRepo := s.signupFlowRoleRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)

		// Verify signup flow exists
		signupFlow, err := txSignupFlowRepo.FindByUUID(signupFlowUUID)
		if err != nil || signupFlow == nil {
			return errors.New("signup flow not found")
		}

		// Assign each role
		for _, roleUUID := range roleUUIDs {
			role, err := txRoleRepo.FindByUUID(roleUUID)
			if err != nil || role == nil {
				return errors.New("role not found: " + roleUUID.String())
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
		return nil, err
	}

	return assignedRoles, nil
}

func (s *signupFlowService) GetRoles(signupFlowUUID uuid.UUID, page, limit int) (*SignupFlowRoleServiceListResult, error) {
	// Verify signup flow exists
	signupFlow, err := s.signupFlowRepo.FindByUUID(signupFlowUUID)
	if err != nil || signupFlow == nil {
		return nil, errors.New("signup flow not found")
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

	return &SignupFlowRoleServiceListResult{
		Data:       roles,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (s *signupFlowService) RemoveRole(signupFlowUUID uuid.UUID, roleUUID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		txSignupFlowRepo := s.signupFlowRepo.WithTx(tx)
		txSignupFlowRoleRepo := s.signupFlowRoleRepo.WithTx(tx)
		txRoleRepo := s.roleRepo.WithTx(tx)

		// Verify signup flow exists
		signupFlow, err := txSignupFlowRepo.FindByUUID(signupFlowUUID)
		if err != nil || signupFlow == nil {
			return errors.New("signup flow not found")
		}

		// Verify role exists
		role, err := txRoleRepo.FindByUUID(roleUUID)
		if err != nil || role == nil {
			return errors.New("role not found")
		}

		// Delete signup flow role
		return txSignupFlowRoleRepo.DeleteBySignupFlowIDAndRoleID(signupFlow.SignupFlowID, role.RoleID)
	})
}
