package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/gorm"
)

type SmsTemplateServiceDataResult struct {
	SmsTemplateUUID uuid.UUID
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

type SmsTemplateServiceListResult struct {
	Data       []SmsTemplateServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type SmsTemplateService interface {
	GetAll(tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*SmsTemplateServiceListResult, error)
	GetByUUID(uuid uuid.UUID, tenantID int64) (*SmsTemplateServiceDataResult, error)
	Create(tenantID int64, name string, description *string, message string, senderID *string, status string) (*SmsTemplateServiceDataResult, error)
	Update(uuid uuid.UUID, tenantID int64, name string, description *string, message string, senderID *string, status string) (*SmsTemplateServiceDataResult, error)
	UpdateStatus(uuid uuid.UUID, tenantID int64, status string) (*SmsTemplateServiceDataResult, error)
	Delete(uuid uuid.UUID, tenantID int64) (*SmsTemplateServiceDataResult, error)
}

type smsTemplateService struct {
	db              *gorm.DB
	smsTemplateRepo repository.SmsTemplateRepository
}

func NewSmsTemplateService(
	db *gorm.DB,
	smsTemplateRepo repository.SmsTemplateRepository,
) SmsTemplateService {
	return &smsTemplateService{
		db:              db,
		smsTemplateRepo: smsTemplateRepo,
	}
}

func (s *smsTemplateService) GetAll(tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*SmsTemplateServiceListResult, error) {
	filter := repository.SmsTemplateRepositoryGetFilter{
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

	data := make([]SmsTemplateServiceDataResult, len(result.Data))
	for i, template := range result.Data {
		data[i] = toSmsTemplateServiceDataResult(&template)
	}

	return &SmsTemplateServiceListResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *smsTemplateService) GetByUUID(smsTemplateUUID uuid.UUID, tenantID int64) (*SmsTemplateServiceDataResult, error) {
	template, err := s.smsTemplateRepo.FindByUUIDAndTenantID(smsTemplateUUID.String(), tenantID)
	if err != nil {
		return nil, err
	}

	if template == nil {
		return nil, errors.New("SMS template not found or access denied")
	}

	result := toSmsTemplateServiceDataResult(template)
	return &result, nil
}

func (s *smsTemplateService) Create(tenantID int64, name string, description *string, message string, senderID *string, status string) (*SmsTemplateServiceDataResult, error) {
	template := &model.SmsTemplate{
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

	result := toSmsTemplateServiceDataResult(createdTemplate)
	return &result, nil
}

func (s *smsTemplateService) Update(smsTemplateUUID uuid.UUID, tenantID int64, name string, description *string, message string, senderID *string, status string) (*SmsTemplateServiceDataResult, error) {
	template, err := s.smsTemplateRepo.FindByUUIDAndTenantID(smsTemplateUUID.String(), tenantID)
	if err != nil {
		return nil, err
	}

	if template == nil {
		return nil, errors.New("SMS template not found or access denied")
	}

	// Prevent updating system templates
	if template.IsSystem {
		return nil, errors.New("cannot update system SMS template")
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

	result := toSmsTemplateServiceDataResult(updatedTemplate)
	return &result, nil
}

func (s *smsTemplateService) UpdateStatus(smsTemplateUUID uuid.UUID, tenantID int64, status string) (*SmsTemplateServiceDataResult, error) {
	template, err := s.smsTemplateRepo.FindByUUIDAndTenantID(smsTemplateUUID.String(), tenantID)
	if err != nil {
		return nil, err
	}

	if template == nil {
		return nil, errors.New("SMS template not found or access denied")
	}

	// Prevent updating system templates
	if template.IsSystem {
		return nil, errors.New("cannot update status of system SMS template")
	}

	template.Status = status

	updatedTemplate, err := s.smsTemplateRepo.UpdateByUUID(smsTemplateUUID, template)
	if err != nil {
		return nil, err
	}

	result := toSmsTemplateServiceDataResult(updatedTemplate)
	return &result, nil
}

func (s *smsTemplateService) Delete(smsTemplateUUID uuid.UUID, tenantID int64) (*SmsTemplateServiceDataResult, error) {
	template, err := s.smsTemplateRepo.FindByUUIDAndTenantID(smsTemplateUUID.String(), tenantID)
	if err != nil {
		return nil, err
	}

	if template == nil {
		return nil, errors.New("SMS template not found or access denied")
	}

	// Prevent deleting system templates
	if template.IsSystem {
		return nil, errors.New("cannot delete system SMS template")
	}

	err = s.smsTemplateRepo.DeleteByUUID(smsTemplateUUID)
	if err != nil {
		return nil, err
	}

	result := toSmsTemplateServiceDataResult(template)
	return &result, nil
}

// Helper function to convert model to service data result
func toSmsTemplateServiceDataResult(template *model.SmsTemplate) SmsTemplateServiceDataResult {
	return SmsTemplateServiceDataResult{
		SmsTemplateUUID: template.SmsTemplateUUID,
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
