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
	IsDefault        bool
	IsRoot           bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type OrganizationServiceGetFilter struct {
	Name        *string
	Description *string
	Email       *string
	Phone       *string
	IsActive    *bool
	IsDefault   *bool
	IsRoot      *bool
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
	Create(name string, description string, email string, phone string, isActive bool, isDefault bool) (*OrganizationServiceDataResult, error)
	CreateExternal(name string, description string, email string, phone string) (*OrganizationServiceDataResult, error)
	Update(organizationUUID uuid.UUID, name string, description string, email string, phone string, isActive bool, isDefault bool, userUUID uuid.UUID) (*OrganizationServiceDataResult, error)
	SetActiveStatusByUUID(organizationUUID uuid.UUID, userUUID uuid.UUID) (*OrganizationServiceDataResult, error)
	DeleteByUUID(organizationUUID uuid.UUID, userUUID uuid.UUID) (*OrganizationServiceDataResult, error)
	GetRootOrganization() (*OrganizationServiceDataResult, error)
	GetDefaultOrganization() (*OrganizationServiceDataResult, error)
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
		IsDefault:   filter.IsDefault,
		IsRoot:      filter.IsRoot,
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

func (s *organizationService) Create(name string, description string, email string, phone string, isActive bool, isDefault bool) (*OrganizationServiceDataResult, error) {
	// Prevent creation of root or default organizations by users
	if isDefault {
		return nil, errors.New("cannot create default organization: default organization can only be created during system initialization")
	}

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

		// Create organization (always external: not root, not default)
		newOrg := &model.Organization{
			Name:        name,
			Description: &description,
			Email:       &email,
			Phone:       &phone,
			IsActive:    isActive,
			IsDefault:   false, // Force to false - only external orgs can be created
			IsRoot:      false, // Force to false - root orgs cannot be created by users
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

func (s *organizationService) Update(organizationUUID uuid.UUID, name string, description string, email string, phone string, isActive bool, isDefault bool, userUUID uuid.UUID) (*OrganizationServiceDataResult, error) {
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

		// Apply logic-based access control for different organization types
		if org.IsRoot {
			// Root organizations: Only allow updating basic info (name, description, email, phone)
			// Cannot change IsActive, IsDefault, or IsRoot status
			if org.IsActive != isActive {
				return errors.New("cannot deactivate root organization")
			}
			if org.IsDefault != isDefault {
				return errors.New("cannot change default status of root organization")
			}
		} else if org.IsDefault {
			// Default organizations: Only allow updating basic info (name, description, email, phone)
			// Cannot change IsActive, IsDefault, or IsRoot status
			if org.IsActive != isActive {
				return errors.New("cannot deactivate default organization")
			}
			if org.IsDefault != isDefault {
				return errors.New("cannot change default status of default organization")
			}
		}
		// External organizations: Allow all updates including IsActive and IsDefault

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

		// Apply updates based on organization type
		if org.IsRoot {
			// Root organizations: Preserve IsRoot, IsDefault, and IsActive status
			// (IsRoot and IsDefault are never changed, IsActive is preserved)
		} else if org.IsDefault {
			// Default organizations: Preserve IsRoot, IsDefault, and IsActive status
			// (IsRoot and IsDefault are never changed, IsActive is preserved)
		} else {
			// External organizations: Allow IsActive and IsDefault updates
			org.IsActive = isActive
			org.IsDefault = isDefault
			// Note: IsRoot is never changed for any organization type
		}

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

		// Prevent deactivating root or default organizations
		if (org.IsRoot || org.IsDefault) && org.IsActive {
			return errors.New("cannot deactivate root or default organization")
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

	// Prevent deletion of root or default organizations
	if org.IsRoot {
		return nil, errors.New("root organization cannot be deleted")
	}
	if org.IsDefault {
		return nil, errors.New("default organization cannot be deleted")
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
		IsDefault:        org.IsDefault,
		IsRoot:           org.IsRoot,
		CreatedAt:        org.CreatedAt,
		UpdatedAt:        org.UpdatedAt,
	}
}

// CreateExternal creates a new external organization (not root, not default)
// This is used when external organizations register to the system
func (s *organizationService) CreateExternal(name string, description string, email string, phone string) (*OrganizationServiceDataResult, error) {
	organization := &model.Organization{
		Name:        name,
		Description: &description,
		Email:       &email,
		Phone:       &phone,
		IsActive:    true,
		IsDefault:   false,
		IsRoot:      false,
	}

	_, err := s.organizationRepo.Create(organization)
	if err != nil {
		return nil, err
	}

	return toOrganizationServiceDataResult(organization), nil
}

// GetRootOrganization returns the root organization
func (s *organizationService) GetRootOrganization() (*OrganizationServiceDataResult, error) {
	filter := repository.OrganizationRepositoryGetFilter{
		IsRoot:    &[]bool{true}[0],
		IsActive:  &[]bool{true}[0],
		Page:      1,
		Limit:     1,
		SortBy:    "created_at",
		SortOrder: "asc",
	}

	result, err := s.organizationRepo.FindPaginated(filter)
	if err != nil {
		return nil, err
	}

	if len(result.Data) == 0 {
		return nil, errors.New("root organization not found")
	}

	return toOrganizationServiceDataResult(&result.Data[0]), nil
}

// GetDefaultOrganization returns the default organization
func (s *organizationService) GetDefaultOrganization() (*OrganizationServiceDataResult, error) {
	filter := repository.OrganizationRepositoryGetFilter{
		IsDefault: &[]bool{true}[0],
		IsActive:  &[]bool{true}[0],
		Page:      1,
		Limit:     1,
		SortBy:    "created_at",
		SortOrder: "asc",
	}

	result, err := s.organizationRepo.FindPaginated(filter)
	if err != nil {
		return nil, err
	}

	if len(result.Data) == 0 {
		return nil, errors.New("default organization not found")
	}

	return toOrganizationServiceDataResult(&result.Data[0]), nil
}
