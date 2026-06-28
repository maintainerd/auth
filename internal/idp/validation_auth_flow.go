package idp

import (
	"encoding/json"
	"fmt"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/auth/internal/shared"
)

func validateRequiredFieldsJSON(raw *string) error {
	if raw == nil {
		return nil
	}
	var fields []string
	if err := json.Unmarshal([]byte(*raw), &fields); err != nil {
		return fmt.Errorf("required_fields must be a JSON string array")
	}
	allowed := map[string]bool{"username": true, "password": true, "fullname": true, "email": true, "phone": true}
	seen := make(map[string]bool, len(fields))
	for _, field := range fields {
		field = strings.ToLower(strings.TrimSpace(field))
		if !allowed[field] {
			return fmt.Errorf("unsupported required field: %s", field)
		}
		if seen[field] {
			return fmt.Errorf("duplicate required field: %s", field)
		}
		seen[field] = true
	}
	return nil
}

func (r AuthFlowCreateRequestDTO) Validate() error {
	if err := validation.ValidateStruct(&r,
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
	); err != nil {
		return err
	}
	return validateRequiredFieldsJSON(r.RequiredFields)
}

func (r AuthFlowUpdateRequestDTO) Validate() error {
	if err := validation.ValidateStruct(&r,
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
	); err != nil {
		return err
	}
	return validateRequiredFieldsJSON(r.RequiredFields)
}

func (r AuthFlowUpdateStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

func (f AuthFlowFilterDTO) Validate() error {
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

func (r AuthFlowAssignRolesRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.RoleUUIDs,
			validation.Required.Error("Role UUIDs are required"),
			validation.Length(1, 0).Error("At least one role UUID is required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}

func (r AuthFlowAssignCallbackURIsRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.ClientURIUUIDs,
			validation.Required.Error("Client URI UUIDs are required"),
			validation.Length(1, 0).Error("At least one client URI UUID is required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}
