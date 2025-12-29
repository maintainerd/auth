package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

// TenantUserResponseDto for tenant user output
type TenantUserResponseDto struct {
	TenantUserUUID uuid.UUID `json:"tenant_user_id"`
	TenantID       int64     `json:"tenant_id"`
	UserID         int64     `json:"user_id"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TenantUserAddMemberRequestDto for adding member to tenant
type TenantUserAddMemberRequestDto struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}

func (r TenantUserAddMemberRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.UserID,
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
