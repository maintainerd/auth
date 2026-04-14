package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Mock: AuthEventRepository
// ---------------------------------------------------------------------------

type mockAuthEventRepo struct {
	createFn           func(e *model.AuthEvent) (*model.AuthEvent, error)
	findPaginatedFn    func(filter repository.AuthEventRepositoryGetFilter) (*repository.PaginationResult[model.AuthEvent], error)
	findByUUIDAndTIDFn func(uuid string, tenantID int64) (*model.AuthEvent, error)
	findByDateRangeFn  func(tenantID int64, from, to time.Time) ([]model.AuthEvent, error)
	deleteOlderThanFn  func(cutoff time.Time) (int64, error)
	countByEventTypeFn func(eventType string, tenantID int64) (int64, error)
}

func (m *mockAuthEventRepo) WithTx(_ *gorm.DB) repository.AuthEventRepository { return m }
func (m *mockAuthEventRepo) Create(e *model.AuthEvent) (*model.AuthEvent, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockAuthEventRepo) CreateOrUpdate(e *model.AuthEvent) (*model.AuthEvent, error) {
	return e, nil
}
func (m *mockAuthEventRepo) FindAll(_ ...string) ([]model.AuthEvent, error) { return nil, nil }
func (m *mockAuthEventRepo) FindByUUID(_ any, _ ...string) (*model.AuthEvent, error) {
	return nil, nil
}
func (m *mockAuthEventRepo) FindByUUIDs(_ []string, _ ...string) ([]model.AuthEvent, error) {
	return nil, nil
}
func (m *mockAuthEventRepo) FindByID(_ any, _ ...string) (*model.AuthEvent, error) {
	return nil, nil
}
func (m *mockAuthEventRepo) UpdateByUUID(_ any, _ any) (*model.AuthEvent, error) { return nil, nil }
func (m *mockAuthEventRepo) UpdateByID(_ any, _ any) (*model.AuthEvent, error)   { return nil, nil }
func (m *mockAuthEventRepo) DeleteByUUID(_ any) error                            { return nil }
func (m *mockAuthEventRepo) DeleteByID(_ any) error                              { return nil }
func (m *mockAuthEventRepo) Paginate(_ map[string]any, _ int, _ int, _ ...string) (*repository.PaginationResult[model.AuthEvent], error) {
	return nil, nil
}
func (m *mockAuthEventRepo) FindPaginated(filter repository.AuthEventRepositoryGetFilter) (*repository.PaginationResult[model.AuthEvent], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(filter)
	}
	return &repository.PaginationResult[model.AuthEvent]{}, nil
}
func (m *mockAuthEventRepo) FindByUUIDAndTenantID(uid string, tid int64) (*model.AuthEvent, error) {
	if m.findByUUIDAndTIDFn != nil {
		return m.findByUUIDAndTIDFn(uid, tid)
	}
	return nil, nil
}
func (m *mockAuthEventRepo) FindByDateRange(tid int64, from, to time.Time) ([]model.AuthEvent, error) {
	if m.findByDateRangeFn != nil {
		return m.findByDateRangeFn(tid, from, to)
	}
	return nil, nil
}
func (m *mockAuthEventRepo) DeleteOlderThan(cutoff time.Time) (int64, error) {
	if m.deleteOlderThanFn != nil {
		return m.deleteOlderThanFn(cutoff)
	}
	return 0, nil
}
func (m *mockAuthEventRepo) CountByEventType(eventType string, tenantID int64) (int64, error) {
	if m.countByEventTypeFn != nil {
		return m.countByEventTypeFn(eventType, tenantID)
	}
	return 0, nil
}

// ---------------------------------------------------------------------------
// Log
// ---------------------------------------------------------------------------

func TestAuthEventService_Log(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var created *model.AuthEvent
		repo := &mockAuthEventRepo{
			createFn: func(e *model.AuthEvent) (*model.AuthEvent, error) {
				created = e
				return e, nil
			},
		}
		svc := NewAuthEventService(repo)
		svc.Log(context.Background(), AuthEventInput{
			TenantID:  1,
			IPAddress: "10.0.0.1",
			Category:  model.AuthEventCategoryAuthn,
			EventType: model.AuthEventTypeLoginSuccess,
			Severity:  model.AuthEventSeverityInfo,
			Result:    model.AuthEventResultSuccess,
		})
		require.NotNil(t, created)
		assert.Equal(t, int64(1), created.TenantID)
		assert.Equal(t, "10.0.0.1", created.IPAddress)
		assert.Equal(t, model.AuthEventCategoryAuthn, created.Category)
		assert.Equal(t, model.AuthEventTypeLoginSuccess, created.EventType)
	})

	t.Run("repo error logged but not propagated", func(t *testing.T) {
		repo := &mockAuthEventRepo{
			createFn: func(_ *model.AuthEvent) (*model.AuthEvent, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewAuthEventService(repo)
		// Should not panic — errors are swallowed
		svc.Log(context.Background(), AuthEventInput{
			TenantID:  1,
			IPAddress: "10.0.0.1",
			Category:  model.AuthEventCategoryAuthn,
			EventType: model.AuthEventTypeLoginFail,
			Severity:  model.AuthEventSeverityWarn,
			Result:    model.AuthEventResultFailure,
		})
	})

	t.Run("optional fields passed through", func(t *testing.T) {
		var created *model.AuthEvent
		actorID := int64(42)
		targetID := int64(99)
		desc := "test description"
		reason := "bad credentials"
		ua := "TestAgent/1.0"
		repo := &mockAuthEventRepo{
			createFn: func(e *model.AuthEvent) (*model.AuthEvent, error) {
				created = e
				return e, nil
			},
		}
		svc := NewAuthEventService(repo)
		svc.Log(context.Background(), AuthEventInput{
			TenantID:     1,
			ActorUserID:  &actorID,
			TargetUserID: &targetID,
			IPAddress:    "10.0.0.1",
			UserAgent:    &ua,
			Category:     model.AuthEventCategoryUser,
			EventType:    model.AuthEventTypeUserDeleted,
			Severity:     model.AuthEventSeverityCritical,
			Result:       model.AuthEventResultSuccess,
			Description:  &desc,
			ErrorReason:  &reason,
			Metadata:     datatypes.JSON([]byte(`{"key":"val"}`)),
		})
		require.NotNil(t, created)
		assert.Equal(t, &actorID, created.ActorUserID)
		assert.Equal(t, &targetID, created.TargetUserID)
		assert.Equal(t, &ua, created.UserAgent)
		assert.Equal(t, &desc, created.Description)
		assert.Equal(t, &reason, created.ErrorReason)
		assert.NotNil(t, created.Metadata)
	})

	t.Run("trace ID extracted from context", func(t *testing.T) {
		var created *model.AuthEvent
		repo := &mockAuthEventRepo{
			createFn: func(e *model.AuthEvent) (*model.AuthEvent, error) {
				created = e
				return e, nil
			},
		}
		svc := NewAuthEventService(repo)

		traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
		spanID, _ := trace.SpanIDFromHex("0102030405060708")
		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		})
		ctx := trace.ContextWithSpanContext(context.Background(), sc)

		svc.Log(ctx, AuthEventInput{
			TenantID:  1,
			IPAddress: "10.0.0.1",
			Category:  model.AuthEventCategoryAuthn,
			EventType: model.AuthEventTypeLoginSuccess,
			Severity:  model.AuthEventSeverityInfo,
			Result:    model.AuthEventResultSuccess,
		})
		require.NotNil(t, created)
		require.NotNil(t, created.TraceID)
		assert.Equal(t, "0102030405060708090a0b0c0d0e0f10", *created.TraceID)
	})
}

// ---------------------------------------------------------------------------
// FindPaginated
// ---------------------------------------------------------------------------

func TestAuthEventService_FindPaginated(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		eventUUID := uuid.New()
		repo := &mockAuthEventRepo{
			findPaginatedFn: func(_ repository.AuthEventRepositoryGetFilter) (*repository.PaginationResult[model.AuthEvent], error) {
				return &repository.PaginationResult[model.AuthEvent]{
					Data: []model.AuthEvent{
						{AuthEventUUID: eventUUID, TenantID: 1, IPAddress: "10.0.0.1", Category: model.AuthEventCategoryAuthn, EventType: model.AuthEventTypeLoginSuccess, Severity: model.AuthEventSeverityInfo, Result: model.AuthEventResultSuccess, CreatedAt: time.Now()},
					},
					Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}
		svc := NewAuthEventService(repo)
		tid := int64(1)
		result, err := svc.FindPaginated(context.Background(), repository.AuthEventRepositoryGetFilter{TenantID: &tid})
		require.NoError(t, err)
		require.Len(t, result.Data, 1)
		assert.Equal(t, eventUUID, result.Data[0].AuthEventUUID)
		assert.Equal(t, int64(1), result.Total)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &mockAuthEventRepo{
			findPaginatedFn: func(_ repository.AuthEventRepositoryGetFilter) (*repository.PaginationResult[model.AuthEvent], error) {
				return nil, errors.New("query failed")
			},
		}
		svc := NewAuthEventService(repo)
		_, err := svc.FindPaginated(context.Background(), repository.AuthEventRepositoryGetFilter{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to query auth events")
	})
}

// ---------------------------------------------------------------------------
// FindByUUID
// ---------------------------------------------------------------------------

func TestAuthEventService_FindByUUID(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		eventUUID := uuid.New()
		repo := &mockAuthEventRepo{
			findByUUIDAndTIDFn: func(uid string, tid int64) (*model.AuthEvent, error) {
				return &model.AuthEvent{
					AuthEventUUID: uuid.MustParse(uid),
					TenantID:      tid,
					IPAddress:     "10.0.0.1",
					Category:      model.AuthEventCategoryAuthn,
					EventType:     model.AuthEventTypeLoginSuccess,
					Severity:      model.AuthEventSeverityInfo,
					Result:        model.AuthEventResultSuccess,
				}, nil
			},
		}
		svc := NewAuthEventService(repo)
		result, err := svc.FindByUUID(context.Background(), 1, eventUUID)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, eventUUID, result.AuthEventUUID)
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockAuthEventRepo{
			findByUUIDAndTIDFn: func(_ string, _ int64) (*model.AuthEvent, error) {
				return nil, nil
			},
		}
		svc := NewAuthEventService(repo)
		_, err := svc.FindByUUID(context.Background(), 1, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &mockAuthEventRepo{
			findByUUIDAndTIDFn: func(_ string, _ int64) (*model.AuthEvent, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewAuthEventService(repo)
		_, err := svc.FindByUUID(context.Background(), 1, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find auth event")
	})
}

// ---------------------------------------------------------------------------
// CountByEventType
// ---------------------------------------------------------------------------

func TestAuthEventService_CountByEventType(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockAuthEventRepo{
			countByEventTypeFn: func(_ string, _ int64) (int64, error) {
				return 42, nil
			},
		}
		svc := NewAuthEventService(repo)
		count, err := svc.CountByEventType(context.Background(), model.AuthEventTypeLoginFail, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(42), count)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &mockAuthEventRepo{
			countByEventTypeFn: func(_ string, _ int64) (int64, error) {
				return 0, errors.New("count failed")
			},
		}
		svc := NewAuthEventService(repo)
		_, err := svc.CountByEventType(context.Background(), model.AuthEventTypeLoginFail, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to count auth events by type")
	})
}

// ---------------------------------------------------------------------------
// DeleteOlderThan
// ---------------------------------------------------------------------------

func TestAuthEventService_DeleteOlderThan(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockAuthEventRepo{
			deleteOlderThanFn: func(_ time.Time) (int64, error) {
				return 100, nil
			},
		}
		svc := NewAuthEventService(repo)
		count, err := svc.DeleteOlderThan(context.Background(), time.Now().Add(-365*24*time.Hour))
		require.NoError(t, err)
		assert.Equal(t, int64(100), count)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &mockAuthEventRepo{
			deleteOlderThanFn: func(_ time.Time) (int64, error) {
				return 0, errors.New("delete failed")
			},
		}
		svc := NewAuthEventService(repo)
		_, err := svc.DeleteOlderThan(context.Background(), time.Now())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete old auth events")
	})
}
