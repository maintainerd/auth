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
}

func (m *mockWebhookEndpointRepo) WithTx(_ *gorm.DB) WebhookEndpointRepository { return m }

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
