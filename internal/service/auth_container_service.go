package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"gorm.io/gorm"
)

type AuthContainerServiceDataResult struct {
	AuthContainerUUID uuid.UUID
	Name              string
	Description       string
	Identifier        string
	Organization      *OrganizationServiceDataResult
	IsActive          bool
	IsPublic          bool
	IsDefault         bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type AuthContainerServiceGetFilter struct {
	Name             *string
	Description      *string
	Identifier       *string
	OrganizationUUID *string
	IsActive         *bool
	IsPublic         *bool
	IsDefault        *bool
	Page             int
	Limit            int
	SortBy           string
	SortOrder        string
}

type AuthContainerServiceGetResult struct {
	Data       []AuthContainerServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type AuthContainerService interface {
	Get(filter AuthContainerServiceGetFilter) (*AuthContainerServiceGetResult, error)
	GetByUUID(authContainerUUID uuid.UUID) (*AuthContainerServiceDataResult, error)
	Create(name string, description string, isActive bool, isPublic bool, isDefault bool, organizationUUID string) (*AuthContainerServiceDataResult, error)
	Update(authContainerUUID uuid.UUID, name string, description string, isActive bool, isPublic bool, isDefault bool) (*AuthContainerServiceDataResult, error)
	SetActiveStatusByUUID(authContainerUUID uuid.UUID) (*AuthContainerServiceDataResult, error)
	SetActivePublicByUUID(authContainerUUID uuid.UUID) (*AuthContainerServiceDataResult, error)
	DeleteByUUID(authContainerUUID uuid.UUID) (*AuthContainerServiceDataResult, error)
}

type authContainerService struct {
	db                *gorm.DB
	authContainerRepo repository.AuthContainerRepository
	organizationRepo  repository.OrganizationRepository
}

func NewAuthContainerService(
	db *gorm.DB,
	authContainerRepo repository.AuthContainerRepository,
	organizationRepo repository.OrganizationRepository,
) AuthContainerService {
	return &authContainerService{
		db:                db,
		authContainerRepo: authContainerRepo,
		organizationRepo:  organizationRepo,
	}
}

func (s *authContainerService) Get(filter AuthContainerServiceGetFilter) (*AuthContainerServiceGetResult, error) {
	var orgID *int64

	// Get organization
	if filter.OrganizationUUID != nil {
		org, err := s.organizationRepo.FindByUUID(*filter.OrganizationUUID)
		if err != nil || org == nil {
			return nil, errors.New("organization not found")
		}
		orgID = &org.OrganizationID
	}

	// Build query filter
	queryFilter := repository.AuthContainerRepositoryGetFilter{
		Name:           filter.Name,
		Description:    filter.Description,
		Identifier:     filter.Identifier,
		OrganizationID: orgID,
		IsActive:       filter.IsActive,
		IsPublic:       filter.IsPublic,
		IsDefault:      filter.IsDefault,
		Page:           filter.Page,
		Limit:          filter.Limit,
		SortBy:         filter.SortBy,
		SortOrder:      filter.SortOrder,
	}

	result, err := s.authContainerRepo.FindPaginated(queryFilter)
	if err != nil {
		return nil, err
	}

	// Build response data
	resData := make([]AuthContainerServiceDataResult, len(result.Data))
	for i, rdata := range result.Data {
		resData[i] = *toAuthContainerServiceDataResult(&rdata)
	}

	return &AuthContainerServiceGetResult{
		Data:       resData,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *authContainerService) GetByUUID(authContainerUUID uuid.UUID) (*AuthContainerServiceDataResult, error) {
	authContainer, err := s.authContainerRepo.FindByUUID(authContainerUUID, "Organization")
	if err != nil || authContainer == nil {
		return nil, errors.New("auth container not found")
	}

	return toAuthContainerServiceDataResult(authContainer), nil
}

func (s *authContainerService) Create(name string, description string, isActive bool, isPublic bool, isDefault bool, organizationUUID string) (*AuthContainerServiceDataResult, error) {
	var createdAuthContainer *model.AuthContainer

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthContainerRepo := s.authContainerRepo.WithTx(tx)
		txOrganizationRepo := s.organizationRepo.WithTx(tx)

		// Check if auth container already exists
		existingAuthContainer, err := txAuthContainerRepo.FindByName(name)
		if err != nil {
			return err
		}
		if existingAuthContainer != nil {
			return errors.New(name + " auth container already exists")
		}

		// Check if organization exist
		organization, err := txOrganizationRepo.FindByUUID(organizationUUID)
		if err != nil || organization == nil {
			return errors.New("organization not found")
		}

		// Generate identifier
		identifier := util.GenerateIdentifier(12)

		// Create auth container
		newAuthContainer := &model.AuthContainer{
			Name:           name,
			Description:    description,
			Identifier:     identifier,
			OrganizationID: organization.OrganizationID,
			IsActive:       isActive,
			IsPublic:       isActive,
			IsDefault:      isDefault,
		}

		_, err = txAuthContainerRepo.CreateOrUpdate(newAuthContainer)
		if err != nil {
			return err
		}

		// Fetch AuthContainer with Service preloaded
		createdAuthContainer, err = txAuthContainerRepo.FindByUUID(newAuthContainer.AuthContainerUUID, "Organization")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toAuthContainerServiceDataResult(createdAuthContainer), nil
}

func (s *authContainerService) Update(authContainerUUID uuid.UUID, name string, description string, isActive bool, isPublic bool, isDefault bool) (*AuthContainerServiceDataResult, error) {
	var updatedAuthContainer *model.AuthContainer

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthContainerRepo := s.authContainerRepo.WithTx(tx)

		// Get auth container
		authContainer, err := txAuthContainerRepo.FindByUUID(authContainerUUID, "Organization")
		if err != nil || authContainer == nil {
			return errors.New("auth container not found")
		}

		// Check if default
		if authContainer.IsDefault {
			return errors.New("default auth cannot cannot be updated")
		}

		// Check if auth container already exist
		if authContainer.Name != name {
			existingAuthContainer, err := txAuthContainerRepo.FindByName(name)
			if err != nil {
				return err
			}
			if existingAuthContainer != nil && existingAuthContainer.AuthContainerUUID != authContainerUUID {
				return errors.New(name + " auth container already exists")
			}
		}

		// Set values
		authContainer.Name = name
		authContainer.Description = description
		authContainer.IsActive = isActive
		authContainer.IsPublic = isPublic
		authContainer.IsDefault = isDefault

		// Update
		_, err = txAuthContainerRepo.CreateOrUpdate(authContainer)
		if err != nil {
			return err
		}

		updatedAuthContainer = authContainer

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toAuthContainerServiceDataResult(updatedAuthContainer), nil
}

func (s *authContainerService) SetActiveStatusByUUID(authContainerUUID uuid.UUID) (*AuthContainerServiceDataResult, error) {
	var updatedAuthContainer *model.AuthContainer

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthContainerRepo := s.authContainerRepo.WithTx(tx)

		// Get auth container
		authContainer, err := txAuthContainerRepo.FindByUUID(authContainerUUID, "Organization")
		if err != nil || authContainer == nil {
			return errors.New("auth container not found")
		}

		// Check if default
		if authContainer.IsDefault {
			return errors.New("default auth container cannot be updated")
		}

		authContainer.IsActive = !authContainer.IsActive

		_, err = txAuthContainerRepo.CreateOrUpdate(authContainer)
		if err != nil {
			return err
		}

		updatedAuthContainer = authContainer

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toAuthContainerServiceDataResult(updatedAuthContainer), nil
}

func (s *authContainerService) SetActivePublicByUUID(authContainerUUID uuid.UUID) (*AuthContainerServiceDataResult, error) {
	var updatedAuthContainer *model.AuthContainer

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthContainerRepo := s.authContainerRepo.WithTx(tx)

		// Get auth container
		authContainer, err := txAuthContainerRepo.FindByUUID(authContainerUUID, "Organization")
		if err != nil || authContainer == nil {
			return errors.New("auth container not found")
		}

		// Check if default
		if authContainer.IsDefault {
			return errors.New("default auth container cannot be updated")
		}

		authContainer.IsPublic = !authContainer.IsPublic

		_, err = txAuthContainerRepo.CreateOrUpdate(authContainer)
		if err != nil {
			return err
		}

		updatedAuthContainer = authContainer

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toAuthContainerServiceDataResult(updatedAuthContainer), nil
}

func (s *authContainerService) DeleteByUUID(authContainerUUID uuid.UUID) (*AuthContainerServiceDataResult, error) {
	// Get auth container
	authContainer, err := s.authContainerRepo.FindByUUID(authContainerUUID, "Organization")
	if err != nil || authContainer == nil {
		return nil, errors.New("auth container not found")
	}

	// Check if default
	if authContainer.IsDefault {
		return nil, errors.New("default auth container cannot be deleted")
	}

	err = s.authContainerRepo.DeleteByUUID(authContainerUUID)
	if err != nil {
		return nil, err
	}

	return toAuthContainerServiceDataResult(authContainer), nil
}

// Reponse builder
func toAuthContainerServiceDataResult(authContainer *model.AuthContainer) *AuthContainerServiceDataResult {
	if authContainer == nil {
		return nil
	}

	result := &AuthContainerServiceDataResult{
		AuthContainerUUID: authContainer.AuthContainerUUID,
		Name:              authContainer.Name,
		Description:       authContainer.Description,
		Identifier:        authContainer.Identifier,
		IsActive:          authContainer.IsActive,
		IsPublic:          authContainer.IsPublic,
		IsDefault:         authContainer.IsDefault,
		CreatedAt:         authContainer.CreatedAt,
		UpdatedAt:         authContainer.UpdatedAt,
	}

	if authContainer.Organization != nil {
		result.Organization = &OrganizationServiceDataResult{
			OrganizationUUID: authContainer.Organization.OrganizationUUID,
			Name:             authContainer.Organization.Name,
			Description:      authContainer.Organization.Description,
			Email:            authContainer.Organization.Email,
			Phone:            authContainer.Organization.Phone,
			IsActive:         authContainer.Organization.IsActive,
			IsDefault:        authContainer.Organization.IsDefault,
			CreatedAt:        authContainer.Organization.CreatedAt,
			UpdatedAt:        authContainer.Organization.UpdatedAt,
		}
	}

	return result
}
