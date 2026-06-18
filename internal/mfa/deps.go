package mfa

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	UserID            int64
	UserUUID          uuid.UUID
	Email             string
	Username          string
	Phone             string `gorm:"column:phone"`
	IsPhoneVerified   bool   `gorm:"column:is_phone_verified"`
	IsTOTPEnabled     bool
	IsWebAuthnEnabled bool `gorm:"column:is_webauthn_enabled"`
	MFAEnabledAt      *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (User) TableName() string { return "users" }

type UserRepository interface {
	BaseRepositoryMethods[User]
	FindByID(id any, preloads ...string) (*User, error)
	FindByUUID(uuid any, preloads ...string) (*User, error)
	WithTx(tx *gorm.DB) UserRepository
}
