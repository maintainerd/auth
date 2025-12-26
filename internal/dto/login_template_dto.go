package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Login template list response DTO (without metadata)
type LoginTemplateListResponseDto struct {
	LoginTemplateID string    `json:"login_template_id"`
	Name            string    `json:"name"`
	Description     *string   `json:"description"`
	Template        string    `json:"template"`
	Status          string    `json:"status"`
	IsDefault       bool      `json:"is_default"`
	IsSystem        bool      `json:"is_system"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Login template response DTO (full details with metadata)
type LoginTemplateResponseDto struct {
	LoginTemplateID string         `json:"login_template_id"`
	Name            string         `json:"name"`
	Description     *string        `json:"description"`
	Template        string         `json:"template"`
	Status          string         `json:"status"`
	Metadata        map[string]any `json:"metadata"`
	IsDefault       bool           `json:"is_default"`
	IsSystem        bool           `json:"is_system"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// Create login template request DTO
type LoginTemplateCreateRequestDto struct {
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	Template    string         `json:"template"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Status      *string        `json:"status,omitempty"`
}

func (r LoginTemplateCreateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Template,
			validation.Required.Error("Template is required"),
			validation.In("modern", "classic", "minimal", "corporate", "creative", "custom").Error("Template must be one of: modern, classic, minimal, corporate, creative, custom"),
		),
		validation.Field(&r.Status,
			validation.In("active", "inactive").Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update login template request DTO
type LoginTemplateUpdateRequestDto struct {
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	Template    string         `json:"template"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Status      *string        `json:"status,omitempty"`
}

func (r LoginTemplateUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Template,
			validation.Required.Error("Template is required"),
			validation.In("modern", "classic", "minimal", "corporate", "creative", "custom").Error("Template must be one of: modern, classic, minimal, corporate, creative, custom"),
		),
		validation.Field(&r.Status,
			validation.In("active", "inactive").Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update login template status request DTO
type LoginTemplateUpdateStatusRequestDto struct {
	Status string `json:"status"`
}

func (r LoginTemplateUpdateStatusRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In("active", "inactive").Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Login template filter DTO
type LoginTemplateFilterDto struct {
	Name      *string  `json:"name"`
	Status    []string `json:"status"`
	Template  *string  `json:"template"`
	IsDefault *bool    `json:"is_default"`
	IsSystem  *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDto
}
