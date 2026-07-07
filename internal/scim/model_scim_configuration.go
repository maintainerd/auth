package scim

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SCIMConfiguration struct {
	SCIMConfigurationID   int64          `gorm:"column:scim_configuration_id;primaryKey" json:"-"`
	SCIMConfigurationUUID uuid.UUID      `gorm:"column:scim_configuration_uuid;type:uuid;uniqueIndex;not null" json:"scim_configuration_uuid"`
	TenantID              int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	IdentityProviderID    *int64         `gorm:"column:identity_provider_id" json:"identity_provider_id"`
	DisplayName           string         `gorm:"column:display_name;type:varchar(255);not null" json:"display_name"`
	BaseURL               *string        `gorm:"column:base_url;type:varchar(2048)" json:"base_url"`
	BearerTokenHash       *string        `gorm:"column:bearer_token_hash;type:varchar(255)" json:"-"`
	SyncUsers             bool           `gorm:"column:sync_users;not null;default:true" json:"sync_users"`
	SyncGroups            bool           `gorm:"column:sync_groups;not null;default:false" json:"sync_groups"`
	SyncDirection         string         `gorm:"column:sync_direction;type:varchar(20);not null;default:inbound" json:"sync_direction"`
	AttributeMapping      datatypes.JSON `gorm:"column:attribute_mapping;type:jsonb;not null;default:'{}'" json:"attribute_mapping"`
	IsActive              bool           `gorm:"column:is_active;not null;default:true" json:"is_active"`
	LastSyncAt            *time.Time     `gorm:"column:last_sync_at" json:"last_sync_at"`
	LastSyncStatus        *string        `gorm:"column:last_sync_status;type:varchar(20)" json:"last_sync_status"`
	LastSyncError         *string        `gorm:"column:last_sync_error;type:text" json:"last_sync_error"`
	CreatedBy             *int64         `gorm:"column:created_by" json:"created_by"`
	UpdatedBy             *int64         `gorm:"column:updated_by" json:"updated_by"`
	CreatedAt             time.Time      `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time      `gorm:"column:updated_at;not null;autoUpdateTime" json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (SCIMConfiguration) TableName() string {
	return "scim_configurations"
}

func (s *SCIMConfiguration) BeforeCreate(tx *gorm.DB) (err error) {
	if s.SCIMConfigurationUUID == uuid.Nil {
		s.SCIMConfigurationUUID = uuid.New()
	}
	return
}
