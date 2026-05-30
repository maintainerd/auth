package tenant

import validation "github.com/go-ozzo/ozzo-validation/v4"

func (r TenantMemberAddMemberRequestDTO) Validate() error {
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
