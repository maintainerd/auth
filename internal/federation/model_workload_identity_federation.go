package federation

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// WorkloadIdentityFederation is the trust configuration that lets an external
// workload (Kubernetes pod, GitHub Actions job, GitLab CI job, …) exchange its
// OIDC token for a platform access token without a long-lived credential.
//
// Backed by migration 079_create_workload_identity_federations_table.go.
type WorkloadIdentityFederation struct {
	WorkloadIdentityFederationID   int64          `gorm:"column:workload_identity_federation_id;primaryKey;autoIncrement" json:"-"`
	WorkloadIdentityFederationUUID uuid.UUID      `gorm:"column:workload_identity_federation_uuid;type:uuid;uniqueIndex;not null" json:"workload_identity_federation_uuid"`
	TenantID                       int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	ClientID                       int64          `gorm:"column:client_id;not null" json:"client_id"`
	Name                           string         `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Description                    *string        `gorm:"column:description;type:text" json:"description,omitempty"`
	IssuerURL                      string         `gorm:"column:issuer_url;type:varchar(2048);not null" json:"issuer_url"`
	Audience                       string         `gorm:"column:audience;type:varchar(512);not null" json:"audience"`
	SubjectClaim                   string         `gorm:"column:subject_claim;type:varchar(100);not null;default:'sub'" json:"subject_claim"`
	SubjectPattern                 string         `gorm:"column:subject_pattern;type:varchar(512);not null" json:"subject_pattern"`
	AllowedScopes                  pq.StringArray `gorm:"column:allowed_scopes;type:text[];not null;default:'{}'" json:"allowed_scopes"`
	AttributeMapping               datatypes.JSON `gorm:"column:attribute_mapping;type:jsonb;not null;default:'{}'" json:"attribute_mapping"`
	IsActive                       bool           `gorm:"column:is_active;not null;default:true" json:"is_active"`
	CreatedBy                      *int64         `gorm:"column:created_by" json:"-"`
	UpdatedBy                      *int64         `gorm:"column:updated_by" json:"-"`
	CreatedAt                      time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt                      time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt                      gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName returns the database table name for WorkloadIdentityFederation.
func (WorkloadIdentityFederation) TableName() string {
	return "workload_identity_federations"
}

// BeforeCreate assigns a UUID before insert when one has not been set.
func (w *WorkloadIdentityFederation) BeforeCreate(tx *gorm.DB) error {
	if w.WorkloadIdentityFederationUUID == uuid.Nil {
		w.WorkloadIdentityFederationUUID = uuid.New()
	}
	return nil
}
