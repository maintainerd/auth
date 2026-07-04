package oauth

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// OAuthRefreshToken represents a refresh token with family tracking for
// rotation and reuse detection. When a refresh token is rotated, the new
// token inherits the family_id so that if the old token is replayed, the
// entire family can be revoked.
type OAuthRefreshToken struct {
	OAuthRefreshTokenID   int64      `gorm:"column:oauth_refresh_token_id;primaryKey;autoIncrement"`
	OAuthRefreshTokenUUID uuid.UUID  `gorm:"column:oauth_refresh_token_uuid;type:uuid;uniqueIndex;not null"`
	TokenHash             string     `gorm:"column:token_hash;uniqueIndex;not null"`
	FamilyID              uuid.UUID  `gorm:"column:family_id;index:idx_oauth_refresh_family;not null"`
	ClientID              int64      `gorm:"column:client_id;index:idx_oauth_refresh_user_client;not null"`
	UserID                int64      `gorm:"column:user_id;index:idx_oauth_refresh_user_client;not null"`
	TenantID              int64      `gorm:"column:tenant_id;not null"`
	Scope                 pq.StringArray `gorm:"column:scope;type:text[];not null;default:'{}'"`
	IsRevoked             bool       `gorm:"column:is_revoked;not null;default:false"`
	RevokedAt             *time.Time `gorm:"column:revoked_at"`
	ExpiresAt             time.Time  `gorm:"column:expires_at;index:idx_oauth_refresh_expires;not null"`
	LastUsedAt            *time.Time `gorm:"column:last_used_at"`
	CreatedAt             time.Time  `gorm:"column:created_at;autoCreateTime;not null"`

	// Relationships
	Client *Client `gorm:"foreignKey:ClientID;references:ClientID"`
}

// TableName returns the database table name for GORM.
func (OAuthRefreshToken) TableName() string {
	return "oauth_refresh_tokens"
}

// BeforeCreate generates a UUID if one is not already set.
func (o *OAuthRefreshToken) BeforeCreate(_ *gorm.DB) error {
	if o.OAuthRefreshTokenUUID == uuid.Nil {
		o.OAuthRefreshTokenUUID = uuid.New()
	}
	return nil
}

// IsExpired returns true if the refresh token has passed its expiry time.
func (o *OAuthRefreshToken) IsExpired() bool {
	return time.Now().After(o.ExpiresAt)
}

// IsActive returns true if the token is neither revoked nor expired.
func (o *OAuthRefreshToken) IsActive() bool {
	return !o.IsRevoked && !o.IsExpired()
}
