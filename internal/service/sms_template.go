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

type SMSTemplateServiceDataResult struct {
	SMSTemplateUUID uuid.UUID
	Name            string
	Description     *string
	Message         string
	SenderID        *string
	Status          string
	IsDefault       bool
	IsSystem        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type SMSTemplateServiceListResult struct {
	Data       []SMSTemplateServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type SMSTemplateService interface {
	GetAll(ctx context.Context, tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*SMSTemplateServiceListResult, error)
	GetByUUID(ctx context.Context, uuid uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error)
	Create(ctx context.Context, tenantID int64, name string, description *string, message string, senderID *string, status string) (*SMSTemplateServiceDataResult, error)
	Update(ctx context.Context, uuid uuid.UUID, tenantID int64, name string, description *string, message string, senderID *string, status string) (*SMSTemplateServiceDataResult, error)
	UpdateStatus(ctx context.Context, uuid uuid.UUID, tenantID int64, status string) (*SMSTemplateServiceDataResult, error)
	Delete(ctx context.Context, uuid uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error)
}

type smsTemplateService struct {
	db              *gorm.DB
	smsTemplateRepo repository.SMSTemplateRepository
}

func NewSMSTemplateService(
	db *gorm.DB,
	smsTemplateRepo repository.SMSTemplateRepository,
) SMSTemplateService {
	return &smsTemplateService{
		db:              db,
		smsTemplateRepo: smsTemplateRepo,
	}
}

func (s *smsTemplateService) GetAll(ctx context.Context, tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*SMSTemplateServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "smsTemplate.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	filter := repository.SMSTemplateRepositoryGetFilter{
		TenantID:  &tenantID,
		Name:      name,
		Status:    status,
		IsDefault: isDefault,
		IsSystem:  isSystem,
		Page:      page,
		Limit:     limit,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	result, err := s.smsTemplateRepo.FindPaginated(filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list sms templates failed")
		return nil, err
	}

	data := make([]SMSTemplateServiceDataResult, len(result.Data))
	for i, template := range result.Data {
		data[i] = toSMSTemplateServiceDataResult(&template)
	}

	span.SetStatus(codes.Ok, "")
	return &SMSTemplateServiceListResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *smsTemplateService) GetByUUID(ctx context.Context, smsTemplateUUID uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "smsTemplate.getByUUID")
	defer span.End()
	span.SetAttributes(attribute.String("smsTemplate.uuid", smsTemplateUUID.String()), attribute.Int64("tenant.id", tenantID))

	template, err := s.smsTemplateRepo.FindByUUIDAndTenantID(smsTemplateUUID.String(), tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get sms template failed")
		return nil, err
	}

	if template == nil {
		span.SetStatus(codes.Error, "sms template not found or access denied")
		return nil, apperror.NewNotFoundWithReason("SMS template not found or access denied")
	}

	span.SetStatus(codes.Ok, "")
	result := toSMSTemplateServiceDataResult(template)
	return &result, nil
}

func (s *smsTemplateService) Create(ctx context.Context, tenantID int64, name string, description *string, message string, senderID *string, status string) (*SMSTemplateServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "smsTemplate.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	template := &model.SMSTemplate{
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		Message:     message,
		SenderID:    senderID,
		Status:      status,
		IsDefault:   false,
		IsSystem:    false,
	}

	createdTemplate, err := s.smsTemplateRepo.Create(template)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create sms template failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	result := toSMSTemplateServiceDataResult(createdTemplate)
	return &result, nil
}

func (s *smsTemplateService) Update(ctx context.Context, smsTemplateUUID uuid.UUID, tenantID int64, name string, description *string, message string, senderID *string, status string) (*SMSTemplateServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "smsTemplate.update")
	defer span.End()
	span.SetAttributes(attribute.String("smsTemplate.uuid", smsTemplateUUID.String()), attribute.Int64("tenant.id", tenantID))

	template, err := s.smsTemplateRepo.FindByUUIDAndTenantID(smsTemplateUUID.String(), tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update sms template failed")
		return nil, err
	}

	if template == nil {
		span.SetStatus(codes.Error, "sms template not found or access denied")
		return nil, apperror.NewNotFoundWithReason("SMS template not found or access denied")
	}

	// Prevent updating system templates
	if template.IsSystem {
		span.SetStatus(codes.Error, "cannot update system SMS template")
		return nil, apperror.NewValidation("cannot update system SMS template")
	}

	template.Name = name
	template.Description = description
	template.Message = message
	template.SenderID = senderID
	template.Status = status
	// is_default is preserved from existing template

	updatedTemplate, err := s.smsTemplateRepo.UpdateByUUID(smsTemplateUUID, template)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update sms template failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	result := toSMSTemplateServiceDataResult(updatedTemplate)
	return &result, nil
}

func (s *smsTemplateService) UpdateStatus(ctx context.Context, smsTemplateUUID uuid.UUID, tenantID int64, status string) (*SMSTemplateServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "smsTemplate.updateStatus")
	defer span.End()
	span.SetAttributes(attribute.String("smsTemplate.uuid", smsTemplateUUID.String()), attribute.Int64("tenant.id", tenantID))

	template, err := s.smsTemplateRepo.FindByUUIDAndTenantID(smsTemplateUUID.String(), tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update sms template status failed")
		return nil, err
	}

	if template == nil {
		span.SetStatus(codes.Error, "sms template not found or access denied")
		return nil, apperror.NewNotFoundWithReason("SMS template not found or access denied")
	}

	// Prevent updating system templates
	if template.IsSystem {
		span.SetStatus(codes.Error, "cannot update status of system SMS template")
		return nil, apperror.NewValidation("cannot update status of system SMS template")
	}

	template.Status = status

	updatedTemplate, err := s.smsTemplateRepo.UpdateByUUID(smsTemplateUUID, template)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update sms template status failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	result := toSMSTemplateServiceDataResult(updatedTemplate)
	return &result, nil
}

func (s *smsTemplateService) Delete(ctx context.Context, smsTemplateUUID uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "smsTemplate.delete")
	defer span.End()
	span.SetAttributes(attribute.String("smsTemplate.uuid", smsTemplateUUID.String()), attribute.Int64("tenant.id", tenantID))

	template, err := s.smsTemplateRepo.FindByUUIDAndTenantID(smsTemplateUUID.String(), tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete sms template failed")
		return nil, err
	}

	if template == nil {
		span.SetStatus(codes.Error, "sms template not found or access denied")
		return nil, apperror.NewNotFoundWithReason("SMS template not found or access denied")
	}

	// Prevent deleting system templates
	if template.IsSystem {
		span.SetStatus(codes.Error, "cannot delete system SMS template")
		return nil, apperror.NewValidation("cannot delete system SMS template")
	}

	err = s.smsTemplateRepo.DeleteByUUID(smsTemplateUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete sms template failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	result := toSMSTemplateServiceDataResult(template)
	return &result, nil
}

// Helper function to convert model to service data result
func toSMSTemplateServiceDataResult(template *model.SMSTemplate) SMSTemplateServiceDataResult {
	return SMSTemplateServiceDataResult{
		SMSTemplateUUID: template.SMSTemplateUUID,
		Name:            template.Name,
		Description:     template.Description,
		Message:         template.Message,
		SenderID:        template.SenderID,
		Status:          template.Status,
		IsDefault:       template.IsDefault,
		IsSystem:        template.IsSystem,
		CreatedAt:       template.CreatedAt,
		UpdatedAt:       template.UpdatedAt,
	}
}
