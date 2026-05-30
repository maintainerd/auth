package iam

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"time"
)

// API output structure
type APIResponseDTO struct {
	APIUUID     uuid.UUID           `json:"api_id"`
	Name        string              `json:"name"`
	DisplayName string              `json:"display_name"`
	Description string              `json:"description"`
	APIType     string              `json:"api_type"`
	Identifier  string              `json:"identifier"`
	Service     *ServiceResponseDTO `json:"service,omitempty"`
	Status      string              `json:"status"`
	IsSystem    bool                `json:"is_system"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// Create API request DTO
type APICreateRequestDTO struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	APIType     string `json:"api_type"`
	Status      string `json:"status"`
	ServiceUUID string `json:"service_id"`
}

// Update API request DTO
type APIUpdateRequestDTO struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	APIType     string `json:"api_type"`
	Status      string `json:"status"`
	ServiceUUID string `json:"service_id"`
}

// API listing / filter DTO
type APIFilterDTO struct {
	Name        *string  `json:"name"`
	DisplayName *string  `json:"display_name"`
	Description *string  `json:"description"`
	APIType     *string  `json:"api_type"`
	Identifier  *string  `json:"identifier"`
	ServiceUUID *string  `json:"service_id"`
	Status      []string `json:"status"`
	IsSystem    *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// API status update DTO
type APIStatusUpdateDTO struct {
	Status string `json:"status"`
}

// Permission output structure
type PermissionResponseDTO struct {
	PermissionUUID uuid.UUID       `json:"permission_id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	API            *APIResponseDTO `json:"api,omitempty"`
	Status         string          `json:"status"`
	IsDefault      bool            `json:"is_default"`
	IsSystem       bool            `json:"is_system"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// Create Permission request DTO
type PermissionCreateRequestDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	APIUUID     string `json:"api_id"`
}

// Update Permission request DTO
type PermissionUpdateRequestDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// API listing / filter DTO
type PermissionFilterDTO struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	APIUUID     *string `json:"api_id"`
	RoleUUID    *string `json:"role_id"`
	ClientUUID  *string `json:"client_id"`
	Status      *string `json:"status"`
	IsDefault   *bool   `json:"is_default"`
	IsSystem    *bool   `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Permission status update DTO
type PermissionStatusUpdateDTO struct {
	Status string `json:"status"`
}

// Policy document structure types for reference
// These are not enforced but show the expected format
//
// Example policy document:
// {
//   "version": "v1",
//   "statement": [
//     {
//       "effect": "allow",
//       "action": ["user:*", "role:create"],
//       "resource": ["auth:*", "account:profile"]
//     }
//   ]
// }
//
// Action format:
// - All permissions for a resource: "user:*"
// - Specific permission: "user:create"
//
// Resource format:
// - Service and all APIs: "auth:*"
// - Service and specific API: "auth:login"
//
// Note: Action and resource values are not validated against existing
// permissions/services. Invalid values will simply result in no access.

// PolicyDocument represents the structure of a policy document
type PolicyDocument struct {
	Version   string            `json:"version"` // e.g., "v1"
	Statement []PolicyStatement `json:"statement"`
}

// PolicyStatement represents a single statement in a policy
type PolicyStatement struct {
	Effect   string   `json:"effect"`   // "allow" or "deny"
	Action   []string `json:"action"`   // e.g., ["user:*", "role:create"]
	Resource []string `json:"resource"` // e.g., ["auth:*", "account:profile"]
}

// Policy output structure for listing (without document)
type PolicyResponseDTO struct {
	PolicyUUID  uuid.UUID `json:"policy_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Version     string    `json:"version"`
	Status      string    `json:"status"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Policy output structure for individual retrieval (with document)
type PolicyDetailResponseDTO struct {
	PolicyUUID  uuid.UUID      `json:"policy_id"`
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	Document    datatypes.JSON `json:"document"`
	Version     string         `json:"version"`
	Status      string         `json:"status"`
	IsSystem    bool           `json:"is_system"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// Create Policy request DTO
type PolicyCreateRequestDTO struct {
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	Document    datatypes.JSON `json:"document"`
	Version     string         `json:"version"`
	Status      string         `json:"status"`
}

// Update Policy request DTO
type PolicyUpdateRequestDTO struct {
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	Document    datatypes.JSON `json:"document"`
	Version     string         `json:"version"`
	Status      string         `json:"status"`
}

// Policy listing / filter DTO
type PolicyFilterDTO struct {
	Name        *string    `json:"name"`
	Description *string    `json:"description"`
	Version     *string    `json:"version"`
	Status      []string   `json:"status"`
	IsSystem    *bool      `json:"is_system"`
	ServiceID   *uuid.UUID `json:"service_id"` // Filter policies by service UUID

	// Pagination and sorting
	PaginationRequestDTO
}

// Policy status update DTO
type PolicyStatusUpdateDTO struct {
	Status string `json:"status"`
}

// Policy services filter DTO - for getting services that use a specific policy
type PolicyServicesFilterDTO struct {
	Name        *string `json:"name"`
	DisplayName *string `json:"display_name"`
	Description *string `json:"description"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Role output structure
type RoleResponseDTO struct {
	RoleUUID    uuid.UUID                `json:"role_id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Permissions *[]PermissionResponseDTO `json:"permissions,omitempty"`
	IsDefault   bool                     `json:"is_default"`
	IsSystem    bool                     `json:"is_system"`
	Status      string                   `json:"status"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
}

// Create or update role request dto
type RoleCreateOrUpdateRequestDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// Add permissions to role request dto
type RoleAddPermissionsRequestDTO struct {
	Permissions []uuid.UUID `json:"permissions"`
}

// Role listing
type RoleFilterDTO struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsDefault   *bool   `json:"is_default"`
	IsSystem    *bool   `json:"is_system"`
	Status      *string `json:"status"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Service output structure
type ServiceResponseDTO struct {
	ServiceUUID uuid.UUID `json:"service_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Status      string    `json:"status"`
	IsSystem    bool      `json:"is_system"`
	APICount    int64     `json:"api_count"`
	PolicyCount int64     `json:"policy_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Create or update service request dto
type ServiceCreateOrUpdateRequestDTO struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Status      string `json:"status"`
}

// Service listing filters
type ServiceFilterDTO struct {
	Name        *string  `json:"name"`
	DisplayName *string  `json:"display_name"`
	Description *string  `json:"description"`
	Version     *string  `json:"version"`
	Status      []string `json:"status"`
	IsSystem    *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Service status update request dto
type ServiceStatusUpdateRequestDTO struct {
	Status string `json:"status"`
}
