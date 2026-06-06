package webhook

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mockBaseRepo[T any] struct{}

func (m *mockBaseRepo[T]) Create(e *T) (*T, error)                            { return e, nil }
func (m *mockBaseRepo[T]) CreateOrUpdate(e *T) (*T, error)                    { return e, nil }
func (m *mockBaseRepo[T]) FindAll(preloads ...string) ([]T, error)            { return nil, nil }
func (m *mockBaseRepo[T]) FindByUUID(id any, p ...string) (*T, error)         { return nil, nil }
func (m *mockBaseRepo[T]) FindByUUIDs(ids []string, p ...string) ([]T, error) { return nil, nil }
func (m *mockBaseRepo[T]) FindByID(id any, p ...string) (*T, error)           { return nil, nil }
func (m *mockBaseRepo[T]) UpdateByUUID(id, data any) (*T, error)              { return nil, nil }
func (m *mockBaseRepo[T]) UpdateByID(id, data any) (*T, error)                { return nil, nil }
func (m *mockBaseRepo[T]) DeleteByUUID(id any) error                          { return nil }
func (m *mockBaseRepo[T]) DeleteByID(id any) error                            { return nil }
func (m *mockBaseRepo[T]) Paginate(c map[string]any, page, limit int, p ...string) (*PaginationResult[T], error) {
	return nil, nil
}

type mockWebhookEndpointRepo struct {
	mockBaseRepo[WebhookEndpoint]
	findByTenantIDFn       func(int64) ([]WebhookEndpoint, error)
	findActiveByTenantIDFn func(int64) ([]WebhookEndpoint, error)
	findByUUIDAndTenantFn  func(uuid.UUID, int64) (*WebhookEndpoint, error)
	findPaginatedFn        func(WebhookEndpointRepositoryGetFilter) (*PaginationResult[WebhookEndpoint], error)
	createFn               func(*WebhookEndpoint) (*WebhookEndpoint, error)
	updateByUUIDFn         func(any, any) (*WebhookEndpoint, error)
	deleteByUUIDFn         func(any) error
	updateLastTriggeredFn  func(int64, time.Time) error
	countByTenantIDFn      func(int64) (int64, error)
	incrementFailuresFn    func(int64) (int, error)
	resetFailuresFn        func(int64) error
	quarantineFn           func(int64) error
	withTxFn               func(*gorm.DB) WebhookEndpointRepository
}

func (m *mockWebhookEndpointRepo) WithTx(tx *gorm.DB) WebhookEndpointRepository {
	if m.withTxFn != nil {
		return m.withTxFn(tx)
	}
	return m
}

func (m *mockWebhookEndpointRepo) FindByTenantID(tenantID int64) ([]WebhookEndpoint, error) {
	if m.findByTenantIDFn != nil {
		return m.findByTenantIDFn(tenantID)
	}
	return nil, nil
}

func (m *mockWebhookEndpointRepo) FindActiveByTenantID(tenantID int64) ([]WebhookEndpoint, error) {
	if m.findActiveByTenantIDFn != nil {
		return m.findActiveByTenantIDFn(tenantID)
	}
	return nil, nil
}

func (m *mockWebhookEndpointRepo) FindByUUIDAndTenantID(webhookEndpointUUID uuid.UUID, tenantID int64) (*WebhookEndpoint, error) {
	if m.findByUUIDAndTenantFn != nil {
		return m.findByUUIDAndTenantFn(webhookEndpointUUID, tenantID)
	}
	return nil, nil
}

func (m *mockWebhookEndpointRepo) FindPaginated(filter WebhookEndpointRepositoryGetFilter) (*PaginationResult[WebhookEndpoint], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(filter)
	}
	return &PaginationResult[WebhookEndpoint]{}, nil
}

func (m *mockWebhookEndpointRepo) Create(e *WebhookEndpoint) (*WebhookEndpoint, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}

func (m *mockWebhookEndpointRepo) UpdateByUUID(id, data any) (*WebhookEndpoint, error) {
	if m.updateByUUIDFn != nil {
		return m.updateByUUIDFn(id, data)
	}
	if endpoint, ok := data.(*WebhookEndpoint); ok {
		return endpoint, nil
	}
	return nil, nil
}

func (m *mockWebhookEndpointRepo) DeleteByUUID(id any) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}

func (m *mockWebhookEndpointRepo) UpdateLastTriggeredAt(webhookEndpointID int64, t time.Time) error {
	if m.updateLastTriggeredFn != nil {
		return m.updateLastTriggeredFn(webhookEndpointID, t)
	}
	return nil
}

func (m *mockWebhookEndpointRepo) CountByTenantID(tenantID int64) (int64, error) {
	if m.countByTenantIDFn != nil {
		return m.countByTenantIDFn(tenantID)
	}
	return 0, nil
}

func (m *mockWebhookEndpointRepo) IncrementConsecutiveFailures(webhookEndpointID int64) (int, error) {
	if m.incrementFailuresFn != nil {
		return m.incrementFailuresFn(webhookEndpointID)
	}
	return 0, nil
}

func (m *mockWebhookEndpointRepo) ResetConsecutiveFailures(webhookEndpointID int64) error {
	if m.resetFailuresFn != nil {
		return m.resetFailuresFn(webhookEndpointID)
	}
	return nil
}

func (m *mockWebhookEndpointRepo) Quarantine(webhookEndpointID int64) error {
	if m.quarantineFn != nil {
		return m.quarantineFn(webhookEndpointID)
	}
	return nil
}

// mockWebhookEndpointEventRepo is a test mock for WebhookEndpointEventRepository.
type mockWebhookEndpointEventRepo struct {
	mockBaseRepo[WebhookEndpointEvent]
	findByEndpointIDFn       func(int64) ([]WebhookEndpointEvent, error)
	existsByEndpointAndKeyFn func(int64, string) (bool, error)
	findByEndpointAndEvtFn   func(int64, int64) (*WebhookEndpointEvent, error)
	deleteByIdAndEvtFn     func(int64, int64) error
	deleteByEndpointIDFn   func(int64) error
	bulkCreateFn           func([]WebhookEndpointEvent) error
	createFn               func(*WebhookEndpointEvent) (*WebhookEndpointEvent, error)
	withTxFn               func(*gorm.DB) WebhookEndpointEventRepository
}

func (m *mockWebhookEndpointEventRepo) WithTx(tx *gorm.DB) WebhookEndpointEventRepository {
	if m.withTxFn != nil { return m.withTxFn(tx) }
	return m
}

func (m *mockWebhookEndpointEventRepo) FindByEndpointID(id int64) ([]WebhookEndpointEvent, error) {
	if m.findByEndpointIDFn != nil { return m.findByEndpointIDFn(id) }
	return nil, nil
}

func (m *mockWebhookEndpointEventRepo) ExistsByEndpointAndEventKey(epID int64, eventTypeKey string) (bool, error) {
	if m.existsByEndpointAndKeyFn != nil { return m.existsByEndpointAndKeyFn(epID, eventTypeKey) }
	return false, nil
}

func (m *mockWebhookEndpointEventRepo) FindByEndpointIDAndEventTypeID(epID, etID int64) (*WebhookEndpointEvent, error) {
	if m.findByEndpointAndEvtFn != nil { return m.findByEndpointAndEvtFn(epID, etID) }
	return nil, nil
}

func (m *mockWebhookEndpointEventRepo) DeleteByEndpointIDAndEventTypeID(epID, etID int64) error {
	if m.deleteByIdAndEvtFn != nil { return m.deleteByIdAndEvtFn(epID, etID) }
	return nil
}

func (m *mockWebhookEndpointEventRepo) DeleteByEndpointID(id int64) error {
	if m.deleteByEndpointIDFn != nil { return m.deleteByEndpointIDFn(id) }
	return nil
}

func (m *mockWebhookEndpointEventRepo) BulkCreate(entries []WebhookEndpointEvent) error {
	if m.bulkCreateFn != nil { return m.bulkCreateFn(entries) }
	return nil
}

func (m *mockWebhookEndpointEventRepo) Create(e *WebhookEndpointEvent) (*WebhookEndpointEvent, error) {
	if m.createFn != nil { return m.createFn(e) }
	return e, nil
}
