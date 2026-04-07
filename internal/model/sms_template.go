package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SMSTemplate struct {
	SMSTemplateID   int64     `gorm:"column:sms_template_id;primaryKey"`
	SMSTemplateUUID uuid.UUID `gorm:"column:sms_template_uuid;unique"`
	TenantID        int64     `gorm:"column:tenant_id;not null"`
	Name            string    `gorm:"column:name;unique"`
	Description     *string   `gorm:"column:description"`
	Message         string    `gorm:"column:message"`
	SenderID        *string   `gorm:"column:sender_id"`
	Status          string    `gorm:"column:status;default:'active'"`
	IsDefault       bool      `gorm:"column:is_default;default:false"`
	IsSystem        bool      `gorm:"column:is_system;default:false"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (SMSTemplate) TableName() string {
	return "sms_templates"
}

func (s *SMSTemplate) BeforeCreate(tx *gorm.DB) (err error) {
	if s.SMSTemplateUUID == uuid.Nil {
		s.SMSTemplateUUID = uuid.New()
	}
	return
}
