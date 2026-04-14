package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OAuthConsentGrant represents a user's consent decision for a specific client.
// Each user-client pair has at most one row. The scopes field is a
// space-delimited list of the scopes the user has approved.
type OAuthConsentGrant struct {
	OAuthConsentGrantID   int64     `gorm:"column:oauth_consent_grant_id;primaryKey;autoIncrement"`
	OAuthConsentGrantUUID uuid.UUID `gorm:"column:oauth_consent_grant_uuid;type:uuid;uniqueIndex;not null"`
	UserID                int64     `gorm:"column:user_id;not null"`
	ClientID              int64     `gorm:"column:client_id;not null"`
	TenantID              int64     `gorm:"column:tenant_id;not null"`
	Scopes                string    `gorm:"column:scopes;not null;default:''"`
	CreatedAt             time.Time `gorm:"column:created_at;autoCreateTime;not null"`
	UpdatedAt             time.Time `gorm:"column:updated_at;autoUpdateTime;not null"`

	// Relationships
	User   *User   `gorm:"foreignKey:UserID;references:UserID"`
	Client *Client `gorm:"foreignKey:ClientID;references:ClientID"`
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
}

// TableName returns the database table name for GORM.
func (OAuthConsentGrant) TableName() string {
	return "oauth_consent_grants"
}

// BeforeCreate generates a UUID if one is not already set.
func (o *OAuthConsentGrant) BeforeCreate(_ *gorm.DB) error {
	if o.OAuthConsentGrantUUID == uuid.Nil {
		o.OAuthConsentGrantUUID = uuid.New()
	}
	return nil
}
