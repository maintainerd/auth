package scim

import (
	"encoding/json"

	"gorm.io/datatypes"
)

// ---- SCIM Configuration DTOs ----

type SCIMConfigurationResponseDTO struct {
	SCIMConfigurationUUID string          `json:"scim_configuration_uuid"`
	TenantID              int64           `json:"tenant_id"`
	IdentityProviderID    *int64          `json:"identity_provider_id,omitempty"`
	DisplayName           string          `json:"display_name"`
	BaseURL               *string         `json:"base_url,omitempty"`
	SyncUsers             bool            `json:"sync_users"`
	SyncGroups            bool            `json:"sync_groups"`
	SyncDirection         string          `json:"sync_direction"`
	AttributeMapping      json.RawMessage `json:"attribute_mapping"`
	IsActive              bool            `json:"is_active"`
	LastSyncAt            *string         `json:"last_sync_at,omitempty"`
	LastSyncStatus        *string         `json:"last_sync_status,omitempty"`
	LastSyncError         *string         `json:"last_sync_error,omitempty"`
	CreatedAt             string          `json:"created_at"`
	UpdatedAt             string          `json:"updated_at"`
}

type SCIMConfigurationCreateDTO struct {
	IdentityProviderID *int64           `json:"identity_provider_id"`
	DisplayName        string           `json:"display_name"`
	BaseURL            *string          `json:"base_url"`
	BearerToken        *string          `json:"bearer_token"`
	SyncUsers          *bool            `json:"sync_users"`
	SyncGroups         *bool            `json:"sync_groups"`
	SyncDirection      *string          `json:"sync_direction"`
	AttributeMapping   *json.RawMessage `json:"attribute_mapping"`
	IsActive           *bool            `json:"is_active"`
}

type SCIMConfigurationUpdateDTO struct {
	IdentityProviderID *int64           `json:"identity_provider_id"`
	DisplayName        *string          `json:"display_name"`
	BaseURL            *string          `json:"base_url"`
	BearerToken        *string          `json:"bearer_token"`
	SyncUsers          *bool            `json:"sync_users"`
	SyncGroups         *bool            `json:"sync_groups"`
	SyncDirection      *string          `json:"sync_direction"`
	AttributeMapping   *json.RawMessage `json:"attribute_mapping"`
	IsActive           *bool            `json:"is_active"`
}

type SCIMConfigurationFilterDTO struct {
	Search    *string `json:"search"`
	IsActive  *bool   `json:"is_active"`
	Page      int     `json:"page"`
	Limit     int     `json:"limit"`
	SortBy    string  `json:"sort_by"`
	SortOrder string  `json:"sort_order"`
}

// ---- SCIM Configuration Service Types ----

type SCIMConfigurationServiceDataResult struct {
	SCIMConfigurationUUID string          `json:"scim_configuration_uuid"`
	TenantID              int64           `json:"tenant_id"`
	IdentityProviderID    *int64          `json:"identity_provider_id,omitempty"`
	DisplayName           string          `json:"display_name"`
	BaseURL               *string         `json:"base_url,omitempty"`
	SyncUsers             bool            `json:"sync_users"`
	SyncGroups            bool            `json:"sync_groups"`
	SyncDirection         string          `json:"sync_direction"`
	AttributeMapping      datatypes.JSON  `json:"attribute_mapping"`
	IsActive              bool            `json:"is_active"`
	LastSyncAt            *string         `json:"last_sync_at,omitempty"`
	LastSyncStatus        *string         `json:"last_sync_status,omitempty"`
	LastSyncError         *string         `json:"last_sync_error,omitempty"`
	CreatedAt             string          `json:"created_at"`
	UpdatedAt             string          `json:"updated_at"`
}

// ---- SCIM Protocol DTOs (RFC 7643 / 7644) ----

const SCIMUserSchema = "urn:ietf:params:scim:schemas:core:2.0:User"
const SCIMServiceProviderConfigSchema = "urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"
const SCIMListResponseSchema = "urn:ietf:params:scim:api:messages:2.0:ListResponse"

type SCIMListResponse struct {
	Schemas      []string    `json:"schemas"`
	TotalResults int         `json:"totalResults"`
	StartIndex   int         `json:"startIndex"`
	ItemsPerPage int         `json:"itemsPerPage"`
	Resources    interface{} `json:"Resources"`
}

type SCIMUserResource struct {
	Schemas    []string            `json:"schemas"`
	ID         string              `json:"id"`
	ExternalID *string             `json:"externalId,omitempty"`
	UserName   string              `json:"userName"`
	Name       *SCIMUserName       `json:"name,omitempty"`
	DisplayName *string            `json:"displayName,omitempty"`
	Emails     []SCIMEmail         `json:"emails,omitempty"`
	PhoneNumbers []SCIMPhoneNumber `json:"phoneNumbers,omitempty"`
	Active     bool               `json:"active"`
	Meta       *SCIMMeta           `json:"meta,omitempty"`
}

type SCIMUserName struct {
	Formatted  string `json:"formatted,omitempty"`
	FamilyName string `json:"familyName,omitempty"`
	GivenName  string `json:"givenName,omitempty"`
	MiddleName string `json:"middleName,omitempty"`
}

type SCIMEmail struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary"`
}

type SCIMPhoneNumber struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary"`
}

type SCIMMeta struct {
	ResourceType string `json:"resourceType"`
	Created      string `json:"created"`
	LastModified string `json:"lastModified"`
	Location     string `json:"location"`
}

type SCIMUserCreateRequest struct {
	Schemas      []string            `json:"schemas"`
	ExternalID   *string             `json:"externalId,omitempty"`
	UserName     string              `json:"userName"`
	Name         *SCIMUserName       `json:"name,omitempty"`
	DisplayName  *string             `json:"displayName,omitempty"`
	Emails       []SCIMEmail         `json:"emails,omitempty"`
	PhoneNumbers []SCIMPhoneNumber   `json:"phoneNumbers,omitempty"`
	Active       *bool               `json:"active,omitempty"`
}

type SCIMUserUpdateRequest struct {
	Schemas      []string            `json:"schemas"`
	ID           *string             `json:"id,omitempty"`
	ExternalID   *string             `json:"externalId,omitempty"`
	UserName     *string             `json:"userName,omitempty"`
	Name         *SCIMUserName       `json:"name,omitempty"`
	DisplayName  *string             `json:"displayName,omitempty"`
	Emails       []SCIMEmail         `json:"emails,omitempty"`
	PhoneNumbers []SCIMPhoneNumber   `json:"phoneNumbers,omitempty"`
	Active       *bool               `json:"active,omitempty"`
}

type SCIMPatchOperation struct {
	Op    string          `json:"op"`
	Path  *string         `json:"path,omitempty"`
	Value json.RawMessage `json:"value,omitempty"`
}

type SCIMPatchRequest struct {
	Schemas    []string              `json:"schemas"`
	Operations []SCIMPatchOperation `json:"Operations"`
}

type SCIMServiceProviderConfig struct {
	Schemas               []string             `json:"schemas"`
	DocumentationURI      string               `json:"documentationUri"`
	Patch                 SCIMSupported        `json:"patch"`
	Bulk                  SCIMBulkSupported    `json:"bulk"`
	Filter                SCIMFilterSupported  `json:"filter"`
	ChangePassword        SCIMSupported        `json:"changePassword"`
	Sort                  SCIMSupported        `json:"sort"`
	Etag                  SCIMSupported        `json:"etag"`
	AuthenticationSchemes []SCIMAuthScheme     `json:"authenticationSchemes"`
}

type SCIMSupported struct {
	Supported bool `json:"supported"`
}

type SCIMBulkSupported struct {
	Supported      bool `json:"supported"`
	MaxOperations  int  `json:"maxOperations"`
	MaxPayloadSize int  `json:"maxPayloadSize"`
}

type SCIMFilterSupported struct {
	Supported  bool `json:"supported"`
	MaxResults int  `json:"maxResults"`
}

type SCIMAuthScheme struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type SCIMSchema struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Attributes  json.RawMessage `json:"attributes"`
}

type SCIMError struct {
	Schemas    []string `json:"schemas"`
	Detail     string   `json:"detail"`
	Status     int      `json:"status"`
	ScimType   string   `json:"scimType,omitempty"`
}

func newSCIMError(status int, detail, scimType string) SCIMError {
	return SCIMError{
		Schemas:  []string{"urn:ietf:params:scim:api:messages:2.0:Error"},
		Detail:   detail,
		Status:   status,
		ScimType: scimType,
	}
}

// ---- DTO conversion helpers ----

func toSCIMConfigurationResponseDTO(cfg *SCIMConfiguration) SCIMConfigurationResponseDTO {
	dto := SCIMConfigurationResponseDTO{
		SCIMConfigurationUUID: cfg.SCIMConfigurationUUID.String(),
		TenantID:              cfg.TenantID,
		IdentityProviderID:    cfg.IdentityProviderID,
		DisplayName:           cfg.DisplayName,
		BaseURL:               cfg.BaseURL,
		SyncUsers:             cfg.SyncUsers,
		SyncGroups:            cfg.SyncGroups,
		SyncDirection:         cfg.SyncDirection,
		AttributeMapping:      []byte(cfg.AttributeMapping),
		IsActive:              cfg.IsActive,
		CreatedAt:             cfg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:             cfg.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if cfg.LastSyncAt != nil {
		s := cfg.LastSyncAt.Format("2006-01-02T15:04:05Z07:00")
		dto.LastSyncAt = &s
	}
	dto.LastSyncStatus = cfg.LastSyncStatus
	dto.LastSyncError = cfg.LastSyncError
	return dto
}
