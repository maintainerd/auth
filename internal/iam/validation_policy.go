package iam

import (
	"encoding/json"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/datatypes"
	"regexp"
)

// policyNamePattern matches valid policy name characters (compiled once for performance).
var policyNamePattern = regexp.MustCompile(`^[a-z0-9_:/\\-]+$`)

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

func (r PolicyStatusUpdateDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be one of: active, inactive"),
		),
	)
}

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
