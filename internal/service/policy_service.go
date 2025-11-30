package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PolicyServiceDataResult struct {
	PolicyUUID  uuid.UUID
	Name        string
	Description *string
	Document    datatypes.JSON
	Version     string
	Status      string
	IsDefault   bool
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PolicyServiceGetFilter struct {
	Name        *string
	Description *string
	Version     *string
	Status      []string
	IsDefault   *bool
	IsSystem    *bool
	ServiceID   *uuid.UUID
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type PolicyServiceGetResult struct {
	Data       []PolicyServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type PolicyService interface {
	Get(filter PolicyServiceGetFilter) (*PolicyServiceGetResult, error)
	GetByUUID(policyUUID uuid.UUID) (*PolicyServiceDataResult, error)
	Create(name string, description *string, document datatypes.JSON, version string, status string, isDefault bool, isSystem bool) (*PolicyServiceDataResult, error)
	Update(policyUUID uuid.UUID, name string, description *string, document datatypes.JSON, version string, status string) (*PolicyServiceDataResult, error)
	SetStatusByUUID(policyUUID uuid.UUID, status string) (*PolicyServiceDataResult, error)
	DeleteByUUID(policyUUID uuid.UUID) (*PolicyServiceDataResult, error)
}

type policyService struct {
	db         *gorm.DB
	policyRepo repository.PolicyRepository
}

func NewPolicyService(
	db *gorm.DB,
	policyRepo repository.PolicyRepository,
) PolicyService {
	return &policyService{
		db:         db,
		policyRepo: policyRepo,
	}
}

func (s *policyService) Get(filter PolicyServiceGetFilter) (*PolicyServiceGetResult, error) {
	repoFilter := repository.PolicyRepositoryGetFilter{
		Name:        filter.Name,
		Description: filter.Description,
		Version:     filter.Version,
		Status:      filter.Status,
		IsDefault:   filter.IsDefault,
		IsSystem:    filter.IsSystem,
		ServiceID:   filter.ServiceID,
		Page:        filter.Page,
		Limit:       filter.Limit,
		SortBy:      filter.SortBy,
		SortOrder:   filter.SortOrder,
	}

	result, err := s.policyRepo.FindPaginated(repoFilter)
	if err != nil {
		return nil, err
	}

	var data []PolicyServiceDataResult
	for _, policy := range result.Data {
		data = append(data, PolicyServiceDataResult{
			PolicyUUID:  policy.PolicyUUID,
			Name:        policy.Name,
			Description: policy.Description,
			Document:    policy.Document,
			Version:     policy.Version,
			Status:      policy.Status,
			IsDefault:   policy.IsDefault,
			IsSystem:    policy.IsSystem,
			CreatedAt:   policy.CreatedAt,
			UpdatedAt:   policy.UpdatedAt,
		})
	}

	return &PolicyServiceGetResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *policyService) GetByUUID(policyUUID uuid.UUID) (*PolicyServiceDataResult, error) {
	policy, err := s.policyRepo.FindByUUID(policyUUID)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, errors.New("policy not found")
	}

	return &PolicyServiceDataResult{
		PolicyUUID:  policy.PolicyUUID,
		Name:        policy.Name,
		Description: policy.Description,
		Document:    policy.Document,
		Version:     policy.Version,
		Status:      policy.Status,
		IsDefault:   policy.IsDefault,
		IsSystem:    policy.IsSystem,
		CreatedAt:   policy.CreatedAt,
		UpdatedAt:   policy.UpdatedAt,
	}, nil
}

func (s *policyService) Create(name string, description *string, document datatypes.JSON, version string, status string, isDefault bool, isSystem bool) (*PolicyServiceDataResult, error) {
	var createdPolicy *model.Policy

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPolicyRepo := s.policyRepo.WithTx(tx)

		// Check if policy with same name and version already exists
		existingPolicy, err := txPolicyRepo.FindByNameAndVersion(name, version)
		if err != nil {
			return err
		}
		if existingPolicy != nil {
			return errors.New("policy with name '" + name + "' and version '" + version + "' already exists")
		}

		// Create new policy
		policy := &model.Policy{
			PolicyUUID:  uuid.New(),
			Name:        name,
			Description: description,
			Document:    document,
			Version:     version,
			Status:      status,
			IsDefault:   isDefault,
			IsSystem:    isSystem,
		}

		createdPolicy, err = txPolicyRepo.Create(policy)
		if err != nil {
			return err
		}

		createdPolicy = policy
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &PolicyServiceDataResult{
		PolicyUUID:  createdPolicy.PolicyUUID,
		Name:        createdPolicy.Name,
		Description: createdPolicy.Description,
		Document:    createdPolicy.Document,
		Version:     createdPolicy.Version,
		Status:      createdPolicy.Status,
		IsDefault:   createdPolicy.IsDefault,
		IsSystem:    createdPolicy.IsSystem,
		CreatedAt:   createdPolicy.CreatedAt,
		UpdatedAt:   createdPolicy.UpdatedAt,
	}, nil
}

func (s *policyService) Update(policyUUID uuid.UUID, name string, description *string, document datatypes.JSON, version string, status string) (*PolicyServiceDataResult, error) {
	var updatedPolicy *model.Policy

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPolicyRepo := s.policyRepo.WithTx(tx)

		// Check if policy exists
		policy, err := txPolicyRepo.FindByUUID(policyUUID)
		if err != nil {
			return err
		}
		if policy == nil {
			return errors.New("policy not found")
		}

		// Check if another policy with same name and version exists (excluding current policy)
		if policy.Name != name || policy.Version != version {
			existingPolicy, err := txPolicyRepo.FindByNameAndVersion(name, version)
			if err != nil {
				return err
			}
			if existingPolicy != nil && existingPolicy.PolicyUUID != policyUUID {
				return errors.New("policy with name '" + name + "' and version '" + version + "' already exists")
			}
		}

		// Update policy
		policy.Name = name
		policy.Description = description
		policy.Document = document
		policy.Version = version
		policy.Status = status

		updatedPolicy, err = txPolicyRepo.UpdateByUUID(policy.PolicyUUID, policy)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &PolicyServiceDataResult{
		PolicyUUID:  updatedPolicy.PolicyUUID,
		Name:        updatedPolicy.Name,
		Description: updatedPolicy.Description,
		Document:    updatedPolicy.Document,
		Version:     updatedPolicy.Version,
		Status:      updatedPolicy.Status,
		IsDefault:   updatedPolicy.IsDefault,
		IsSystem:    updatedPolicy.IsSystem,
		CreatedAt:   updatedPolicy.CreatedAt,
		UpdatedAt:   updatedPolicy.UpdatedAt,
	}, nil
}

func (s *policyService) SetStatusByUUID(policyUUID uuid.UUID, status string) (*PolicyServiceDataResult, error) {
	var updatedPolicy *model.Policy

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPolicyRepo := s.policyRepo.WithTx(tx)

		// Check if policy exists
		policy, err := txPolicyRepo.FindByUUID(policyUUID)
		if err != nil {
			return err
		}
		if policy == nil {
			return errors.New("policy not found")
		}

		// Update status
		if err := txPolicyRepo.SetStatusByUUID(policyUUID, status); err != nil {
			return err
		}

		// Get updated policy
		updatedPolicy, err = txPolicyRepo.FindByUUID(policyUUID)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &PolicyServiceDataResult{
		PolicyUUID:  updatedPolicy.PolicyUUID,
		Name:        updatedPolicy.Name,
		Description: updatedPolicy.Description,
		Document:    updatedPolicy.Document,
		Version:     updatedPolicy.Version,
		Status:      updatedPolicy.Status,
		IsDefault:   updatedPolicy.IsDefault,
		IsSystem:    updatedPolicy.IsSystem,
		CreatedAt:   updatedPolicy.CreatedAt,
		UpdatedAt:   updatedPolicy.UpdatedAt,
	}, nil
}

func (s *policyService) DeleteByUUID(policyUUID uuid.UUID) (*PolicyServiceDataResult, error) {
	var deletedPolicy *model.Policy

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPolicyRepo := s.policyRepo.WithTx(tx)

		// Check if policy exists
		policy, err := txPolicyRepo.FindByUUID(policyUUID)
		if err != nil {
			return err
		}
		if policy == nil {
			return errors.New("policy not found")
		}

		// Check if policy is system policy (cannot be deleted)
		if policy.IsSystem {
			return errors.New("system policies cannot be deleted")
		}

		deletedPolicy = policy

		// Delete policy
		if err := txPolicyRepo.DeleteByUUID(policyUUID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &PolicyServiceDataResult{
		PolicyUUID:  deletedPolicy.PolicyUUID,
		Name:        deletedPolicy.Name,
		Description: deletedPolicy.Description,
		Document:    deletedPolicy.Document,
		Version:     deletedPolicy.Version,
		Status:      deletedPolicy.Status,
		IsDefault:   deletedPolicy.IsDefault,
		IsSystem:    deletedPolicy.IsSystem,
		CreatedAt:   deletedPolicy.CreatedAt,
		UpdatedAt:   deletedPolicy.UpdatedAt,
	}, nil
}
