package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/gorm"
)

type IpRestrictionRuleServiceDataResult struct {
	IpRestrictionRuleUUID uuid.UUID
	TenantID              int64
	Description           string
	Type                  string
	IpAddress             string
	Status                string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type IpRestrictionRuleServiceListResult struct {
	Data       []IpRestrictionRuleServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type IpRestrictionRuleService interface {
	GetAll(tenantID int64, ruleType *string, status []string, ipAddress, description *string, page, limit int, sortBy, sortOrder string) (*IpRestrictionRuleServiceListResult, error)
	GetByUUID(tenantID int64, ipRestrictionRuleUUID uuid.UUID) (*IpRestrictionRuleServiceDataResult, error)
	Create(tenantID int64, description, ruleType, ipAddress, status string, createdBy int64) (*IpRestrictionRuleServiceDataResult, error)
	Update(tenantID int64, ipRestrictionRuleUUID uuid.UUID, description, ruleType, ipAddress, status string, updatedBy int64) (*IpRestrictionRuleServiceDataResult, error)
	UpdateStatus(tenantID int64, ipRestrictionRuleUUID uuid.UUID, status string, updatedBy int64) (*IpRestrictionRuleServiceDataResult, error)
	Delete(tenantID int64, ipRestrictionRuleUUID uuid.UUID) (*IpRestrictionRuleServiceDataResult, error)
}

type ipRestrictionRuleService struct {
	db                    *gorm.DB
	ipRestrictionRuleRepo repository.IpRestrictionRuleRepository
}

func NewIpRestrictionRuleService(
	db *gorm.DB,
	ipRestrictionRuleRepo repository.IpRestrictionRuleRepository,
) IpRestrictionRuleService {
	return &ipRestrictionRuleService{
		db:                    db,
		ipRestrictionRuleRepo: ipRestrictionRuleRepo,
	}
}

func (s *ipRestrictionRuleService) GetAll(tenantID int64, ruleType *string, status []string, ipAddress, description *string, page, limit int, sortBy, sortOrder string) (*IpRestrictionRuleServiceListResult, error) {
	filter := repository.IpRestrictionRuleRepositoryGetFilter{
		TenantID:    &tenantID,
		Type:        ruleType,
		Status:      status,
		IpAddress:   ipAddress,
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

	data := make([]IpRestrictionRuleServiceDataResult, len(result.Data))
	for i, rule := range result.Data {
		data[i] = toIpRestrictionRuleServiceDataResult(&rule)
	}

	return &IpRestrictionRuleServiceListResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *ipRestrictionRuleService) GetByUUID(tenantID int64, ipRestrictionRuleUUID uuid.UUID) (*IpRestrictionRuleServiceDataResult, error) {
	rule, err := s.ipRestrictionRuleRepo.FindByUUID(ipRestrictionRuleUUID)
	if err != nil {
		return nil, err
	}

	if rule == nil {
		return nil, errors.New("ip restriction rule not found")
	}

	// Verify tenant ownership
	if rule.TenantID != tenantID {
		return nil, errors.New("ip restriction rule not found")
	}

	result := toIpRestrictionRuleServiceDataResult(rule)
	return &result, nil
}

func (s *ipRestrictionRuleService) Create(tenantID int64, description, ruleType, ipAddress, status string, createdBy int64) (*IpRestrictionRuleServiceDataResult, error) {
	rule := &model.IpRestrictionRule{
		TenantID:    tenantID,
		Description: description,
		Type:        ruleType,
		IpAddress:   ipAddress,
		Status:      status,
		CreatedBy:   &createdBy,
	}

	createdRule, err := s.ipRestrictionRuleRepo.Create(rule)
	if err != nil {
		return nil, err
	}

	result := toIpRestrictionRuleServiceDataResult(createdRule)
	return &result, nil
}

func (s *ipRestrictionRuleService) Update(tenantID int64, ipRestrictionRuleUUID uuid.UUID, description, ruleType, ipAddress, status string, updatedBy int64) (*IpRestrictionRuleServiceDataResult, error) {
	rule, err := s.ipRestrictionRuleRepo.FindByUUID(ipRestrictionRuleUUID)
	if err != nil {
		return nil, err
	}

	if rule == nil {
		return nil, errors.New("ip restriction rule not found")
	}

	// Verify tenant ownership
	if rule.TenantID != tenantID {
		return nil, errors.New("ip restriction rule not found")
	}

	rule.Description = description
	rule.Type = ruleType
	rule.IpAddress = ipAddress
	rule.Status = status
	rule.UpdatedBy = &updatedBy

	updatedRule, err := s.ipRestrictionRuleRepo.UpdateByUUID(ipRestrictionRuleUUID, rule)
	if err != nil {
		return nil, err
	}

	result := toIpRestrictionRuleServiceDataResult(updatedRule)
	return &result, nil
}

func (s *ipRestrictionRuleService) UpdateStatus(tenantID int64, ipRestrictionRuleUUID uuid.UUID, status string, updatedBy int64) (*IpRestrictionRuleServiceDataResult, error) {
	rule, err := s.ipRestrictionRuleRepo.FindByUUID(ipRestrictionRuleUUID)
	if err != nil {
		return nil, err
	}

	if rule == nil {
		return nil, errors.New("ip restriction rule not found")
	}

	// Verify tenant ownership
	if rule.TenantID != tenantID {
		return nil, errors.New("ip restriction rule not found")
	}

	rule.Status = status
	rule.UpdatedBy = &updatedBy

	updatedRule, err := s.ipRestrictionRuleRepo.UpdateByUUID(ipRestrictionRuleUUID, rule)
	if err != nil {
		return nil, err
	}

	result := toIpRestrictionRuleServiceDataResult(updatedRule)
	return &result, nil
}

func (s *ipRestrictionRuleService) Delete(tenantID int64, ipRestrictionRuleUUID uuid.UUID) (*IpRestrictionRuleServiceDataResult, error) {
	rule, err := s.ipRestrictionRuleRepo.FindByUUID(ipRestrictionRuleUUID)
	if err != nil {
		return nil, err
	}

	if rule == nil {
		return nil, errors.New("ip restriction rule not found")
	}

	// Verify tenant ownership
	if rule.TenantID != tenantID {
		return nil, errors.New("ip restriction rule not found")
	}

	if err := s.ipRestrictionRuleRepo.DeleteByUUID(ipRestrictionRuleUUID); err != nil {
		return nil, err
	}

	result := toIpRestrictionRuleServiceDataResult(rule)
	return &result, nil
}

// Helper functions
func toIpRestrictionRuleServiceDataResult(rule *model.IpRestrictionRule) IpRestrictionRuleServiceDataResult {
	return IpRestrictionRuleServiceDataResult{
		IpRestrictionRuleUUID: rule.IpRestrictionRuleUUID,
		TenantID:              rule.TenantID,
		Description:           rule.Description,
		Type:                  rule.Type,
		IpAddress:             rule.IpAddress,
		Status:                rule.Status,
		CreatedAt:             rule.CreatedAt,
		UpdatedAt:             rule.UpdatedAt,
	}
}
