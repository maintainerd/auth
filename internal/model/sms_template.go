package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SmsTemplate struct {
	SmsTemplateID   int64     `gorm:"column:sms_template_id;primaryKey"`
	SmsTemplateUUID uuid.UUID `gorm:"column:sms_template_uuid;unique"`
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

func (SmsTemplate) TableName() string {
	return "sms_templates"
}

func (s *SmsTemplate) BeforeCreate(tx *gorm.DB) (err error) {
	if s.SmsTemplateUUID == uuid.Nil {
		s.SmsTemplateUUID = uuid.New()
	}
	return
}
