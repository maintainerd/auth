package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/maintainerd/auth/internal/model"
)

// User output structure
type UserResponseDTO struct {
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
	Tenant             *TenantResponseDTO `json:"tenant,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

type UserIdentityResponseDTO struct {
	UserIdentityUUID uuid.UUID          `json:"user_identity_id"`
	Provider         string             `json:"provider"`
	Sub              string             `json:"sub"`
	Metadata         datatypes.JSON     `json:"metadata"`
	Client           *ClientResponseDTO `json:"client,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// User input structures
type UserCreateRequestDTO struct {
	Username   string         `json:"username"`
	Fullname   string         `json:"fullname"`
	Email      *string        `json:"email,omitempty"`
	Phone      *string        `json:"phone,omitempty"`
	Password   string         `json:"password"`
	Status     string         `json:"status"`
	Metadata   datatypes.JSON `json:"metadata,omitempty"`
	TenantUUID string         `json:"tenant_id"`
}

func (dto UserCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Username, validation.Required, validation.Length(3, 50)),
		validation.Field(&dto.Fullname, validation.Required, validation.Length(1, 255)),
		validation.Field(&dto.Email, validation.When(dto.Email != nil, is.Email)),
		validation.Field(&dto.Phone, validation.When(dto.Phone != nil, validation.Length(10, 20))),
		validation.Field(&dto.Password, validation.Required, validation.Length(8, 100)),
		validation.Field(&dto.Status, validation.Required, validation.In(model.StatusActive, model.StatusInactive, model.StatusPending, model.StatusSuspended)),
		validation.Field(&dto.TenantUUID, validation.Required, is.UUID),
	)
}

type UserUpdateRequestDTO struct {
	Username string         `json:"username"`
	Fullname string         `json:"fullname"`
	Email    *string        `json:"email,omitempty"`
	Phone    *string        `json:"phone,omitempty"`
	Status   string         `json:"status"`
	Metadata datatypes.JSON `json:"metadata,omitempty"`
}

func (dto UserUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Username, validation.Required, validation.Length(3, 50)),
		validation.Field(&dto.Fullname, validation.Required, validation.Length(1, 255)),
		validation.Field(&dto.Email, validation.When(dto.Email != nil, is.Email)),
		validation.Field(&dto.Phone, validation.When(dto.Phone != nil, validation.Length(10, 20))),
		validation.Field(&dto.Status, validation.Required, validation.In(model.StatusActive, model.StatusInactive, model.StatusPending, model.StatusSuspended)),
	)
}

type UserSetStatusRequestDTO struct {
	Status string `json:"status"`
}

func (dto UserSetStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Status, validation.Required, validation.In(model.StatusActive, model.StatusInactive, model.StatusPending, model.StatusSuspended)),
	)
}

type UserAssignRolesRequestDTO struct {
	RoleUUIDs []uuid.UUID `json:"role_ids"`
}

func (dto UserAssignRolesRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.RoleUUIDs, validation.Required, validation.Length(1, 10)),
	)
}

// User filter structure
type UserFilterDTO struct {
	Username     *string  `json:"username,omitempty"`
	Email        *string  `json:"email,omitempty"`
	Phone        *string  `json:"phone,omitempty"`
	Status       []string `json:"status,omitempty"`
	TenantUUID   *string  `json:"tenant_id,omitempty"`
	RoleUUID     *string  `json:"role_id,omitempty"`
	UserPoolUUID *string  `json:"user_pool_id,omitempty"`
	ClientUUID   *string  `json:"client_id,omitempty"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validate validates the user filter DTO.
func (f UserFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.TenantUUID,
			validation.When(f.TenantUUID != nil,
				is.UUID.Error("Tenant ID must be a valid UUID"),
			),
		),
		validation.Field(&f.RoleUUID,
			validation.When(f.RoleUUID != nil,
				is.UUID.Error("Role ID must be a valid UUID"),
			),
		),
		validation.Field(&f.UserPoolUUID,
			validation.When(f.UserPoolUUID != nil,
				is.UUID.Error("User pool ID must be a valid UUID"),
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

// User role filter structure
type UserRoleFilterDTO struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`

	// Pagination and sorting
	PaginationRequestDTO
}

func (r UserRoleFilterDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PaginationRequestDTO),
	)
}

// User identity filter structure
type UserIdentityFilterDTO struct {
	Provider *string `json:"provider,omitempty"`

	// Pagination and sorting
	PaginationRequestDTO
}

func (r UserIdentityFilterDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PaginationRequestDTO),
	)
}
