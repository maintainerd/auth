package idp

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

type RegistrationFlow struct {
	RegistrationFlowID   int64          `gorm:"column:registration_flow_id;primaryKey;autoIncrement" json:"registration_flow_id"`
	RegistrationFlowUUID uuid.UUID      `gorm:"column:registration_flow_uuid;type:uuid;uniqueIndex;not null" json:"registration_flow_uuid"`
	TenantID             int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	Name                 string         `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Description          string         `gorm:"column:description;type:text;not null" json:"description"`
	Identifier           string         `gorm:"column:identifier;type:varchar(255);not null" json:"identifier"`
	IsSystem             bool           `gorm:"column:is_system;default:false" json:"is_system"`
	VerificationRequired bool           `gorm:"column:verification_required;default:false" json:"verification_required"`
	RequiredFields       string         `gorm:"column:required_fields;default:'[]'" json:"required_fields"`
	Status               string         `gorm:"column:status;type:varchar(20);default:'active'" json:"status"`
	ClientID             int64          `gorm:"column:client_id;not null" json:"client_id"`
	CreatedBy            *int64         `gorm:"column:created_by" json:"created_by,omitempty"`
	UpdatedBy            *int64         `gorm:"column:updated_by" json:"updated_by,omitempty"`
	CreatedAt            time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`

	Client *Client `gorm:"foreignKey:ClientID;references:ClientID" json:"client,omitempty"`
}

func (RegistrationFlow) TableName() string {
	return "registration_flows"
}

func (flow *RegistrationFlow) BeforeCreate(tx *gorm.DB) error {
	if flow.RegistrationFlowUUID == uuid.Nil {
		flow.RegistrationFlowUUID = uuid.New()
	}
	if flow.Status == "" {
		flow.Status = shared.StatusActive
	}
	return nil
}
