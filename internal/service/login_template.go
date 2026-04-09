package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
)

type LoginTemplateServiceDataResult struct {
	LoginTemplateUUID uuid.UUID
	Name              string
	Description       *string
	Template          string
	Status            string
	Metadata          map[string]any
	IsDefault         bool
	IsSystem          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type LoginTemplateServiceListResult struct {
	Data       []LoginTemplateServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type LoginTemplateService interface {
	GetAll(ctx context.Context, tenantID int64, name *string, status []string, template *string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*LoginTemplateServiceListResult, error)
	GetByUUID(ctx context.Context, loginTemplateUUID uuid.UUID, tenantID int64) (*LoginTemplateServiceDataResult, error)
	Create(ctx context.Context, tenantID int64, name string, description *string, template string, metadata map[string]any, status string) (*LoginTemplateServiceDataResult, error)
	Update(ctx context.Context, loginTemplateUUID uuid.UUID, tenantID int64, name string, description *string, template string, metadata map[string]any, status string) (*LoginTemplateServiceDataResult, error)
	UpdateStatus(ctx context.Context, loginTemplateUUID uuid.UUID, tenantID int64, status string) (*LoginTemplateServiceDataResult, error)
	Delete(ctx context.Context, loginTemplateUUID uuid.UUID, tenantID int64) (*LoginTemplateServiceDataResult, error)
}

type loginTemplateService struct {
	loginTemplateRepo repository.LoginTemplateRepository
}

func NewLoginTemplateService(loginTemplateRepo repository.LoginTemplateRepository) LoginTemplateService {
	return &loginTemplateService{
		loginTemplateRepo: loginTemplateRepo,
	}
}

func (s *loginTemplateService) GetAll(ctx context.Context, tenantID int64, name *string, status []string, template *string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*LoginTemplateServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "loginTemplate.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))
	filter := repository.LoginTemplateRepositoryGetFilter{
		Name:      name,
		Status:    status,
		Template:  template,
		TenantID:  &tenantID,
		IsDefault: isDefault,
		IsSystem:  isSystem,
		Page:      page,
		Limit:     limit,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	result, err := s.loginTemplateRepo.FindPaginated(filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list login templates failed")
		return nil, err
	}

	dataResults := make([]LoginTemplateServiceDataResult, len(result.Data))
	for i, template := range result.Data {
		dataResults[i] = toLoginTemplateServiceDataResult(&template)
	}

	return &LoginTemplateServiceListResult{
		Data:       dataResults,
		Total:      int64(result.Total),
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *loginTemplateService) GetByUUID(ctx context.Context, loginTemplateUUID uuid.UUID, tenantID int64) (*LoginTemplateServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "loginTemplate.get")
	defer span.End()
	span.SetAttributes(attribute.String("login_template.uuid", loginTemplateUUID.String()), attribute.Int64("tenant.id", tenantID))
	template, err := s.loginTemplateRepo.FindByUUIDAndTenantID(loginTemplateUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get login template failed")
		return nil, err
	}

	if template == nil {
		return nil, apperror.NewNotFoundWithReason("login template not found or access denied")
	}

	result := toLoginTemplateServiceDataResult(template)
	span.SetStatus(codes.Ok, "")
	return &result, nil
}

func (s *loginTemplateService) Create(ctx context.Context, tenantID int64, name string, description *string, template string, metadata map[string]any, status string) (*LoginTemplateServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "loginTemplate.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))
	var metadataJSON datatypes.JSON
	if metadata != nil {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return nil, err
		}
		metadataJSON = datatypes.JSON(metadataBytes)
	} else {
		metadataJSON = datatypes.JSON([]byte("{}"))
	}

	loginTemplate := &model.LoginTemplate{
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		Template:    template,
		Metadata:    metadataJSON,
		Status:      status,
		IsDefault:   false,
		IsSystem:    false,
	}

	created, err := s.loginTemplateRepo.Create(loginTemplate)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create login template failed")
		return nil, err
	}

	result := toLoginTemplateServiceDataResult(created)
	span.SetStatus(codes.Ok, "")
	return &result, nil
}

func (s *loginTemplateService) Update(ctx context.Context, loginTemplateUUID uuid.UUID, tenantID int64, name string, description *string, template string, metadata map[string]any, status string) (*LoginTemplateServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "loginTemplate.update")
	defer span.End()
	span.SetAttributes(attribute.String("login_template.uuid", loginTemplateUUID.String()), attribute.Int64("tenant.id", tenantID))
	loginTemplate, err := s.loginTemplateRepo.FindByUUIDAndTenantID(loginTemplateUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update login template failed")
		return nil, err
	}

	if loginTemplate == nil {
		return nil, apperror.NewNotFoundWithReason("login template not found or access denied")
	}

	// Prevent updating system templates
	if loginTemplate.IsSystem {
		return nil, apperror.NewValidation("cannot update system login template")
	}

	var metadataJSON datatypes.JSON
	if metadata != nil {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return nil, err
		}
		metadataJSON = datatypes.JSON(metadataBytes)
	} else {
		metadataJSON = datatypes.JSON([]byte("{}"))
	}

	loginTemplate.Name = name
	loginTemplate.Description = description
	loginTemplate.Template = template
	loginTemplate.Metadata = metadataJSON
	loginTemplate.Status = status
	// is_default is preserved from existing template

	updatedTemplate, err := s.loginTemplateRepo.UpdateByUUID(loginTemplateUUID, loginTemplate)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update login template failed")
		return nil, err
	}

	result := toLoginTemplateServiceDataResult(updatedTemplate)
	span.SetStatus(codes.Ok, "")
	return &result, nil
}

func (s *loginTemplateService) UpdateStatus(ctx context.Context, loginTemplateUUID uuid.UUID, tenantID int64, status string) (*LoginTemplateServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "loginTemplate.updateStatus")
	defer span.End()
	span.SetAttributes(attribute.String("login_template.uuid", loginTemplateUUID.String()), attribute.Int64("tenant.id", tenantID))
	template, err := s.loginTemplateRepo.FindByUUIDAndTenantID(loginTemplateUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update login template status failed")
		return nil, err
	}

	if template == nil {
		return nil, apperror.NewNotFoundWithReason("login template not found or access denied")
	}

	// Prevent updating system templates
	if template.IsSystem {
		return nil, apperror.NewValidation("cannot update system login template")
	}

	template.Status = status

	updatedTemplate, err := s.loginTemplateRepo.UpdateByUUID(loginTemplateUUID, template)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update login template status failed")
		return nil, err
	}

	result := toLoginTemplateServiceDataResult(updatedTemplate)
	span.SetStatus(codes.Ok, "")
	return &result, nil
}

func (s *loginTemplateService) Delete(ctx context.Context, loginTemplateUUID uuid.UUID, tenantID int64) (*LoginTemplateServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "loginTemplate.delete")
	defer span.End()
	span.SetAttributes(attribute.String("login_template.uuid", loginTemplateUUID.String()), attribute.Int64("tenant.id", tenantID))
	template, err := s.loginTemplateRepo.FindByUUIDAndTenantID(loginTemplateUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete login template failed")
		return nil, err
	}

	if template == nil {
		return nil, apperror.NewNotFoundWithReason("login template not found or access denied")
	}

	// Prevent deleting system templates
	if template.IsSystem {
		return nil, apperror.NewValidation("cannot delete system login template")
	}

	if err := s.loginTemplateRepo.DeleteByUUID(loginTemplateUUID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete login template failed")
		return nil, err
	}

	result := toLoginTemplateServiceDataResult(template)
	span.SetStatus(codes.Ok, "")
	return &result, nil
}

func toLoginTemplateServiceDataResult(template *model.LoginTemplate) LoginTemplateServiceDataResult {
	var metadata map[string]any
	if len(template.Metadata) > 0 {
		if err := json.Unmarshal(template.Metadata, &metadata); err != nil {
			metadata = nil // fall through to empty-map default below
		}
	}
	if metadata == nil {
		metadata = make(map[string]any)
	}

	return LoginTemplateServiceDataResult{
		LoginTemplateUUID: template.LoginTemplateUUID,
		Name:              template.Name,
		Description:       template.Description,
		Template:          template.Template,
		Status:            template.Status,
		Metadata:          metadata,
		IsDefault:         template.IsDefault,
		IsSystem:          template.IsSystem,
		CreatedAt:         template.CreatedAt,
		UpdatedAt:         template.UpdatedAt,
	}
}
