package dto

import (
	"regexp"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

// Organization output structure
type OrganizationResponseDto struct {
	OrganizationUUID uuid.UUID `json:"organization_uuid"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Email            string    `json:"email"`
	Phone            string    `json:"phone"`
	IsActive         bool      `json:"is_active"`
	IsDefault        bool      `json:"is_default"`
	IsRoot           bool      `json:"is_root"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Create or update orgainzation request dto
type OrganizationCreateOrUpdateRequestDto struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Email       *string `json:"email"`
	Phone       *string `json:"phone"`
	IsActive    bool    `json:"is_active"`
}

func (r OrganizationCreateOrUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 20).Error("Name must be between 3 and 20 characters"),
		),
		validation.Field(&r.Description,
			validation.When(r.Description != nil,
				validation.Length(8, 100).Error("Description must be between 8 and 100 characters"),
			),
		),
		validation.Field(&r.Email,
			validation.When(r.Email != nil,
				validation.Length(8, 100).Error("Email must be between 8 and 100 characters"),
				validation.Match(regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)).Error("Invalid email format"),
			),
		),
		validation.Field(&r.Phone,
			validation.When(r.Phone != nil,
				validation.Length(8, 100).Error("Phone must be between 8 and 100 characters"),
			),
		),
		validation.Field(&r.IsActive,
			validation.In(true, false).Error("Is active is required"),
		),
	)
}

// Organization listing
type OrganizationFilterDto struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Email       *string `json:"email"`
	Phone       *string `json:"phone"`
	IsDefault   *bool   `json:"is_default"`
	IsRoot      *bool   `json:"is_root"`
	IsActive    *bool   `json:"is_active"`

	// Pagination and sorting
	PaginationRequestDto
}
