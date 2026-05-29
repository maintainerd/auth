package tenant

import (
	"regexp"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var tenantNamePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// Tenant output structure
type TenantResponseDTO struct {
	TenantUUID  uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Identifier  string    `json:"identifier"`
	Status      string    `json:"status"`
	IsPublic    bool      `json:"is_public"`
	IsSystem    bool      `json:"is_system"`
	Metadata    any       `json:"metadata,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Create Tenant request DTO
type TenantCreateRequestDTO struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	IsPublic    bool   `json:"is_public"`
}

// Validation
func (r TenantCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
			validation.Match(tenantNamePattern).Error("Name must contain only lowercase letters, numbers, and hyphens"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended).Error("Status must be active, inactive, pending, or suspended"),
		),
		validation.Field(&r.IsPublic,
			validation.In(true, false).Error("Is public is required"),
		),
	)
}

// Update Tenant request DTO
type TenantUpdateRequestDTO struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	IsPublic    bool   `json:"is_public"`
}

// Validation
func (r TenantUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
			validation.Match(tenantNamePattern).Error("Name must contain only lowercase letters, numbers, and hyphens"),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended).Error("Status must be active, inactive, pending, or suspended"),
		),
		validation.Field(&r.IsPublic,
			validation.In(true, false).Error("Is public is required"),
		),
	)
}

// API listing / filter DTO
type TenantFilterDTO struct {
	Name        *string  `json:"name"`
	DisplayName *string  `json:"display_name"`
	Description *string  `json:"description"`
	Identifier  *string  `json:"identifier"`
	Status      []string `json:"status"`
	IsPublic    *bool    `json:"is_public"`
	IsSystem    *bool    `json:"is_system"`

	// Pagination and sorting
	PaginationRequestDTO
}

// Validation
func (r TenantFilterDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PaginationRequestDTO),
	)
}

// TenantMemberResponseDTO for tenant member output
type TenantMemberResponseDTO struct {
	TenantMemberUUID uuid.UUID              `json:"tenant_member_id"`
	Role             string                 `json:"role"`
	User             *MemberUserResponseDTO `json:"user"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// MemberUserResponseDTO is tenant's view of a user in member responses.
// It mirrors the user domain's user response shape to preserve the API.
type MemberUserResponseDTO struct {
	UserUUID           uuid.UUID      `json:"user_id"`
	Username           string         `json:"username"`
	Fullname           string         `json:"fullname"`
	Email              string         `json:"email"`
	Phone              string         `json:"phone"`
	IsEmailVerified    bool           `json:"is_email_verified"`
	IsPhoneVerified    bool           `json:"is_phone_verified"`
	IsProfileCompleted bool           `json:"is_profile_completed"`
	IsAccountCompleted bool           `json:"is_account_completed"`
	Status             string         `json:"status"`
	Metadata           datatypes.JSON `json:"metadata"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// TenantMemberAddMemberRequestDTO for adding member to tenant
type TenantMemberAddMemberRequestDTO struct {
	UserUUID uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
}

func (r TenantMemberAddMemberRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.UserUUID,
			validation.Required.Error("User ID is required"),
		),
		validation.Field(&r.Role,
			validation.Required.Error("Role is required"),
			validation.In("owner", "member").Error("Role must be 'owner' or 'member'"),
		),
	)
}

// TenantMemberUpdateRoleRequestDTO for updating member role
type TenantMemberUpdateRoleRequestDTO struct {
	Role string `json:"role"`
}

func (r TenantMemberUpdateRoleRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Role,
			validation.Required.Error("Role is required"),
			validation.In("owner", "member").Error("Role must be 'owner' or 'member'"),
		),
	)
}

// TenantMemberFilterDTO for filtering tenant members
type TenantMemberFilterDTO struct {
	Role *string `json:"role"`
	PaginationRequestDTO
}

func (r TenantMemberFilterDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Role,
			validation.When(r.Role != nil,
				validation.In("owner", "member").Error("Role must be 'owner' or 'member'"),
			),
		),
		validation.Field(&r.PaginationRequestDTO),
	)
}

// TenantSettingConfigResponseDTO returns a single JSONB config as a map.
type TenantSettingConfigResponseDTO map[string]any

// TenantSettingUpdateConfigRequestDTO is the request body for updating a
// tenant setting config section.
type TenantSettingUpdateConfigRequestDTO map[string]any

// Validate ensures the request body is not empty.
func (r TenantSettingUpdateConfigRequestDTO) Validate() error {
	if len(r) == 0 {
		return validation.NewError("validation_error", "Config cannot be empty")
	}
	return nil
}

type TenantMember struct {
	TenantMemberID   int64          `gorm:"column:tenant_member_id;primaryKey"`
	TenantMemberUUID uuid.UUID      `gorm:"column:tenant_member_uuid;unique;not null"`
	TenantID         int64          `gorm:"column:tenant_id;not null"`
	UserID           int64          `gorm:"column:user_id;not null"`
	Role             string         `gorm:"column:role;not null;default:'member'"`
	CreatedBy        *int64         `gorm:"column:created_by"`
	UpdatedBy        *int64         `gorm:"column:updated_by"`
	CreatedAt        time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (TenantMember) TableName() string {
	return "tenant_members"
}

func (t *TenantMember) BeforeCreate(tx *gorm.DB) (err error) {
	if t.TenantMemberUUID == uuid.Nil {
		t.TenantMemberUUID = uuid.New()
	}
	return
}

// TenantSetting holds tenant-level operational configuration such as rate
// limits, audit settings, maintenance windows, and feature flags.
type TenantSetting struct {
	TenantSettingID   int64          `gorm:"column:tenant_setting_id;primaryKey;autoIncrement" json:"tenant_setting_id"`
	TenantSettingUUID uuid.UUID      `gorm:"column:tenant_setting_uuid;type:uuid;uniqueIndex;not null" json:"tenant_setting_uuid"`
	TenantID          int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	RateLimitConfig   datatypes.JSON `gorm:"column:rate_limit_config;type:jsonb;default:'{}'" json:"rate_limit_config"`
	AuditConfig       datatypes.JSON `gorm:"column:audit_config;type:jsonb;default:'{}'" json:"audit_config"`
	MaintenanceConfig datatypes.JSON `gorm:"column:maintenance_config;type:jsonb;default:'{}'" json:"maintenance_config"`
	FeatureFlags      datatypes.JSON `gorm:"column:feature_flags;type:jsonb;default:'{}'" json:"feature_flags"`
	CreatedAt         time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
}

// TableName returns the database table name for TenantSetting.
func (TenantSetting) TableName() string {
	return "tenant_settings"
}

// BeforeCreate sets a new UUID on the TenantSetting before it is inserted into
// the database if one has not already been assigned.
func (ts *TenantSetting) BeforeCreate(tx *gorm.DB) error {
	if ts.TenantSettingUUID == uuid.Nil {
		ts.TenantSettingUUID = uuid.New()
	}
	return nil
}
