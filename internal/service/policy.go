package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PolicyServiceGetFilter struct {
	TenantID    int64
	Name        *string
	Description *string
	Version     *string
	Status      []string
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

type PolicyServiceServiceDataResult struct {
	ServiceUUID uuid.UUID
	Name        string
	DisplayName string
	Description string
	Version     string
	Status      string
	IsSystem    bool
	APICount    int64
	PolicyCount int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PolicyServiceServicesFilter struct {
	Name        *string
	DisplayName *string
	Description *string
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type PolicyServiceServicesResult struct {
	Data       []PolicyServiceServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type PolicyService interface {
	Get(ctx context.Context, filter PolicyServiceGetFilter) (*PolicyServiceGetResult, error)
	GetByUUID(ctx context.Context, policyUUID uuid.UUID, tenantID int64) (*PolicyServiceDataResult, error)
	GetServicesByPolicyUUID(ctx context.Context, policyUUID uuid.UUID, tenantID int64, filter PolicyServiceServicesFilter) (*PolicyServiceServicesResult, error)
	Create(ctx context.Context, tenantID int64, name string, description *string, document datatypes.JSON, version string, status string, isSystem bool) (*PolicyServiceDataResult, error)
	Update(ctx context.Context, policyUUID uuid.UUID, tenantID int64, name string, description *string, document datatypes.JSON, version string, status string) (*PolicyServiceDataResult, error)
	SetStatusByUUID(ctx context.Context, policyUUID uuid.UUID, tenantID int64, status string) (*PolicyServiceDataResult, error)
	DeleteByUUID(ctx context.Context, policyUUID uuid.UUID, tenantID int64) (*PolicyServiceDataResult, error)
}

type policyService struct {
	db          *gorm.DB
	policyRepo  repository.PolicyRepository
	serviceRepo repository.ServiceRepository
	apiRepo     repository.APIRepository
}

func NewPolicyService(
	db *gorm.DB,
	policyRepo repository.PolicyRepository,
	serviceRepo repository.ServiceRepository,
	apiRepo repository.APIRepository,
) PolicyService {
	return &policyService{
		db:          db,
		policyRepo:  policyRepo,
		serviceRepo: serviceRepo,
		apiRepo:     apiRepo,
	}
}

func (s *policyService) Get(ctx context.Context, filter PolicyServiceGetFilter) (*PolicyServiceGetResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "policy.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", filter.TenantID))
	repoFilter := repository.PolicyRepositoryGetFilter{
		TenantID:    filter.TenantID,
		Name:        filter.Name,
		Description: filter.Description,
		Version:     filter.Version,
		Status:      filter.Status,
		IsSystem:    filter.IsSystem,
		ServiceID:   filter.ServiceID,
		Page:        filter.Page,
		Limit:       filter.Limit,
		SortBy:      filter.SortBy,
		SortOrder:   filter.SortOrder,
	}

	result, err := s.policyRepo.FindPaginated(repoFilter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list policies failed")
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

func (s *policyService) GetByUUID(ctx context.Context, policyUUID uuid.UUID, tenantID int64) (*PolicyServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "policy.get")
	defer span.End()
	span.SetAttributes(attribute.String("policy.uuid", policyUUID.String()), attribute.Int64("tenant.id", tenantID))
	policy, err := s.policyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get policy failed")
		return nil, err
	}
	if policy == nil {
		return nil, apperror.NewNotFound("policy")
	}

	return &PolicyServiceDataResult{
		PolicyUUID:  policy.PolicyUUID,
		Name:        policy.Name,
		Description: policy.Description,
		Document:    policy.Document,
		Version:     policy.Version,
		Status:      policy.Status,
		IsSystem:    policy.IsSystem,
		CreatedAt:   policy.CreatedAt,
		UpdatedAt:   policy.UpdatedAt,
	}, nil
}

func (s *policyService) GetServicesByPolicyUUID(ctx context.Context, policyUUID uuid.UUID, tenantID int64, filter PolicyServiceServicesFilter) (*PolicyServiceServicesResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "policy.getServices")
	defer span.End()
	span.SetAttributes(attribute.String("policy.uuid", policyUUID.String()), attribute.Int64("tenant.id", tenantID))
	// First check if policy exists and belongs to tenant
	_, err := s.policyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get services by policy failed")
		return nil, err
	}

	// Convert filter to repository filter
	repoFilter := repository.ServiceRepositoryGetFilter{
		Name:        filter.Name,
		DisplayName: filter.DisplayName,
		Description: filter.Description,
		Page:        filter.Page,
		Limit:       filter.Limit,
		SortBy:      filter.SortBy,
		SortOrder:   filter.SortOrder,
	}

	// Get services that use this policy
	result, err := s.serviceRepo.FindServicesByPolicyUUID(policyUUID, repoFilter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get services by policy failed")
		return nil, err
	}

	// Convert to service data results
	var data []PolicyServiceServiceDataResult
	for _, service := range result.Data {
		// Get API count and policy count for each service, scoped to the caller's tenant
		apiCount, _ := s.apiRepo.CountByServiceID(service.ServiceID, tenantID)
		policyCount, _ := s.serviceRepo.CountPoliciesByServiceID(service.ServiceID)

		data = append(data, PolicyServiceServiceDataResult{
			ServiceUUID: service.ServiceUUID,
			Name:        service.Name,
			DisplayName: service.DisplayName,
			Description: service.Description,
			Version:     service.Version,
			Status:      service.Status,
			IsSystem:    service.IsSystem,
			APICount:    apiCount,
			PolicyCount: policyCount,
			CreatedAt:   service.CreatedAt,
			UpdatedAt:   service.UpdatedAt,
		})
	}

	return &PolicyServiceServicesResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *policyService) Create(ctx context.Context, tenantID int64, name string, description *string, document datatypes.JSON, version string, status string, isSystem bool) (*PolicyServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "policy.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))
	var createdPolicy *model.Policy

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPolicyRepo := s.policyRepo.WithTx(tx)

		// Check if policy with same name and version already exists
		existingPolicy, err := txPolicyRepo.FindByNameAndVersion(name, version, tenantID)
		if err != nil {
			return err
		}
		if existingPolicy != nil {
			return apperror.NewConflict("policy with name '" + name + "' and version '" + version + "' already exists")
		}

		// Create new policy
		policy := &model.Policy{
			PolicyUUID:  uuid.New(),
			TenantID:    tenantID,
			Name:        name,
			Description: description,
			Document:    document,
			Version:     version,
			Status:      status,
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "create policy failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return &PolicyServiceDataResult{
		PolicyUUID:  createdPolicy.PolicyUUID,
		Name:        createdPolicy.Name,
		Description: createdPolicy.Description,
		Document:    createdPolicy.Document,
		Version:     createdPolicy.Version,
		Status:      createdPolicy.Status,
		IsSystem:    createdPolicy.IsSystem,
		CreatedAt:   createdPolicy.CreatedAt,
		UpdatedAt:   createdPolicy.UpdatedAt,
	}, nil
}

func (s *policyService) Update(ctx context.Context, policyUUID uuid.UUID, tenantID int64, name string, description *string, document datatypes.JSON, version string, status string) (*PolicyServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "policy.update")
	defer span.End()
	span.SetAttributes(attribute.String("policy.uuid", policyUUID.String()), attribute.Int64("tenant.id", tenantID))
	var updatedPolicy *model.Policy

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPolicyRepo := s.policyRepo.WithTx(tx)

		// Check if policy exists and belongs to tenant
		policy, err := txPolicyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
		if err != nil {
			return err
		}
		if policy == nil {
			return apperror.NewNotFoundWithReason("policy not found or access denied")
		}

		// Check if policy is a system record (critical for app functionality)
		if policy.IsSystem {
			return apperror.NewValidation("system policy cannot be updated")
		}

		// Check if another policy with same name and version exists (excluding current policy)
		if policy.Name != name || policy.Version != version {
			existingPolicy, err := txPolicyRepo.FindByNameAndVersion(name, version, tenantID)
			if err != nil {
				return err
			}
			if existingPolicy != nil && existingPolicy.PolicyUUID != policyUUID {
				return apperror.NewConflict("policy with name '" + name + "' and version '" + version + "' already exists")
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "update policy failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return &PolicyServiceDataResult{
		PolicyUUID:  updatedPolicy.PolicyUUID,
		Name:        updatedPolicy.Name,
		Description: updatedPolicy.Description,
		Document:    updatedPolicy.Document,
		Version:     updatedPolicy.Version,
		Status:      updatedPolicy.Status,
		IsSystem:    updatedPolicy.IsSystem,
		CreatedAt:   updatedPolicy.CreatedAt,
		UpdatedAt:   updatedPolicy.UpdatedAt,
	}, nil
}

func (s *policyService) SetStatusByUUID(ctx context.Context, policyUUID uuid.UUID, tenantID int64, status string) (*PolicyServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "policy.setStatus")
	defer span.End()
	span.SetAttributes(attribute.String("policy.uuid", policyUUID.String()), attribute.Int64("tenant.id", tenantID))
	var updatedPolicy *model.Policy

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPolicyRepo := s.policyRepo.WithTx(tx)

		// Check if policy exists and belongs to tenant
		policy, err := txPolicyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
		if err != nil {
			return err
		}
		if policy == nil {
			return apperror.NewNotFoundWithReason("policy not found or access denied")
		}

		// Check if policy is a system record (critical for app functionality)
		if policy.IsSystem {
			return apperror.NewValidation("system policy status cannot be updated")
		}

		// Update status
		if err := txPolicyRepo.SetStatusByUUID(policyUUID, tenantID, status); err != nil {
			return err
		}

		// Get updated policy
		updatedPolicy, err = txPolicyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "set policy status failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return &PolicyServiceDataResult{
		PolicyUUID:  updatedPolicy.PolicyUUID,
		Name:        updatedPolicy.Name,
		Description: updatedPolicy.Description,
		Document:    updatedPolicy.Document,
		Version:     updatedPolicy.Version,
		Status:      updatedPolicy.Status,
		IsSystem:    updatedPolicy.IsSystem,
		CreatedAt:   updatedPolicy.CreatedAt,
		UpdatedAt:   updatedPolicy.UpdatedAt,
	}, nil
}

func (s *policyService) DeleteByUUID(ctx context.Context, policyUUID uuid.UUID, tenantID int64) (*PolicyServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "policy.delete")
	defer span.End()
	span.SetAttributes(attribute.String("policy.uuid", policyUUID.String()), attribute.Int64("tenant.id", tenantID))
	var deletedPolicy *model.Policy

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPolicyRepo := s.policyRepo.WithTx(tx)

		// Check if policy exists and belongs to tenant
		policy, err := txPolicyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
		if err != nil {
			return err
		}
		if policy == nil {
			return apperror.NewNotFoundWithReason("policy not found or access denied")
		}

		// Check if policy is system policy (cannot be deleted)
		if policy.IsSystem {
			return apperror.NewValidation("system policies cannot be deleted")
		}

		deletedPolicy = policy

		// Delete policy
		if err := txPolicyRepo.DeleteByUUIDAndTenantID(policyUUID, tenantID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete policy failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return &PolicyServiceDataResult{
		PolicyUUID:  deletedPolicy.PolicyUUID,
		Name:        deletedPolicy.Name,
		Description: deletedPolicy.Description,
		Document:    deletedPolicy.Document,
		Version:     deletedPolicy.Version,
		Status:      deletedPolicy.Status,
		IsSystem:    deletedPolicy.IsSystem,
		CreatedAt:   deletedPolicy.CreatedAt,
		UpdatedAt:   deletedPolicy.UpdatedAt,
	}, nil
}
