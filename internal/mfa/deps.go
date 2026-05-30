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
	IsTOTPEnabled     bool
	IsWebAuthnEnabled bool
	MFAEnabledAt      *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (User) TableName() string { return "users" }

type UserRepository interface {
	BaseRepositoryMethods[User]
	WithTx(tx *gorm.DB) UserRepository
}
