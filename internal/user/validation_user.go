package user

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/auth/internal/shared"
)

func (dto UserCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Username, validation.Required, validation.Length(3, 50)),
		validation.Field(&dto.Fullname, validation.Required, validation.Length(1, 255)),
		validation.Field(&dto.Email, validation.When(dto.Email != nil, is.Email)),
		validation.Field(&dto.Phone, validation.When(dto.Phone != nil, validation.Length(10, 20))),
		validation.Field(&dto.Password, validation.Required, validation.Length(8, 100)),
		validation.Field(&dto.Status, validation.Required, validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended)),
		validation.Field(&dto.TenantUUID, validation.Required, is.UUID),
	)
}

func (dto UserUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Username, validation.Required, validation.Length(3, 50)),
		validation.Field(&dto.Fullname, validation.Required, validation.Length(1, 255)),
		validation.Field(&dto.Email, validation.When(dto.Email != nil, is.Email)),
		validation.Field(&dto.Phone, validation.When(dto.Phone != nil, validation.Length(10, 20))),
		validation.Field(&dto.Status, validation.Required, validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended)),
	)
}

func (dto UserSetStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Status, validation.Required, validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended)),
	)
}

func (dto UserAssignRolesRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.RoleUUIDs, validation.Required, validation.Length(1, 10)),
	)
}

func (f UserFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.TenantUUID, validation.When(f.TenantUUID != nil, is.UUID.Error("Tenant ID must be a valid UUID"))),
		validation.Field(&f.RoleUUID, validation.When(f.RoleUUID != nil, is.UUID.Error("Role ID must be a valid UUID"))),
		validation.Field(&f.ClientUUID, validation.When(f.ClientUUID != nil, is.UUID.Error("Client ID must be a valid UUID"))),
		validation.Field(&f.PaginationRequestDTO),
	)
}

func (r UserRoleFilterDTO) Validate() error {
	return validation.ValidateStruct(&r, validation.Field(&r.PaginationRequestDTO))
}

func (r UserIdentityFilterDTO) Validate() error {
	return validation.ValidateStruct(&r, validation.Field(&r.PaginationRequestDTO))
}
