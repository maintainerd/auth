package dto

import (
	"encoding/json"
	"regexp"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

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
			validation.In("allow", "deny").Error("Statement effect must be 'allow' or 'deny'"),
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
type PolicyResponseDto struct {
	PolicyUUID  uuid.UUID `json:"policy_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Version     string    `json:"version"`
	Status      string    `json:"status"`
	IsDefault   bool      `json:"is_default"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Policy output structure for individual retrieval (with document)
type PolicyDetailResponseDto struct {
	PolicyUUID  uuid.UUID      `json:"policy_id"`
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	Document    datatypes.JSON `json:"document"`
	Version     string         `json:"version"`
	Status      string         `json:"status"`
	IsDefault   bool           `json:"is_default"`
	IsSystem    bool           `json:"is_system"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// Create Policy request DTO
type PolicyCreateRequestDto struct {
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	Document    datatypes.JSON `json:"document"`
	Version     string         `json:"version"`
	Status      string         `json:"status"`
}

// Validation
func (r PolicyCreateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Policy name is required"),
			validation.Length(3, 150).Error("Policy name must be between 3 and 150 characters"),
			validation.Match(regexp.MustCompile(`^[a-z0-9_:/\\-]+$`)).Error("Policy name must contain only lowercase letters, numbers, underscores, colons, forward slashes, backslashes, and hyphens"),
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
			validation.In("active", "inactive").Error("Status must be one of: active, inactive"),
		),
	)
}

// Update Policy request DTO
type PolicyUpdateRequestDto struct {
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	Document    datatypes.JSON `json:"document"`
	Version     string         `json:"version"`
	Status      string         `json:"status"`
}

// Validation
func (r PolicyUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Policy name is required"),
			validation.Length(3, 150).Error("Policy name must be between 3 and 150 characters"),
			validation.Match(regexp.MustCompile(`^[a-z0-9_:/\\-]+$`)).Error("Policy name must contain only lowercase letters, numbers, underscores, colons, forward slashes, backslashes, and hyphens"),
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
			validation.In("active", "inactive").Error("Status must be one of: active, inactive"),
		),
	)
}

// Policy listing / filter DTO
type PolicyFilterDto struct {
	Name        *string    `json:"name"`
	Description *string    `json:"description"`
	Version     *string    `json:"version"`
	Status      []string   `json:"status"`
	IsDefault   *bool      `json:"is_default"`
	IsSystem    *bool      `json:"is_system"`
	ServiceID   *uuid.UUID `json:"service_id"` // Filter policies by service UUID

	// Pagination and sorting
	PaginationRequestDto
}

// Validation
func (r PolicyFilterDto) Validate() error {
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
				validation.Each(validation.In("active", "inactive").Error("Status must be 'active' or 'inactive'")),
				validation.Length(1, 2).Error("Status filter can have at most 2 values"),
			),
		),
		validation.Field(&r.ServiceID,
			validation.When(r.ServiceID != nil,
				is.UUID.Error("Service ID must be a valid UUID"),
			),
		),
		validation.Field(&r.PaginationRequestDto),
	)
}

// Policy status update DTO
type PolicyStatusUpdateDto struct {
	Status string `json:"status"`
}

func (r PolicyStatusUpdateDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In("active", "inactive").Error("Status must be one of: active, inactive"),
		),
	)
}

// validatePolicyDocumentStructure validates the JSON structure of a policy document
func validatePolicyDocumentStructure(value interface{}) error {
	document, ok := value.(datatypes.JSON)
	if !ok {
		return validation.NewError("validation_error", "Document must be valid JSON")
	}

	// Parse the JSON into PolicyDocument struct
	var policyDoc PolicyDocument
	if err := json.Unmarshal(document, &policyDoc); err != nil {
		return validation.NewError("validation_error", "Document must be valid JSON: "+err.Error())
	}

	// Validate the document structure
	if err := policyDoc.Validate(); err != nil {
		return err
	}

	// Validate each statement
	for i, statement := range policyDoc.Statement {
		if err := statement.Validate(); err != nil {
			return validation.NewError("validation_error", "Statement "+string(rune(i+1))+": "+err.Error())
		}
	}

	return nil
}
