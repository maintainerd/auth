package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

// TenantUserResponseDto for tenant user output
type TenantUserResponseDto struct {
	TenantUserUUID uuid.UUID        `json:"tenant_user_id"`
	Role           string           `json:"role"`
	User           *UserResponseDto `json:"user"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// TenantUserAddMemberRequestDto for adding member to tenant
type TenantUserAddMemberRequestDto struct {
	UserUUID uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
}

func (r TenantUserAddMemberRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.UserUUID,
			validation.Required.Error("User ID is required"),
		),
		validation.Field(&r.Role,
			validation.Required.Error("Role is required"),
			validation.In("owner", "member").Error("Role must be 'owner' or 'member'"),
		),
	)
}

// TenantUserUpdateRoleRequestDto for updating member role
type TenantUserUpdateRoleRequestDto struct {
	Role string `json:"role"`
}

func (r TenantUserUpdateRoleRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Role,
			validation.Required.Error("Role is required"),
			validation.In("owner", "member").Error("Role must be 'owner' or 'member'"),
		),
	)
}

// TenantUserFilterDto for filtering tenant members
type TenantUserFilterDto struct {
	Role *string `json:"role"`
	PaginationRequestDto
}

func (r TenantUserFilterDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Role,
			validation.When(r.Role != nil,
				validation.In("owner", "member").Error("Role must be 'owner' or 'member'"),
			),
		),
		validation.Field(&r.PaginationRequestDto),
	)
}
