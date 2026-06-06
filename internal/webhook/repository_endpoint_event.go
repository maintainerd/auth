package webhook

import (
	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

// WebhookEndpointEventRepository defines persistence for webhook_endpoint_events M:N.
type WebhookEndpointEventRepository interface {
	BaseRepositoryMethods[WebhookEndpointEvent]
	FindByEndpointID(webhookEndpointID int64) ([]WebhookEndpointEvent, error)
	// ExistsByEndpointAndEventKey reports whether the endpoint subscribes to the
	// exact event type (resolved via the event_types catalog by canonical key).
	ExistsByEndpointAndEventKey(webhookEndpointID int64, eventTypeKey string) (bool, error)
	FindByEndpointIDAndEventTypeID(webhookEndpointID, eventTypeID int64) (*WebhookEndpointEvent, error)
	DeleteByEndpointIDAndEventTypeID(webhookEndpointID, eventTypeID int64) error
	DeleteByEndpointID(webhookEndpointID int64) error
	BulkCreate(entries []WebhookEndpointEvent) error
	WithTx(tx *gorm.DB) WebhookEndpointEventRepository
}

type webhookEndpointEventRepository struct {
	*BaseRepository[WebhookEndpointEvent]
}

func NewWebhookEndpointEventRepository(db *gorm.DB) WebhookEndpointEventRepository {
	return &webhookEndpointEventRepository{
		BaseRepository: database.NewBaseRepository[WebhookEndpointEvent](db, "", "webhook_endpoint_event_id"),
	}
}

func (r *webhookEndpointEventRepository) WithTx(tx *gorm.DB) WebhookEndpointEventRepository {
	return &webhookEndpointEventRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *webhookEndpointEventRepository) FindByEndpointID(webhookEndpointID int64) ([]WebhookEndpointEvent, error) {
	var subs []WebhookEndpointEvent
	err := r.DB().Where("webhook_endpoint_id = ?", webhookEndpointID).Find(&subs).Error
	return subs, err
}

func (r *webhookEndpointEventRepository) ExistsByEndpointAndEventKey(webhookEndpointID int64, eventTypeKey string) (bool, error) {
	var count int64
	err := r.DB().
		Table("webhook_endpoint_events AS wee").
		Joins("JOIN event_types et ON et.event_type_id = wee.event_type_id").
		Where("wee.webhook_endpoint_id = ? AND et.key = ?", webhookEndpointID, eventTypeKey).
		Count(&count).Error
	return count > 0, err
}

func (r *webhookEndpointEventRepository) FindByEndpointIDAndEventTypeID(webhookEndpointID, eventTypeID int64) (*WebhookEndpointEvent, error) {
	var sub WebhookEndpointEvent
	err := r.DB().Where("webhook_endpoint_id = ? AND event_type_id = ?", webhookEndpointID, eventTypeID).First(&sub).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *webhookEndpointEventRepository) DeleteByEndpointIDAndEventTypeID(webhookEndpointID, eventTypeID int64) error {
	return r.DB().Where("webhook_endpoint_id = ? AND event_type_id = ?", webhookEndpointID, eventTypeID).Delete(&WebhookEndpointEvent{}).Error
}

func (r *webhookEndpointEventRepository) DeleteByEndpointID(webhookEndpointID int64) error {
	return r.DB().Where("webhook_endpoint_id = ?", webhookEndpointID).Delete(&WebhookEndpointEvent{}).Error
}

func (r *webhookEndpointEventRepository) BulkCreate(entries []WebhookEndpointEvent) error {
	if len(entries) == 0 {
		return nil
	}
	return r.DB().Create(&entries).Error
}
