package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SignupFlow struct {
	SignupFlowID   int64          `gorm:"column:signup_flow_id;primaryKey;autoIncrement" json:"signup_flow_id"`
	SignupFlowUUID uuid.UUID      `gorm:"column:signup_flow_uuid;type:uuid;uniqueIndex;not null" json:"signup_flow_uuid"`
	TenantID       int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	Name           string         `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Description    string         `gorm:"column:description;type:text;not null" json:"description"`
	Identifier     string         `gorm:"column:identifier;type:varchar(255);not null" json:"identifier"`
	Config         datatypes.JSON `gorm:"column:config;type:jsonb;default:'{}'" json:"config"`
	Status         string         `gorm:"column:status;type:varchar(20);default:'active'" json:"status"`
	ClientID       int64          `gorm:"column:client_id;not null" json:"client_id"`
	Client         *Client        `gorm:"foreignKey:ClientID;references:ClientID" json:"client,omitempty"`
	CreatedBy      *int64         `gorm:"column:created_by" json:"created_by,omitempty"`
	UpdatedBy      *int64         `gorm:"column:updated_by" json:"updated_by,omitempty"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`
}

func (SignupFlow) TableName() string {
	return "signup_flows"
}

func (sf *SignupFlow) BeforeCreate(tx *gorm.DB) error {
	if sf.SignupFlowUUID == uuid.Nil {
		sf.SignupFlowUUID = uuid.New()
	}
	if sf.Status == "" {
		sf.Status = StatusActive
	}
	return nil
}
