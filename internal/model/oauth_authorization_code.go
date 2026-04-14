package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OAuthAuthorizationCode represents a pending or consumed authorization code.
// Codes are short-lived (10 minutes), single-use, and bound to a PKCE
// code_challenge for protection against interception attacks.
type OAuthAuthorizationCode struct {
	OAuthAuthorizationCodeID   int64      `gorm:"column:oauth_authorization_code_id;primaryKey;autoIncrement"`
	OAuthAuthorizationCodeUUID uuid.UUID  `gorm:"column:oauth_authorization_code_uuid;type:uuid;uniqueIndex;not null"`
	CodeHash                   string     `gorm:"column:code_hash;uniqueIndex;not null"`
	ClientID                   int64      `gorm:"column:client_id;not null"`
	UserID                     int64      `gorm:"column:user_id;not null"`
	TenantID                   int64      `gorm:"column:tenant_id;not null"`
	RedirectURI                string     `gorm:"column:redirect_uri;not null"`
	Scope                      string     `gorm:"column:scope;not null;default:''"`
	State                      *string    `gorm:"column:state"`
	Nonce                      *string    `gorm:"column:nonce"`
	CodeChallenge              string     `gorm:"column:code_challenge;not null"`
	CodeChallengeMethod        string     `gorm:"column:code_challenge_method;not null;default:'S256'"`
	IsUsed                     bool       `gorm:"column:is_used;not null;default:false"`
	UsedAt                     *time.Time `gorm:"column:used_at"`
	ExpiresAt                  time.Time  `gorm:"column:expires_at;not null"`
	CreatedAt                  time.Time  `gorm:"column:created_at;autoCreateTime;not null"`

	// Relationships
	Client *Client `gorm:"foreignKey:ClientID;references:ClientID"`
	User   *User   `gorm:"foreignKey:UserID;references:UserID"`
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
}

// TableName returns the database table name for GORM.
func (OAuthAuthorizationCode) TableName() string {
	return "oauth_authorization_codes"
}

// BeforeCreate generates a UUID if one is not already set.
func (o *OAuthAuthorizationCode) BeforeCreate(_ *gorm.DB) error {
	if o.OAuthAuthorizationCodeUUID == uuid.Nil {
		o.OAuthAuthorizationCodeUUID = uuid.New()
	}
	return nil
}

// IsExpired returns true if the authorization code has passed its expiry time.
func (o *OAuthAuthorizationCode) IsExpired() bool {
	return time.Now().After(o.ExpiresAt)
}
