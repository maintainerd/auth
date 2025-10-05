package dto

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// CreateOrganizationRequestDto for initial organization setup
type CreateOrganizationRequestDto struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Email       *string `json:"email,omitempty"`
	Phone       *string `json:"phone,omitempty"`
}

func (dto CreateOrganizationRequestDto) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Name,
			validation.Required.Error("Organization name is required"),
			validation.Length(2, 100).Error("Organization name must be between 2 and 100 characters"),
			validation.Match(regexp.MustCompile(`^[a-zA-Z0-9\s\-_\.]+$`)).Error("Organization name contains invalid characters"),
		),
		validation.Field(&dto.Description,
			validation.When(dto.Description != nil,
				validation.Length(0, 500).Error("Description must not exceed 500 characters"),
			),
		),
		validation.Field(&dto.Email,
			validation.When(dto.Email != nil,
				is.Email.Error("Invalid email format"),
				validation.Length(0, 100).Error("Email must not exceed 100 characters"),
			),
		),
		validation.Field(&dto.Phone,
			validation.When(dto.Phone != nil,
				validation.Length(0, 20).Error("Phone must not exceed 20 characters"),
			),
		),
	)
}

// CreateAdminRequestDto for initial admin user setup
type CreateAdminRequestDto struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

func (dto CreateAdminRequestDto) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.Username,
			validation.Required.Error("Username is required"),
			validation.Length(3, 50).Error("Username must be between 3 and 50 characters"),
			validation.Match(regexp.MustCompile(`^[a-zA-Z0-9_\-\.@]+$`)).Error("Username contains invalid characters"),
		),
		validation.Field(&dto.Password,
			validation.Required.Error("Password is required"),
			validation.Length(8, 100).Error("Password must be between 8 and 100 characters"),
		),
		validation.Field(&dto.Email,
			validation.Required.Error("Email is required"),
			is.Email.Error("Invalid email format"),
			validation.Length(0, 100).Error("Email must not exceed 100 characters"),
		),
	)
}

// SetupStatusResponseDto for checking setup status
type SetupStatusResponseDto struct {
	IsOrganizationSetup bool `json:"is_organization_setup"`
	IsAdminSetup        bool `json:"is_admin_setup"`
	IsSetupComplete     bool `json:"is_setup_complete"`
}

// CreateOrganizationResponseDto for organization creation response
type CreateOrganizationResponseDto struct {
	Message      string                  `json:"message"`
	Organization OrganizationResponseDto `json:"organization"`
}

// CreateAdminResponseDto for admin creation response
type CreateAdminResponseDto struct {
	Message      string          `json:"message"`
	User         UserResponseDto `json:"user"`
	TokenResponse *AuthResponseDto `json:"token_response,omitempty"`
}
