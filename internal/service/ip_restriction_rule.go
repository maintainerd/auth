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
	"gorm.io/gorm"
)

// IPRestrictionRuleServiceDataResult is the service-layer representation of a
// single IP restriction rule.
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

// IPRestrictionRuleServiceListResult is the paginated result returned by
// listing IP restriction rules.
type IPRestrictionRuleServiceListResult struct {
	Data       []IPRestrictionRuleServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

// IPRestrictionRuleService defines business operations on IP restriction rules.
type IPRestrictionRuleService interface {
	GetAll(ctx context.Context, tenantID int64, ruleType *string, status []string, ipAddress, description *string, page, limit int, sortBy, sortOrder string) (*IPRestrictionRuleServiceListResult, error)
	GetByUUID(ctx context.Context, tenantID int64, ipRestrictionRuleUUID uuid.UUID) (*IPRestrictionRuleServiceDataResult, error)
	Create(ctx context.Context, tenantID int64, description, ruleType, ipAddress, status string, createdBy int64) (*IPRestrictionRuleServiceDataResult, error)
	Update(ctx context.Context, tenantID int64, ipRestrictionRuleUUID uuid.UUID, description, ruleType, ipAddress, status string, updatedBy int64) (*IPRestrictionRuleServiceDataResult, error)
	UpdateStatus(ctx context.Context, tenantID int64, ipRestrictionRuleUUID uuid.UUID, status string, updatedBy int64) (*IPRestrictionRuleServiceDataResult, error)
	Delete(ctx context.Context, tenantID int64, ipRestrictionRuleUUID uuid.UUID) (*IPRestrictionRuleServiceDataResult, error)
}

type ipRestrictionRuleService struct {
	db                    *gorm.DB
	ipRestrictionRuleRepo repository.IPRestrictionRuleRepository
}

// NewIPRestrictionRuleService creates a new IPRestrictionRuleService.
func NewIPRestrictionRuleService(
	db *gorm.DB,
	ipRestrictionRuleRepo repository.IPRestrictionRuleRepository,
) IPRestrictionRuleService {
	return &ipRestrictionRuleService{
		db:                    db,
		ipRestrictionRuleRepo: ipRestrictionRuleRepo,
	}
}

func (s *ipRestrictionRuleService) GetAll(ctx context.Context, tenantID int64, ruleType *string, status []string, ipAddress, description *string, page, limit int, sortBy, sortOrder string) (*IPRestrictionRuleServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "ipRestrictionRule.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to list ip restriction rules")
		return nil, err
	}

	data := make([]IPRestrictionRuleServiceDataResult, len(result.Data))
	for i, rule := range result.Data {
		data[i] = toIPRestrictionRuleServiceDataResult(&rule)
	}

	span.SetStatus(codes.Ok, "")
	return &IPRestrictionRuleServiceListResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *ipRestrictionRuleService) GetByUUID(ctx context.Context, tenantID int64, ipRestrictionRuleUUID uuid.UUID) (*IPRestrictionRuleServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "ipRestrictionRule.get")
	defer span.End()
	span.SetAttributes(
		attribute.String("ip_rule.uuid", ipRestrictionRuleUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	rule, err := s.ipRestrictionRuleRepo.FindByUUID(ipRestrictionRuleUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch ip restriction rule")
		return nil, err
	}

	if rule == nil {
		span.SetStatus(codes.Error, "ip restriction rule not found")
		return nil, apperror.NewNotFoundWithReason("ip restriction rule not found")
	}

	// Verify tenant ownership
	if rule.TenantID != tenantID {
		span.SetStatus(codes.Error, "ip restriction rule not found")
		return nil, apperror.NewNotFoundWithReason("ip restriction rule not found")
	}

	span.SetStatus(codes.Ok, "")
	result := toIPRestrictionRuleServiceDataResult(rule)
	return &result, nil
}

func (s *ipRestrictionRuleService) Create(ctx context.Context, tenantID int64, description, ruleType, ipAddress, status string, createdBy int64) (*IPRestrictionRuleServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "ipRestrictionRule.create")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("tenant.id", tenantID),
		attribute.String("ip_rule.type", ruleType),
	)

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
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create ip restriction rule")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	result := toIPRestrictionRuleServiceDataResult(createdRule)
	return &result, nil
}

func (s *ipRestrictionRuleService) Update(ctx context.Context, tenantID int64, ipRestrictionRuleUUID uuid.UUID, description, ruleType, ipAddress, status string, updatedBy int64) (*IPRestrictionRuleServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "ipRestrictionRule.update")
	defer span.End()
	span.SetAttributes(
		attribute.String("ip_rule.uuid", ipRestrictionRuleUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	rule, err := s.ipRestrictionRuleRepo.FindByUUID(ipRestrictionRuleUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch ip restriction rule")
		return nil, err
	}

	if rule == nil {
		span.SetStatus(codes.Error, "ip restriction rule not found")
		return nil, apperror.NewNotFoundWithReason("ip restriction rule not found")
	}

	// Verify tenant ownership
	if rule.TenantID != tenantID {
		span.SetStatus(codes.Error, "ip restriction rule not found")
		return nil, apperror.NewNotFoundWithReason("ip restriction rule not found")
	}

	rule.Description = description
	rule.Type = ruleType
	rule.IPAddress = ipAddress
	rule.Status = status
	rule.UpdatedBy = &updatedBy

	updatedRule, err := s.ipRestrictionRuleRepo.UpdateByUUID(ipRestrictionRuleUUID, rule)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update ip restriction rule")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	result := toIPRestrictionRuleServiceDataResult(updatedRule)
	return &result, nil
}

func (s *ipRestrictionRuleService) UpdateStatus(ctx context.Context, tenantID int64, ipRestrictionRuleUUID uuid.UUID, status string, updatedBy int64) (*IPRestrictionRuleServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "ipRestrictionRule.updateStatus")
	defer span.End()
	span.SetAttributes(
		attribute.String("ip_rule.uuid", ipRestrictionRuleUUID.String()),
		attribute.Int64("tenant.id", tenantID),
		attribute.String("ip_rule.status", status),
	)

	rule, err := s.ipRestrictionRuleRepo.FindByUUID(ipRestrictionRuleUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch ip restriction rule")
		return nil, err
	}

	if rule == nil {
		span.SetStatus(codes.Error, "ip restriction rule not found")
		return nil, apperror.NewNotFoundWithReason("ip restriction rule not found")
	}

	// Verify tenant ownership
	if rule.TenantID != tenantID {
		span.SetStatus(codes.Error, "ip restriction rule not found")
		return nil, apperror.NewNotFoundWithReason("ip restriction rule not found")
	}

	rule.Status = status
	rule.UpdatedBy = &updatedBy

	updatedRule, err := s.ipRestrictionRuleRepo.UpdateByUUID(ipRestrictionRuleUUID, rule)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update ip restriction rule status")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	result := toIPRestrictionRuleServiceDataResult(updatedRule)
	return &result, nil
}

func (s *ipRestrictionRuleService) Delete(ctx context.Context, tenantID int64, ipRestrictionRuleUUID uuid.UUID) (*IPRestrictionRuleServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "ipRestrictionRule.delete")
	defer span.End()
	span.SetAttributes(
		attribute.String("ip_rule.uuid", ipRestrictionRuleUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	rule, err := s.ipRestrictionRuleRepo.FindByUUID(ipRestrictionRuleUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch ip restriction rule")
		return nil, err
	}

	if rule == nil {
		span.SetStatus(codes.Error, "ip restriction rule not found")
		return nil, apperror.NewNotFoundWithReason("ip restriction rule not found")
	}

	// Verify tenant ownership
	if rule.TenantID != tenantID {
		span.SetStatus(codes.Error, "ip restriction rule not found")
		return nil, apperror.NewNotFoundWithReason("ip restriction rule not found")
	}

	if err := s.ipRestrictionRuleRepo.DeleteByUUID(ipRestrictionRuleUUID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete ip restriction rule")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	result := toIPRestrictionRuleServiceDataResult(rule)
	return &result, nil
}

// toIPRestrictionRuleServiceDataResult converts a model.IPRestrictionRule into
// its service-layer representation.
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
