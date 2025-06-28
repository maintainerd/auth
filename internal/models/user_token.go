package models

import (
	"time"

	"github.com/google/uuid"
)

type UserToken struct {
	TokenID   int64     `gorm:"primaryKey;column:token_id"`
	TokenUUID uuid.UUID `gorm:"type:uuid;not null;unique;column:token_uuid;index:idx_user_tokens_token_uuid"`

	UserID    int64      `gorm:"not null;index:idx_user_tokens_user_id"`
	TokenType string     `gorm:"type:varchar(50);not null;index:idx_user_tokens_token_type"`
	Token     string     `gorm:"type:text;not null"`
	UserAgent *string    `gorm:"type:text"`
	IPAddress *string    `gorm:"type:varchar(50)"`
	IsRevoked bool       `gorm:"default:false"`
	ExpiresAt *time.Time `gorm:"column:expires_at"`

	CreatedAt time.Time  `gorm:"autoCreateTime"`
	UpdatedAt *time.Time `gorm:"autoUpdateTime"`

	// Relations
	User User `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

// TableName overrides the default table name
func (UserToken) TableName() string {
	return "user_tokens"
}
