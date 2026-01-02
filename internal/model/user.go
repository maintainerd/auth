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
	TenantID           int64          `gorm:"column:tenant_id"`
	Username           string         `gorm:"column:username"`
	Fullname           string         `gorm:"column:fullname"`
	Email              string         `gorm:"column:email"`
	Phone              string         `gorm:"column:phone"`
	Password           *string        `gorm:"column:password"` // nullable for external users
	IsEmailVerified    bool           `gorm:"column:is_email_verified;default:false"`
	IsPhoneVerified    bool           `gorm:"column:is_phone_verified;default:false"`
	IsProfileCompleted bool           `gorm:"column:is_profile_completed;default:false"`
	IsAccountCompleted bool           `gorm:"column:is_account_completed;default:false"`
	Status             string         `gorm:"column:status;default:'active'"`
	Metadata           datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'"`
	CreatedAt          time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Tenant         *Tenant        `gorm:"foreignKey:TenantID;references:TenantID;constraint:OnDelete:CASCADE"`
	UserIdentities []UserIdentity `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
	UserRoles      []UserRole     `gorm:"foreignKey:UserID;references:UserID"`
	Roles          []Role         `gorm:"many2many:user_roles;joinForeignKey:UserID;joinReferences:RoleID"`
	UserTokens     []UserToken    `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
	AuthLogs       []AuthLog      `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:SET NULL"`
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
