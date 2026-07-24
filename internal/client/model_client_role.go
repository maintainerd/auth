package client

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClientRole struct {
	ClientRoleID   int64     `gorm:"column:client_role_id;primaryKey;autoIncrement"`
	ClientRoleUUID uuid.UUID `gorm:"column:client_role_uuid;type:uuid;uniqueIndex;not null;default:gen_random_uuid()"`
	ClientID       int64     `gorm:"column:client_id;not null"`
	RoleID         int64     `gorm:"column:role_id;not null"`
	CreatedBy      *int64    `gorm:"column:created_by"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime;not null"`

	// Role is the joined role. Without it the listing endpoint could only return
	// the join row's own UUID, which tells an operator nothing about which role is
	// actually attached.
	Role *Role `gorm:"foreignKey:RoleID;references:RoleID"`
}

func (ClientRole) TableName() string { return "client_roles" }

func (c *ClientRole) BeforeCreate(_ *gorm.DB) error {
	if c.ClientRoleUUID == uuid.Nil {
		c.ClientRoleUUID = uuid.New()
	}
	return nil
}
