package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
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
	GetAll(name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*EmailTemplateServiceListResult, error)
	GetByUUID(emailTemplateUUID uuid.UUID) (*EmailTemplateServiceDataResult, error)
	Create(name, subject, bodyHTML string, bodyPlain *string, status string, isDefault bool) (*EmailTemplateServiceDataResult, error)
	Update(emailTemplateUUID uuid.UUID, name, subject, bodyHTML string, bodyPlain *string, status string) (*EmailTemplateServiceDataResult, error)
	UpdateStatus(emailTemplateUUID uuid.UUID, status string) (*EmailTemplateServiceDataResult, error)
	Delete(emailTemplateUUID uuid.UUID) (*EmailTemplateServiceDataResult, error)
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

func (s *emailTemplateService) GetAll(name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*EmailTemplateServiceListResult, error) {
	filter := repository.EmailTemplateRepositoryGetFilter{
		Name:      name,
		Status:    status,
		IsDefault: isDefault,
		IsSystem:  isSystem,
		Page:      page,
		Limit:     limit,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	result, err := s.emailTemplateRepo.FindPaginated(filter)
	if err != nil {
		return nil, err
	}

	data := make([]EmailTemplateServiceDataResult, len(result.Data))
	for i, template := range result.Data {
		data[i] = toEmailTemplateServiceDataResult(&template)
	}

	return &EmailTemplateServiceListResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *emailTemplateService) GetByUUID(emailTemplateUUID uuid.UUID) (*EmailTemplateServiceDataResult, error) {
	template, err := s.emailTemplateRepo.FindByUUID(emailTemplateUUID)
	if err != nil {
		return nil, err
	}

	if template == nil {
		return nil, errors.New("email template not found")
	}

	result := toEmailTemplateServiceDataResult(template)
	return &result, nil
}

func (s *emailTemplateService) Create(name, subject, bodyHTML string, bodyPlain *string, status string, isDefault bool) (*EmailTemplateServiceDataResult, error) {
	template := &model.EmailTemplate{
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
		return nil, err
	}

	result := toEmailTemplateServiceDataResult(createdTemplate)
	return &result, nil
}

func (s *emailTemplateService) Update(emailTemplateUUID uuid.UUID, name, subject, bodyHTML string, bodyPlain *string, status string) (*EmailTemplateServiceDataResult, error) {
	template, err := s.emailTemplateRepo.FindByUUID(emailTemplateUUID)
	if err != nil {
		return nil, err
	}

	if template == nil {
		return nil, errors.New("email template not found")
	}

	// Prevent updating system templates
	if template.IsSystem {
		return nil, errors.New("cannot update system email template")
	}

	template.Name = name
	template.Subject = subject
	template.BodyHTML = bodyHTML
	template.BodyPlain = bodyPlain
	template.Status = status
	// is_default is preserved from existing template

	updatedTemplate, err := s.emailTemplateRepo.UpdateByUUID(emailTemplateUUID, template)
	if err != nil {
		return nil, err
	}

	result := toEmailTemplateServiceDataResult(updatedTemplate)
	return &result, nil
}

func (s *emailTemplateService) UpdateStatus(emailTemplateUUID uuid.UUID, status string) (*EmailTemplateServiceDataResult, error) {
	template, err := s.emailTemplateRepo.FindByUUID(emailTemplateUUID)
	if err != nil {
		return nil, err
	}

	if template == nil {
		return nil, errors.New("email template not found")
	}

	// Prevent updating system templates
	if template.IsSystem {
		return nil, errors.New("cannot update status of system email template")
	}

	template.Status = status

	updatedTemplate, err := s.emailTemplateRepo.UpdateByUUID(emailTemplateUUID, template)
	if err != nil {
		return nil, err
	}

	result := toEmailTemplateServiceDataResult(updatedTemplate)
	return &result, nil
}

func (s *emailTemplateService) Delete(emailTemplateUUID uuid.UUID) (*EmailTemplateServiceDataResult, error) {
	template, err := s.emailTemplateRepo.FindByUUID(emailTemplateUUID)
	if err != nil {
		return nil, err
	}

	if template == nil {
		return nil, errors.New("email template not found")
	}

	// Prevent deleting system templates
	if template.IsSystem {
		return nil, errors.New("cannot delete system email template")
	}

	err = s.emailTemplateRepo.DeleteByUUID(emailTemplateUUID)
	if err != nil {
		return nil, err
	}

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
