package tenant

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Tenant output structure
type TenantResponseDTO struct {
	TenantUUID  uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Identifier  string    `json:"identifier"`
	Status      string    `json:"status"`
	IsPublic    bool      `json:"is_public"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Create Tenant request DTO
type TenantCreateRequestDTO struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	IsPublic    bool   `json:"is_public"`
}

// Update Tenant request DTO
type TenantUpdateRequestDTO struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	IsPublic    bool   `json:"is_public"`
}

// TenantSetStatusRequestDTO is the request body for changing tenant status.
type TenantSetStatusRequestDTO struct {
	Status string `json:"status"`
}

// API listing / filter DTO
type TenantFilterDTO struct {
	Name        *string  `json:"name"`
	DisplayName *string  `json:"display_name"`
	Description *string  `json:"description"`
	Identifier  *string  `json:"identifier"`
	Status      []string `json:"status"`
	IsPublic    *bool    `json:"is_public"`
	IsSystem    *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// TenantMemberResponseDTO for tenant member output
type TenantMemberResponseDTO struct {
	TenantMemberUUID uuid.UUID              `json:"tenant_member_id"`
	Role             string                 `json:"role"`
	User             *MemberUserResponseDTO `json:"user"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// MemberUserResponseDTO is tenant's view of a user in member responses.
// It mirrors the user domain's user response shape to preserve the API.
type MemberUserResponseDTO struct {
	UserUUID           uuid.UUID      `json:"user_id"`
	Username           string         `json:"username"`
	Fullname           string         `json:"fullname"`
	Email              string         `json:"email"`
	Phone              string         `json:"phone"`
	IsEmailVerified    bool           `json:"is_email_verified"`
	IsPhoneVerified    bool           `json:"is_phone_verified"`
	IsProfileCompleted bool           `json:"is_profile_completed"`
	IsAccountCompleted bool           `json:"is_account_completed"`
	Status             string         `json:"status"`
	Metadata           datatypes.JSON `json:"metadata"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// TenantMemberAddMemberRequestDTO for adding member to tenant
type TenantMemberAddMemberRequestDTO struct {
	UserUUID uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
}

// TenantMemberUpdateRoleRequestDTO for updating member role
type TenantMemberUpdateRoleRequestDTO struct {
	Role string `json:"role"`
}

// TenantMemberFilterDTO for filtering tenant members
type TenantMemberFilterDTO struct {
	Role *string `json:"role"`
	PaginationRequestDTO
}

// TenantSettingConfigResponseDTO returns a single JSONB config as a map.
type TenantSettingConfigResponseDTO map[string]any

// TenantSettingUpdateConfigRequestDTO is the request body for updating a
// tenant setting config section.
type TenantSettingUpdateConfigRequestDTO map[string]any
