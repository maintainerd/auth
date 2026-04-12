package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// UserPool is the isolation boundary for users, roles, clients, and settings
// within a single tenant deployment. Each pool represents an independent
// application's user namespace, analogous to an AWS Cognito User Pool.
type UserPool struct {
	UserPoolID   int64          `gorm:"column:user_pool_id;primaryKey"`
	UserPoolUUID uuid.UUID      `gorm:"column:user_pool_uuid;unique;not null"`
	TenantID     int64          `gorm:"column:tenant_id;not null"`
	Name         string         `gorm:"column:name;not null"`
	DisplayName  string         `gorm:"column:display_name"`
	Identifier   string         `gorm:"column:identifier;not null"`
	IsDefault    bool           `gorm:"column:is_default;default:false"`
	IsSystem     bool           `gorm:"column:is_system;default:false"`
	Status       string         `gorm:"column:status;type:varchar(16);default:'active'"`
	Metadata     datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt    *time.Time     `gorm:"column:deleted_at"`

	// Relationships
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
}

// TableName returns the database table name for UserPool.
func (UserPool) TableName() string {
	return "user_pools"
}

// BeforeCreate sets a new UUID on the UserPool before it is inserted into the
// database if one has not already been assigned.
func (up *UserPool) BeforeCreate(tx *gorm.DB) error {
	if up.UserPoolUUID == uuid.Nil {
		up.UserPoolUUID = uuid.New()
	}
	return nil
}
