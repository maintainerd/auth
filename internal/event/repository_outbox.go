package event

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

// OutboxRepository defines persistence operations for integration_event_outbox.
type OutboxRepository interface {
	BaseRepositoryMethods[Outbox]
	FindUnpublished(batchSize int) ([]Outbox, error)
	ClaimUnpublished(batchSize int) ([]Outbox, error)
	FindByTenantID(tenantID int64) ([]Outbox, error)
	FindByEventID(eventID uuid.UUID) (*Outbox, error)
	MarkPublished(outboxID int64) error
	DeleteOlderThan(cutoff time.Time) (int64, error)
	DeleteBySubjectUUID(subjectUUID uuid.UUID) (int64, error)
	WithTx(tx *gorm.DB) OutboxRepository
}

type outboxRepository struct {
	*BaseRepository[Outbox]
}

func NewOutboxRepository(db *gorm.DB) OutboxRepository {
	return &outboxRepository{
		BaseRepository: database.NewBaseRepository[Outbox](db, "outbox_uuid", "outbox_id"),
	}
}

func (r *outboxRepository) WithTx(tx *gorm.DB) OutboxRepository {
	return &outboxRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *outboxRepository) FindUnpublished(batchSize int) ([]Outbox, error) {
	var rows []Outbox
	err := r.DB().Where("is_published = ?", false).
		Order("created_at ASC").
		Limit(batchSize).
		Find(&rows).Error
	return rows, err
}

// claimVisibilityTimeout is how long a claimed-but-unpublished row stays
// invisible to other relay workers before it can be re-claimed.
const claimVisibilityTimeout = "5 minutes"

// ClaimUnpublished atomically claims a batch of unpublished outbox rows using
// FOR UPDATE SKIP LOCKED so that concurrent relay workers (multiple replicas)
// never process the same row. A claimed row is hidden for claimVisibilityTimeout;
// if delivery hand-off fails and the row is never marked published, the claim
// expires and the row becomes re-claimable.
func (r *outboxRepository) ClaimUnpublished(batchSize int) ([]Outbox, error) {
	var rows []Outbox
	err := r.DB().Raw(`
		UPDATE integration_event_outbox SET claimed_at = now()
		WHERE outbox_id IN (
			SELECT outbox_id FROM integration_event_outbox
			WHERE is_published = false
			  AND (claimed_at IS NULL OR claimed_at < now() - interval '`+claimVisibilityTimeout+`')
			ORDER BY created_at ASC
			LIMIT ?
			FOR UPDATE SKIP LOCKED
		)
		RETURNING *
	`, batchSize).Scan(&rows).Error
	return rows, err
}

func (r *outboxRepository) FindByTenantID(tenantID int64) ([]Outbox, error) {
	var rows []Outbox
	err := r.DB().Where("tenant_id = ?", tenantID).
		Order("created_at ASC").
		Find(&rows).Error
	return rows, err
}

func (r *outboxRepository) FindByEventID(eventID uuid.UUID) (*Outbox, error) {
	var row Outbox
	err := r.DB().Where("event_id = ?", eventID).First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *outboxRepository) MarkPublished(outboxID int64) error {
	now := time.Now().UTC()
	return r.DB().Model(&Outbox{}).
		Where("outbox_id = ?", outboxID).
		Updates(map[string]any{
			"is_published": true,
			"published_at": now,
		}).Error
}

func (r *outboxRepository) DeleteOlderThan(cutoff time.Time) (int64, error) {
	result := r.DB().Where("created_at < ? AND is_published = ?", cutoff, true).Delete(&Outbox{})
	return result.RowsAffected, result.Error
}

func (r *outboxRepository) DeleteBySubjectUUID(subjectUUID uuid.UUID) (int64, error) {
	result := r.DB().Where("subject_uuid = ?", subjectUUID).Delete(&Outbox{})
	return result.RowsAffected, result.Error
}
