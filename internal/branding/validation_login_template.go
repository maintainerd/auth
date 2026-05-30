package branding

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/shared"
)

func (r LoginTemplateCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Template,
			validation.Required.Error("Template is required"),
			validation.In(shared.LoginTemplateModern, shared.LoginTemplateClassic, shared.LoginTemplateMinimal, shared.LoginTemplateCorporate, shared.LoginTemplateCreative, shared.LoginTemplateCustom).Error("Template must be one of: modern, classic, minimal, corporate, creative, custom"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

func (r LoginTemplateUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Template,
			validation.Required.Error("Template is required"),
			validation.In(shared.LoginTemplateModern, shared.LoginTemplateClassic, shared.LoginTemplateMinimal, shared.LoginTemplateCorporate, shared.LoginTemplateCreative, shared.LoginTemplateCustom).Error("Template must be one of: modern, classic, minimal, corporate, creative, custom"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

func (r LoginTemplateUpdateStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

func (f LoginTemplateFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Template,
			validation.When(f.Template != nil,
				validation.In(shared.LoginTemplateModern, shared.LoginTemplateClassic, shared.LoginTemplateMinimal, shared.LoginTemplateCorporate, shared.LoginTemplateCreative, shared.LoginTemplateCustom).Error("Template must be one of: modern, classic, minimal, corporate, creative, custom"),
			),
		),
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}
