package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserToken stores short-lived tokens (email verification, password reset,
// magic links) and persistent session records.
//
// Session-specific fields (LastUsedAt, IdleTimeoutSeconds, AbsoluteExpiresAt)
// are populated only when TokenType == shared.TokenTypeSession; they are NULL for
// all other token types.
type UserToken struct {
	UserTokenID   int64      `gorm:"column:user_token_id;primaryKey"`
	UserTokenUUID uuid.UUID  `gorm:"column:user_token_uuid;not null;unique"`
	UserID        int64      `gorm:"column:user_id;not null"`
	TokenType     string     `gorm:"column:token_type;not null"`
	Token         string     `gorm:"column:token;uniqueIndex:idx_user_tokens_token_unique;not null"` // hashed
	UserAgent     *string    `gorm:"column:user_agent"`
	IPAddress     *string    `gorm:"column:ip_address"`
	IsRevoked     bool       `gorm:"column:is_revoked;not null;default:false"`
	ExpiresAt     *time.Time `gorm:"column:expires_at;index:idx_user_tokens_expires_at,where:expires_at IS NOT NULL"`

	// Session-specific fields — only populated for shared.TokenTypeSession records.
	LastUsedAt         *time.Time `gorm:"column:last_used_at"`
	IdleTimeoutSeconds *int       `gorm:"column:idle_timeout_seconds"`
	AbsoluteExpiresAt  *time.Time `gorm:"column:absolute_expires_at"`

	CreatedAt time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt *time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

func (UserToken) TableName() string {
	return "user_tokens"
}

func (ut *UserToken) BeforeCreate(tx *gorm.DB) (err error) {
	if ut.UserTokenUUID == uuid.Nil {
		ut.UserTokenUUID = uuid.New()
	}
	return
}
