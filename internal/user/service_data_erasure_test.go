package user

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// mockErasureRepo is a function-field mock of DataErasureRequestRepository.
type mockErasureRepo struct {
	createFn         func(*DataErasureRequest) (*DataErasureRequest, error)
	findActiveFn     func(int64) (*DataErasureRequest, error)
	findDueFn        func(time.Time, int) ([]DataErasureRequest, error)
	markInProgressFn func(int64, time.Time) error
	markCompletedFn  func(int64, time.Time) error
	markPendingFn    func(int64) error
}

func (m *mockErasureRepo) Create(e *DataErasureRequest) (*DataErasureRequest, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockErasureRepo) CreateOrUpdate(e *DataErasureRequest) (*DataErasureRequest, error) {
	return e, nil
}
func (m *mockErasureRepo) WithTx(*gorm.DB) DataErasureRequestRepository { return m }
func (m *mockErasureRepo) FindActiveByUserID(userID int64) (*DataErasureRequest, error) {
	if m.findActiveFn != nil {
		return m.findActiveFn(userID)
	}
	return nil, nil
}
func (m *mockErasureRepo) FindDueForProcessing(now time.Time, limit int) ([]DataErasureRequest, error) {
	if m.findDueFn != nil {
		return m.findDueFn(now, limit)
	}
	return nil, nil
}
func (m *mockErasureRepo) MarkInProgress(id int64, at time.Time) error {
	if m.markInProgressFn != nil {
		return m.markInProgressFn(id, at)
	}
	return nil
}
func (m *mockErasureRepo) MarkCompleted(id int64, at time.Time) error {
	if m.markCompletedFn != nil {
		return m.markCompletedFn(id, at)
	}
	return nil
}
func (m *mockErasureRepo) MarkPending(id int64) error {
	if m.markPendingFn != nil {
		return m.markPendingFn(id)
	}
	return nil
}

// mockAnonymizer is a function-field mock of UserAnonymizer.
type mockAnonymizer struct {
	fn func(int64) error
}

func (m *mockAnonymizer) AnonymizeUser(_ context.Context, userID int64) error {
	if m.fn != nil {
		return m.fn(userID)
	}
	return nil
}

func TestDataErasureService_RequestErasure(t *testing.T) {
	t.Run("creates a new pending request scheduled ~30 days out", func(t *testing.T) {
		var created *DataErasureRequest
		repo := &mockErasureRepo{createFn: func(e *DataErasureRequest) (*DataErasureRequest, error) {
			created = e
			e.DataErasureRequestID = 1
			return e, nil
		}}
		svc := NewDataErasureService(repo, &mockAnonymizer{})
		res, err := svc.RequestErasure(context.Background(), RequestErasureInput{TenantID: 1, UserID: 9, Reason: "gdpr"})
		assert.NoError(t, err)
		assert.Equal(t, "pending", res.Status)
		assert.Equal(t, "gdpr", created.Reason)
		// scheduled ~30 days out.
		delta := time.Until(created.ScheduledAt)
		assert.Greater(t, delta, 29*24*time.Hour)
		assert.Less(t, delta, 31*24*time.Hour)
	})

	t.Run("is idempotent when a pending request already exists", func(t *testing.T) {
		existing := &DataErasureRequest{DataErasureRequestID: 3, Status: "pending"}
		createCalled := false
		repo := &mockErasureRepo{
			findActiveFn: func(int64) (*DataErasureRequest, error) { return existing, nil },
			createFn: func(e *DataErasureRequest) (*DataErasureRequest, error) {
				createCalled = true
				return e, nil
			},
		}
		svc := NewDataErasureService(repo, &mockAnonymizer{})
		res, err := svc.RequestErasure(context.Background(), RequestErasureInput{TenantID: 1, UserID: 9})
		assert.NoError(t, err)
		assert.Equal(t, "pending", res.Status)
		assert.False(t, createCalled, "must not create a second request")
	})
}

func TestDataErasureService_ProcessPendingErasureRequests(t *testing.T) {
	t.Run("anonymizes due requests and marks completed", func(t *testing.T) {
		completed := false
		repo := &mockErasureRepo{
			findDueFn: func(time.Time, int) ([]DataErasureRequest, error) {
				return []DataErasureRequest{{DataErasureRequestID: 1, UserID: 42}}, nil
			},
			markCompletedFn: func(int64, time.Time) error { completed = true; return nil },
		}
		anonymizedUser := int64(0)
		anon := &mockAnonymizer{fn: func(userID int64) error { anonymizedUser = userID; return nil }}
		svc := NewDataErasureService(repo, anon)
		assert.NoError(t, svc.ProcessPendingErasureRequests(context.Background()))
		assert.Equal(t, int64(42), anonymizedUser)
		assert.True(t, completed)
	})

	t.Run("reverts to pending when anonymize fails (no 'failed' status)", func(t *testing.T) {
		reverted := false
		completed := false
		repo := &mockErasureRepo{
			findDueFn: func(time.Time, int) ([]DataErasureRequest, error) {
				return []DataErasureRequest{{DataErasureRequestID: 1, UserID: 42}}, nil
			},
			markPendingFn:   func(int64) error { reverted = true; return nil },
			markCompletedFn: func(int64, time.Time) error { completed = true; return nil },
		}
		anon := &mockAnonymizer{fn: func(int64) error { return assert.AnError }}
		svc := NewDataErasureService(repo, anon)
		assert.NoError(t, svc.ProcessPendingErasureRequests(context.Background()))
		assert.True(t, reverted, "failed request must revert to pending for retry")
		assert.False(t, completed, "failed request must not be marked completed")
	})
}
