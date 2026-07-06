package iam

import (
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

// PolicyVersionHistoryRepository provides append-only persistence and read
// access for policy version snapshots. There is no update or delete method —
// the table is immutable by design (and enforced by a DB trigger).
type PolicyVersionHistoryRepository interface {
	WithTx(tx *gorm.DB) PolicyVersionHistoryRepository
	// Create appends a snapshot row.
	Create(entry *PolicyVersionHistory) (*PolicyVersionHistory, error)
	// NextVersionNumber returns MAX(version_number)+1 for a policy (1 when none).
	NextVersionNumber(policyID int64) (int, error)
	// FindByPolicyIDPaginated lists snapshots for a policy, newest first.
	FindByPolicyIDPaginated(policyID int64, page, limit int) (*PaginationResult[PolicyVersionHistory], error)
	// FindByPolicyIDAndVersion returns a single snapshot; nil when absent.
	FindByPolicyIDAndVersion(policyID int64, versionNumber int) (*PolicyVersionHistory, error)
}

type policyVersionHistoryRepository struct {
	db *gorm.DB
}

// NewPolicyVersionHistoryRepository creates a new repository backed by db.
func NewPolicyVersionHistoryRepository(db *gorm.DB) PolicyVersionHistoryRepository {
	return &policyVersionHistoryRepository{db: db}
}

// WithTx returns a copy of the repository bound to the supplied transaction so
// the snapshot insert participates in the policy-update transaction.
func (r *policyVersionHistoryRepository) WithTx(tx *gorm.DB) PolicyVersionHistoryRepository {
	return &policyVersionHistoryRepository{db: tx}
}

func (r *policyVersionHistoryRepository) Create(entry *PolicyVersionHistory) (*PolicyVersionHistory, error) {
	if err := r.db.Create(entry).Error; err != nil {
		return nil, err
	}
	return entry, nil
}

func (r *policyVersionHistoryRepository) NextVersionNumber(policyID int64) (int, error) {
	var next int
	err := r.db.Model(&PolicyVersionHistory{}).
		Select("COALESCE(MAX(version_number), 0) + 1").
		Where("policy_id = ?", policyID).
		Scan(&next).Error
	if err != nil {
		return 0, err
	}
	return next, nil
}

func (r *policyVersionHistoryRepository) FindByPolicyIDPaginated(policyID int64, page, limit int) (*PaginationResult[PolicyVersionHistory], error) {
	query := r.db.Model(&PolicyVersionHistory{}).
		Where("policy_id = ?", policyID).
		Order("version_number DESC")
	return database.PaginateQuery[PolicyVersionHistory](query, page, limit)
}

func (r *policyVersionHistoryRepository) FindByPolicyIDAndVersion(policyID int64, versionNumber int) (*PolicyVersionHistory, error) {
	var entry PolicyVersionHistory
	err := r.db.Where("policy_id = ? AND version_number = ?", policyID, versionNumber).First(&entry).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &entry, nil
}
