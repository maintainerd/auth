package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/maintainerd/auth/internal/model"
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

// Validate validates the IP restriction rule create request.
func (r IPRestrictionRuleCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Description,
			validation.Length(0, 500).Error("Description must not exceed 500 characters"),
		),
		validation.Field(&r.Type,
			validation.Required.Error("Type is required"),
			validation.In(model.IPRuleTypeAllow, model.IPRuleTypeDeny, model.IPRuleTypeWhitelist, model.IPRuleTypeBlacklist).Error("Type must be 'allow', 'deny', 'whitelist', or 'blacklist'"),
		),
		validation.Field(&r.IPAddress,
			validation.Required.Error("IP address is required"),
			is.IPv4.Error("Invalid IPv4 address format"),
			validation.Length(1, 50).Error("IP address must be between 1 and 50 characters"),
		),
		validation.Field(&r.Status,
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// IPRestrictionRuleUpdateRequestDTO is the request body for updating an IP
// restriction rule.
type IPRestrictionRuleUpdateRequestDTO struct {
	Description string  `json:"description"`
	Type        string  `json:"type"`
	IPAddress   string  `json:"ip_address"`
	Status      *string `json:"status,omitempty"`
}

// Validate validates the IP restriction rule update request.
func (r IPRestrictionRuleUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Description,
			validation.Length(0, 500).Error("Description must not exceed 500 characters"),
		),
		validation.Field(&r.Type,
			validation.Required.Error("Type is required"),
			validation.In(model.IPRuleTypeAllow, model.IPRuleTypeDeny, model.IPRuleTypeWhitelist, model.IPRuleTypeBlacklist).Error("Type must be 'allow', 'deny', 'whitelist', or 'blacklist'"),
		),
		validation.Field(&r.IPAddress,
			validation.Required.Error("IP address is required"),
			is.IPv4.Error("Invalid IPv4 address format"),
			validation.Length(1, 50).Error("IP address must be between 1 and 50 characters"),
		),
		validation.Field(&r.Status,
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// IPRestrictionRuleUpdateStatusRequestDTO is the request body for updating an
// IP restriction rule's status.
type IPRestrictionRuleUpdateStatusRequestDTO struct {
	Status string `json:"status"`
}

// Validate validates the IP restriction rule status update request.
func (r IPRestrictionRuleUpdateStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
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

// Validate validates the IP restriction rule filter parameters.
func (f IPRestrictionRuleFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Type,
			validation.When(f.Type != nil, validation.In(model.IPRuleTypeAllow, model.IPRuleTypeDeny, model.IPRuleTypeWhitelist, model.IPRuleTypeBlacklist).Error("Type must be 'allow', 'deny', 'whitelist', or 'blacklist'")),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}
