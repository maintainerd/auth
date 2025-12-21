package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// Signup flow role output structure
type SignupFlowRoleResponseDto struct {
	SignupFlowRoleUUID string    `json:"signup_flow_role_id"`
	SignupFlowUUID     string    `json:"signup_flow_id"`
	RoleUUID           string    `json:"role_id"`
	RoleName           string    `json:"role_name,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

// Assign roles to signup flow request dto
type SignupFlowAssignRolesRequestDto struct {
	RoleUUIDs []string `json:"role_uuids"`
}

func (r SignupFlowAssignRolesRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.RoleUUIDs,
			validation.Required.Error("Role UUIDs are required"),
			validation.Length(1, 0).Error("At least one role UUID is required"),
			validation.Each(is.UUID.Error("Invalid UUID provided")),
		),
	)
}
