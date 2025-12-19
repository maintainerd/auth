package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// User output structure
type UserResponseDto struct {
	UserUUID           uuid.UUID          `json:"user_id"`
	Username           string             `json:"username"`
	Fullname           string             `json:"fullname"`
	Email              string             `json:"email"`
	Phone              string             `json:"phone"`
	IsEmailVerified    bool               `json:"is_email_verified"`
	IsPhoneVerified    bool               `json:"is_phone_verified"`
	IsProfileCompleted bool               `json:"is_profile_completed"`
	IsAccountCompleted bool               `json:"is_account_completed"`
	Status             string             `json:"status"`
	Metadata           datatypes.JSON     `json:"metadata"`
	Tenant             *TenantResponseDto `json:"tenant,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

type UserIdentityResponseDto struct {
	UserIdentityUUID uuid.UUID              `json:"user_identity_id"`
	Provider         string                 `json:"provider"`
	Sub              string                 `json:"sub"`
	Metadata         datatypes.JSON         `json:"metadata"`
	AuthClient       *AuthClientResponseDto `json:"client,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// User input structures
type UserCreateRequestDto struct {
	Username   string         `json:"username"`
	Fullname   string         `json:"fullname"`
	Email      *string        `json:"email,omitempty"`
	Phone      *string        `json:"phone,omitempty"`
	Password   string         `json:"password"`
	Status     string         `json:"status"`
	Metadata   datatypes.JSON `json:"metadata,omitempty"`
	TenantUUID string         `json:"tenant_id"`
}

func (dto UserCreateRequestDto) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Username, validation.Required, validation.Length(3, 50)),
		validation.Field(&dto.Fullname, validation.Required, validation.Length(1, 255)),
		validation.Field(&dto.Email, validation.When(dto.Email != nil, is.Email)),
		validation.Field(&dto.Phone, validation.When(dto.Phone != nil, validation.Length(10, 20))),
		validation.Field(&dto.Password, validation.Required, validation.Length(8, 100)),
		validation.Field(&dto.Status, validation.Required, validation.In("active", "inactive", "pending", "suspended")),
		validation.Field(&dto.TenantUUID, validation.Required, is.UUID),
	)
}

type UserUpdateRequestDto struct {
	Username string         `json:"username"`
	Fullname string         `json:"fullname"`
	Email    *string        `json:"email,omitempty"`
	Phone    *string        `json:"phone,omitempty"`
	Status   string         `json:"status"`
	Metadata datatypes.JSON `json:"metadata,omitempty"`
}

func (dto UserUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Username, validation.Required, validation.Length(3, 50)),
		validation.Field(&dto.Fullname, validation.Required, validation.Length(1, 255)),
		validation.Field(&dto.Email, validation.When(dto.Email != nil, is.Email)),
		validation.Field(&dto.Phone, validation.When(dto.Phone != nil, validation.Length(10, 20))),
		validation.Field(&dto.Status, validation.Required, validation.In("active", "inactive", "pending", "suspended")),
	)
}

type UserSetStatusRequestDto struct {
	Status string `json:"status"`
}

func (dto UserSetStatusRequestDto) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Status, validation.Required, validation.In("active", "inactive", "pending", "suspended")),
	)
}

type UserAssignRolesRequestDto struct {
	RoleUUIDs []uuid.UUID `json:"role_ids"`
}

func (dto UserAssignRolesRequestDto) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.RoleUUIDs, validation.Required, validation.Length(1, 10)),
	)
}

// User filter structure
type UserFilterDto struct {
	Username   *string  `json:"username,omitempty"`
	Email      *string  `json:"email,omitempty"`
	Phone      *string  `json:"phone,omitempty"`
	Status     []string `json:"status,omitempty"`
	TenantUUID *string  `json:"tenant_id,omitempty"`
	RoleUUID   *string  `json:"role_id,omitempty"`

	// Pagination and sorting
	PaginationRequestDto
}

// User role filter structure
type UserRoleFilterDto struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`

	// Pagination and sorting
	PaginationRequestDto
}

func (r UserRoleFilterDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PaginationRequestDto),
	)
}

// User identity filter structure
type UserIdentityFilterDto struct {
	Provider *string `json:"provider,omitempty"`

	// Pagination and sorting
	PaginationRequestDto
}

func (r UserIdentityFilterDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PaginationRequestDto),
	)
}
