package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type User struct {
	UserID   int64     `gorm:"column:user_id;primaryKey"`
	UserUUID uuid.UUID `gorm:"column:user_uuid;unique"`
	// TenantID scopes the user to a tenant. Users are isolated per tenant:
	// email/username are unique per (tenant_id, ...), so the same address can
	// exist as a separate account in different tenants.
	TenantID int64  `gorm:"column:tenant_id"`
	Username string `gorm:"column:username"`
	// Fullname is not persisted on users; it lives in Profile.
	Fullname                   string         `gorm:"-"`
	Email                      string         `gorm:"column:email"`
	Phone                      string         `gorm:"column:phone"`
	Password                   *string        `gorm:"column:password" json:"-"`
	IsEmailVerified            bool           `gorm:"column:is_email_verified;default:false"`
	IsPhoneVerified            bool           `gorm:"column:is_phone_verified;default:false"`
	IsProfileCompleted         bool           `gorm:"column:is_profile_completed;default:false"`
	IsAccountCompleted         bool           `gorm:"column:is_account_completed;default:false"`
	Status                     string         `gorm:"column:status;default:'active'"`
	Metadata                   datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'"`
	ForcePasswordChange        bool           `gorm:"column:force_password_change;default:false"`
	PasswordChangedAt          *time.Time     `gorm:"column:password_changed_at"`
	TemporaryPasswordExpiresAt *time.Time     `gorm:"column:temporary_password_expires_at"`
	IsTOTPEnabled              bool           `gorm:"column:is_totp_enabled;default:false"`
	IsWebAuthnEnabled          bool           `gorm:"column:is_webauthn_enabled;default:false"`
	FirstMFAEnrolledAt         *time.Time     `gorm:"column:first_mfa_enrolled_at"`
	CreatedAt                  time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt                  time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt                  gorm.DeletedAt `gorm:"column:deleted_at;index"`

	UserIdentities []UserIdentity `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
	UserRoles      []UserRole     `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
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
