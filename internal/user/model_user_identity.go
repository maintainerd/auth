package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserIdentity struct {
	UserIdentityID     int64          `gorm:"column:user_identity_id;primaryKey"`
	UserIdentityUUID   uuid.UUID      `gorm:"column:user_identity_uuid;unique"`
	TenantID           int64          `gorm:"column:tenant_id"`
	UserID             int64          `gorm:"column:user_id"`
	ClientID           *int64         `gorm:"column:client_id"`
	IdentityProviderID *int64         `gorm:"column:identity_provider_id"`
	Provider           string         `gorm:"column:provider"`
	Sub                string         `gorm:"column:sub"`
	Metadata           datatypes.JSON `gorm:"column:metadata"`
	JITProvisionedAt   *time.Time     `gorm:"column:jit_provisioned_at"`
	ProvisioningSource *string        `gorm:"column:provisioning_source"`
	CreatedAt          time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Tenant           *Tenant           `gorm:"foreignKey:TenantID;references:TenantID;constraint:OnDelete:CASCADE"`
	User             *User             `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
	Client           *Client           `gorm:"foreignKey:ClientID;references:ClientID;constraint:OnDelete:SET NULL"`
	IdentityProvider *IdentityProvider `gorm:"foreignKey:IdentityProviderID;references:IdentityProviderID"`
}

func (UserIdentity) TableName() string {
	return "user_identities"
}

func (ui *UserIdentity) BeforeCreate(tx *gorm.DB) (err error) {
	if ui.UserIdentityUUID == uuid.Nil {
		ui.UserIdentityUUID = uuid.New()
	}
	return
}
