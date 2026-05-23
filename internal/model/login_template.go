package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type LoginTemplate struct {
	LoginTemplateID   int64          `gorm:"primaryKey;column:login_template_id"`
	LoginTemplateUUID uuid.UUID      `gorm:"type:uuid;uniqueIndex;column:login_template_uuid"`
	TenantID          int64          `gorm:"column:tenant_id;not null"`
	Name              string         `gorm:"type:varchar(100);not null;column:name"`
	Description       *string        `gorm:"type:text;column:description"`
	Template          string         `gorm:"type:varchar(20);not null;column:template"`
	Status            string         `gorm:"type:varchar(20);not null;default:'active';column:status"`
	Metadata          datatypes.JSON `gorm:"type:jsonb;default:'{}';column:metadata"`
	IsDefault         bool           `gorm:"default:false;column:is_default"`
	IsSystem          bool           `gorm:"default:false;column:is_system"`
	CreatedBy         *int64         `gorm:"column:created_by"`
	UpdatedBy         *int64         `gorm:"column:updated_by"`
	CreatedAt         time.Time      `gorm:"column:created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (LoginTemplate) TableName() string {
	return "login_templates"
}

func (t *LoginTemplate) BeforeCreate(tx *gorm.DB) error {
	if t.LoginTemplateUUID == uuid.Nil {
		t.LoginTemplateUUID = uuid.New()
	}
	return nil
}
