package iam

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// PolicyVersionHistory is an append-only snapshot of a policy's state taken
// before each edit. It captures the exact policy document so a prior version
// can be audited or rolled back. The management_audit_log records who made the
// change; this table records what the policy looked like.
//
// The row is immutable at the DB level (BEFORE UPDATE/DELETE trigger) and must
// only ever be inserted. Backed by migration
// 082_create_policy_version_history_table.go.
//
// Note: the plan's DDL modelled the snapshot as effect/actions/resources/
// conditions columns, but the policies table stores a single multi-statement
// `document` JSONB plus a string `version`. This model snapshots those directly
// (document + policy_version) so the full policy — not a lossy single-statement
// projection — is preserved.
type PolicyVersionHistory struct {
	PolicyVersionHistoryID   int64          `gorm:"column:policy_version_history_id;primaryKey;autoIncrement"`
	PolicyVersionHistoryUUID uuid.UUID      `gorm:"column:policy_version_history_uuid;type:uuid;uniqueIndex;not null"`
	TenantID                 int64          `gorm:"column:tenant_id;not null"`
	PolicyID                 int64          `gorm:"column:policy_id;not null"`
	VersionNumber            int            `gorm:"column:version_number;not null"`
	Name                     string         `gorm:"column:name;type:varchar(255);not null"`
	Description              *string        `gorm:"column:description;type:text"`
	Document                 datatypes.JSON `gorm:"column:document;type:jsonb;not null;default:'{}'"`
	PolicyVersion            string         `gorm:"column:policy_version;type:varchar(20);not null"`
	ChangedByUserID          *int64         `gorm:"column:changed_by_user_id"`
	ChangedByClientID        *int64         `gorm:"column:changed_by_client_id"`
	ChangeReason             *string        `gorm:"column:change_reason;type:text"`
	SnapshotAt               time.Time      `gorm:"column:snapshot_at;not null;autoCreateTime"`
}

// TableName returns the database table name for PolicyVersionHistory.
func (PolicyVersionHistory) TableName() string {
	return "policy_version_history"
}

// BeforeCreate assigns a UUID before insert when one has not been set.
func (p *PolicyVersionHistory) BeforeCreate(tx *gorm.DB) error {
	if p.PolicyVersionHistoryUUID == uuid.Nil {
		p.PolicyVersionHistoryUUID = uuid.New()
	}
	return nil
}
