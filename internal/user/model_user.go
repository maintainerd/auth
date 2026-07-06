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
	IsEmailVerified            bool           `gorm:"column:is_email_verified;not null;default:false"`
	IsPhoneVerified            bool           `gorm:"column:is_phone_verified;not null;default:false"`
	Status                     string         `gorm:"column:status;not null;default:'active'"`
	Metadata                   datatypes.JSON `gorm:"column:metadata;type:jsonb;not null;default:'{}'"`
	ForcePasswordChange        bool           `gorm:"column:force_password_change;default:false"`
	PasswordChangedAt          *time.Time     `gorm:"column:password_changed_at"`
	TemporaryPasswordExpiresAt *time.Time     `gorm:"column:temporary_password_expires_at"`
	IsTOTPEnabled              bool           `gorm:"column:is_totp_enabled;default:false"`
	IsWebAuthnEnabled          bool           `gorm:"column:is_webauthn_enabled;default:false"`
	FirstMFAEnrolledAt         *time.Time     `gorm:"column:first_mfa_enrolled_at"`
	LastLoginAt                *time.Time     `gorm:"column:last_login_at"`
	LoginCount                 int            `gorm:"column:login_count;not null;default:0"`
	EmailVerifiedAt            *time.Time     `gorm:"column:email_verified_at"`
	PhoneVerifiedAt            *time.Time     `gorm:"column:phone_verified_at"`
	ExternalID                 *string        `gorm:"column:external_id"`
	CreatedBy                  *int64         `gorm:"column:created_by"`
	UpdatedBy                  *int64         `gorm:"column:updated_by"`
	CreatedAt                  time.Time      `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt                  time.Time      `gorm:"column:updated_at;not null;autoUpdateTime"`
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
