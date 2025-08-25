package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Invite struct {
	InviteID        int64      `gorm:"column:invite_id;primaryKey"`
	InviteUUID      uuid.UUID  `gorm:"column:invite_uuid;unique"`
	AuthClientID    int64      `gorm:"column:auth_client_id"`
	InvitedEmail    string     `gorm:"column:invited_email"`
	InvitedByUserID int64      `gorm:"column:invited_by_user_id"`
	InviteToken     string     `gorm:"column:invite_token;unique"`
	Status          string     `gorm:"column:status;default:pending"` // pending, accepted, expired, revoked
	ExpiresAt       *time.Time `gorm:"column:expires_at"`
	UsedAt          *time.Time `gorm:"column:used_at"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       *time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	AuthClient    *AuthClient `gorm:"foreignKey:AuthClientID;references:AuthClientID;constraint:OnDelete:CASCADE"`
	InvitedByUser *User       `gorm:"foreignKey:InvitedByUserID;references:UserID;constraint:OnDelete:SET NULL"`
	Roles         []Role      `gorm:"many2many:invite_roles;joinForeignKey:InviteID;joinReferences:RoleID;constraint:OnDelete:CASCADE"`
}

func (Invite) TableName() string {
	return "invites"
}

func (i *Invite) BeforeCreate(tx *gorm.DB) (err error) {
	if i.InviteUUID == uuid.Nil {
		i.InviteUUID = uuid.New()
	}
	return
}
