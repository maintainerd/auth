package idp

import (
	"fmt"
	"regexp"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// registrationFlowNamePattern constrains a flow name to a URL-safe slug, because
// the name IS the public selector in a registration link
// (?registration_flow=<name>). Lowercase alphanumerics separated by single
// hyphens or underscores, starting with an alphanumeric.
var registrationFlowNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9]*([-_][a-z0-9]+)*$`)

const registrationFlowNameMessage = "Name must be lowercase letters, numbers, hyphens and underscores (e.g. partner-signup)"

// maxRegistrationFlowRoles caps a single role-membership payload. Every entry
// costs a role lookup plus a permission lookup inside the write transaction, so
// an unbounded list is a cheap way to hold one open. No real flow needs more.
const maxRegistrationFlowRoles = 50

// maxRegistrationFlowFilterTerm caps free-text filter values. They reach the
// database as LOWER(col) LIKE '%term%', so an unbounded term is wasted work on a
// query that cannot use an index anyway.
const maxRegistrationFlowFilterTerm = 100

func validateRequiredFieldsJSON(raw *[]string) error {
	if raw == nil || len(*raw) == 0 {
		return nil
	}
	allowed := map[string]bool{"fullname": true, "email": true, "phone": true}
	seen := make(map[string]bool, len(*raw))
	for _, field := range *raw {
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

func (r RegistrationFlowCreateRequestDTO) Validate() error {
	if err := validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Registration flow name is required"),
			validation.Length(1, 100).Error("Registration flow name must be between 1 and 100 characters"),
			validation.Match(registrationFlowNamePattern).Error(registrationFlowNameMessage),
		),
		validation.Field(&r.Description,
			validation.Length(0, 500).Error("Description must not exceed 500 characters"),
		),
		validation.Field(&r.Status,
			validation.When(r.Status != nil,
				validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
			),
		),
		validation.Field(&r.ClientUUID,
			validation.Required.Error("Auth client UUID is required"),
			is.UUID.Error("Invalid auth client UUID format"),
		),
		validation.Field(&r.RoleIDs,
			validation.Length(0, maxRegistrationFlowRoles).Error("At most 50 roles can be assigned to a registration flow"),
			validation.Each(is.UUID.Error("Invalid role UUID provided")),
		),
	); err != nil {
		return err
	}
	return validateRequiredFieldsJSON(r.RequiredFields)
}

func (r RegistrationFlowUpdateRequestDTO) Validate() error {
	if err := validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.When(r.Name != nil,
				validation.Required.Error("Registration flow name is required"),
				validation.Length(1, 100).Error("Registration flow name must be between 1 and 100 characters"),
				validation.Match(registrationFlowNamePattern).Error(registrationFlowNameMessage),
			),
		),
		validation.Field(&r.Description,
			validation.When(r.Description != nil,
				validation.Length(0, 500).Error("Description must not exceed 500 characters"),
			),
		),
		validation.Field(&r.Status,
			validation.When(r.Status != nil,
				validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
			),
		),
		validation.Field(&r.RoleIDs,
			validation.Length(0, maxRegistrationFlowRoles).Error("At most 50 roles can be assigned to a registration flow"),
			validation.Each(is.UUID.Error("Invalid role UUID provided")),
		),
	); err != nil {
		return err
	}
	return validateRequiredFieldsJSON(r.RequiredFields)
}

func (r RegistrationFlowUpdateStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Validate validates the registration flow filter DTO.
func (f RegistrationFlowFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Name,
			validation.When(f.Name != nil,
				validation.Length(0, maxRegistrationFlowFilterTerm).Error("Name filter must not exceed 100 characters"),
			),
		),
		validation.Field(&f.Search,
			validation.When(f.Search != nil,
				validation.Length(0, maxRegistrationFlowFilterTerm).Error("Search term must not exceed 100 characters"),
			),
		),
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

func (r RegistrationFlowAssignRolesRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.RoleUUIDs,
			validation.Required.Error("Role UUIDs are required"),
			validation.Length(1, maxRegistrationFlowRoles).Error("Between 1 and 50 role UUIDs are required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}
