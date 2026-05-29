package idp

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type FederationTokenRequestDTO struct {
	// ProviderIdentifier is the identifier of the configured IdentityProvider
	// record (e.g. "idp-abc123xyz").
	ProviderIdentifier string `json:"provider_identifier"`
	// ExternalToken is the raw OIDC ID token (JWT) from the upstream provider.
	ExternalToken string `json:"external_token"`
	// ClientID is our OAuth2 client identifier used to scope the issued tokens.
	ClientID string `json:"client_id"`
}

// Identity link / unlink

// LinkIdentityRequestDTO is the body for POST /account/identities/link.
type LinkIdentityRequestDTO struct {
	ProviderIdentifier string `json:"provider_identifier"`
	ExternalToken      string `json:"external_token"`
}

// IdentityDTO is the public view of a UserIdentity record.
type IdentityDTO struct {
	IdentityUUID string  `json:"identity_uuid"`
	Provider     string  `json:"provider"`
	Sub          string  `json:"sub"`
	IsDefault    bool    `json:"is_default"`
	CreatedAt    string  `json:"created_at"`
	Email        *string `json:"email,omitempty"`
	Name         *string `json:"name,omitempty"`
	Picture      *string `json:"picture,omitempty"`
}

// Home Realm Discovery

// HRDResponseDTO tells the frontend which provider handles the given email.
type HRDResponseDTO struct {
	ProviderIdentifier string `json:"provider_identifier"`
	Provider           string `json:"provider"`
	DisplayName        string `json:"display_name"`
}

// Identity provider list response structure (without config and tenant)
type IdentityProviderResponseDTO struct {
	IdentityProviderUUID uuid.UUID `json:"identity_provider_id"`
	Name                 string    `json:"name"`
	DisplayName          string    `json:"display_name"`
	Provider             string    `json:"provider"`
	ProviderType         string    `json:"provider_type"`
	Identifier           string    `json:"identifier"`
	Status               string    `json:"status"`
	IsDefault            bool      `json:"is_default"`
	IsSystem             bool      `json:"is_system"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// Identity provider detail response structure (with config and tenant)
type IdentityProviderDetailResponseDTO struct {
	IdentityProviderUUID uuid.UUID          `json:"identity_provider_id"`
	Name                 string             `json:"name"`
	DisplayName          string             `json:"display_name"`
	Provider             string             `json:"provider"`
	ProviderType         string             `json:"provider_type"`
	Identifier           string             `json:"identifier"`
	Config               *datatypes.JSON    `json:"config,omitempty"`
	Tenant               *TenantResponseDTO `json:"tenant,omitempty"`
	Status               string             `json:"status"`
	IsDefault            bool               `json:"is_default"`
	IsSystem             bool               `json:"is_system"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

// Create identity provider request DTO
type IdentityProviderCreateRequestDTO struct {
	Name         string         `json:"name"`
	DisplayName  string         `json:"display_name"`
	Provider     string         `json:"provider"`
	ProviderType string         `json:"provider_type"`
	Config       datatypes.JSON `json:"config"`
	Status       string         `json:"status"`
	TenantUUID   string         `json:"tenant_id"`
}

// Validation for create request
func (r IdentityProviderCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(8, 200).Error("Display name must be between 8 and 200 characters"),
		),
		validation.Field(&r.Provider,
			validation.Required.Error("Provider is required"),
			validation.In(shared.IDPProviderInternal, shared.IDPProviderCognito, shared.IDPProviderAuth0, shared.IDPProviderGoogle, shared.IDPProviderFacebook, shared.IDPProviderGitHub, shared.IDPProviderMicrosoft, shared.IDPProviderApple, shared.IDPProviderLinkedIn, shared.IDPProviderTwitter).Error("Provider must be one of: internal, cognito, auth0, google, facebook, github, microsoft, apple, linkedin, twitter"),
		),
		validation.Field(&r.ProviderType,
			validation.Required.Error("Provider type is required"),
			validation.In(shared.IDPTypeIdentity, shared.IDPTypeSocial).Error("Provider type must be either 'identity' or 'social'"),
		),
		validation.Field(&r.Config,
			validation.Required.Error("Config is required"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be either 'active' or 'inactive'"),
		),
		validation.Field(&r.TenantUUID,
			validation.Required.Error("Tenant UUID is required"),
			is.UUID.Error("Tenant UUID must be a valid UUID"),
		),
	)
}

// Update identity provider request DTO (without tenant_id)
type IdentityProviderUpdateRequestDTO struct {
	Name         string         `json:"name"`
	DisplayName  string         `json:"display_name"`
	Provider     string         `json:"provider"`
	ProviderType string         `json:"provider_type"`
	Config       datatypes.JSON `json:"config"`
	Status       string         `json:"status"`
}

// Validation for update request
func (r IdentityProviderUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(8, 200).Error("Display name must be between 8 and 200 characters"),
		),
		validation.Field(&r.Provider,
			validation.Required.Error("Provider is required"),
			validation.In(shared.IDPProviderInternal, shared.IDPProviderCognito, shared.IDPProviderAuth0, shared.IDPProviderGoogle, shared.IDPProviderFacebook, shared.IDPProviderGitHub, shared.IDPProviderMicrosoft, shared.IDPProviderApple, shared.IDPProviderLinkedIn, shared.IDPProviderTwitter).Error("Provider must be one of: internal, cognito, auth0, google, facebook, github, microsoft, apple, linkedin, twitter"),
		),
		validation.Field(&r.ProviderType,
			validation.Required.Error("Provider type is required"),
			validation.In(shared.IDPTypeIdentity, shared.IDPTypeSocial).Error("Provider type must be either 'identity' or 'social'"),
		),
		validation.Field(&r.Config,
			validation.Required.Error("Config is required"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be either 'active' or 'inactive'"),
		),
	)
}

// Identity provider status update DTO
type IdentityProviderStatusUpdateDTO struct {
	Status string `json:"status"`
}

func (r IdentityProviderStatusUpdateDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be either 'active' or 'inactive'"),
		),
	)
}

// Identity provider listing / filter DTO
type IdentityProviderFilterDTO struct {
	Name         *string  `json:"name"`
	DisplayName  *string  `json:"display_name"`
	Provider     []string `json:"provider"`
	ProviderType *string  `json:"provider_type"`
	Identifier   *string  `json:"identifier"`
	Status       []string `json:"status"`
	IsDefault    *bool    `json:"is_default"`
	IsSystem     *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validate validates the identity provider filter DTO.
func (f IdentityProviderFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Provider,
			validation.When(len(f.Provider) > 0,
				validation.Each(validation.In(shared.IDPProviderInternal, shared.IDPProviderCognito, shared.IDPProviderAuth0, shared.IDPProviderGoogle, shared.IDPProviderFacebook, shared.IDPProviderGitHub, shared.IDPProviderMicrosoft, shared.IDPProviderApple, shared.IDPProviderLinkedIn, shared.IDPProviderTwitter).Error("Invalid identity provider")),
			),
		),
		validation.Field(&f.ProviderType,
			validation.When(f.ProviderType != nil,
				validation.In(shared.IDPTypeIdentity, shared.IDPTypeSocial).Error("Provider type must be one of: identity, social"),
			),
		),
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}

// Signup flow output structure
type SignupFlowResponseDTO struct {
	SignupFlowUUID string         `json:"signup_flow_id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Identifier     string         `json:"identifier"`
	Config         map[string]any `json:"config"`
	Status         string         `json:"status"`
	ClientUUID     string         `json:"client_id,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// Create signup flow request dto
type SignupFlowCreateRequestDTO struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Config      map[string]any `json:"config,omitempty"`
	Status      *string        `json:"status,omitempty"`
	ClientUUID  string         `json:"client_id"`
}

func (r SignupFlowCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Signup flow name is required"),
			validation.Length(1, 100).Error("Signup flow name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
		validation.Field(&r.ClientUUID,
			validation.Required.Error("Auth client UUID is required"),
			is.UUID.Error("Invalid auth client UUID format"),
		),
	)
}

// Update signup flow request dto
type SignupFlowUpdateRequestDTO struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Config      map[string]any `json:"config,omitempty"`
	Status      *string        `json:"status,omitempty"`
}

func (r SignupFlowUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Signup flow name is required"),
			validation.Length(1, 100).Error("Signup flow name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update signup flow status request dto
type SignupFlowUpdateStatusRequestDTO struct {
	Status string `json:"status"`
}

func (r SignupFlowUpdateStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Signup flow listing request dto
type SignupFlowFilterDTO struct {
	Name       *string  `json:"name"`
	Identifier *string  `json:"identifier"`
	Status     []string `json:"status"`
	ClientUUID *string  `json:"client_id"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validate validates the signup flow filter DTO.
func (f SignupFlowFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.ClientUUID,
			validation.When(f.ClientUUID != nil,
				is.UUID.Error("Client ID must be a valid UUID"),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}

// Signup flow role output structure
type SignupFlowRoleResponseDTO struct {
	SignupFlowRoleUUID string    `json:"signup_flow_role_id"`
	SignupFlowUUID     string    `json:"signup_flow_id"`
	RoleUUID           string    `json:"role_id"`
	RoleName           string    `json:"role_name,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

// Assign roles to signup flow request dto
type SignupFlowAssignRolesRequestDTO struct {
	RoleUUIDs []string `json:"role_uuids"`
}

func (r SignupFlowAssignRolesRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.RoleUUIDs,
			validation.Required.Error("Role UUIDs are required"),
			validation.Length(1, 0).Error("At least one role UUID is required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

type IdentityProvider struct {
	IdentityProviderID   int64          `gorm:"column:identity_provider_id;primaryKey"`
	IdentityProviderUUID uuid.UUID      `gorm:"column:identity_provider_uuid"`
	TenantID             int64          `gorm:"column:tenant_id"`
	Name                 string         `gorm:"column:name"`
	DisplayName          string         `gorm:"column:display_name"`
	Provider             string         `gorm:"column:provider"`
	ProviderType         string         `gorm:"column:provider_type"`
	Identifier           string         `gorm:"column:identifier"`
	Config               datatypes.JSON `gorm:"column:config"`
	Status               string         `gorm:"column:status;default:'inactive'"`
	IsDefault            bool           `gorm:"column:is_default;default:false"`
	IsSystem             bool           `gorm:"column:is_system;default:false"`
	CreatedBy            *int64         `gorm:"column:created_by"`
	UpdatedBy            *int64         `gorm:"column:updated_by"`
	CreatedAt            time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt            time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt            gorm.DeletedAt `gorm:"column:deleted_at;index"`

	// Relationships
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
}

func (IdentityProvider) TableName() string {
	return "identity_providers"
}

func (ip *IdentityProvider) BeforeCreate(tx *gorm.DB) (err error) {
	if ip.IdentityProviderUUID == uuid.Nil {
		ip.IdentityProviderUUID = uuid.New()
	}
	return
}
