package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserToken struct {
	TokenID   int64      `gorm:"column:token_id;primaryKey"`
	TokenUUID uuid.UUID  `gorm:"column:token_uuid;type:uuid;not null;unique;index:idx_user_tokens_token_uuid"`
	UserID    int64      `gorm:"column:user_id;type:integer;not null;index:idx_user_tokens_user_id"`
	TokenType string     `gorm:"column:token_type;type:varchar(50);not null;index:idx_user_tokens_token_type"`
	Token     string     `gorm:"column:token;type:text;not null"` // hashed
	UserAgent *string    `gorm:"column:user_agent;type:text"`
	IPAddress *string    `gorm:"column:ip_address;type:varchar(50)"`
	IsRevoked bool       `gorm:"column:is_revoked;type:boolean;default:false"`
	ExpiresAt *time.Time `gorm:"column:expires_at;type:timestamptz"`
	CreatedAt time.Time  `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt *time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

func (UserToken) TableName() string {
	return "user_tokens"
}

func (ut *UserToken) BeforeCreate(tx *gorm.DB) (err error) {
	if ut.TokenUUID == uuid.Nil {
		ut.TokenUUID = uuid.New()
	}
	return
}
