package branding

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/shared"
)

func (r SMSTemplateCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Message,
			validation.Required.Error("Message is required"),
		),
		validation.Field(&r.SenderID,
			validation.Length(0, 20).Error("Sender ID must not exceed 20 characters"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

func (r SMSTemplateUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Message,
			validation.Required.Error("Message is required"),
		),
		validation.Field(&r.SenderID,
			validation.Length(0, 20).Error("Sender ID must not exceed 20 characters"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

func (r SMSTemplateUpdateStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

func (f SMSTemplateFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}
