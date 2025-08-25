package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EmailTemplate struct {
	TemplateID   int64     `gorm:"column:template_id;primaryKey"`
	TemplateUUID uuid.UUID `gorm:"column:template_uuid"`
	Name         string    `gorm:"column:name"`
	Subject      string    `gorm:"column:subject"`
	BodyHTML     string    `gorm:"column:body_html"`
	BodyPlain    *string   `gorm:"column:body_plain"`
	IsActive     bool      `gorm:"column:is_active"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (EmailTemplate) TableName() string {
	return "email_templates"
}

func (e *EmailTemplate) BeforeCreate(tx *gorm.DB) (err error) {
	if e.TemplateUUID == uuid.Nil {
		e.TemplateUUID = uuid.New()
	}
	return
}
