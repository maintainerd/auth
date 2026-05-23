package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type User struct {
	UserID             int64          `gorm:"column:user_id;primaryKey"`
	UserUUID           uuid.UUID      `gorm:"column:user_uuid;unique"`
	Username           string         `gorm:"column:username"`
	// Fullname is not persisted on users — it lives in Profile (first_name + last_name + display_name).
	// Kept as a transient field for API compatibility; populated by services when loading users
	// with their Profile, and split into Profile fields when creating/updating.
	Fullname           string         `gorm:"-"`
	Email              string         `gorm:"column:email"`
	Phone              string         `gorm:"column:phone"`
	Password           *string        `gorm:"column:password" json:"-"` // nullable for external users
	IsEmailVerified    bool           `gorm:"column:is_email_verified;default:false"`
	IsPhoneVerified    bool           `gorm:"column:is_phone_verified;default:false"`
	IsProfileCompleted bool           `gorm:"column:is_profile_completed;default:false"`
	IsAccountCompleted bool           `gorm:"column:is_account_completed;default:false"`
	Status             string         `gorm:"column:status;default:'active'"`
	Metadata           datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'"`
	// Feature: Force password change on next login
	ForcePasswordChange bool `gorm:"column:force_password_change;default:false"`
	// Feature: Email change with re-verification
	PendingEmail            *string        `gorm:"column:pending_email"`
	EmailChangeOTP          *string        `gorm:"column:email_change_otp"`
	EmailChangeOTPExpiresAt *time.Time     `gorm:"column:email_change_otp_expires_at"`
	CreatedAt               time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt               time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt               gorm.DeletedAt `gorm:"column:deleted_at;index"`

	// Relationships
	UserIdentities []UserIdentity `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
	UserRoles      []UserRole     `gorm:"foreignKey:UserID;references:UserID"`
	Roles          []Role         `gorm:"many2many:user_roles;joinForeignKey:UserID;joinReferences:RoleID"`
	UserTokens     []UserToken    `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
	Profile        *Profile       `gorm:"foreignKey:UserID;references:UserID"`
	UserSetting    *UserSetting   `gorm:"foreignKey:UserID;references:UserID"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.UserUUID == uuid.Nil {
		u.UserUUID = uuid.New()
	}
	return
}
