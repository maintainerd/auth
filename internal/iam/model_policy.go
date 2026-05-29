package iam

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Policy struct {
	PolicyID    int64          `gorm:"column:policy_id;primaryKey"`
	PolicyUUID  uuid.UUID      `gorm:"column:policy_uuid;unique"`
	TenantID    int64          `gorm:"column:tenant_id;not null"`
	Name        string         `gorm:"column:name"`
	Description *string        `gorm:"column:description"`
	Document    datatypes.JSON `gorm:"column:document"`
	Version     string         `gorm:"column:version"`
	Status      string         `gorm:"column:status;default:'inactive'"`
	IsSystem    bool           `gorm:"column:is_system;default:false"`
	CreatedBy   *int64         `gorm:"column:created_by"`
	UpdatedBy   *int64         `gorm:"column:updated_by"`
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (Policy) TableName() string {
	return "policies"
}

func (p *Policy) BeforeCreate(tx *gorm.DB) (err error) {
	if p.PolicyUUID == uuid.Nil {
		p.PolicyUUID = uuid.New()
	}
	return
}
