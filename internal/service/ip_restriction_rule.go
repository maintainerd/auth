package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/gorm"
	"github.com/maintainerd/auth/internal/apperror"
)

type IPRestrictionRuleServiceDataResult struct {
	IPRestrictionRuleUUID uuid.UUID
	TenantID              int64
	Description           string
	Type                  string
	IPAddress             string
	Status                string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type IPRestrictionRuleServiceListResult struct {
	Data       []IPRestrictionRuleServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type IPRestrictionRuleService interface {
	GetAll(tenantID int64, ruleType *string, status []string, ipAddress, description *string, page, limit int, sortBy, sortOrder string) (*IPRestrictionRuleServiceListResult, error)
	GetByUUID(tenantID int64, ipRestrictionRuleUUID uuid.UUID) (*IPRestrictionRuleServiceDataResult, error)
	Create(tenantID int64, description, ruleType, ipAddress, status string, createdBy int64) (*IPRestrictionRuleServiceDataResult, error)
	Update(tenantID int64, ipRestrictionRuleUUID uuid.UUID, description, ruleType, ipAddress, status string, updatedBy int64) (*IPRestrictionRuleServiceDataResult, error)
	UpdateStatus(tenantID int64, ipRestrictionRuleUUID uuid.UUID, status string, updatedBy int64) (*IPRestrictionRuleServiceDataResult, error)
	Delete(tenantID int64, ipRestrictionRuleUUID uuid.UUID) (*IPRestrictionRuleServiceDataResult, error)
}

type ipRestrictionRuleService struct {
	db                    *gorm.DB
	ipRestrictionRuleRepo repository.IPRestrictionRuleRepository
}

func NewIPRestrictionRuleService(
	db *gorm.DB,
	ipRestrictionRuleRepo repository.IPRestrictionRuleRepository,
) IPRestrictionRuleService {
	return &ipRestrictionRuleService{
		db:                    db,
		ipRestrictionRuleRepo: ipRestrictionRuleRepo,
	}
}

func (s *ipRestrictionRuleService) GetAll(tenantID int64, ruleType *string, status []string, ipAddress, description *string, page, limit int, sortBy, sortOrder string) (*IPRestrictionRuleServiceListResult, error) {
	filter := repository.IPRestrictionRuleRepositoryGetFilter{
		TenantID:    &tenantID,
		Type:        ruleType,
		Status:      status,
		IPAddress:   ipAddress,
		Description: description,
		Page:        page,
		Limit:       limit,
		SortBy:      sortBy,
		SortOrder:   sortOrder,
	}

	result, err := s.ipRestrictionRuleRepo.FindPaginated(filter)
	if err != nil {
		return nil, err
	}

	data := make([]IPRestrictionRuleServiceDataResult, len(result.Data))
	for i, rule := range result.Data {
		data[i] = toIPRestrictionRuleServiceDataResult(&rule)
	}

	return &IPRestrictionRuleServiceListResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *ipRestrictionRuleService) GetByUUID(tenantID int64, ipRestrictionRuleUUID uuid.UUID) (*IPRestrictionRuleServiceDataResult, error) {
	rule, err := s.ipRestrictionRuleRepo.FindByUUID(ipRestrictionRuleUUID)
	if err != nil {
		return nil, err
	}

	if rule == nil {
		return nil, apperror.NewNotFoundWithReason("ip restriction rule not found")
	}

	// Verify tenant ownership
	if rule.TenantID != tenantID {
		return nil, apperror.NewNotFoundWithReason("ip restriction rule not found")
	}

	result := toIPRestrictionRuleServiceDataResult(rule)
	return &result, nil
}

func (s *ipRestrictionRuleService) Create(tenantID int64, description, ruleType, ipAddress, status string, createdBy int64) (*IPRestrictionRuleServiceDataResult, error) {
	rule := &model.IPRestrictionRule{
		TenantID:    tenantID,
		Description: description,
		Type:        ruleType,
		IPAddress:   ipAddress,
		Status:      status,
		CreatedBy:   &createdBy,
	}

	createdRule, err := s.ipRestrictionRuleRepo.Create(rule)
	if err != nil {
		return nil, err
	}

	result := toIPRestrictionRuleServiceDataResult(createdRule)
	return &result, nil
}

func (s *ipRestrictionRuleService) Update(tenantID int64, ipRestrictionRuleUUID uuid.UUID, description, ruleType, ipAddress, status string, updatedBy int64) (*IPRestrictionRuleServiceDataResult, error) {
	rule, err := s.ipRestrictionRuleRepo.FindByUUID(ipRestrictionRuleUUID)
	if err != nil {
		return nil, err
	}

	if rule == nil {
		return nil, apperror.NewNotFoundWithReason("ip restriction rule not found")
	}

	// Verify tenant ownership
	if rule.TenantID != tenantID {
		return nil, apperror.NewNotFoundWithReason("ip restriction rule not found")
	}

	rule.Description = description
	rule.Type = ruleType
	rule.IPAddress = ipAddress
	rule.Status = status
	rule.UpdatedBy = &updatedBy

	updatedRule, err := s.ipRestrictionRuleRepo.UpdateByUUID(ipRestrictionRuleUUID, rule)
	if err != nil {
		return nil, err
	}

	result := toIPRestrictionRuleServiceDataResult(updatedRule)
	return &result, nil
}

func (s *ipRestrictionRuleService) UpdateStatus(tenantID int64, ipRestrictionRuleUUID uuid.UUID, status string, updatedBy int64) (*IPRestrictionRuleServiceDataResult, error) {
	rule, err := s.ipRestrictionRuleRepo.FindByUUID(ipRestrictionRuleUUID)
	if err != nil {
		return nil, err
	}

	if rule == nil {
		return nil, apperror.NewNotFoundWithReason("ip restriction rule not found")
	}

	// Verify tenant ownership
	if rule.TenantID != tenantID {
		return nil, apperror.NewNotFoundWithReason("ip restriction rule not found")
	}

	rule.Status = status
	rule.UpdatedBy = &updatedBy

	updatedRule, err := s.ipRestrictionRuleRepo.UpdateByUUID(ipRestrictionRuleUUID, rule)
	if err != nil {
		return nil, err
	}

	result := toIPRestrictionRuleServiceDataResult(updatedRule)
	return &result, nil
}

func (s *ipRestrictionRuleService) Delete(tenantID int64, ipRestrictionRuleUUID uuid.UUID) (*IPRestrictionRuleServiceDataResult, error) {
	rule, err := s.ipRestrictionRuleRepo.FindByUUID(ipRestrictionRuleUUID)
	if err != nil {
		return nil, err
	}

	if rule == nil {
		return nil, apperror.NewNotFoundWithReason("ip restriction rule not found")
	}

	// Verify tenant ownership
	if rule.TenantID != tenantID {
		return nil, apperror.NewNotFoundWithReason("ip restriction rule not found")
	}

	if err := s.ipRestrictionRuleRepo.DeleteByUUID(ipRestrictionRuleUUID); err != nil {
		return nil, err
	}

	result := toIPRestrictionRuleServiceDataResult(rule)
	return &result, nil
}

// Helper functions
func toIPRestrictionRuleServiceDataResult(rule *model.IPRestrictionRule) IPRestrictionRuleServiceDataResult {
	return IPRestrictionRuleServiceDataResult{
		IPRestrictionRuleUUID: rule.IPRestrictionRuleUUID,
		TenantID:              rule.TenantID,
		Description:           rule.Description,
		Type:                  rule.Type,
		IPAddress:             rule.IPAddress,
		Status:                rule.Status,
		CreatedAt:             rule.CreatedAt,
		UpdatedAt:             rule.UpdatedAt,
	}
}
