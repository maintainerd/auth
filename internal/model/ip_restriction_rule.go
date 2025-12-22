package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IpRestrictionRule struct {
	IpRestrictionRuleID   int64     `gorm:"column:ip_restriction_rule_id;primaryKey;autoIncrement" json:"ip_restriction_rule_id"`
	IpRestrictionRuleUUID uuid.UUID `gorm:"column:ip_restriction_rule_uuid;type:uuid;uniqueIndex;not null" json:"ip_restriction_rule_uuid"`
	TenantID              int64     `gorm:"column:tenant_id;not null" json:"tenant_id"`
	Description           string    `gorm:"column:description;type:text" json:"description"`
	Type                  string    `gorm:"column:type;type:varchar(20);not null" json:"type"`
	IpAddress             string    `gorm:"column:ip_address;type:varchar(50);not null" json:"ip_address"`
	Status                string    `gorm:"column:status;type:varchar(20);not null;default:'active'" json:"status"`
	CreatedBy             *int64    `gorm:"column:created_by" json:"created_by"`
	UpdatedBy             *int64    `gorm:"column:updated_by" json:"updated_by"`
	CreatedAt             time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relationships
	Tenant  *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
	Creator *User   `gorm:"foreignKey:CreatedBy;references:UserID"`
	Updater *User   `gorm:"foreignKey:UpdatedBy;references:UserID"`
}

func (IpRestrictionRule) TableName() string {
	return "ip_restriction_rules"
}

func (irr *IpRestrictionRule) BeforeCreate(tx *gorm.DB) error {
	if irr.IpRestrictionRuleUUID == uuid.Nil {
		irr.IpRestrictionRuleUUID = uuid.New()
	}
	return nil
}
