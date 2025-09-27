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

type AuthClientSecretServiceDataResult struct {
	ClientID     string
	ClientSecret *string
}

type AuthClientConfigServiceDataResult struct {
	Config datatypes.JSON
}

type AuthClientRedirectURIServiceDataResult struct {
	AuthClientRedirectURIUUID uuid.UUID
	RedirectURI               string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

type AuthClientServiceDataResult struct {
	AuthClientUUID         uuid.UUID
	Name                   string
	DisplayName            string
	ClientType             string
	Domain                 *string
	AuthClientRedirectURIs *[]AuthClientRedirectURIServiceDataResult
	IdentityProvider       *IdentityProviderServiceDataResult
	Permissions            *[]PermissionServiceDataResult
	IsActive               bool
	IsDefault              bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
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
	GetSecretByUUID(authClientUUID uuid.UUID) (*AuthClientSecretServiceDataResult, error)
	GetConfigByUUID(authClientUUID uuid.UUID) (*AuthClientConfigServiceDataResult, error)
	Create(name string, displayName string, clientType string, domain string, config datatypes.JSON, isActive bool, isDefault bool, identityProviderUUID string) (*AuthClientServiceDataResult, error)
	Update(authClientUUID uuid.UUID, name string, displayName string, clientType string, domain string, config datatypes.JSON, isActive bool, isDefault bool) (*AuthClientServiceDataResult, error)
	SetActiveStatusByUUID(authClientUUID uuid.UUID) (*AuthClientServiceDataResult, error)
	DeleteByUUID(authClientUUID uuid.UUID) (*AuthClientServiceDataResult, error)
	CreateRedirectURI(authClientUUID uuid.UUID, redirectURI string) (*AuthClientServiceDataResult, error)
	UpdateRedirectURI(authClientUUID uuid.UUID, authClientRedirectURIUUID uuid.UUID, redirectURI string) (*AuthClientServiceDataResult, error)
	DeleteRedirectURI(authClientUUID uuid.UUID, authClientRedirectURIUUID uuid.UUID) (*AuthClientServiceDataResult, error)
	AddAuthClientPermissions(authClientUUID uuid.UUID, permissionUUIDs []uuid.UUID) (*AuthClientServiceDataResult, error)
	RemoveAuthClientPermissions(authClientUUID uuid.UUID, permissionUUID uuid.UUID) (*AuthClientServiceDataResult, error)
}

type authClientService struct {
	db                        *gorm.DB
	authClientRepo            repository.AuthClientRepository
	authClientRedirectUrlRepo repository.AuthClientRedirectURIRepository
	idpRepo                   repository.IdentityProviderRepository
	permissionRepo            repository.PermissionRepository
	authClientPermissionRepo  repository.AuthClientPermissionRepository
}

func NewAuthClientService(
	db *gorm.DB,
	authClientRepo repository.AuthClientRepository,
	authClientRedirectUrlRepo repository.AuthClientRedirectURIRepository,
	idpRepo repository.IdentityProviderRepository,
	permissionRepo repository.PermissionRepository,
	authClientPermissionRepo repository.AuthClientPermissionRepository,
) AuthClientService {
	return &authClientService{
		db:                        db,
		authClientRepo:            authClientRepo,
		authClientRedirectUrlRepo: authClientRedirectUrlRepo,
		idpRepo:                   idpRepo,
		permissionRepo:            permissionRepo,
		authClientPermissionRepo:  authClientPermissionRepo,
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
	authClient, err := s.authClientRepo.FindByUUID(authClientUUID, "IdentityProvider", "AuthClientRedirectURIs")
	if err != nil || authClient == nil {
		return nil, errors.New("auth client not found")
	}

	return toAuthClientServiceDataResult(authClient), nil
}

func (s *authClientService) GetSecretByUUID(authClientUUID uuid.UUID) (*AuthClientSecretServiceDataResult, error) {
	authClient, err := s.authClientRepo.FindByUUID(authClientUUID)
	if err != nil || authClient == nil {
		return nil, errors.New("auth client not found")
	}

	return &AuthClientSecretServiceDataResult{
		ClientID:     *authClient.ClientID,
		ClientSecret: authClient.ClientSecret,
	}, nil
}

func (s *authClientService) GetConfigByUUID(authClientUUID uuid.UUID) (*AuthClientConfigServiceDataResult, error) {
	authClient, err := s.authClientRepo.FindByUUID(authClientUUID)
	if err != nil || authClient == nil {
		return nil, errors.New("auth client not found")
	}

	return &AuthClientConfigServiceDataResult{
		Config: authClient.Config,
	}, nil
}

func (s *authClientService) Create(name string, displayName string, clientType string, domain string, config datatypes.JSON, isActive bool, isDefault bool, identityProviderUUID string) (*AuthClientServiceDataResult, error) {
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
		createdAuthClient, err = txAuthClientRepo.FindByUUID(newAuthClient.AuthClientUUID, "IdentityProvider", "AuthClientRedirectURIs")
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

func (s *authClientService) Update(authClientUUID uuid.UUID, name string, displayName string, clientType string, domain string, config datatypes.JSON, isActive bool, isDefault bool) (*AuthClientServiceDataResult, error) {
	var updatedAuthClient *model.AuthClient

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthClientRepo := s.authClientRepo.WithTx(tx)

		// Get auth client
		authClient, err := txAuthClientRepo.FindByUUID(authClientUUID, "IdentityProvider", "AuthClientRedirectURIs")
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
		authClient, err := txAuthClientRepo.FindByUUID(authClientUUID, "IdentityProvider", "AuthClientRedirectURIs")
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
	var deletedAuthClient *model.AuthClient

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthClientRepo := s.authClientRepo.WithTx(tx)

		// Get auth client
		authClient, err := txAuthClientRepo.FindByUUID(authClientUUID, "IdentityProvider", "AuthClientRedirectURIs")
		if err != nil || authClient == nil {
			return errors.New("auth client not found")
		}

		// Check if default
		if authClient.IsDefault {
			return errors.New("default auth client cannot be deleted")
		}

		// Delete
		if err := txAuthClientRepo.DeleteByUUID(authClientUUID); err != nil {
			return err
		}

		deletedAuthClient = authClient

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toAuthClientServiceDataResult(deletedAuthClient), nil
}

func (s *authClientService) CreateRedirectURI(authClientUUID uuid.UUID, redirectURI string) (*AuthClientServiceDataResult, error) {
	var createdAuthClient *model.AuthClient

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txRedirectURIRepo := s.authClientRedirectUrlRepo.WithTx(tx)

		// Find the auth client by UUID
		authClient, err := txAuthClientRepo.FindByUUID(authClientUUID)
		if err != nil || authClient == nil {
			return errors.New("auth client not found")
		}

		// Create the redirect URI entry
		newRedirectURI := &model.AuthClientRedirectURI{
			AuthClientID: authClient.AuthClientID,
			RedirectURI:  redirectURI,
		}

		_, err = txRedirectURIRepo.CreateOrUpdate(newRedirectURI)
		if err != nil {
			return err
		}

		// Find the auth client updated with the new redirect URI
		authClientCreated, err := txAuthClientRepo.FindByUUID(authClientUUID, "IdentityProvider", "AuthClientRedirectURIs")
		if err != nil || authClientCreated == nil {
			return errors.New("auth client not found")
		}

		createdAuthClient = authClientCreated

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toAuthClientServiceDataResult(createdAuthClient), nil
}

func (s *authClientService) UpdateRedirectURI(authClientUUID uuid.UUID, authClientRedirectURIUUID uuid.UUID, redirectURI string) (*AuthClientServiceDataResult, error) {
	var updatedAuthClient *model.AuthClient

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txRedirectURIRepo := s.authClientRedirectUrlRepo.WithTx(tx)

		// Find the auth client by UUID
		authClient, err := txAuthClientRepo.FindByUUID(authClientUUID)
		if err != nil || authClient == nil {
			return errors.New("auth client not found")
		}

		// Find the redirect URI entry by UUID
		existingRedirectURI, err := txRedirectURIRepo.FindByUUID(authClientRedirectURIUUID)
		if err != nil || existingRedirectURI == nil {
			return errors.New("redirect URI not found")
		}

		// Check if the redirect URI belongs to the auth client
		if existingRedirectURI.AuthClientID != authClient.AuthClientID {
			return errors.New("redirect URI does not belong to the specified auth client")
		}

		// Set new value
		existingRedirectURI.RedirectURI = redirectURI

		// Update
		_, err = txRedirectURIRepo.CreateOrUpdate(existingRedirectURI)
		if err != nil {
			return err
		}

		// Find the auth client updated with the new redirect URI
		authClientUpdated, err := txAuthClientRepo.FindByUUID(authClientUUID, "IdentityProvider", "AuthClientRedirectURIs")
		if err != nil || authClientUpdated == nil {
			return errors.New("auth client not found")
		}

		updatedAuthClient = authClientUpdated

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toAuthClientServiceDataResult(updatedAuthClient), nil
}

func (s *authClientService) DeleteRedirectURI(authClientUUID uuid.UUID, authClientRedirectURIUUID uuid.UUID) (*AuthClientServiceDataResult, error) {
	var deletedAuthClient *model.AuthClient

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txRedirectURIRepo := s.authClientRedirectUrlRepo.WithTx(tx)

		// Find the auth client by UUID
		authClient, err := txAuthClientRepo.FindByUUID(authClientUUID, "IdentityProvider", "AuthClientRedirectURIs")
		if err != nil || authClient == nil {
			return errors.New("auth client not found")
		}

		// Find the redirect URI entry by UUID
		existingRedirectURI, err := txRedirectURIRepo.FindByUUID(authClientRedirectURIUUID)
		if err != nil || existingRedirectURI == nil {
			return errors.New("redirect URI not found")
		}

		// Check if the redirect URI belongs to the auth client
		if existingRedirectURI.AuthClientID != authClient.AuthClientID {
			return errors.New("redirect URI does not belong to the specified auth client")
		}

		// Delete the entry
		if err := txRedirectURIRepo.DeleteByUUID(authClientRedirectURIUUID); err != nil {
			return err
		}

		deletedAuthClient = authClient

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toAuthClientServiceDataResult(deletedAuthClient), nil
}

func (s *authClientService) AddAuthClientPermissions(authClientUUID uuid.UUID, permissionUUIDs []uuid.UUID) (*AuthClientServiceDataResult, error) {
	var authClientWithPermissions *model.AuthClient

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txPermissionRepo := s.permissionRepo.WithTx(tx)
		txAuthClientPermissionRepo := s.authClientPermissionRepo.WithTx(tx)

		// Find existing auth client
		authClient, err := txAuthClientRepo.FindByUUID(authClientUUID)
		if err != nil {
			return err
		}
		if authClient == nil {
			return errors.New("auth client not found")
		}

		// Convert UUIDs to strings for the repository method
		permissionUUIDStrings := make([]string, len(permissionUUIDs))
		for i, uuid := range permissionUUIDs {
			permissionUUIDStrings[i] = uuid.String()
		}

		// Find permissions by UUIDs
		permissions, err := txPermissionRepo.FindByUUIDs(permissionUUIDStrings)
		if err != nil {
			return err
		}

		// Validate that all permissions were found
		if len(permissions) != len(permissionUUIDs) {
			return errors.New("one or more permissions not found")
		}

		// Create auth client-permission associations using the dedicated repository
		for _, permission := range permissions {
			// Check if association already exists
			existing, err := txAuthClientPermissionRepo.FindByAuthClientAndPermission(authClient.AuthClientID, permission.PermissionID)
			if err != nil {
				return err
			}

			// Skip if association already exists
			if existing != nil {
				continue
			}

			// Create new auth client-permission association
			authClientPermission := &model.AuthClientPermission{
				AuthClientID: authClient.AuthClientID,
				PermissionID: permission.PermissionID,
			}

			_, err = txAuthClientPermissionRepo.Create(authClientPermission)
			if err != nil {
				return err
			}
		}

		// Fetch the auth client with permissions for the response
		authClientWithPermissions, err = txAuthClientRepo.FindByUUID(authClientUUID, "Permissions")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toAuthClientServiceDataResult(authClientWithPermissions), nil
}

func (s *authClientService) RemoveAuthClientPermissions(authClientUUID uuid.UUID, permissionUUID uuid.UUID) (*AuthClientServiceDataResult, error) {
	var authClientWithPermissions *model.AuthClient

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txAuthClientRepo := s.authClientRepo.WithTx(tx)
		txPermissionRepo := s.permissionRepo.WithTx(tx)
		txAuthClientPermissionRepo := s.authClientPermissionRepo.WithTx(tx)

		// Find existing auth client
		authClient, err := txAuthClientRepo.FindByUUID(authClientUUID)
		if err != nil {
			return err
		}
		if authClient == nil {
			return errors.New("auth client not found")
		}

		// Find permission by UUID
		permission, err := txPermissionRepo.FindByUUID(permissionUUID.String())
		if err != nil {
			return err
		}
		if permission == nil {
			return errors.New("permission not found")
		}

		// Check if association exists
		existing, err := txAuthClientPermissionRepo.FindByAuthClientAndPermission(authClient.AuthClientID, permission.PermissionID)
		if err != nil {
			return err
		}

		// Skip if association doesn't exist
		if existing == nil {
			// Association doesn't exist, but we'll still return success for idempotency
			authClientWithPermissions, err = txAuthClientRepo.FindByUUID(authClientUUID, "Permissions")
			if err != nil {
				return err
			}
			return nil
		}

		// Remove the auth client-permission association
		err = txAuthClientPermissionRepo.RemoveByAuthClientAndPermission(authClient.AuthClientID, permission.PermissionID)
		if err != nil {
			return err
		}

		// Fetch the auth client with permissions for the response
		authClientWithPermissions, err = txAuthClientRepo.FindByUUID(authClientUUID, "Permissions")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return toAuthClientServiceDataResult(authClientWithPermissions), nil
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

	// Map Redirect URIs
	if authClient.AuthClientRedirectURIs != nil && len(*authClient.AuthClientRedirectURIs) > 0 {
		redirects := make([]AuthClientRedirectURIServiceDataResult, len(*authClient.AuthClientRedirectURIs))
		for i, r := range *authClient.AuthClientRedirectURIs {
			redirects[i] = AuthClientRedirectURIServiceDataResult{
				AuthClientRedirectURIUUID: r.AuthClientRedirectURIUUID,
				RedirectURI:               r.RedirectURI,
				CreatedAt:                 r.CreatedAt,
				UpdatedAt:                 r.UpdatedAt,
			}
		}
		result.AuthClientRedirectURIs = &redirects
	}

	// Map Permissions if present
	if authClient.Permissions != nil {
		permissions := make([]PermissionServiceDataResult, len(*authClient.Permissions))
		for i, permission := range *authClient.Permissions {
			permissions[i] = PermissionServiceDataResult{
				PermissionUUID: permission.PermissionUUID,
				Name:           permission.Name,
				Description:    permission.Description,
				IsActive:       permission.IsActive,
				IsDefault:      permission.IsDefault,
				CreatedAt:      permission.CreatedAt,
				UpdatedAt:      permission.UpdatedAt,
			}
		}
		result.Permissions = &permissions
	}

	return result
}
