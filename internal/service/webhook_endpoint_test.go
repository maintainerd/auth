package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func newWebhookEndpointSvc(repo *mockWebhookEndpointRepo) WebhookEndpointService {
	return NewWebhookEndpointService(repo)
}

func newWebhookEndpoint(tenantID int64) *model.WebhookEndpoint {
	evts, _ := json.Marshal([]string{"user.created", "user.deleted"})
	now := time.Now()
	return &model.WebhookEndpoint{
		WebhookEndpointID:   1,
		WebhookEndpointUUID: uuid.New(),
		TenantID:            tenantID,
		URL:                 "https://example.com/webhook",
		SecretEncrypted:     "sec123",
		Events:              datatypes.JSON(evts),
		MaxRetries:          3,
		TimeoutSeconds:      30,
		Status:              model.StatusActive,
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
			findPaginatedFn: func(f repository.WebhookEndpointRepositoryGetFilter) (*repository.PaginationResult[model.WebhookEndpoint], error) {
				return &repository.PaginationResult[model.WebhookEndpoint]{
					Data:       []model.WebhookEndpoint{*ep},
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
			findPaginatedFn: func(_ repository.WebhookEndpointRepositoryGetFilter) (*repository.PaginationResult[model.WebhookEndpoint], error) {
				return &repository.PaginationResult[model.WebhookEndpoint]{
					Data:       []model.WebhookEndpoint{},
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
			findPaginatedFn: func(_ repository.WebhookEndpointRepositoryGetFilter) (*repository.PaginationResult[model.WebhookEndpoint], error) {
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
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*model.WebhookEndpoint, error) { return ep, nil },
		})
		res, err := svc.GetByUUID(context.Background(), 1, ep.WebhookEndpointUUID)
		require.NoError(t, err)
		assert.Equal(t, ep.WebhookEndpointUUID, res.WebhookEndpointUUID)
	})

	t.Run("not found", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*model.WebhookEndpoint, error) { return nil, nil },
		})
		_, err := svc.GetByUUID(context.Background(), 1, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*model.WebhookEndpoint, error) {
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
			createFn: func(e *model.WebhookEndpoint) (*model.WebhookEndpoint, error) {
				e.WebhookEndpointUUID = uuid.New()
				return e, nil
			},
		})
		res, err := svc.Create(context.Background(), 1,
			"https://example.com/hook", "secret",
			[]string{"user.created"}, nil, nil, "desc", model.StatusActive,
		)
		require.NoError(t, err)
		assert.Equal(t, "https://example.com/hook", res.URL)
		assert.Equal(t, 3, res.MaxRetries)
		assert.Equal(t, 30, res.TimeoutSeconds)
	})

	t.Run("success with custom retries and timeout", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			createFn: func(e *model.WebhookEndpoint) (*model.WebhookEndpoint, error) {
				e.WebhookEndpointUUID = uuid.New()
				return e, nil
			},
		})
		res, err := svc.Create(context.Background(), 1,
			"https://example.com/hook", "secret",
			[]string{"user.created"}, intPtr(5), intPtr(60), "desc", model.StatusActive,
		)
		require.NoError(t, err)
		assert.Equal(t, 5, res.MaxRetries)
		assert.Equal(t, 60, res.TimeoutSeconds)
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			createFn: func(_ *model.WebhookEndpoint) (*model.WebhookEndpoint, error) {
				return nil, errors.New("db")
			},
		})
		_, err := svc.Create(context.Background(), 1,
			"https://example.com/hook", "",
			[]string{"user.created"}, nil, nil, "", model.StatusActive,
		)
		require.Error(t, err)
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
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*model.WebhookEndpoint, error) { return ep, nil },
			updateByUUIDFn: func(_ any, data any) (*model.WebhookEndpoint, error) {
				return data.(*model.WebhookEndpoint), nil
			},
		})
		res, err := svc.Update(context.Background(), 1, ep.WebhookEndpointUUID,
			"https://new.example.com/hook", "new-secret",
			[]string{"user.updated"}, intPtr(5), intPtr(60), "updated", model.StatusActive,
		)
		require.NoError(t, err)
		assert.Equal(t, "https://new.example.com/hook", res.URL)
	})

	t.Run("secret preserved on blank", func(t *testing.T) {
		ep := newWebhookEndpoint(1)
		original := ep.SecretEncrypted
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*model.WebhookEndpoint, error) { return ep, nil },
			updateByUUIDFn: func(_ any, data any) (*model.WebhookEndpoint, error) {
				return data.(*model.WebhookEndpoint), nil
			},
		})
		_, err := svc.Update(context.Background(), 1, ep.WebhookEndpointUUID,
			"https://example.com/hook", "",
			[]string{"user.created"}, nil, nil, "", model.StatusActive,
		)
		require.NoError(t, err)
		assert.Equal(t, original, ep.SecretEncrypted)
	})

	t.Run("not found", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*model.WebhookEndpoint, error) { return nil, nil },
		})
		_, err := svc.Update(context.Background(), 1, uuid.New(),
			"https://example.com", "",
			[]string{"user.created"}, nil, nil, "", model.StatusActive,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("find repo error", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*model.WebhookEndpoint, error) {
				return nil, errors.New("db")
			},
		})
		_, err := svc.Update(context.Background(), 1, uuid.New(),
			"https://example.com", "",
			[]string{"user.created"}, nil, nil, "", model.StatusActive,
		)
		require.Error(t, err)
	})

	t.Run("UpdateByUUID error", func(t *testing.T) {
		ep := newWebhookEndpoint(1)
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*model.WebhookEndpoint, error) { return ep, nil },
			updateByUUIDFn: func(_ any, _ any) (*model.WebhookEndpoint, error) {
				return nil, errors.New("save err")
			},
		})
		_, err := svc.Update(context.Background(), 1, ep.WebhookEndpointUUID,
			"https://example.com", "",
			[]string{}, nil, nil, "", model.StatusActive,
		)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// UpdateStatus
// ---------------------------------------------------------------------------

func TestWebhookEndpointService_UpdateStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ep := newWebhookEndpoint(1)
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*model.WebhookEndpoint, error) { return ep, nil },
			updateByUUIDFn: func(_ any, data any) (*model.WebhookEndpoint, error) {
				return data.(*model.WebhookEndpoint), nil
			},
		})
		res, err := svc.UpdateStatus(context.Background(), 1, ep.WebhookEndpointUUID, "inactive")
		require.NoError(t, err)
		assert.Equal(t, "inactive", res.Status)
	})

	t.Run("not found", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*model.WebhookEndpoint, error) { return nil, nil },
		})
		_, err := svc.UpdateStatus(context.Background(), 1, uuid.New(), "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("find repo error", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*model.WebhookEndpoint, error) {
				return nil, errors.New("db")
			},
		})
		_, err := svc.UpdateStatus(context.Background(), 1, uuid.New(), "inactive")
		require.Error(t, err)
	})

	t.Run("UpdateByUUID error", func(t *testing.T) {
		ep := newWebhookEndpoint(1)
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*model.WebhookEndpoint, error) { return ep, nil },
			updateByUUIDFn: func(_ any, _ any) (*model.WebhookEndpoint, error) {
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
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*model.WebhookEndpoint, error) { return ep, nil },
			deleteByUUIDFn:        func(_ any) error { return nil },
		})
		res, err := svc.Delete(context.Background(), 1, ep.WebhookEndpointUUID)
		require.NoError(t, err)
		assert.Equal(t, ep.WebhookEndpointUUID, res.WebhookEndpointUUID)
	})

	t.Run("not found", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*model.WebhookEndpoint, error) { return nil, nil },
		})
		_, err := svc.Delete(context.Background(), 1, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("find repo error", func(t *testing.T) {
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*model.WebhookEndpoint, error) {
				return nil, errors.New("db")
			},
		})
		_, err := svc.Delete(context.Background(), 1, uuid.New())
		require.Error(t, err)
	})

	t.Run("DeleteByUUID error", func(t *testing.T) {
		ep := newWebhookEndpoint(1)
		svc := newWebhookEndpointSvc(&mockWebhookEndpointRepo{
			findByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) (*model.WebhookEndpoint, error) { return ep, nil },
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
	t.Run("nil events produces empty array", func(t *testing.T) {
		ep := &model.WebhookEndpoint{WebhookEndpointUUID: uuid.New(), TenantID: 1}
		result := toWebhookEndpointServiceDataResult(ep)
		assert.Equal(t, []any{}, result.Events)
	})

	t.Run("valid events are unmarshalled", func(t *testing.T) {
		evts, _ := json.Marshal([]string{"user.created"})
		ep := &model.WebhookEndpoint{
			WebhookEndpointUUID: uuid.New(),
			TenantID:            1,
			Events:              datatypes.JSON(evts),
		}
		result := toWebhookEndpointServiceDataResult(ep)
		arr, ok := result.Events.([]any)
		require.True(t, ok)
		assert.Len(t, arr, 1)
	})

	t.Run("invalid JSON events falls back to empty array", func(t *testing.T) {
		ep := &model.WebhookEndpoint{
			WebhookEndpointUUID: uuid.New(),
			TenantID:            1,
			Events:              datatypes.JSON([]byte(`not json`)),
		}
		result := toWebhookEndpointServiceDataResult(ep)
		assert.Equal(t, []any{}, result.Events)
	})
}
