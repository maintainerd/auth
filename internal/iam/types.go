package iam

import (
	"encoding/json"
	"regexp"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/datatypes"
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

// Validation
func (r APICreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("API name is required"),
			validation.Length(3, 50).Error("API name must be between 3 and 50 characters"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(3, 50).Error("Display name must be between 3 and 50 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.APIType,
			validation.Required.Error("API type is required"),
			validation.In(shared.APITypeRest, shared.APITypeGRPC, shared.APITypeGraphQL, shared.APITypeSOAP, shared.APITypeWebhook, shared.APITypeWebSocket, shared.APITypeRPC).Error("API type must be one of: rest, grpc, graphql, soap, webhook, websocket, rpc"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be one of: active, inactive"),
		),

		validation.Field(&r.ServiceUUID,
			validation.Required.Error("Service ID is required"),
			is.UUID.Error("Service ID must be a valid UUID"),
		),
	)
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

// Validation
func (r APIUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("API name is required"),
			validation.Length(3, 50).Error("API name must be between 3 and 50 characters"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(3, 50).Error("Display name must be between 3 and 50 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.APIType,
			validation.Required.Error("API type is required"),
			validation.In(shared.APITypeRest, shared.APITypeGRPC, shared.APITypeGraphQL, shared.APITypeSOAP, shared.APITypeWebhook, shared.APITypeWebSocket, shared.APITypeRPC).Error("API type must be one of: rest, grpc, graphql, soap, webhook, websocket, rpc"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be one of: active, inactive"),
		),
		validation.Field(&r.ServiceUUID,
			validation.Required.Error("Service ID is required"),
			is.UUID.Error("Service ID must be a valid UUID"),
		),
	)
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

// Validate validates the API filter DTO.
func (f APIFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.APIType,
			validation.When(f.APIType != nil,
				validation.In(shared.APITypeRest, shared.APITypeGRPC, shared.APITypeGraphQL, shared.APITypeSOAP, shared.APITypeWebhook, shared.APITypeWebSocket, shared.APITypeRPC).Error("API type must be one of: rest, grpc, graphql, soap, webhook, websocket, rpc"),
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

// API status update DTO
type APIStatusUpdateDTO struct {
	Status string `json:"status"`
}

func (r APIStatusUpdateDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be one of: active, inactive"),
		),
	)
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

// Validation
func (r PermissionCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be one of: active, inactive"),
		),
		validation.Field(&r.APIUUID,
			validation.Required.Error("API ID is required"),
			is.UUID.Error("API ID must be a valid UUID"),
		),
	)
}

// Update Permission request DTO
type PermissionUpdateRequestDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// Validation
func (r PermissionUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be one of: active, inactive"),
		),
	)
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

// Validate validates the permission filter DTO.
func (f PermissionFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(f.Status != nil,
				validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}

// Permission status update DTO
type PermissionStatusUpdateDTO struct {
	Status string `json:"status"`
}

func (r PermissionStatusUpdateDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be one of: active, inactive"),
		),
	)
}

// policyNamePattern matches valid policy name characters (compiled once for performance).
var policyNamePattern = regexp.MustCompile(`^[a-z0-9_:/\\-]+$`)

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

// Validate validates the PolicyDocument structure using ozzo-validation
func (p PolicyDocument) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Version,
			validation.Required.Error("Document must contain a 'version' field"),
		),
		validation.Field(&p.Statement,
			validation.Required.Error("Document must contain at least one 'statement'"),
			validation.Length(1, 0).Error("Document must contain at least one 'statement'"),
		),
	)
}

// PolicyStatement represents a single statement in a policy
type PolicyStatement struct {
	Effect   string   `json:"effect"`   // "allow" or "deny"
	Action   []string `json:"action"`   // e.g., ["user:*", "role:create"]
	Resource []string `json:"resource"` // e.g., ["auth:*", "account:profile"]
}

// Validate validates the PolicyStatement structure using ozzo-validation
func (s PolicyStatement) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.Effect,
			validation.Required.Error("Statement effect is required"),
			validation.In(shared.PolicyEffectAllow, shared.PolicyEffectDeny).Error("Statement effect must be 'allow' or 'deny'"),
		),
		validation.Field(&s.Action,
			validation.Required.Error("Statement must contain at least one action"),
			validation.Length(1, 0).Error("Statement must contain at least one action"),
		),
		validation.Field(&s.Resource,
			validation.Required.Error("Statement must contain at least one resource"),
			validation.Length(1, 0).Error("Statement must contain at least one resource"),
		),
	)
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

// Validation
func (r PolicyCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Policy name is required"),
			validation.Length(3, 150).Error("Policy name must be between 3 and 150 characters"),
			validation.Match(policyNamePattern).Error("Policy name must contain only lowercase letters, numbers, underscores, colons, forward slashes, backslashes, and hyphens"),
		),
		validation.Field(&r.Description,
			validation.Length(0, 500).Error("Description must be at most 500 characters"),
		),
		validation.Field(&r.Document,
			validation.Required.Error("Policy document is required"),
			validation.By(validatePolicyDocumentStructure),
		),
		validation.Field(&r.Version,
			validation.Required.Error("Version is required"),
			validation.Length(1, 20).Error("Version must be between 1 and 20 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be one of: active, inactive"),
		),
	)
}

// Update Policy request DTO
type PolicyUpdateRequestDTO struct {
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	Document    datatypes.JSON `json:"document"`
	Version     string         `json:"version"`
	Status      string         `json:"status"`
}

// Validation
func (r PolicyUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Policy name is required"),
			validation.Length(3, 150).Error("Policy name must be between 3 and 150 characters"),
			validation.Match(policyNamePattern).Error("Policy name must contain only lowercase letters, numbers, underscores, colons, forward slashes, backslashes, and hyphens"),
		),
		validation.Field(&r.Description,
			validation.Length(0, 500).Error("Description must be at most 500 characters"),
		),
		validation.Field(&r.Document,
			validation.Required.Error("Policy document is required"),
			validation.By(validatePolicyDocumentStructure),
		),
		validation.Field(&r.Version,
			validation.Required.Error("Version is required"),
			validation.Length(1, 20).Error("Version must be between 1 and 20 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be one of: active, inactive"),
		),
	)
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

// Validation
func (r PolicyFilterDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.When(r.Name != nil,
				validation.Length(1, 150).Error("Name filter must be between 1 and 150 characters"),
			),
		),
		validation.Field(&r.Description,
			validation.When(r.Description != nil,
				validation.Length(1, 500).Error("Description filter must be between 1 and 500 characters"),
			),
		),
		validation.Field(&r.Version,
			validation.When(r.Version != nil,
				validation.Length(1, 20).Error("Version filter must be between 1 and 20 characters"),
			),
		),
		validation.Field(&r.Status,
			validation.When(len(r.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
				validation.Length(1, 2).Error("Status filter can have at most 2 values"),
			),
		),
		validation.Field(&r.ServiceID,
			validation.When(r.ServiceID != nil,
				is.UUID.Error("Service ID must be a valid UUID"),
			),
		),
		validation.Field(&r.PaginationRequestDTO),
	)
}

// Policy status update DTO
type PolicyStatusUpdateDTO struct {
	Status string `json:"status"`
}

func (r PolicyStatusUpdateDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be one of: active, inactive"),
		),
	)
}

// Policy services filter DTO - for getting services that use a specific policy
type PolicyServicesFilterDTO struct {
	Name        *string `json:"name"`
	DisplayName *string `json:"display_name"`
	Description *string `json:"description"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validation
func (r PolicyServicesFilterDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.When(r.Name != nil,
				validation.Length(0, 150).Error("Name filter must be at most 150 characters"),
			),
		),
		validation.Field(&r.DisplayName,
			validation.When(r.DisplayName != nil,
				validation.Length(0, 150).Error("Display name filter must be at most 150 characters"),
			),
		),
		validation.Field(&r.Description,
			validation.When(r.Description != nil,
				validation.Length(0, 500).Error("Description filter must be at most 500 characters"),
			),
		),
		validation.Field(&r.PaginationRequestDTO),
	)
}

// validatePolicyDocumentStructure validates the JSON structure of a policy document
func validatePolicyDocumentStructure(value any) error {
	document, ok := value.(datatypes.JSON)
	if !ok {
		return validation.NewError("validation_error", "Document must be valid JSON")
	}

	// Parse the JSON into PolicyDocument struct
	var policyDoc PolicyDocument
	if err := json.Unmarshal(document, &policyDoc); err != nil {
		return validation.NewError("validation_error", "Document must be valid JSON: "+err.Error())
	}

	// Validate the document structure (ozzo-validation auto-validates each PolicyStatement element)
	if err := policyDoc.Validate(); err != nil {
		return err
	}

	return nil
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

func (r RoleCreateOrUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Role name is required"),
			validation.Length(3, 20).Error("Role name must be between 3 and 20 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 100).Error("Description must be between 8 and 100 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Add permissions to role request dto
type RoleAddPermissionsRequestDTO struct {
	Permissions []uuid.UUID `json:"permissions"`
}

func (r RoleAddPermissionsRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Permissions,
			validation.Required.Error("Permission UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
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

// Validate validates the role filter DTO.
func (f RoleFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(f.Status != nil,
				validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
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

func (r ServiceCreateOrUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Service name is required"),
			validation.Length(3, 50).Error("Service name must be between 3 and 50 characters"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(3, 100).Error("Display name must be between 3 and 100 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 255).Error("Description must be between 8 and 255 characters"),
		),
		validation.Field(&r.Version,
			validation.Required.Error("Version is required"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusMaintenance, shared.StatusDeprecated, shared.StatusInactive).Error("Status must be one of: active, maintenance, deprecated, inactive"),
		),
	)
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

// Validate validates the service filter DTO.
func (f ServiceFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusMaintenance, shared.StatusDeprecated, shared.StatusInactive).Error("Status must be one of: active, maintenance, deprecated, inactive")),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}

// Service status update request dto
type ServiceStatusUpdateRequestDTO struct {
	Status string `json:"status"`
}

func (r ServiceStatusUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusMaintenance, shared.StatusDeprecated, shared.StatusInactive).Error("Status must be one of: active, maintenance, deprecated, inactive"),
		),
	)
}
