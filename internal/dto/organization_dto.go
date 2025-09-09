package dto

import (
	"time"

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
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
