package dto

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Sort order constants used across all filter DTOs.
const (
	SortOrderAsc  = "asc"
	SortOrderDesc = "desc"
)

// SuccessResponseDTO is a generic success message response used across all handlers.
type SuccessResponseDTO struct {
	Message string `json:"message"`
}

// PaginationRequest makes pagination and sorting reusable
type PaginationRequestDTO struct {
	Page      int    `json:"page"`
	Limit     int    `json:"limit"`
	SortBy    string `json:"sort_by"`
	SortOrder string `json:"sort_order"`
}

// Validate validates the pagination request
func (p PaginationRequestDTO) Validate() error {
	return validation.ValidateStruct(&p,
		validation.Field(&p.Page,
			validation.Required.Error("Page is required"),
			validation.Min(1).Error("Page must be greater than 0"),
		),
		validation.Field(&p.Limit,
			validation.Required.Error("Limit is required"),
			validation.Min(1).Error("Limit must be greater than 0"),
		),
		validation.Field(&p.SortBy,
			validation.Length(0, 50).Error("SortBy cannot exceed 50 characters"),
		),
		validation.Field(&p.SortOrder,
			validation.In(SortOrderAsc, SortOrderDesc).Error("Order must be either 'asc' or 'desc'"),
		),
	)
}

// Generic paginated response
type PaginatedResponseDTO[T any] struct {
	Rows       []T   `json:"rows"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}
