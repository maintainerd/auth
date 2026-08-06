package invite

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Invite struct {
	InviteID           int64     `gorm:"column:invite_id;primaryKey"`
	InviteUUID         uuid.UUID `gorm:"column:invite_uuid;unique"`
	TenantID           int64     `gorm:"column:tenant_id;not null"`
	ClientID           int64     `gorm:"column:client_id"`
	RegistrationFlowID *int64    `gorm:"column:registration_flow_id"`
	InvitedByUserID    *int64    `gorm:"column:invited_by_user_id"`
	InvitedEmail       string    `gorm:"column:invited_email"`
	// InviteTokenHash stores only the digest of the emailed invite token. The raw
	// token is a bearer credential that creates an account and carries the
	// registration flow's role grants, so persisting it in cleartext turned every
	// read of this table — a backup, a read replica, a support query, a SQL
	// injection — into a set of live, unexpired account-creation credentials.
	// json:"-" keeps the digest out of the audit-log "after" snapshot the handler
	// marshals from this struct, which previously recorded the raw token.
	// The column keeps its historical `invite_token` name; the value in it is a
	// digest, never a token.
	InviteTokenHash string         `gorm:"column:invite_token;unique" json:"-"`
	CallbackURL     *string        `gorm:"column:callback_url"`
	Status          string         `gorm:"column:status;not null;default:pending"` // pending, accepted, expired, revoked
	ExpiresAt       *time.Time     `gorm:"column:expires_at"`
	UsedAt          *time.Time     `gorm:"column:used_at"`
	CreatedBy       *int64         `gorm:"column:created_by"`
	UpdatedBy       *int64         `gorm:"column:updated_by"`
	CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at;index"`

	RegistrationFlow *RegistrationFlow `gorm:"foreignKey:RegistrationFlowID;references:RegistrationFlowID"`
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
