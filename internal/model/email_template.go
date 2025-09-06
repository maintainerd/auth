package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EmailTemplate struct {
	EmailTemplateID   int64     `gorm:"column:email_template_id;primaryKey"`
	EmailTemplateUUID uuid.UUID `gorm:"column:email_template_uuid;unique"`
	Name              string    `gorm:"column:name;unique"`
	Subject           string    `gorm:"column:subject"`
	BodyHTML          string    `gorm:"column:body_html"`
	BodyPlain         *string   `gorm:"column:body_plain"`
	IsActive          bool      `gorm:"column:is_active;default:true"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (EmailTemplate) TableName() string {
	return "email_templates"
}

func (e *EmailTemplate) BeforeCreate(tx *gorm.DB) (err error) {
	if e.EmailTemplateUUID == uuid.Nil {
		e.EmailTemplateUUID = uuid.New()
	}
	return
}
