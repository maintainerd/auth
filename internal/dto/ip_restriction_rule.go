package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/maintainerd/auth/internal/model"
)

// IP restriction rule response DTO
type IPRestrictionRuleResponseDTO struct {
	IPRestrictionRuleID string    `json:"ip_restriction_rule_id"`
	Description         string    `json:"description"`
	Type                string    `json:"type"`
	IPAddress           string    `json:"ip_address"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// Create IP restriction rule request DTO
type IPRestrictionRuleCreateRequestDTO struct {
	Description string  `json:"description"`
	Type        string  `json:"type"`
	IPAddress   string  `json:"ip_address"`
	Status      *string `json:"status,omitempty"`
}

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

// Update IP restriction rule request DTO
type IPRestrictionRuleUpdateRequestDTO struct {
	Description string  `json:"description"`
	Type        string  `json:"type"`
	IPAddress   string  `json:"ip_address"`
	Status      *string `json:"status,omitempty"`
}

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

// Update IP restriction rule status request DTO
type IPRestrictionRuleUpdateStatusRequestDTO struct {
	Status string `json:"status"`
}

func (r IPRestrictionRuleUpdateStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(model.StatusActive, model.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// IP restriction rule filter DTO
type IPRestrictionRuleFilterDTO struct {
	Type        *string  `json:"type"`
	Status      []string `json:"status"`
	IPAddress   *string  `json:"ip_address"`
	Description *string  `json:"description"`

	// Pagination and sorting
	PaginationRequestDTO
}

func (f IPRestrictionRuleFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.PaginationRequestDTO),
	)
}
