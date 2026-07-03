package secpolicy

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// Validate validates the IP restriction rule create request.
func (r IPRestrictionRuleCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Description,
			validation.Length(0, 500).Error("Description must not exceed 500 characters"),
		),
		validation.Field(&r.Type,
			validation.Required.Error("Type is required"),
			validation.In(shared.IPRuleTypeAllow, shared.IPRuleTypeDeny, shared.IPRuleTypeWhitelist, shared.IPRuleTypeBlacklist).Error("Type must be 'allow', 'deny', 'whitelist', or 'blacklist'"),
		),
		validation.Field(&r.IPAddress,
			validation.Required.Error("IP address is required"),
			is.IPv4.Error("Invalid IPv4 address format"),
			validation.Length(1, 50).Error("IP address must be between 1 and 50 characters"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Validate validates the IP restriction rule update request.
func (r IPRestrictionRuleUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Description,
			validation.Length(0, 500).Error("Description must not exceed 500 characters"),
		),
		validation.Field(&r.Type,
			validation.Required.Error("Type is required"),
			validation.In(shared.IPRuleTypeAllow, shared.IPRuleTypeDeny, shared.IPRuleTypeWhitelist, shared.IPRuleTypeBlacklist).Error("Type must be 'allow', 'deny', 'whitelist', or 'blacklist'"),
		),
		validation.Field(&r.IPAddress,
			validation.Required.Error("IP address is required"),
			is.IPv4.Error("Invalid IPv4 address format"),
			validation.Length(1, 50).Error("IP address must be between 1 and 50 characters"),
		),
		validation.Field(&r.Status,
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Validate validates the IP restriction rule status update request.
func (r IPRestrictionRuleUpdateStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
		),
	)
}

// Validate validates the IP restriction rule filter parameters.
func (f IPRestrictionRuleFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Type,
			validation.When(f.Type != nil, validation.In(shared.IPRuleTypeAllow, shared.IPRuleTypeDeny, shared.IPRuleTypeWhitelist, shared.IPRuleTypeBlacklist).Error("Type must be 'allow', 'deny', 'whitelist', or 'blacklist'")),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}
