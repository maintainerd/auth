package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/maintainerd/auth/internal/model"
)

// Signup flow output structure
type SignupFlowResponseDto struct {
	SignupFlowUUID string         `json:"signup_flow_id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Identifier     string         `json:"identifier"`
	Config         map[string]any `json:"config"`
	Status         string         `json:"status"`
	ClientUUID     string         `json:"client_id,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// Create signup flow request dto
type SignupFlowCreateRequestDto struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Config      map[string]any `json:"config,omitempty"`
	Status      *string        `json:"status,omitempty"`
	ClientUUID  string         `json:"client_id"`
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
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
		validation.Field(&r.ClientUUID,
			validation.Required.Error("Auth client UUID is required"),
			is.UUID.Error("Invalid auth client UUID format"),
		),
	)
}

// Update signup flow request dto
type SignupFlowUpdateRequestDto struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Config      map[string]any `json:"config,omitempty"`
	Status      *string        `json:"status,omitempty"`
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
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
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
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Signup flow listing request dto
type SignupFlowFilterDto struct {
	Name       *string  `json:"name"`
	Identifier *string  `json:"identifier"`
	Status     []string `json:"status"`
	ClientUUID *string  `json:"client_id"`

	// Pagination and sorting
	PaginationRequestDto
}

// Validate validates the signup flow filter DTO.
func (f SignupFlowFilterDto) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.ClientUUID,
			validation.When(f.ClientUUID != nil,
				is.UUID.Error("Client ID must be a valid UUID"),
			),
		),
		validation.Field(&f.PaginationRequestDto),
	)
}
