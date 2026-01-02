package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

// TenantMemberResponseDto for tenant member output
type TenantMemberResponseDto struct {
	TenantMemberUUID uuid.UUID        `json:"tenant_member_id"`
	Role             string           `json:"role"`
	User             *UserResponseDto `json:"user"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// TenantMemberAddMemberRequestDto for adding member to tenant
type TenantMemberAddMemberRequestDto struct {
	UserUUID uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
}

func (r TenantMemberAddMemberRequestDto) Validate() error {
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

// TenantMemberUpdateRoleRequestDto for updating member role
type TenantMemberUpdateRoleRequestDto struct {
	Role string `json:"role"`
}

func (r TenantMemberUpdateRoleRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Role,
			validation.Required.Error("Role is required"),
			validation.In("owner", "member").Error("Role must be 'owner' or 'member'"),
		),
	)
}

// TenantMemberFilterDto for filtering tenant members
type TenantMemberFilterDto struct {
	Role *string `json:"role"`
	PaginationRequestDto
}

func (r TenantMemberFilterDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Role,
			validation.When(r.Role != nil,
				validation.In("owner", "member").Error("Role must be 'owner' or 'member'"),
			),
		),
		validation.Field(&r.PaginationRequestDto),
	)
}
