package tenant

import (
	"errors"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

func (r TenantMemberAddMemberRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.UserUUID,
			validation.By(requiredUUID("User ID is required")),
		),
		validation.Field(&r.Role,
			validation.Required.Error("Role is required"),
			validation.In("owner", "member").Error("Role must be 'owner' or 'member'"),
		),
	)
}

func requiredUUID(message string) validation.RuleFunc {
	return func(value any) error {
		id, ok := value.(uuid.UUID)
		if !ok || id == uuid.Nil {
			return errors.New(message)
		}
		return nil
	}
}

func (r TenantMemberUpdateRoleRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Role,
			validation.Required.Error("Role is required"),
			validation.In("owner", "member").Error("Role must be 'owner' or 'member'"),
		),
	)
}

func (r TenantMemberFilterDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Role,
			validation.When(r.Role != nil,
				validation.In("owner", "member").Error("Role must be 'owner' or 'member'"),
			),
		),
		validation.Field(&r.PaginationRequestDTO),
	)
}
