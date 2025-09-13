package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AuthClientServiceDataResult struct {
	AuthClientUUID   uuid.UUID
	Name             string
	DisplayName      string
	ClientType       string
	Domain           *string
	RedirectURI      *string
	IdentityProvider *IdentityProviderServiceDataResult
	IsActive         bool
	IsDefault        bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type AuthClientServiceGetFilter struct {
	Name                 *string
	DisplayName          *string
	ClientType           *string
	IdentityProviderUUID *string
	IsActive             *bool
	IsDefault            *bool
	Page                 int
	Limit                int
	SortBy               string
	SortOrder            string
}

type AuthClientServiceGetResult struct {
	Data       []AuthClientServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type AuthClientService interface {
	Get(filter AuthClientServiceGetFilter) (*AuthClientServiceGetResult, error)
	GetByUUID(authClientUUID uuid.UUID) (*AuthClientServiceDataResult, error)
	Create(name string, displayName string, clientType string, domain string, redirectURI string, config datatypes.JSON, isActive bool, isDefault bool, identityProviderUUID string) (*AuthClientServiceDataResult, error)
	Update(authClientUUID uuid.UUID, name string, displayName string, clientType string, domain string, redirectURI string, config datatypes.JSON, isActive bool, isDefault bool) (*AuthClientServiceDataResult, error)
	SetActiveStatusByUUID(authClientUUID uuid.UUID) (*AuthClientServiceDataResult, error)
	DeleteByUUID(authClientUUID uuid.UUID) (*AuthClientServiceDataResult, error)
}

type authClientService struct {
	db             *gorm.DB
	authClientRepo repository.AuthClientRepository
	idpRepo        repository.IdentityProviderRepository
}

func NewAuthClientService(
	db *gorm.DB,
	authClientRepo repository.AuthClientRepository,
	idpRepo repository.IdentityProviderRepository,
) AuthClientService {
	return &authClientService{
		db:             db,
		authClientRepo: authClientRepo,
		idpRepo:        idpRepo,
	}
}

func (s *authClientService) Get(filter AuthClientServiceGetFilter) (*AuthClientServiceGetResult, error) {
	var idpID *int64

	// Get identity provider
	if filter.IdentityProviderUUID != nil {
		idp, err := s.idpRepo.FindByUUID(*filter.IdentityProviderUUID)
		if err != nil || idp == nil {
			return nil, errors.New("identity provider not found")
		}
		idpID = &idp.IdentityProviderID
	}

	// Build query filter
	queryFilter := repository.AuthClientRepositoryGetFilter{
		Name:               filter.Name,
		DisplayName:        filter.DisplayName,
		ClientType:         filter.ClientType,
		IdentityProviderID: idpID,
		IsActive:           filter.IsActive,
		IsDefault:          filter.IsDefault,
		Page:               filter.Page,
		Limit:              filter.Limit,
		SortBy:             filter.SortBy,
		SortOrder:          filter.SortOrder,
	}

	result, err := s.authClientRepo.FindPaginated(queryFilter)
	if err != nil {
		return nil, err
	}

	// Build response data
	resData := make([]AuthClientServiceDataResult, len(result.Data))
	for i, rdata := range result.Data {
		resData[i] = *toAuthClientServiceDataResult(&rdata)
	}

	return &AuthClientServiceGetResult{
		Data:       resData,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *authClientService) GetByUUID(authClientUUID uuid.UUID) (*AuthClientServiceDataResult, error) {
	authClient, err := s.authClientRepo.FindByUUID(authClientUUID, "IdentityProvider")
	if err != nil || authClient == nil {
		return nil, errors.New("auth client not found")
	}

	return toAuthClientServiceDataResult(authClient), nil
}

func (s *authClientService) Create(name string, displayName string, clientType string, domain string, redirectURI string, config datatypes.JSON, isActive bool, isDefault bool, identityProviderUUID string) (*AuthClientServiceDataResult, error) {
	var createdAuthClient *model.AuthClient

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txIdpRepo := s.idpRepo.WithTx(tx)

		// Check if identity provider exists
		identityProvider, err := txIdpRepo.FindByUUID(identityProviderUUID)
		if err != nil || identityProvider == nil {
			return errors.New("identity provider not found")
		}

		// Check if auth client already exists
		existingAuthClient, err := txAuthClientRepo.FindByNameAndIdentityProvider(name, identityProvider.IdentityProviderID)
		if err != nil {
			return err
		}
		if existingAuthClient != nil {
			return errors.New(name + " auth client already exists")
		}

		// Generate identifier
		clientId := util.GenerateIdentifier(12)
		clientSecret := util.GenerateIdentifier(64)

		// Create auth client
		newAuthClient := &model.AuthClient{
			Name:               name,
			DisplayName:        displayName,
			ClientType:         clientType,
			Domain:             &domain,
			ClientID:           &clientId,
			ClientSecret:       &clientSecret,
			RedirectURI:        &redirectURI,
			Config:             config,
			IdentityProviderID: identityProvider.IdentityProviderID,
			IsActive:           isActive,
			IsDefault:          isDefault,
		}

		_, err = txAuthClientRepo.CreateOrUpdate(newAuthClient)
		if err != nil {
			return err
		}

		// Fetch AuthClient with Service preloaded
		createdAuthClient, err = txAuthClientRepo.FindByUUID(newAuthClient.AuthClientUUID, "IdentityProvider")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toAuthClientServiceDataResult(createdAuthClient), nil
}

func (s *authClientService) Update(authClientUUID uuid.UUID, name string, displayName string, clientType string, domain string, redirectURI string, config datatypes.JSON, isActive bool, isDefault bool) (*AuthClientServiceDataResult, error) {
	var updatedAuthClient *model.AuthClient

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthClientRepo := s.authClientRepo.WithTx(tx)

		// Get auth client
		authClient, err := txAuthClientRepo.FindByUUID(authClientUUID, "IdentityProvider")
		if err != nil || authClient == nil {
			return errors.New("auth client not found")
		}

		// Check if default
		if authClient.IsDefault {
			return errors.New("default auth client cannot cannot be updated")
		}

		// Check if auth client already exist
		if authClient.Name != name {
			existingAuthClient, err := txAuthClientRepo.FindByNameAndIdentityProvider(name, authClient.IdentityProviderID)
			if err != nil {
				return err
			}
			if existingAuthClient != nil && existingAuthClient.AuthClientUUID != authClientUUID {
				return errors.New(name + " auth client already exists")
			}
		}

		// Set values
		authClient.Name = name
		authClient.DisplayName = displayName
		authClient.ClientType = clientType
		authClient.Domain = &domain
		authClient.RedirectURI = &redirectURI
		authClient.Config = config
		authClient.IsActive = isActive
		authClient.IsDefault = isDefault

		// Update
		_, err = txAuthClientRepo.CreateOrUpdate(authClient)
		if err != nil {
			return err
		}

		updatedAuthClient = authClient

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toAuthClientServiceDataResult(updatedAuthClient), nil
}

func (s *authClientService) SetActiveStatusByUUID(authClientUUID uuid.UUID) (*AuthClientServiceDataResult, error) {
	var updatedAuthClient *model.AuthClient

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthClientRepo := s.authClientRepo.WithTx(tx)

		// Get auth client
		authClient, err := txAuthClientRepo.FindByUUID(authClientUUID, "IdentityProvider")
		if err != nil || authClient == nil {
			return errors.New("auth client not found")
		}

		// Check if default
		if authClient.IsDefault {
			return errors.New("default auth client cannot cannot be updated")
		}

		// Set values
		authClient.IsActive = !authClient.IsActive

		// Update
		_, err = txAuthClientRepo.CreateOrUpdate(authClient)
		if err != nil {
			return err
		}

		updatedAuthClient = authClient

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toAuthClientServiceDataResult(updatedAuthClient), nil
}

func (s *authClientService) DeleteByUUID(authClientUUID uuid.UUID) (*AuthClientServiceDataResult, error) {
	// Get auth client
	authClient, err := s.authClientRepo.FindByUUID(authClientUUID, "IdentityProvider")
	if err != nil || authClient == nil {
		return nil, errors.New("auth client not found")
	}

	// Check if default
	if authClient.IsDefault {
		return nil, errors.New("default auth client cannot be deleted")
	}

	err = s.authClientRepo.DeleteByUUID(authClientUUID)
	if err != nil {
		return nil, err
	}

	return toAuthClientServiceDataResult(authClient), nil
}

// Reponse builder
func toAuthClientServiceDataResult(authClient *model.AuthClient) *AuthClientServiceDataResult {
	if authClient == nil {
		return nil
	}

	result := &AuthClientServiceDataResult{
		AuthClientUUID: authClient.AuthClientUUID,
		Name:           authClient.Name,
		DisplayName:    authClient.DisplayName,
		ClientType:     authClient.ClientType,
		Domain:         authClient.Domain,
		RedirectURI:    authClient.RedirectURI,
		IsActive:       authClient.IsActive,
		IsDefault:      authClient.IsDefault,
		CreatedAt:      authClient.CreatedAt,
		UpdatedAt:      authClient.UpdatedAt,
	}

	if authClient.IdentityProvider != nil {
		result.IdentityProvider = &IdentityProviderServiceDataResult{
			IdentityProviderUUID: authClient.IdentityProvider.IdentityProviderUUID,
			Name:                 authClient.IdentityProvider.Name,
			DisplayName:          authClient.IdentityProvider.DisplayName,
			ProviderType:         authClient.IdentityProvider.ProviderType,
			Identifier:           authClient.IdentityProvider.Identifier,
			IsActive:             authClient.IdentityProvider.IsActive,
			IsDefault:            authClient.IdentityProvider.IsDefault,
			CreatedAt:            authClient.IdentityProvider.CreatedAt,
			UpdatedAt:            authClient.IdentityProvider.UpdatedAt,
		}
	}

	return result
}
