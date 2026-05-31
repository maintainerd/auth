package secpolicy

import (
	"time"
)

// IPRestrictionRuleResponseDTO is the JSON representation of an IP restriction
// rule.
type IPRestrictionRuleResponseDTO struct {
	IPRestrictionRuleID string    `json:"ip_restriction_rule_id"`
	Description         string    `json:"description"`
	Type                string    `json:"type"`
	IPAddress           string    `json:"ip_address"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// IPRestrictionRuleCreateRequestDTO is the request body for creating an IP
// restriction rule.
type IPRestrictionRuleCreateRequestDTO struct {
	Description string  `json:"description"`
	Type        string  `json:"type"`
	IPAddress   string  `json:"ip_address"`
	Status      *string `json:"status,omitempty"`
}

// IPRestrictionRuleUpdateRequestDTO is the request body for updating an IP
// restriction rule.
type IPRestrictionRuleUpdateRequestDTO struct {
	Description string  `json:"description"`
	Type        string  `json:"type"`
	IPAddress   string  `json:"ip_address"`
	Status      *string `json:"status,omitempty"`
}

// IPRestrictionRuleUpdateStatusRequestDTO is the request body for updating an
// IP restriction rule's status.
type IPRestrictionRuleUpdateStatusRequestDTO struct {
	Status string `json:"status"`
}

// IPRestrictionRuleFilterDTO holds query parameters for listing and filtering
// IP restriction rules.
type IPRestrictionRuleFilterDTO struct {
	Type        *string  `json:"type"`
	Status      []string `json:"status"`
	IPAddress   *string  `json:"ip_address"`
	Description *string  `json:"description"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Security setting config response - returns config directly
type SecuritySettingConfigResponseDTO map[string]any

// Update config request - accepts config directly
type SecuritySettingUpdateConfigRequestDTO map[string]any
