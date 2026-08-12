package idp

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type RegistrationFlow struct {
	RegistrationFlowID   int64     `gorm:"column:registration_flow_id;primaryKey;autoIncrement"`
	RegistrationFlowUUID uuid.UUID `gorm:"column:registration_flow_uuid;type:uuid;uniqueIndex;not null"`
	TenantID             int64     `gorm:"column:tenant_id;not null"`
	ClientID             int64     `gorm:"column:client_id;not null"`
	// Name doubles as the public selector an external app puts in a registration
	// link (?registration_flow=<name>), so it is validated as a slug and is
	// unique per tenant. Renaming a flow therefore changes its link — see
	// docs/features/registration-and-invites.md.
	//
	// It is NOT a secret: it travels in URLs, browser history and referrers.
	// Authorization lives in the client binding, the flow's own status, the
	// grantable-role guard, and (for system flows) the invite requirement.
	Name                 string         `gorm:"column:name;type:varchar(100);not null"`
	Description          string         `gorm:"column:description;type:text;not null;default:''"`
	RequiredFields       datatypes.JSON `gorm:"column:required_fields;type:jsonb;default:'[]'"`
	VerificationRequired bool           `gorm:"column:verification_required;default:false"`
	IsSystem             bool           `gorm:"column:is_system;default:false"`
	Status               string         `gorm:"column:status;type:varchar(20);not null;default:'active'"`
	CreatedBy            *int64         `gorm:"column:created_by"`
	UpdatedBy            *int64         `gorm:"column:updated_by"`
	CreatedAt            time.Time      `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt            time.Time      `gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt            gorm.DeletedAt `gorm:"column:deleted_at;index"`

	Client *Client `gorm:"foreignKey:ClientID;references:ClientID"`
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
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
