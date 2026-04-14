package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OAuthConsentChallenge represents a pending consent challenge. Created when
// the authorization endpoint determines that the user must approve scopes.
// The frontend retrieves the challenge details, displays them, and submits
// the user's decision. Challenges are short-lived (10 minutes).
type OAuthConsentChallenge struct {
	OAuthConsentChallengeID   int64     `gorm:"column:oauth_consent_challenge_id;primaryKey;autoIncrement"`
	OAuthConsentChallengeUUID uuid.UUID `gorm:"column:oauth_consent_challenge_uuid;type:uuid;uniqueIndex;not null"`
	ClientID                  int64     `gorm:"column:client_id;not null"`
	UserID                    int64     `gorm:"column:user_id;not null"`
	TenantID                  int64     `gorm:"column:tenant_id;not null"`
	RedirectURI               string    `gorm:"column:redirect_uri;not null"`
	Scope                     string    `gorm:"column:scope;not null;default:''"`
	State                     *string   `gorm:"column:state"`
	Nonce                     *string   `gorm:"column:nonce"`
	CodeChallenge             string    `gorm:"column:code_challenge;not null"`
	CodeChallengeMethod       string    `gorm:"column:code_challenge_method;not null;default:'S256'"`
	ResponseType              string    `gorm:"column:response_type;not null;default:'code'"`
	ExpiresAt                 time.Time `gorm:"column:expires_at;not null"`
	CreatedAt                 time.Time `gorm:"column:created_at;autoCreateTime;not null"`

	// Relationships
	Client *Client `gorm:"foreignKey:ClientID;references:ClientID"`
	User   *User   `gorm:"foreignKey:UserID;references:UserID"`
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
}

// TableName returns the database table name for GORM.
func (OAuthConsentChallenge) TableName() string {
	return "oauth_consent_challenges"
}

// BeforeCreate generates a UUID if one is not already set.
func (o *OAuthConsentChallenge) BeforeCreate(_ *gorm.DB) error {
	if o.OAuthConsentChallengeUUID == uuid.Nil {
		o.OAuthConsentChallengeUUID = uuid.New()
	}
	return nil
}

// IsExpired returns true if the consent challenge has passed its expiry time.
func (o *OAuthConsentChallenge) IsExpired() bool {
	return time.Now().After(o.ExpiresAt)
}
