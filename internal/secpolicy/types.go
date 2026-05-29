package secpolicy

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
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
			validation.When(f.Type != nil, validation.In(shared.IPRuleTypeAllow, shared.IPRuleTypeDeny, shared.IPRuleTypeWhitelist, shared.IPRuleTypeBlacklist).Error("Type must be 'allow', 'deny', 'whitelist', or 'blacklist'")),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}

// Security setting config response - returns config directly
type SecuritySettingConfigResponseDTO map[string]any

// Update config request - accepts config directly
type SecuritySettingUpdateConfigRequestDTO map[string]any

func (r SecuritySettingUpdateConfigRequestDTO) Validate() error {
	if len(r) == 0 {
		return validation.NewError("validation_error", "Config cannot be empty")
	}
	return nil
}

// IPRestrictionRule represents a tenant-scoped IP allow/deny rule that
// controls access to authentication endpoints.
type IPRestrictionRule struct {
	IPRestrictionRuleID   int64          `gorm:"column:ip_restriction_rule_id;primaryKey;autoIncrement" json:"ip_restriction_rule_id"`
	IPRestrictionRuleUUID uuid.UUID      `gorm:"column:ip_restriction_rule_uuid;type:uuid;uniqueIndex;not null" json:"ip_restriction_rule_uuid"`
	TenantID              int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	Description           string         `gorm:"column:description;type:text" json:"description"`
	Type                  string         `gorm:"column:type;type:varchar(20);not null" json:"type"`
	IPAddress             string         `gorm:"column:ip_address;type:inet;not null" json:"ip_address"`
	Status                string         `gorm:"column:status;type:varchar(20);not null;default:'active'" json:"status"`
	CreatedBy             *int64         `gorm:"column:created_by" json:"created_by"`
	UpdatedBy             *int64         `gorm:"column:updated_by" json:"updated_by"`
	CreatedAt             time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`

	// Relationships
}

// TableName returns the database table name for IPRestrictionRule.
func (IPRestrictionRule) TableName() string {
	return "ip_restriction_rules"
}

// BeforeCreate sets a new UUID on the IPRestrictionRule before it is inserted
// into the database if one has not already been assigned.
func (irr *IPRestrictionRule) BeforeCreate(tx *gorm.DB) error {
	if irr.IPRestrictionRuleUUID == uuid.Nil {
		irr.IPRestrictionRuleUUID = uuid.New()
	}
	return nil
}

// SecuritySetting holds pool-level security configuration as a set of JSONB
// columns. Each user pool has exactly one SecuritySetting row.
type SecuritySetting struct {
	SecuritySettingID   int64          `gorm:"column:security_setting_id;primaryKey;autoIncrement" json:"security_setting_id"`
	SecuritySettingUUID uuid.UUID      `gorm:"column:security_setting_uuid;type:uuid;uniqueIndex;not null" json:"security_setting_uuid"`
	UserPoolID          int64          `gorm:"column:user_pool_id;not null" json:"user_pool_id"`
	MFAConfig           datatypes.JSON `gorm:"column:mfa_config;type:jsonb;default:'{}'" json:"mfa_config"`
	PasswordConfig      datatypes.JSON `gorm:"column:password_config;type:jsonb;default:'{}'" json:"password_config"`
	SessionConfig       datatypes.JSON `gorm:"column:session_config;type:jsonb;default:'{}'" json:"session_config"`
	ThreatConfig        datatypes.JSON `gorm:"column:threat_config;type:jsonb;default:'{}'" json:"threat_config"`
	LockoutConfig       datatypes.JSON `gorm:"column:lockout_config;type:jsonb;default:'{}'" json:"lockout_config"`
	RegistrationConfig  datatypes.JSON `gorm:"column:registration_config;type:jsonb;default:'{}'" json:"registration_config"`
	TokenConfig         datatypes.JSON `gorm:"column:token_config;type:jsonb;default:'{}'" json:"token_config"`
	Version             int            `gorm:"column:version;default:1" json:"version"`
	CreatedBy           *int64         `gorm:"column:created_by" json:"created_by"`
	UpdatedBy           *int64         `gorm:"column:updated_by" json:"updated_by"`
	CreatedAt           time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
}

func (SecuritySetting) TableName() string {
	return "security_settings"
}

func (ss *SecuritySetting) BeforeCreate(tx *gorm.DB) error {
	if ss.SecuritySettingUUID == uuid.Nil {
		ss.SecuritySettingUUID = uuid.New()
	}
	return nil
}
