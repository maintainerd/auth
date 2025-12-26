package service

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
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
	GetAll(name *string, status []string, template *string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*LoginTemplateServiceListResult, error)
	GetByUUID(loginTemplateUUID uuid.UUID) (*LoginTemplateServiceDataResult, error)
	Create(name string, description *string, template string, metadata map[string]any, status string) (*LoginTemplateServiceDataResult, error)
	Update(loginTemplateUUID uuid.UUID, name string, description *string, template string, metadata map[string]any, status string) (*LoginTemplateServiceDataResult, error)
	UpdateStatus(loginTemplateUUID uuid.UUID, status string) (*LoginTemplateServiceDataResult, error)
	Delete(loginTemplateUUID uuid.UUID) (*LoginTemplateServiceDataResult, error)
}

type loginTemplateService struct {
	loginTemplateRepo repository.LoginTemplateRepository
}

func NewLoginTemplateService(loginTemplateRepo repository.LoginTemplateRepository) LoginTemplateService {
	return &loginTemplateService{
		loginTemplateRepo: loginTemplateRepo,
	}
}

func (s *loginTemplateService) GetAll(name *string, status []string, template *string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*LoginTemplateServiceListResult, error) {
	filter := repository.LoginTemplateRepositoryGetFilter{
		Name:      name,
		Status:    status,
		Template:  template,
		IsDefault: isDefault,
		IsSystem:  isSystem,
		Page:      page,
		Limit:     limit,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	result, err := s.loginTemplateRepo.FindPaginated(filter)
	if err != nil {
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

func (s *loginTemplateService) GetByUUID(loginTemplateUUID uuid.UUID) (*LoginTemplateServiceDataResult, error) {
	template, err := s.loginTemplateRepo.FindByUUID(loginTemplateUUID)
	if err != nil {
		return nil, err
	}

	if template == nil {
		return nil, errors.New("login template not found")
	}

	result := toLoginTemplateServiceDataResult(template)
	return &result, nil
}

func (s *loginTemplateService) Create(name string, description *string, template string, metadata map[string]any, status string) (*LoginTemplateServiceDataResult, error) {
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
		return nil, err
	}

	result := toLoginTemplateServiceDataResult(created)
	return &result, nil
}

func (s *loginTemplateService) Update(loginTemplateUUID uuid.UUID, name string, description *string, template string, metadata map[string]any, status string) (*LoginTemplateServiceDataResult, error) {
	loginTemplate, err := s.loginTemplateRepo.FindByUUID(loginTemplateUUID)
	if err != nil {
		return nil, err
	}

	if loginTemplate == nil {
		return nil, errors.New("login template not found")
	}

	// Prevent updating system templates
	if loginTemplate.IsSystem {
		return nil, errors.New("cannot update system login template")
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
		return nil, err
	}

	result := toLoginTemplateServiceDataResult(updatedTemplate)
	return &result, nil
}

func (s *loginTemplateService) UpdateStatus(loginTemplateUUID uuid.UUID, status string) (*LoginTemplateServiceDataResult, error) {
	template, err := s.loginTemplateRepo.FindByUUID(loginTemplateUUID)
	if err != nil {
		return nil, err
	}

	if template == nil {
		return nil, errors.New("login template not found")
	}

	// Prevent updating system templates
	if template.IsSystem {
		return nil, errors.New("cannot update system login template")
	}

	template.Status = status

	updatedTemplate, err := s.loginTemplateRepo.UpdateByUUID(loginTemplateUUID, template)
	if err != nil {
		return nil, err
	}

	result := toLoginTemplateServiceDataResult(updatedTemplate)
	return &result, nil
}

func (s *loginTemplateService) Delete(loginTemplateUUID uuid.UUID) (*LoginTemplateServiceDataResult, error) {
	template, err := s.loginTemplateRepo.FindByUUID(loginTemplateUUID)
	if err != nil {
		return nil, err
	}

	if template == nil {
		return nil, errors.New("login template not found")
	}

	// Prevent deleting system templates
	if template.IsSystem {
		return nil, errors.New("cannot delete system login template")
	}

	if err := s.loginTemplateRepo.DeleteByUUID(loginTemplateUUID); err != nil {
		return nil, err
	}

	result := toLoginTemplateServiceDataResult(template)
	return &result, nil
}

func toLoginTemplateServiceDataResult(template *model.LoginTemplate) LoginTemplateServiceDataResult {
	var metadata map[string]interface{}
	if len(template.Metadata) > 0 {
		json.Unmarshal(template.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
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
