package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/maintainerd/auth/internal/model"
)

// Login template list response DTO (without metadata)
type LoginTemplateListResponseDTO struct {
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
type LoginTemplateResponseDTO struct {
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
type LoginTemplateCreateRequestDTO struct {
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	Template    string         `json:"template"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Status      *string        `json:"status,omitempty"`
}

func (r LoginTemplateCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Template,
			validation.Required.Error("Template is required"),
			validation.In(model.LoginTemplateModern, model.LoginTemplateClassic, model.LoginTemplateMinimal, model.LoginTemplateCorporate, model.LoginTemplateCreative, model.LoginTemplateCustom).Error("Template must be one of: modern, classic, minimal, corporate, creative, custom"),
		),
		validation.Field(&r.Status,
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update login template request DTO
type LoginTemplateUpdateRequestDTO struct {
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	Template    string         `json:"template"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Status      *string        `json:"status,omitempty"`
}

func (r LoginTemplateUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Template,
			validation.Required.Error("Template is required"),
			validation.In(model.LoginTemplateModern, model.LoginTemplateClassic, model.LoginTemplateMinimal, model.LoginTemplateCorporate, model.LoginTemplateCreative, model.LoginTemplateCustom).Error("Template must be one of: modern, classic, minimal, corporate, creative, custom"),
		),
		validation.Field(&r.Status,
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update login template status request DTO
type LoginTemplateUpdateStatusRequestDTO struct {
	Status string `json:"status"`
}

func (r LoginTemplateUpdateStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Login template filter DTO
type LoginTemplateFilterDTO struct {
	Name      *string  `json:"name"`
	Status    []string `json:"status"`
	Template  *string  `json:"template"`
	IsDefault *bool    `json:"is_default"`
	IsSystem  *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validate validates the login template filter DTO.
func (f LoginTemplateFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Template,
			validation.When(f.Template != nil,
				validation.In(model.LoginTemplateModern, model.LoginTemplateClassic, model.LoginTemplateMinimal, model.LoginTemplateCorporate, model.LoginTemplateCreative, model.LoginTemplateCustom).Error("Template must be one of: modern, classic, minimal, corporate, creative, custom"),
			),
		),
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}
