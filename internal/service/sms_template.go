package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
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
	GetAll(tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*SMSTemplateServiceListResult, error)
	GetByUUID(uuid uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error)
	Create(tenantID int64, name string, description *string, message string, senderID *string, status string) (*SMSTemplateServiceDataResult, error)
	Update(uuid uuid.UUID, tenantID int64, name string, description *string, message string, senderID *string, status string) (*SMSTemplateServiceDataResult, error)
	UpdateStatus(uuid uuid.UUID, tenantID int64, status string) (*SMSTemplateServiceDataResult, error)
	Delete(uuid uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error)
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

func (s *smsTemplateService) GetAll(tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*SMSTemplateServiceListResult, error) {
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
		return nil, err
	}

	data := make([]SMSTemplateServiceDataResult, len(result.Data))
	for i, template := range result.Data {
		data[i] = toSMSTemplateServiceDataResult(&template)
	}

	return &SMSTemplateServiceListResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *smsTemplateService) GetByUUID(smsTemplateUUID uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error) {
	template, err := s.smsTemplateRepo.FindByUUIDAndTenantID(smsTemplateUUID.String(), tenantID)
	if err != nil {
		return nil, err
	}

	if template == nil {
		return nil, apperror.NewNotFoundWithReason("SMS template not found or access denied")
	}

	result := toSMSTemplateServiceDataResult(template)
	return &result, nil
}

func (s *smsTemplateService) Create(tenantID int64, name string, description *string, message string, senderID *string, status string) (*SMSTemplateServiceDataResult, error) {
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
		return nil, err
	}

	result := toSMSTemplateServiceDataResult(createdTemplate)
	return &result, nil
}

func (s *smsTemplateService) Update(smsTemplateUUID uuid.UUID, tenantID int64, name string, description *string, message string, senderID *string, status string) (*SMSTemplateServiceDataResult, error) {
	template, err := s.smsTemplateRepo.FindByUUIDAndTenantID(smsTemplateUUID.String(), tenantID)
	if err != nil {
		return nil, err
	}

	if template == nil {
		return nil, apperror.NewNotFoundWithReason("SMS template not found or access denied")
	}

	// Prevent updating system templates
	if template.IsSystem {
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
		return nil, err
	}

	result := toSMSTemplateServiceDataResult(updatedTemplate)
	return &result, nil
}

func (s *smsTemplateService) UpdateStatus(smsTemplateUUID uuid.UUID, tenantID int64, status string) (*SMSTemplateServiceDataResult, error) {
	template, err := s.smsTemplateRepo.FindByUUIDAndTenantID(smsTemplateUUID.String(), tenantID)
	if err != nil {
		return nil, err
	}

	if template == nil {
		return nil, apperror.NewNotFoundWithReason("SMS template not found or access denied")
	}

	// Prevent updating system templates
	if template.IsSystem {
		return nil, apperror.NewValidation("cannot update status of system SMS template")
	}

	template.Status = status

	updatedTemplate, err := s.smsTemplateRepo.UpdateByUUID(smsTemplateUUID, template)
	if err != nil {
		return nil, err
	}

	result := toSMSTemplateServiceDataResult(updatedTemplate)
	return &result, nil
}

func (s *smsTemplateService) Delete(smsTemplateUUID uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error) {
	template, err := s.smsTemplateRepo.FindByUUIDAndTenantID(smsTemplateUUID.String(), tenantID)
	if err != nil {
		return nil, err
	}

	if template == nil {
		return nil, apperror.NewNotFoundWithReason("SMS template not found or access denied")
	}

	// Prevent deleting system templates
	if template.IsSystem {
		return nil, apperror.NewValidation("cannot delete system SMS template")
	}

	err = s.smsTemplateRepo.DeleteByUUID(smsTemplateUUID)
	if err != nil {
		return nil, err
	}

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
