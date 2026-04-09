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

type EmailTemplateServiceDataResult struct {
	EmailTemplateUUID uuid.UUID
	Name              string
	Subject           string
	BodyHTML          string
	BodyPlain         *string
	Status            string
	IsDefault         bool
	IsSystem          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type EmailTemplateServiceListResult struct {
	Data       []EmailTemplateServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type EmailTemplateService interface {
	GetAll(ctx context.Context, tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*EmailTemplateServiceListResult, error)
	GetByUUID(ctx context.Context, emailTemplateUUID uuid.UUID, tenantID int64) (*EmailTemplateServiceDataResult, error)
	Create(ctx context.Context, tenantID int64, name, subject, bodyHTML string, bodyPlain *string, status string, isDefault bool) (*EmailTemplateServiceDataResult, error)
	Update(ctx context.Context, emailTemplateUUID uuid.UUID, tenantID int64, name, subject, bodyHTML string, bodyPlain *string, status string) (*EmailTemplateServiceDataResult, error)
	UpdateStatus(ctx context.Context, emailTemplateUUID uuid.UUID, tenantID int64, status string) (*EmailTemplateServiceDataResult, error)
	Delete(ctx context.Context, emailTemplateUUID uuid.UUID, tenantID int64) (*EmailTemplateServiceDataResult, error)
}

type emailTemplateService struct {
	db                *gorm.DB
	emailTemplateRepo repository.EmailTemplateRepository
}

func NewEmailTemplateService(
	db *gorm.DB,
	emailTemplateRepo repository.EmailTemplateRepository,
) EmailTemplateService {
	return &emailTemplateService{
		db:                db,
		emailTemplateRepo: emailTemplateRepo,
	}
}

func (s *emailTemplateService) GetAll(ctx context.Context, tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*EmailTemplateServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "emailTemplate.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	filter := repository.EmailTemplateRepositoryGetFilter{
		Name:      name,
		Status:    status,
		TenantID:  &tenantID,
		IsDefault: isDefault,
		IsSystem:  isSystem,
		Page:      page,
		Limit:     limit,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	result, err := s.emailTemplateRepo.FindPaginated(filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to list email templates")
		return nil, err
	}

	data := make([]EmailTemplateServiceDataResult, len(result.Data))
	for i, template := range result.Data {
		data[i] = toEmailTemplateServiceDataResult(&template)
	}

	span.SetStatus(codes.Ok, "")
	return &EmailTemplateServiceListResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *emailTemplateService) GetByUUID(ctx context.Context, emailTemplateUUID uuid.UUID, tenantID int64) (*EmailTemplateServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "emailTemplate.get")
	defer span.End()
	span.SetAttributes(
		attribute.String("email_template.uuid", emailTemplateUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	template, err := s.emailTemplateRepo.FindByUUIDAndTenantID(emailTemplateUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch email template")
		return nil, err
	}

	if template == nil {
		span.SetStatus(codes.Error, "email template not found")
		return nil, apperror.NewNotFoundWithReason("email template not found or access denied")
	}

	span.SetStatus(codes.Ok, "")
	result := toEmailTemplateServiceDataResult(template)
	return &result, nil
}

func (s *emailTemplateService) Create(ctx context.Context, tenantID int64, name, subject, bodyHTML string, bodyPlain *string, status string, isDefault bool) (*EmailTemplateServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "emailTemplate.create")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("tenant.id", tenantID),
		attribute.String("email_template.name", name),
	)

	template := &model.EmailTemplate{
		TenantID:  tenantID,
		Name:      name,
		Subject:   subject,
		BodyHTML:  bodyHTML,
		BodyPlain: bodyPlain,
		Status:    status,
		IsDefault: isDefault,
		IsSystem:  false,
	}

	createdTemplate, err := s.emailTemplateRepo.Create(template)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create email template")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	result := toEmailTemplateServiceDataResult(createdTemplate)
	return &result, nil
}

func (s *emailTemplateService) Update(ctx context.Context, emailTemplateUUID uuid.UUID, tenantID int64, name, subject, bodyHTML string, bodyPlain *string, status string) (*EmailTemplateServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "emailTemplate.update")
	defer span.End()
	span.SetAttributes(
		attribute.String("email_template.uuid", emailTemplateUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	template, err := s.emailTemplateRepo.FindByUUIDAndTenantID(emailTemplateUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch email template")
		return nil, err
	}

	if template == nil {
		span.SetStatus(codes.Error, "email template not found")
		return nil, apperror.NewNotFoundWithReason("email template not found or access denied")
	}

	// Prevent updating system templates
	if template.IsSystem {
		span.SetStatus(codes.Error, "cannot update system email template")
		return nil, apperror.NewValidation("cannot update system email template")
	}

	template.Name = name
	template.Subject = subject
	template.BodyHTML = bodyHTML
	template.BodyPlain = bodyPlain
	template.Status = status
	// is_default is preserved from existing template

	updatedTemplate, err := s.emailTemplateRepo.UpdateByUUID(emailTemplateUUID, template)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update email template")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	result := toEmailTemplateServiceDataResult(updatedTemplate)
	return &result, nil
}

func (s *emailTemplateService) UpdateStatus(ctx context.Context, emailTemplateUUID uuid.UUID, tenantID int64, status string) (*EmailTemplateServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "emailTemplate.updateStatus")
	defer span.End()
	span.SetAttributes(
		attribute.String("email_template.uuid", emailTemplateUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	template, err := s.emailTemplateRepo.FindByUUIDAndTenantID(emailTemplateUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch email template")
		return nil, err
	}

	if template == nil {
		span.SetStatus(codes.Error, "email template not found")
		return nil, apperror.NewNotFoundWithReason("email template not found or access denied")
	}

	// Prevent updating system templates
	if template.IsSystem {
		span.SetStatus(codes.Error, "cannot update status of system email template")
		return nil, apperror.NewValidation("cannot update status of system email template")
	}

	template.Status = status

	updatedTemplate, err := s.emailTemplateRepo.UpdateByUUID(emailTemplateUUID, template)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update email template status")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	result := toEmailTemplateServiceDataResult(updatedTemplate)
	return &result, nil
}

func (s *emailTemplateService) Delete(ctx context.Context, emailTemplateUUID uuid.UUID, tenantID int64) (*EmailTemplateServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "emailTemplate.delete")
	defer span.End()
	span.SetAttributes(
		attribute.String("email_template.uuid", emailTemplateUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	template, err := s.emailTemplateRepo.FindByUUIDAndTenantID(emailTemplateUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch email template")
		return nil, err
	}

	if template == nil {
		span.SetStatus(codes.Error, "email template not found")
		return nil, apperror.NewNotFoundWithReason("email template not found or access denied")
	}

	// Prevent deleting system templates
	if template.IsSystem {
		span.SetStatus(codes.Error, "cannot delete system email template")
		return nil, apperror.NewValidation("cannot delete system email template")
	}

	err = s.emailTemplateRepo.DeleteByUUID(emailTemplateUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete email template")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	result := toEmailTemplateServiceDataResult(template)
	return &result, nil
}

// Helper function to convert model to service data result
func toEmailTemplateServiceDataResult(template *model.EmailTemplate) EmailTemplateServiceDataResult {
	return EmailTemplateServiceDataResult{
		EmailTemplateUUID: template.EmailTemplateUUID,
		Name:              template.Name,
		Subject:           template.Subject,
		BodyHTML:          template.BodyHTML,
		BodyPlain:         template.BodyPlain,
		Status:            template.Status,
		IsDefault:         template.IsDefault,
		IsSystem:          template.IsSystem,
		CreatedAt:         template.CreatedAt,
		UpdatedAt:         template.UpdatedAt,
	}
}
