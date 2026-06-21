package branding

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/shared"
)

func (r EmailTemplateCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Subject,
			validation.Required.Error("Subject is required"),
			validation.Length(1, 255).Error("Subject must be between 1 and 255 characters"),
		),
		validation.Field(&r.BodyHTML,
			validation.Required.Error("Body HTML is required"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

func (r EmailTemplateUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Subject,
			validation.Required.Error("Subject is required"),
			validation.Length(1, 255).Error("Subject must be between 1 and 255 characters"),
		),
		validation.Field(&r.BodyHTML,
			validation.Required.Error("Body HTML is required"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

func (r EmailTemplateUpdateStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

func (f EmailTemplateFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}
