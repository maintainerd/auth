package idp

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/gorm"
)

type AuthFlow struct {
	AuthFlowID           int64          `gorm:"column:auth_flow_id;primaryKey;autoIncrement" json:"auth_flow_id"`
	AuthFlowUUID         uuid.UUID      `gorm:"column:auth_flow_uuid;type:uuid;uniqueIndex;not null" json:"auth_flow_uuid"`
	TenantID             int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	Name                 string         `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Description          string         `gorm:"column:description;type:text;not null" json:"description"`
	Identifier           string         `gorm:"column:identifier;type:varchar(255);not null" json:"identifier"`
	Status               string         `gorm:"column:status;type:varchar(20);default:'active'" json:"status"`
	ClientID             *int64         `gorm:"column:client_id" json:"client_id,omitempty"`
	BrandingID           *int64         `gorm:"column:branding_id" json:"branding_id,omitempty"`
	CreatedBy            *int64         `gorm:"column:created_by" json:"created_by,omitempty"`
	UpdatedBy            *int64         `gorm:"column:updated_by" json:"updated_by,omitempty"`
	CreatedAt            time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`

	Client   *Client   `gorm:"foreignKey:ClientID;references:ClientID" json:"client,omitempty"`
	Branding *Branding `gorm:"foreignKey:BrandingID;references:BrandingID" json:"branding,omitempty"`
}

func (AuthFlow) TableName() string {
	return "auth_flows"
}

func (af *AuthFlow) BeforeCreate(tx *gorm.DB) error {
	if af.AuthFlowUUID == uuid.Nil {
		af.AuthFlowUUID = uuid.New()
	}
	if af.Status == "" {
		af.Status = shared.StatusActive
	}
	return nil
}
