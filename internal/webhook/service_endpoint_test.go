package webhook

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newWebhookEndpointSvc(repo *mockWebhookEndpointRepo) WebhookEndpointService {
	return NewWebhookEndpointService(repo)
}

func newWebhookEndpoint(tenantID int64) *WebhookEndpoint {
	now := time.Now()
	return &WebhookEndpoint{
		WebhookEndpointID:   1,
		WebhookEndpointUUID: uuid.New(),
		TenantID:            tenantID,
		URL:                 "https://example.com/webhook",
		SecretEncrypted:     "sec123",
		SubscribeAll:        true,
		MaxRetries:          3,
		TimeoutSeconds:      30,
		Status:              shared.StatusActive,
		Description:         "test",
		LastTriggeredAt:     &now,
	}
}

// ---------------------------------------------------------------------------
// GetAll
// ---------------------------------------------------------------------------

func TestWebhookEndpointService_GetAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ep := newWebhookEndpoint(1)
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findPaginatedFn: func(f WebhookEndpointRepositoryGetFilter) (*PaginationResult[WebhookEndpoint], error) {
				return &PaginationResult[WebhookEndpoint]{
					Data:       []WebhookEndpoint{*ep},
					Total:      1,
					Page:       f.Page,
					Limit:      f.Limit,
					TotalPages: 1,
				}, nil
			},
		})
		res, err := svc.GetAll(context.Background(), 1, nil, 1, 10, "created_at", "desc")
		require.NoError(t, err)
		assert.Len(t, res.Data, 1)
		assert.Equal(t, int64(1), res.Total)
	})

	t.Run("empty result", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findPaginatedFn: func(_ WebhookEndpointRepositoryGetFilter) (*PaginationResult[WebhookEndpoint], error) {
				return &PaginationResult[WebhookEndpoint]{
					Data:       []WebhookEndpoint{},
					Total:      0,
					Page:       1,
					Limit:      10,
					TotalPages: 0,
				}, nil
			},
		})
		res, err := svc.GetAll(context.Background(), 1, nil, 1, 10, "created_at", "desc")
		require.NoError(t, err)
		assert.Empty(t, res.Data)
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findPaginatedFn: func(_ WebhookEndpointRepositoryGetFilter) (*PaginationResult[WebhookEndpoint], error) {
				return nil, errors.New("db")
			},
		})
		_, err := svc.GetAll(context.Background(), 1, nil, 1, 10, "", "")
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// GetByUUID
// ---------------------------------------------------------------------------

func TestWebhookEndpointService_GetByUUID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ep := newWebhookEndpoint(1)
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) { return ep, nil },
		})
		res, err := svc.GetByUUID(context.Background(), 1, ep.WebhookEndpointUUID)
		require.NoError(t, err)
		assert.Equal(t, ep.WebhookEndpointUUID, res.WebhookEndpointUUID)
	})

	t.Run("not found", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) { return nil, nil },
		})
		_, err := svc.GetByUUID(context.Background(), 1, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) {
				return nil, errors.New("db")
			},
		})
		_, err := svc.GetByUUID(context.Background(), 1, uuid.New())
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestWebhookEndpointService_Create(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	t.Run("success with defaults", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			createFn: func(e *WebhookEndpoint) (*WebhookEndpoint, error) {
				e.WebhookEndpointUUID = uuid.New()
				return e, nil
			},
		})
		res, err := svc.Create(context.Background(), 1,
			"https://example.com/hook",
			true, nil, nil, "desc", shared.StatusActive,
		)
		require.NoError(t, err)
		assert.Equal(t, "https://example.com/hook", res.URL)
		assert.Equal(t, 3, res.MaxRetries)
		assert.Equal(t, 30, res.TimeoutSeconds)
	})

	t.Run("success with custom retries and timeout", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			createFn: func(e *WebhookEndpoint) (*WebhookEndpoint, error) {
				e.WebhookEndpointUUID = uuid.New()
				return e, nil
			},
		})
		res, err := svc.Create(context.Background(), 1,
			"https://example.com/hook",
			true, intPtr(5), intPtr(60), "desc", shared.StatusActive,
		)
		require.NoError(t, err)
		assert.Equal(t, 5, res.MaxRetries)
		assert.Equal(t, 60, res.TimeoutSeconds)
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			createFn: func(_ *WebhookEndpoint) (*WebhookEndpoint, error) {
				return nil, errors.New("db")
			},
		})
		_, err := svc.Create(context.Background(), 1,
			"https://example.com/hook",
			true, nil, nil, "", shared.StatusActive,
		)
		require.Error(t, err)
	})

	t.Run("encrypt secret error", func(t *testing.T) {
		original := crypto.EncryptAtRest
		crypto.EncryptAtRest = func(string) (string, error) {
			return "", assert.AnError
		}
		t.Cleanup(func() {
			crypto.EncryptAtRest = original
		})
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			createFn: func(_ *WebhookEndpoint) (*WebhookEndpoint, error) {
				t.Fatal("repo should not be called when encryption fails")
				return nil, nil
			},
		})

		_, err := svc.Create(context.Background(), 1,
			"https://example.com/hook",
			true, nil, nil, "", shared.StatusActive,
		)

		require.ErrorIs(t, err, assert.AnError)
	})
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestWebhookEndpointService_Update(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	t.Run("success", func(t *testing.T) {
		ep := newWebhookEndpoint(1)
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) { return ep, nil },
			updateByUUIDFn: func(_ any, data any) (*WebhookEndpoint, error) {
				return data.(*WebhookEndpoint), nil
			},
		})
		res, err := svc.Update(context.Background(), 1, ep.WebhookEndpointUUID,
			"https://new.example.com/hook", true,
		true, intPtr(5), intPtr(60), "updated", shared.StatusActive,
		)
		require.NoError(t, err)
		assert.Equal(t, "https://new.example.com/hook", res.URL)
	})

	t.Run("secret preserved on blank", func(t *testing.T) {
		ep := newWebhookEndpoint(1)
		original := ep.SecretEncrypted
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) { return ep, nil },
			updateByUUIDFn: func(_ any, data any) (*WebhookEndpoint, error) {
				return data.(*WebhookEndpoint), nil
			},
		})
		_, err := svc.Update(context.Background(), 1, ep.WebhookEndpointUUID,
			"https://example.com/hook", false,
			true, nil, nil, "", shared.StatusActive,
		)
		require.NoError(t, err)
		assert.Equal(t, original, ep.SecretEncrypted)
	})

	t.Run("not found", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) { return nil, nil },
		})
		_, err := svc.Update(context.Background(), 1, uuid.New(),
			"https://example.com", false,
			true, nil, nil, "", shared.StatusActive,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("find repo error", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) {
				return nil, errors.New("db")
			},
		})
		_, err := svc.Update(context.Background(), 1, uuid.New(),
			"https://example.com", false,
			true, nil, nil, "", shared.StatusActive,
		)
		require.Error(t, err)
	})

	t.Run("UpdateByUUID error", func(t *testing.T) {
		ep := newWebhookEndpoint(1)
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) { return ep, nil },
			updateByUUIDFn: func(_ any, _ any) (*WebhookEndpoint, error) {
				return nil, errors.New("save err")
			},
		})
		_, err := svc.Update(context.Background(), 1, ep.WebhookEndpointUUID,
			"https://example.com", false,
			true, nil, nil, "", shared.StatusActive,
		)
		require.Error(t, err)
	})

	t.Run("encrypt secret error", func(t *testing.T) {
		original := crypto.EncryptAtRest
		crypto.EncryptAtRest = func(string) (string, error) {
			return "", assert.AnError
		}
		t.Cleanup(func() {
			crypto.EncryptAtRest = original
		})
		ep := newWebhookEndpoint(1)
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) { return ep, nil },
			updateByUUIDFn: func(_ any, _ any) (*WebhookEndpoint, error) {
				t.Fatal("repo update should not be called when encryption fails")
				return nil, nil
			},
		})

		_, err := svc.Update(context.Background(), 1, ep.WebhookEndpointUUID,
			"https://example.com", true,
			true, nil, nil, "", shared.StatusActive,
		)

		require.ErrorIs(t, err, assert.AnError)
	})
}

// ---------------------------------------------------------------------------
// UpdateStatus
// ---------------------------------------------------------------------------

func TestWebhookEndpointService_UpdateStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ep := newWebhookEndpoint(1)
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) { return ep, nil },
			updateByUUIDFn: func(_ any, data any) (*WebhookEndpoint, error) {
				return data.(*WebhookEndpoint), nil
			},
		})
		res, err := svc.UpdateStatus(context.Background(), 1, ep.WebhookEndpointUUID, "inactive")
		require.NoError(t, err)
		assert.Equal(t, "inactive", res.Status)
	})

	t.Run("not found", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) { return nil, nil },
		})
		_, err := svc.UpdateStatus(context.Background(), 1, uuid.New(), "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("find repo error", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) {
				return nil, errors.New("db")
			},
		})
		_, err := svc.UpdateStatus(context.Background(), 1, uuid.New(), "inactive")
		require.Error(t, err)
	})

	t.Run("UpdateByUUID error", func(t *testing.T) {
		ep := newWebhookEndpoint(1)
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) { return ep, nil },
			updateByUUIDFn: func(_ any, _ any) (*WebhookEndpoint, error) {
				return nil, errors.New("save err")
			},
		})
		_, err := svc.UpdateStatus(context.Background(), 1, ep.WebhookEndpointUUID, "inactive")
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestWebhookEndpointService_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ep := newWebhookEndpoint(1)
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) { return ep, nil },
			deleteByUUIDFn:        func(_ any) error { return nil },
		})
		res, err := svc.Delete(context.Background(), 1, ep.WebhookEndpointUUID)
		require.NoError(t, err)
		assert.Equal(t, ep.WebhookEndpointUUID, res.WebhookEndpointUUID)
	})

	t.Run("not found", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) { return nil, nil },
		})
		_, err := svc.Delete(context.Background(), 1, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("find repo error", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) {
				return nil, errors.New("db")
			},
		})
		_, err := svc.Delete(context.Background(), 1, uuid.New())
		require.Error(t, err)
	})

	t.Run("DeleteByUUID error", func(t *testing.T) {
		ep := newWebhookEndpoint(1)
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*WebhookEndpoint, error) { return ep, nil },
			deleteByUUIDFn:        func(_ any) error { return errors.New("delete err") },
		})
		_, err := svc.Delete(context.Background(), 1, ep.WebhookEndpointUUID)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// toWebhookEndpointServiceDataResult — edge cases
// ---------------------------------------------------------------------------

func TestToWebhookEndpointServiceDataResult(t *testing.T) {
	t.Run("subscribe_all false", func(t *testing.T) {
		ep := &WebhookEndpoint{WebhookEndpointUUID: uuid.New(), TenantID: 1, SubscribeAll: false}
		result := toWebhookEndpointServiceDataResult(ep)
		assert.False(t, result.SubscribeAll)
	})

	t.Run("subscribe_all true", func(t *testing.T) {
		ep := &WebhookEndpoint{WebhookEndpointUUID: uuid.New(), TenantID: 1, SubscribeAll: true}
		result := toWebhookEndpointServiceDataResult(ep)
		assert.True(t, result.SubscribeAll)
	})
}
