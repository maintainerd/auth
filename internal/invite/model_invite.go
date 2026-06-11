package invite

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Invite struct {
	InviteID        int64          `gorm:"column:invite_id;primaryKey"`
	InviteUUID      uuid.UUID      `gorm:"column:invite_uuid;unique"`
	TenantID        int64          `gorm:"column:tenant_id;not null"`
	ClientID        int64          `gorm:"column:client_id"`
	InvitedEmail    string         `gorm:"column:invited_email"`
	InvitedByUserID *int64         `gorm:"column:invited_by_user_id"`
	InviteToken     string         `gorm:"column:invite_token;unique"`
	AuthFlowID      *int64         `gorm:"column:auth_flow_id"`
	Status          string         `gorm:"column:status;default:pending"` // pending, accepted, expired, revoked
	ExpiresAt       *time.Time     `gorm:"column:expires_at"`
	UsedAt          *time.Time     `gorm:"column:used_at"`
	CreatedBy       *int64         `gorm:"column:created_by"`
	UpdatedBy       *int64         `gorm:"column:updated_by"`
	CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at;index"`

	AuthFlow *AuthFlow `gorm:"foreignKey:AuthFlowID;references:AuthFlowID"`
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
