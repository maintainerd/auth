package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"

	"github.com/maintainerd/auth/internal/model"
)

// API output structure
type APIResponseDto struct {
	APIUUID     uuid.UUID           `json:"api_id"`
	Name        string              `json:"name"`
	DisplayName string              `json:"display_name"`
	Description string              `json:"description"`
	APIType     string              `json:"api_type"`
	Identifier  string              `json:"identifier"`
	Service     *ServiceResponseDto `json:"service,omitempty"`
	Status      string              `json:"status"`
	IsSystem    bool                `json:"is_system"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// Create API request DTO
type APICreateRequestDto struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	APIType     string `json:"api_type"`
	Status      string `json:"status"`
	ServiceUUID string `json:"service_id"`
}

// Validation
func (r APICreateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("API name is required"),
			validation.Length(3, 50).Error("API name must be between 3 and 50 characters"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(3, 50).Error("Display name must be between 3 and 50 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.APIType,
			validation.Required.Error("API type is required"),
			validation.In(model.APITypeRest, model.APITypeGRPC, model.APITypeGraphQL, model.APITypeSOAP, model.APITypeWebhook, model.APITypeWebSocket, model.APITypeRPC).Error("API type must be one of: rest, grpc, graphql, soap, webhook, websocket, rpc"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be one of: active, inactive"),
		),

		validation.Field(&r.ServiceUUID,
			validation.Required.Error("Service ID is required"),
			is.UUID.Error("Service ID must be a valid UUID"),
		),
	)
}

// Update API request DTO
type APIUpdateRequestDto struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	APIType     string `json:"api_type"`
	Status      string `json:"status"`
	ServiceUUID string `json:"service_id"`
}

// Validation
func (r APIUpdateRequestDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("API name is required"),
			validation.Length(3, 50).Error("API name must be between 3 and 50 characters"),
		),
		validation.Field(&r.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(3, 50).Error("Display name must be between 3 and 50 characters"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.APIType,
			validation.Required.Error("API type is required"),
			validation.In(model.APITypeRest, model.APITypeGRPC, model.APITypeGraphQL, model.APITypeSOAP, model.APITypeWebhook, model.APITypeWebSocket, model.APITypeRPC).Error("API type must be one of: rest, grpc, graphql, soap, webhook, websocket, rpc"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be one of: active, inactive"),
		),
		validation.Field(&r.ServiceUUID,
			validation.Required.Error("Service ID is required"),
			is.UUID.Error("Service ID must be a valid UUID"),
		),
	)
}

// API listing / filter DTO
type APIFilterDto struct {
	Name        *string  `json:"name"`
	DisplayName *string  `json:"display_name"`
	Description *string  `json:"description"`
	APIType     *string  `json:"api_type"`
	Identifier  *string  `json:"identifier"`
	ServiceUUID *string  `json:"service_id"`
	Status      []string `json:"status"`
	IsSystem    *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDto
}

// Validate validates the API filter DTO.
func (f APIFilterDto) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.APIType,
			validation.When(f.APIType != nil,
				validation.In(model.APITypeRest, model.APITypeGRPC, model.APITypeGraphQL, model.APITypeSOAP, model.APITypeWebhook, model.APITypeWebSocket, model.APITypeRPC).Error("API type must be one of: rest, grpc, graphql, soap, webhook, websocket, rpc"),
			),
		),
		validation.Field(&f.Status,
			validation.When(len(f.Status) > 0,
				validation.Each(validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'")),
			),
		),
		validation.Field(&f.PaginationRequestDto),
	)
}

// API status update DTO
type APIStatusUpdateDto struct {
	Status string `json:"status"`
}

func (r APIStatusUpdateDto) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be one of: active, inactive"),
		),
	)
}
