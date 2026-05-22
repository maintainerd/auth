package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Device code status constants.
const (
	DeviceCodeStatusPending  = "pending"
	DeviceCodeStatusApproved = "approved"
	DeviceCodeStatusDenied   = "denied"
	DeviceCodeStatusExpired  = "expired"
)

// Grant type constants for new RFC grants.
const (
	GrantTypeDeviceCode    = "urn:ietf:params:oauth:grant-type:device_code"
	GrantTypeTokenExchange = "urn:ietf:params:oauth:grant-type:token-exchange"
	GrantTypeCIBA          = "urn:ietf:params:oauth:grant-type:ciba"
)

// OAuthDeviceCode represents a pending device authorization request (RFC 8628).
// The device_code is returned to the client (device) and is used to poll the
// token endpoint. The user_code is displayed to the user who then visits the
// verification URI to approve or deny access.
type OAuthDeviceCode struct {
	OAuthDeviceCodeID   int64     `gorm:"column:oauth_device_code_id;primaryKey;autoIncrement"`
	OAuthDeviceCodeUUID uuid.UUID `gorm:"column:oauth_device_code_uuid;type:uuid;uniqueIndex;not null"`
	DeviceCodeHash      string    `gorm:"column:device_code_hash;uniqueIndex;not null"`
	UserCode            string    `gorm:"column:user_code;uniqueIndex;not null"`
	ClientID            int64     `gorm:"column:client_id;not null"`
	TenantID            int64     `gorm:"column:tenant_id;not null"`
	Scope               string    `gorm:"column:scope;not null;default:''"`
	// UserID is set once the user approves the request at the verification URI.
	UserID   *int64 `gorm:"column:user_id"`
	Status   string `gorm:"column:status;not null;default:'pending'"`
	Interval int    `gorm:"column:interval;not null;default:5"`
	// LastPollAt tracks the most recent polling attempt for slow-down enforcement.
	LastPollAt *time.Time `gorm:"column:last_poll_at"`
	ExpiresAt  time.Time  `gorm:"column:expires_at;not null"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime;not null"`

	// Relationships
	Client *Client `gorm:"foreignKey:ClientID;references:ClientID"`
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
	User   *User   `gorm:"foreignKey:UserID;references:UserID"`
}

func (OAuthDeviceCode) TableName() string {
	return "oauth_device_codes"
}

func (o *OAuthDeviceCode) BeforeCreate(_ *gorm.DB) error {
	if o.OAuthDeviceCodeUUID == uuid.Nil {
		o.OAuthDeviceCodeUUID = uuid.New()
	}
	return nil
}

func (o *OAuthDeviceCode) IsExpired() bool {
	return time.Now().After(o.ExpiresAt)
}
