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
	Name           string         `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Description    string         `gorm:"column:description;type:text;not null" json:"description"`
	Identifier     string         `gorm:"column:identifier;type:varchar(255);uniqueIndex;not null" json:"identifier"`
	Config         datatypes.JSON `gorm:"column:config;type:jsonb;default:'{}'" json:"config"`
	Status         string         `gorm:"column:status;type:varchar(20);default:'active'" json:"status"`
	AuthClientID   int64          `gorm:"column:auth_client_id;not null" json:"auth_client_id"`
	AuthClient     *AuthClient    `gorm:"foreignKey:AuthClientID;references:AuthClientID" json:"auth_client,omitempty"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SignupFlow) TableName() string {
	return "signup_flows"
}

func (sf *SignupFlow) BeforeCreate(tx *gorm.DB) error {
	if sf.SignupFlowUUID == uuid.Nil {
		sf.SignupFlowUUID = uuid.New()
	}
	if sf.Status == "" {
		sf.Status = "active"
	}
	return nil
}
