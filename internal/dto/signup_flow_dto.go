package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// Signup flow output structure
type SignupFlowResponseDto struct {
	SignupFlowUUID string                 `json:"signup_flow_id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Identifier     string                 `json:"identifier"`
	Config         map[string]interface{} `json:"config"`
	Status         string                 `json:"status"`
	AuthClientUUID string                 `json:"client_id,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// Create signup flow request dto
type SignupFlowCreateRequestDto struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Config         map[string]interface{} `json:"config,omitempty"`
	Status         *string                `json:"status,omitempty"`
	AuthClientUUID string                 `json:"client_id"`
}

func (r SignupFlowCreateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Signup flow name is required"),
			validation.Length(1, 100).Error("Signup flow name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
		),
		validation.Field(&r.Status,
			validation.In("active", "inactive").Error("Status must be 'active' or 'inactive'"),
		),
		validation.Field(&r.AuthClientUUID,
			validation.Required.Error("Auth client UUID is required"),
			is.UUID.Error("Invalid auth client UUID format"),
		),
	)
}

// Update signup flow request dto
type SignupFlowUpdateRequestDto struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Status      *string                `json:"status,omitempty"`
}

func (r SignupFlowUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Signup flow name is required"),
			validation.Length(1, 100).Error("Signup flow name must be between 1 and 100 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
		),
		validation.Field(&r.Status,
			validation.In("active", "inactive").Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Update signup flow status request dto
type SignupFlowUpdateStatusRequestDto struct {
	Status string `json:"status"`
}

func (r SignupFlowUpdateStatusRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In("active", "inactive").Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Signup flow listing request dto
type SignupFlowFilterDto struct {
	Name           *string  `json:"name"`
	Identifier     *string  `json:"identifier"`
	Status         []string `json:"status"`
	AuthClientUUID *string  `json:"client_id"`

	// Pagination and sorting
	PaginationRequestDto
}
