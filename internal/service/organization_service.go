package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"gorm.io/gorm"
)

type OrganizationServiceDataResult struct {
	OrganizationUUID uuid.UUID
	Name             string
	Description      *string
	Email            *string
	Phone            *string
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type OrganizationServiceGetFilter struct {
	Name        *string
	Description *string
	Email       *string
	Phone       *string
	IsActive    *bool
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type OrganizationServiceGetResult struct {
	Data       []OrganizationServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type OrganizationService interface {
	Get(filter OrganizationServiceGetFilter) (*OrganizationServiceGetResult, error)
	GetByUUID(organizationUUID uuid.UUID) (*OrganizationServiceDataResult, error)
	Create(name string, description string, email string, phone string, isActive bool) (*OrganizationServiceDataResult, error)
	Update(organizationUUID uuid.UUID, name string, description string, email string, phone string, isActive bool, userUUID uuid.UUID) (*OrganizationServiceDataResult, error)
	SetActiveStatusByUUID(organizationUUID uuid.UUID, userUUID uuid.UUID) (*OrganizationServiceDataResult, error)
	DeleteByUUID(organizationUUID uuid.UUID, userUUID uuid.UUID) (*OrganizationServiceDataResult, error)
}

type organizationService struct {
	db               *gorm.DB
	organizationRepo repository.OrganizationRepository
}

func NewOrganizationService(
	db *gorm.DB,
	organizationRepo repository.OrganizationRepository,
) OrganizationService {
	return &organizationService{
		db:               db,
		organizationRepo: organizationRepo,
	}
}

func (s *organizationService) Get(filter OrganizationServiceGetFilter) (*OrganizationServiceGetResult, error) {
	orgFilter := repository.OrganizationRepositoryGetFilter{
		Name:        filter.Name,
		Description: filter.Description,
		Email:       filter.Email,
		Phone:       filter.Phone,
		IsActive:    filter.IsActive,
		Page:        filter.Page,
		Limit:       filter.Limit,
		SortBy:      filter.SortBy,
		SortOrder:   filter.SortOrder,
	}

	// Query paginated organizations
	result, err := s.organizationRepo.FindPaginated(orgFilter)
	if err != nil {
		return nil, err
	}

	orgs := make([]OrganizationServiceDataResult, len(result.Data))
	for i, r := range result.Data {
		orgs[i] = *toOrganizationServiceDataResult(&r)
	}

	return &OrganizationServiceGetResult{
		Data:       orgs,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *organizationService) GetByUUID(organizationUUID uuid.UUID) (*OrganizationServiceDataResult, error) {
	org, err := s.organizationRepo.FindByUUID(organizationUUID)
	if err != nil || org == nil {
		return nil, errors.New("organization not found")
	}

	return toOrganizationServiceDataResult(org), nil
}

func (s *organizationService) Create(name string, description string, email string, phone string, isActive bool) (*OrganizationServiceDataResult, error) {
	var createdOrg *model.Organization

	// Transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txOrgRepo := s.organizationRepo.WithTx(tx)

		// Check if organization already exist
		existingOrg, err := txOrgRepo.FindByName(name)
		if err != nil {
			return err
		}
		if existingOrg != nil {
			return errors.New(name + " organization already exist")
		}

		// Create organization
		newOrg := &model.Organization{
			Name:        name,
			Description: &description,
			Email:       &email,
			Phone:       &phone,
			IsActive:    isActive,
		}

		_, err = txOrgRepo.CreateOrUpdate(newOrg)
		if err != nil {
			return err
		}

		createdOrg = newOrg

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toOrganizationServiceDataResult(createdOrg), nil
}

func (s *organizationService) Update(organizationUUID uuid.UUID, name string, description string, email string, phone string, isActive bool, userUUID uuid.UUID) (*OrganizationServiceDataResult, error) {
	var updatedOrg *model.Organization

	// Transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txOrgRepo := s.organizationRepo.WithTx(tx)

		// Permission check
		hasPermission, err := txOrgRepo.CanUserUpdateOrganization(userUUID, organizationUUID)
		if err != nil || !hasPermission {
			return errors.New("permission denied: you cannot update this organization")
		}

		// Find existing organization
		org, err := txOrgRepo.FindByUUID(organizationUUID)
		if err != nil || org == nil {
			return errors.New("organization not found")
		}

		// If organization name is changed, check if duplicate
		if org.Name != name {
			existingOrg, err := txOrgRepo.FindByName(name)
			if err != nil {
				return err
			}
			if existingOrg != nil && existingOrg.OrganizationUUID != organizationUUID {
				return errors.New(name + " organization already exists")
			}
		}

		// Update organization details
		org.Name = name
		org.Description = &description
		org.Email = &email
		org.Phone = &phone
		org.IsActive = isActive

		_, err = txOrgRepo.CreateOrUpdate(org)
		if err != nil {
			return err
		}

		updatedOrg = org

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toOrganizationServiceDataResult(updatedOrg), nil
}

func (s *organizationService) SetActiveStatusByUUID(organizationUUID uuid.UUID, userUUID uuid.UUID) (*OrganizationServiceDataResult, error) {
	var updatedOrg *model.Organization

	// Transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txOrgRepo := s.organizationRepo.WithTx(tx)

		// Permission check
		hasPermission, err := txOrgRepo.CanUserUpdateOrganization(userUUID, organizationUUID)
		if err != nil || !hasPermission {
			return errors.New("permission denied: you cannot update this organization")
		}

		// Find existing organization
		org, err := txOrgRepo.FindByUUID(organizationUUID)
		if err != nil || org == nil {
			return errors.New("organization not found")
		}

		// Update organization
		org.IsActive = !org.IsActive

		_, err = txOrgRepo.CreateOrUpdate(org)
		if err != nil {
			return err
		}

		updatedOrg = org

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toOrganizationServiceDataResult(updatedOrg), nil
}

func (s *organizationService) DeleteByUUID(organizationUUID uuid.UUID, userUUID uuid.UUID) (*OrganizationServiceDataResult, error) {
	// Permission check
	hasPermission, err := s.organizationRepo.CanUserUpdateOrganization(userUUID, organizationUUID)
	if err != nil || !hasPermission {
		return nil, errors.New("permission denied: you cannot update this organization")
	}

	// Check organization existence
	org, err := s.organizationRepo.FindByUUID(organizationUUID)
	if err != nil || org == nil {
		return nil, errors.New("organization not found")
	}

	// Delete organization
	err = s.organizationRepo.DeleteByUUID(organizationUUID)
	if err != nil {
		return nil, err
	}

	return toOrganizationServiceDataResult(org), nil
}

// Reponse builder
func toOrganizationServiceDataResult(org *model.Organization) *OrganizationServiceDataResult {
	if org == nil {
		return nil
	}

	return &OrganizationServiceDataResult{
		OrganizationUUID: org.OrganizationUUID,
		Name:             org.Name,
		Description:      org.Description,
		Email:            org.Email,
		Phone:            org.Phone,
		IsActive:         org.IsActive,
		CreatedAt:        org.CreatedAt,
		UpdatedAt:        org.UpdatedAt,
	}
}
