package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/event"
	"github.com/maintainerd/auth/internal/platform/database"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type mockEventTypeRepo struct {
	findByUUIDFn func(uuid string) (*event.EventType, error)
}

func (m *mockEventTypeRepo) Create(entity *event.EventType) (*event.EventType, error) {
	return entity, nil
}
func (m *mockEventTypeRepo) CreateOrUpdate(entity *event.EventType) (*event.EventType, error) {
	return entity, nil
}
func (m *mockEventTypeRepo) FindAll(preloads ...string) ([]event.EventType, error) { return nil, nil }
func (m *mockEventTypeRepo) FindByUUID(uuid any, preloads ...string) (*event.EventType, error) {
	if m.findByUUIDFn != nil {
		s, _ := uuid.(string)
		return m.findByUUIDFn(s)
	}
	return nil, nil
}
func (m *mockEventTypeRepo) FindByUUIDs(uuids []string, preloads ...string) ([]event.EventType, error) {
	return nil, nil
}
func (m *mockEventTypeRepo) FindByID(id any, preloads ...string) (*event.EventType, error) {
	return nil, nil
}
func (m *mockEventTypeRepo) UpdateByUUID(uuid any, updatedData any) (*event.EventType, error) {
	return nil, nil
}
func (m *mockEventTypeRepo) UpdateByID(id any, updatedData any) (*event.EventType, error) {
	return nil, nil
}
func (m *mockEventTypeRepo) DeleteByUUID(uuid any) error { return nil }
func (m *mockEventTypeRepo) DeleteByID(id any) error     { return nil }
func (m *mockEventTypeRepo) Paginate(conditions map[string]any, page int, limit int, preloads ...string) (*database.PaginationResult[event.EventType], error) {
	return nil, nil
}
func (m *mockEventTypeRepo) FindAllActive() ([]event.EventType, error) { return nil, nil }
func (m *mockEventTypeRepo) FindActiveByTenantID(int64) ([]event.EventType, error) {
	return nil, nil
}
func (m *mockEventTypeRepo) FindByKey(key string) (*event.EventType, error) { return nil, nil }
func (m *mockEventTypeRepo) FindByKeyAndTenantID(key string, tenantID int64) (*event.EventType, error) {
	return nil, nil
}
func (m *mockEventTypeRepo) FindByKeys(keys []string) ([]event.EventType, error) { return nil, nil }
func (m *mockEventTypeRepo) FindByCategory(category string) ([]event.EventType, error) {
	return nil, nil
}
func (m *mockEventTypeRepo) WithTx(tx *gorm.DB) event.EventTypeRepository { return m }

func noopEventTypeRepo() *mockEventTypeRepo { return &mockEventTypeRepo{} }

func TestSubscriptionHandler_AddSubscription(t *testing.T) {
	t.Run("no tenant in context", func(t *testing.T) {
		h := NewSubscriptionHandler(nil, nil, noopEventTypeRepo())
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/webhook-endpoints/any/subscriptions", nil)
		h.AddSubscription(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid endpoint UUID", func(t *testing.T) {
		h := NewSubscriptionHandler(nil, nil, noopEventTypeRepo())
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/webhook-endpoints/bad-uuid/subscriptions", nil)
		r = withTenant(r)
		r = withChiParam(r, "webhook_endpoint_uuid", "not-a-uuid")
		h.AddSubscription(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("endpoint not found", func(t *testing.T) {
		repo := &mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) {
				return nil, nil
			},
		}
		epUUID := uuid.New()
		etUUID := uuid.New()
		h := NewSubscriptionHandler(&mockWebhookEndpointEventRepo{}, repo, noopEventTypeRepo())
		w := httptest.NewRecorder()
		body := subscriptionRequestDTO{EventTypeUUID: etUUID.String()}
		raw, _ := json.Marshal(body)
		r := httptest.NewRequest("POST", "/webhook-endpoints/"+epUUID.String()+"/subscriptions", bytes.NewReader(raw))
		r = withTenant(r)
		r = withChiParam(r, "webhook_endpoint_uuid", epUUID.String())
		h.AddSubscription(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("event type not found", func(t *testing.T) {
		ep := &WebhookEndpoint{WebhookEndpointID: 1, WebhookEndpointUUID: uuid.New(), TenantID: 1}
		repo := &mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) {
				return ep, nil
			},
		}
		etUUID := uuid.New()
		etRepo := &mockEventTypeRepo{
			findByUUIDFn: func(_ string) (*event.EventType, error) { return nil, nil },
		}
		h := NewSubscriptionHandler(&mockWebhookEndpointEventRepo{}, repo, etRepo)
		w := httptest.NewRecorder()
		body := subscriptionRequestDTO{EventTypeUUID: etUUID.String()}
		raw, _ := json.Marshal(body)
		r := httptest.NewRequest("POST", "/webhook-endpoints/"+ep.WebhookEndpointUUID.String()+"/subscriptions", bytes.NewReader(raw))
		r = withTenant(r)
		r = withChiParam(r, "webhook_endpoint_uuid", ep.WebhookEndpointUUID.String())
		h.AddSubscription(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		etUUID := uuid.New()
		ep := &WebhookEndpoint{WebhookEndpointID: 1, WebhookEndpointUUID: uuid.New(), TenantID: 1}
		repo := &mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) {
				return ep, nil
			},
		}
		etRepo := &mockEventTypeRepo{
			findByUUIDFn: func(_ string) (*event.EventType, error) {
				return &event.EventType{EventTypeID: 5, EventTypeUUID: etUUID, Key: "user.created", TenantID: 1}, nil
			},
		}
		h := NewSubscriptionHandler(&mockWebhookEndpointEventRepo{}, repo, etRepo)
		w := httptest.NewRecorder()
		body := subscriptionRequestDTO{EventTypeUUID: etUUID.String()}
		raw, _ := json.Marshal(body)
		r := httptest.NewRequest("POST", "/webhook-endpoints/"+ep.WebhookEndpointUUID.String()+"/subscriptions", bytes.NewReader(raw))
		r = withTenant(r)
		r = withChiParam(r, "webhook_endpoint_uuid", ep.WebhookEndpointUUID.String())
		h.AddSubscription(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestSubscriptionHandler_RemoveSubscription(t *testing.T) {
	t.Run("no tenant", func(t *testing.T) {
		h := NewSubscriptionHandler(nil, nil, noopEventTypeRepo())
		w := httptest.NewRecorder()
		r := httptest.NewRequest("DELETE", "/webhook-endpoints/any/subscriptions", nil)
		h.RemoveSubscription(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("endpoint not found", func(t *testing.T) {
		repo := &mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) {
				return nil, nil
			},
		}
		epUUID := uuid.New()
		etUUID := uuid.New()
		h := NewSubscriptionHandler(&mockWebhookEndpointEventRepo{}, repo, noopEventTypeRepo())
		w := httptest.NewRecorder()
		body := subscriptionRequestDTO{EventTypeUUID: etUUID.String()}
		raw, _ := json.Marshal(body)
		r := httptest.NewRequest("DELETE", "/webhook-endpoints/"+epUUID.String()+"/subscriptions", bytes.NewReader(raw))
		r = withTenant(r)
		r = withChiParam(r, "webhook_endpoint_uuid", epUUID.String())
		h.RemoveSubscription(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("event type not found", func(t *testing.T) {
		ep := &WebhookEndpoint{WebhookEndpointID: 1, WebhookEndpointUUID: uuid.New(), TenantID: 1}
		repo := &mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) {
				return ep, nil
			},
		}
		etUUID := uuid.New()
		etRepo := &mockEventTypeRepo{
			findByUUIDFn: func(_ string) (*event.EventType, error) { return nil, nil },
		}
		h := NewSubscriptionHandler(&mockWebhookEndpointEventRepo{}, repo, etRepo)
		w := httptest.NewRecorder()
		body := subscriptionRequestDTO{EventTypeUUID: etUUID.String()}
		raw, _ := json.Marshal(body)
		r := httptest.NewRequest("DELETE", "/webhook-endpoints/"+ep.WebhookEndpointUUID.String()+"/subscriptions", bytes.NewReader(raw))
		r = withTenant(r)
		r = withChiParam(r, "webhook_endpoint_uuid", ep.WebhookEndpointUUID.String())
		h.RemoveSubscription(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestReplayHandler_ReplayDelivery(t *testing.T) {
	t.Run("no tenant", func(t *testing.T) {
		h := NewReplayHandler(nil, nil, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/webhook-replay", nil)
		h.ReplayDelivery(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid event_id", func(t *testing.T) {
		h := NewReplayHandler(nil, nil, nil)
		w := httptest.NewRecorder()
		body := replayRequestDTO{EventID: "bad"}
		raw, _ := json.Marshal(body)
		r := httptest.NewRequest("POST", "/webhook-replay", bytes.NewReader(raw))
		r = withTenant(r)
		h.ReplayDelivery(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("replay to all endpoints", func(t *testing.T) {
		repo := &mockWebhookEndpointRepo{
			findActiveByTenantIDFn: func(_ int64) ([]WebhookEndpoint, error) {
				return []WebhookEndpoint{}, nil
			},
		}
		h := NewReplayHandler(nil, repo, func(ctx context.Context, ep WebhookEndpoint, eventID uuid.UUID, isReplay bool) error {
			return nil
		})
		w := httptest.NewRecorder()
		body := replayRequestDTO{EventID: uuid.New().String()}
		raw, _ := json.Marshal(body)
		r := httptest.NewRequest("POST", "/webhook-replay", bytes.NewReader(raw))
		r = withTenant(r)
		h.ReplayDelivery(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRateLimitAndCapMiddleware(t *testing.T) {
	t.Run("no tenant passes through", func(t *testing.T) {
		mw := RateLimitAndCapMiddleware(nil)
		called := false
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/", nil))
		assert.True(t, called)
	})

	t.Run("under limit passes through", func(t *testing.T) {
		repo := &mockWebhookEndpointRepo{
			countByTenantIDFn: func(_ int64) (int64, error) {
				return 10, nil
			},
		}
		mw := RateLimitAndCapMiddleware(repo)
		called := false
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		r := httptest.NewRequest("POST", "/", nil)
		r = withTenant(r)
		handler.ServeHTTP(httptest.NewRecorder(), r)
		assert.True(t, called)
	})

	t.Run("at capacity returns 429", func(t *testing.T) {
		repo := &mockWebhookEndpointRepo{
			countByTenantIDFn: func(_ int64) (int64, error) {
				return 50, nil
			},
		}
		mw := RateLimitAndCapMiddleware(repo)
		called := false
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		r := httptest.NewRequest("POST", "/", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		assert.False(t, called)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})
}
