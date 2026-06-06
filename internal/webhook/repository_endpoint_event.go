package webhook

import (
	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

// WebhookEndpointEventRepository defines persistence for webhook_endpoint_events M:N.
type WebhookEndpointEventRepository interface {
	BaseRepositoryMethods[WebhookEndpointEvent]
	FindByEndpointID(webhookEndpointID int64) ([]WebhookEndpointEvent, error)
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
