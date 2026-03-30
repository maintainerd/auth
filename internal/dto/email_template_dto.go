package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/maintainerd/auth/internal/model"
)

// Email template list response DTO (without body content)
type EmailTemplateListResponseDto struct {
	EmailTemplateID string    `json:"email_template_id"`
	Name            string    `json:"name"`
	Subject         string    `json:"subject"`
	Status          string    `json:"status"`
	IsDefault       bool      `json:"is_default"`
	IsSystem        bool      `json:"is_system"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Email template response DTO (full details with body content)
type EmailTemplateResponseDto struct {
	EmailTemplateID string    `json:"email_template_id"`
	Name            string    `json:"name"`
	Subject         string    `json:"subject"`
	BodyHTML        string    `json:"body_html"`
	BodyPlain       *string   `json:"body_plain"`
	Status          string    `json:"status"`
	IsDefault       bool      `json:"is_default"`
	IsSystem        bool      `json:"is_system"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Create email template request DTO
type EmailTemplateCreateRequestDto struct {
	Name      string  `json:"name"`
	Subject   string  `json:"subject"`
	BodyHTML  string  `json:"body_html"`
	BodyPlain *string `json:"body_plain,omitempty"`
	Status    *string `json:"status,omitempty"`
}

func (r EmailTemplateCreateRequestDto) Validate() error {
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
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update email template request DTO
type EmailTemplateUpdateRequestDto struct {
	Name      string  `json:"name"`
	Subject   string  `json:"subject"`
	BodyHTML  string  `json:"body_html"`
	BodyPlain *string `json:"body_plain,omitempty"`
	Status    *string `json:"status,omitempty"`
}

func (r EmailTemplateUpdateRequestDto) Validate() error {
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
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update email template status request DTO
type EmailTemplateUpdateStatusRequestDto struct {
	Status string `json:"status"`
}

func (r EmailTemplateUpdateStatusRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Email template filter DTO
type EmailTemplateFilterDto struct {
	Name      *string  `json:"name"`
	Status    []string `json:"status"`
	IsDefault *bool    `json:"is_default"`
	IsSystem  *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDto
}

// Validate validates the email template filter DTO.
func (f EmailTemplateFilterDto) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.PaginationRequestDto),
	)
}
