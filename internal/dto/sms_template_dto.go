package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/maintainerd/auth/internal/model"
)

// SMS template list response DTO (without message content)
type SMSTemplateListResponseDto struct {
	SMSTemplateID string    `json:"sms_template_id"`
	Name          string    `json:"name"`
	Description   *string   `json:"description"`
	SenderID      *string   `json:"sender_id"`
	Status        string    `json:"status"`
	IsDefault     bool      `json:"is_default"`
	IsSystem      bool      `json:"is_system"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SMS template response DTO (full details with message content)
type SMSTemplateResponseDto struct {
	SMSTemplateID string    `json:"sms_template_id"`
	Name          string    `json:"name"`
	Description   *string   `json:"description"`
	Message       string    `json:"message"`
	SenderID      *string   `json:"sender_id"`
	Status        string    `json:"status"`
	IsDefault     bool      `json:"is_default"`
	IsSystem      bool      `json:"is_system"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Create SMS template request DTO
type SMSTemplateCreateRequestDto struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Message     string  `json:"message"`
	SenderID    *string `json:"sender_id,omitempty"`
	Status      *string `json:"status,omitempty"`
}

func (r SMSTemplateCreateRequestDto) Validate() error {
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
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update SMS template request DTO
type SMSTemplateUpdateRequestDto struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Message     string  `json:"message"`
	SenderID    *string `json:"sender_id,omitempty"`
	Status      *string `json:"status,omitempty"`
}

func (r SMSTemplateUpdateRequestDto) Validate() error {
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
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update SMS template status request DTO
type SMSTemplateUpdateStatusRequestDto struct {
	Status string `json:"status"`
}

func (r SMSTemplateUpdateStatusRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// SMS template filter DTO
type SMSTemplateFilterDto struct {
	Name      *string  `json:"name"`
	Status    []string `json:"status"`
	IsDefault *bool    `json:"is_default"`
	IsSystem  *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDto
}

// Validate validates the SMS template filter DTO.
func (f SMSTemplateFilterDto) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.PaginationRequestDto),
	)
}
