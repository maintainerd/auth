package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type IdentityProviderServiceDataResult struct {
	IdentityProviderUUID uuid.UUID
	Name                 string
	DisplayName          string
	ProviderType         string
	Identifier           string
	Config               datatypes.JSON
	AuthContainer        *AuthContainerServiceDataResult
	IsActive             bool
	IsDefault            bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type IdentityProviderServiceGetFilter struct {
	Name              *string
	DisplayName       *string
	ProviderType      *string
	Identifier        *string
	AuthContainerUUID *string
	IsActive          *bool
	IsDefault         *bool
	Page              int
	Limit             int
	SortBy            string
	SortOrder         string
}

type IdentityProviderServiceGetResult struct {
	Data       []IdentityProviderServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type IdentityProviderService interface {
	Get(filter IdentityProviderServiceGetFilter) (*IdentityProviderServiceGetResult, error)
	GetByUUID(idpUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
	Create(name string, displayName string, providerType string, config datatypes.JSON, isActive bool, isDefault bool, authContainerUUID string) (*IdentityProviderServiceDataResult, error)
	Update(idpUUID uuid.UUID, name string, displayName string, providerType string, config datatypes.JSON, isActive bool, isDefault bool, authContainerUUID string) (*IdentityProviderServiceDataResult, error)
	SetActiveStatusByUUID(idpUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
	DeleteByUUID(idpUUID uuid.UUID) (*IdentityProviderServiceDataResult, error)
}

type identityProviderService struct {
	db                *gorm.DB
	idpRepo           repository.IdentityProviderRepository
	authContainerRepo repository.AuthContainerRepository
}

func NewIdentityProviderService(
	db *gorm.DB,
	idpRepo repository.IdentityProviderRepository,
	authContainerRepo repository.AuthContainerRepository,
) IdentityProviderService {
	return &identityProviderService{
		db:                db,
		idpRepo:           idpRepo,
		authContainerRepo: authContainerRepo,
	}
}

func (s *identityProviderService) Get(filter IdentityProviderServiceGetFilter) (*IdentityProviderServiceGetResult, error) {
	var authContainerID *int64

	// Get auth container if uuid exist
	if filter.AuthContainerUUID != nil {
		authContainer, err := s.authContainerRepo.FindByUUID(*filter.AuthContainerUUID)
		if err != nil || authContainer == nil {
			return nil, errors.New("auth container not found")
		}
		authContainerID = &authContainer.AuthContainerID
	}

	// Build query filter
	queryFilter := repository.IdentityProviderRepositoryGetFilter{
		Name:            filter.Name,
		DisplayName:     filter.DisplayName,
		ProviderType:    filter.ProviderType,
		Identifier:      filter.Identifier,
		AuthContainerID: authContainerID,
		IsActive:        filter.IsActive,
		IsDefault:       filter.IsDefault,
		Page:            filter.Page,
		Limit:           filter.Limit,
		SortBy:          filter.SortBy,
		SortOrder:       filter.SortOrder,
	}

	result, err := s.idpRepo.FindPaginated(queryFilter)
	if err != nil {
		return nil, err
	}

	idps := make([]IdentityProviderServiceDataResult, len(result.Data))
	for i, idp := range result.Data {
		idps[i] = *toIdpServiceDataResult(&idp)
	}

	return &IdentityProviderServiceGetResult{
		Data:       idps,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *identityProviderService) GetByUUID(idpUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	idp, err := s.idpRepo.FindByUUID(idpUUID, "AuthContainer")
	if err != nil || idp == nil {
		return nil, errors.New("idp not found")
	}

	return toIdpServiceDataResult(idp), nil
}

func (s *identityProviderService) Create(name string, displayName string, providerType string, config datatypes.JSON, isActive bool, isDefault bool, authContainerUUID string) (*IdentityProviderServiceDataResult, error) {
	var createdIdp *model.IdentityProvider

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txIdpRepo := s.idpRepo.WithTx(tx)
		txAuthContainerRepo := s.authContainerRepo.WithTx(tx)

		// Check if auth container exist
		authContainer, err := txAuthContainerRepo.FindByUUID(authContainerUUID)
		if err != nil || authContainer == nil {
			return errors.New("auth container not found")
		}

		// Check if idp already exists
		existingIdp, err := txIdpRepo.FindByName(name, authContainer.AuthContainerID)
		if err != nil {
			return err
		}
		if existingIdp != nil {
			return errors.New(name + " idp already exists")
		}

		// Generate identifier
		identifier := fmt.Sprintf("idp-%s", util.GenerateIdentifier(12))

		// Create idp
		newIdp := &model.IdentityProvider{
			Name:            name,
			DisplayName:     displayName,
			ProviderType:    providerType,
			Identifier:      identifier,
			Config:          config,
			AuthContainerID: authContainer.AuthContainerID,
			IsActive:        isActive,
			IsDefault:       isDefault,
		}

		_, err = txIdpRepo.CreateOrUpdate(newIdp)
		if err != nil {
			return err
		}

		// Fetch idp with Service preloaded
		createdIdp, err = txIdpRepo.FindByUUID(newIdp.IdentityProviderUUID, "AuthContainer")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toIdpServiceDataResult(createdIdp), nil
}

func (s *identityProviderService) Update(idpUUID uuid.UUID, name string, displayName string, providerType string, config datatypes.JSON, isActive bool, isDefault bool, authContainerUUID string) (*IdentityProviderServiceDataResult, error) {
	var updatedIdp *model.IdentityProvider

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txIdpRepo := s.idpRepo.WithTx(tx)
		txAuthContainerRepo := s.authContainerRepo.WithTx(tx)

		// Check if auth container exist
		authContainer, err := txAuthContainerRepo.FindByUUID(authContainerUUID)
		if err != nil || authContainer == nil {
			return errors.New("auth container not found")
		}

		// Get idp
		idp, err := txIdpRepo.FindByUUID(idpUUID, "AuthContainer")
		if err != nil || idp == nil {
			return errors.New("idp not found")
		}

		// Check if default
		if idp.IsDefault {
			return errors.New("default idp cannot be updated")
		}

		// Check if idp already exist
		if idp.Name != name {
			existingIdp, err := txIdpRepo.FindByName(name, authContainer.AuthContainerID)
			if err != nil {
				return err
			}
			if existingIdp != nil && existingIdp.IdentityProviderUUID != idpUUID {
				return errors.New(name + " idp already exists")
			}
		}

		// Set values
		idp.Name = name
		idp.DisplayName = displayName
		idp.ProviderType = providerType
		idp.Config = config
		idp.IsActive = isActive
		idp.IsDefault = isDefault

		// Update
		_, err = txIdpRepo.CreateOrUpdate(idp)
		if err != nil {
			return err
		}

		updatedIdp = idp

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toIdpServiceDataResult(updatedIdp), nil
}

func (s *identityProviderService) SetActiveStatusByUUID(idpUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	var updatedIdp *model.IdentityProvider

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txIdpRepo := s.idpRepo.WithTx(tx)

		// Get idp
		idp, err := txIdpRepo.FindByUUID(idpUUID, "AuthContainer")
		if err != nil || idp == nil {
			return errors.New("idp not found")
		}

		// Check if default
		if idp.IsDefault {
			return errors.New("default idp cannot be updated")
		}

		idp.IsActive = !idp.IsActive

		_, err = txIdpRepo.CreateOrUpdate(idp)
		if err != nil {
			return err
		}

		updatedIdp = idp

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toIdpServiceDataResult(updatedIdp), nil
}

func (s *identityProviderService) DeleteByUUID(idpUUID uuid.UUID) (*IdentityProviderServiceDataResult, error) {
	// Get idp
	idp, err := s.idpRepo.FindByUUID(idpUUID, "AuthContainer")
	if err != nil || idp == nil {
		return nil, errors.New("idp not found")
	}

	// Check if default
	if idp.IsDefault {
		return nil, errors.New("default idp cannot be deleted")
	}

	err = s.idpRepo.DeleteByUUID(idpUUID)
	if err != nil {
		return nil, err
	}

	return toIdpServiceDataResult(idp), nil
}

// Reponse builder
func toIdpServiceDataResult(idp *model.IdentityProvider) *IdentityProviderServiceDataResult {
	if idp == nil {
		return nil
	}

	result := &IdentityProviderServiceDataResult{
		IdentityProviderUUID: idp.IdentityProviderUUID,
		Name:                 idp.Name,
		DisplayName:          idp.DisplayName,
		ProviderType:         idp.ProviderType,
		Identifier:           idp.Identifier,
		Config:               idp.Config,
		IsActive:             idp.IsActive,
		IsDefault:            idp.IsDefault,
		CreatedAt:            idp.CreatedAt,
		UpdatedAt:            idp.UpdatedAt,
	}

	if idp.AuthContainer != nil {
		result.AuthContainer = &AuthContainerServiceDataResult{
			AuthContainerUUID: idp.AuthContainer.AuthContainerUUID,
			Name:              idp.AuthContainer.Name,
			Description:       idp.AuthContainer.Description,
			Identifier:        idp.AuthContainer.Identifier,
			IsActive:          idp.AuthContainer.IsActive,
			IsPublic:          idp.AuthContainer.IsPublic,
			IsDefault:         idp.AuthContainer.IsDefault,
			CreatedAt:         idp.AuthContainer.CreatedAt,
			UpdatedAt:         idp.AuthContainer.UpdatedAt,
		}
	}

	return result
}
