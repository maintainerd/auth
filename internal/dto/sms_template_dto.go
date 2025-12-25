package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// SMS template list response DTO (without message content)
type SmsTemplateListResponseDto struct {
	SmsTemplateID string    `json:"sms_template_id"`
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
type SmsTemplateResponseDto struct {
	SmsTemplateID string    `json:"sms_template_id"`
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
type SmsTemplateCreateRequestDto struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Message     string  `json:"message"`
	SenderID    *string `json:"sender_id,omitempty"`
	Status      *string `json:"status,omitempty"`
}

func (r SmsTemplateCreateRequestDto) Validate() error {
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
			validation.In("active", "inactive").Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update SMS template request DTO
type SmsTemplateUpdateRequestDto struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Message     string  `json:"message"`
	SenderID    *string `json:"sender_id,omitempty"`
	Status      *string `json:"status,omitempty"`
}

func (r SmsTemplateUpdateRequestDto) Validate() error {
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
			validation.In("active", "inactive").Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update SMS template status request DTO
type SmsTemplateUpdateStatusRequestDto struct {
	Status string `json:"status"`
}

func (r SmsTemplateUpdateStatusRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In("active", "inactive").Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// SMS template filter DTO
type SmsTemplateFilterDto struct {
	Name      *string  `json:"name"`
	Status    []string `json:"status"`
	IsDefault *bool    `json:"is_default"`
	IsSystem  *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDto
}
