package user

import (
	"errors"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/maintainerd-auth/internal/platform/valid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

func (dto UserCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Username, validation.Required, validation.Length(3, 50)),
		validation.Field(&dto.Email, validation.When(dto.Email != nil,
			validation.By(func(value interface{}) error {
				if email := value.(*string); email != nil && *email != "" {
					if !valid.IsValidEmail(*email) {
						return errors.New("email must be a valid email address")
					}
				}
				return nil
			}),
		)),
		validation.Field(&dto.Phone, validation.When(dto.Phone != nil,
			validation.By(func(value interface{}) error {
				if phone := value.(*string); phone != nil && *phone != "" {
					if !valid.IsValidPhoneNumber(*phone) {
						return errors.New("phone must be a valid phone number")
					}
				}
				return nil
			}),
		)),
		// Length/complexity are enforced authoritatively by the tenant's password
		// policy (security.ValidatePasswordPolicy) so its configured min/max is the
		// single source of truth; the DTO only requires a value and bounds abuse.
		validation.Field(&dto.Password, validation.Required, validation.Length(1, 4096)),
		validation.Field(&dto.Status, validation.Required, validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended)),
	)
}

func (dto UserUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Username, validation.Required, validation.Length(3, 50)),
		validation.Field(&dto.Email, validation.When(dto.Email != nil,
			validation.By(func(value interface{}) error {
				if email := value.(*string); email != nil && *email != "" {
					if !valid.IsValidEmail(*email) {
						return errors.New("email must be a valid email address")
					}
				}
				return nil
			}),
		)),
		validation.Field(&dto.Phone, validation.When(dto.Phone != nil,
			validation.By(func(value interface{}) error {
				if phone := value.(*string); phone != nil && *phone != "" {
					if !valid.IsValidPhoneNumber(*phone) {
						return errors.New("phone must be a valid phone number")
					}
				}
				return nil
			}),
		)),
		validation.Field(&dto.Status, validation.Required, validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended)),
	)
}

func (dto UserSetStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Status, validation.Required, validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended)),
	)
}

func (dto UserSetPasswordRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		// Length/complexity are enforced authoritatively by the tenant's password
		// policy in userService.SetPassword, so its configured min/max stays the
		// single source of truth; the DTO only requires a value and bounds abuse —
		// an unbounded body would be handed straight to argon2id.
		validation.Field(&dto.Password, validation.Required, validation.Length(1, 4096)),
	)
}

func (dto UserLinkIdentityRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.IdentityProviderUUID, validation.Required, is.UUID.Error("Identity provider ID must be a valid UUID")),
		// The upper bound matches user_identities.sub VARCHAR(255) in migration 022;
		// a longer value would otherwise be rejected by the database as a 500
		// rather than reported as a 400.
		validation.Field(&dto.Sub, validation.Required, validation.Length(1, 255)),
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
				validation.Each(validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended).Error("Status must be 'active', 'inactive', 'pending', or 'suspended'")),
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
