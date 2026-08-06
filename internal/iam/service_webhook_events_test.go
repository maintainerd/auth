package iam

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/event"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type recordingAuthEventService struct {
	events []authevent.AuthEventInput
}

func (r *recordingAuthEventService) Log(_ context.Context, input authevent.AuthEventInput) {
	r.events = append(r.events, input)
}
func (r *recordingAuthEventService) FindPaginated(context.Context, authevent.AuthEventRepositoryGetFilter) (*authevent.PaginationResult[authevent.AuthEventServiceDataResult], error) {
	return nil, nil
}
func (r *recordingAuthEventService) FindByUUID(context.Context, int64, uuid.UUID) (*authevent.AuthEventServiceDataResult, error) {
	return nil, nil
}
func (r *recordingAuthEventService) CountByEventType(context.Context, string, int64) (int64, error) {
	return 0, nil
}
func (r *recordingAuthEventService) DeleteOlderThan(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (r *recordingAuthEventService) Shutdown() {}

// recordingEventService captures the integration events written to the outbox.
type recordingEventService struct {
	emitted []*event.IntegrationEvent
}

func (r *recordingEventService) Emit(_ context.Context, _ *gorm.DB, e *event.IntegrationEvent) (*event.IntegrationEvent, error) {
	r.emitted = append(r.emitted, e)
	return e, nil
}
func (r *recordingEventService) Shutdown() {}

func TestServiceService_AssignAndRemovePolicy_EmitWebhookEvents(t *testing.T) {
	svcUUID := uuid.New()
	polUUID := uuid.New()
	events := &recordingAuthEventService{}

	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()
	svc := NewServiceService(db, &mockServiceRepo{
		findByUUIDFn: func(any, ...string) (*Service, error) {
			return &Service{ServiceID: 1, ServiceUUID: svcUUID, TenantID: tenantID}, nil
		},
	}, &mockAPIRepo{}, &mockServicePolicyRepo{
		findByServiceAndPolicyFn: func(int64, int64) (*ServicePolicy, error) { return nil, nil },
		createFn:                 func(sp *ServicePolicy) (*ServicePolicy, error) { return sp, nil },
	}, &mockPolicyRepo{
		findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Policy, error) {
			return &Policy{PolicyID: 2, PolicyUUID: polUUID}, nil
		},
	}, events)
	require.NoError(t, svc.AssignPolicy(context.Background(), svcUUID, polUUID, tenantID))
	require.Len(t, events.events, 1)
	require.Equal(t, authevent.AuthEventTypeIAMServicePolicyAssigned, events.events[0].EventType)

	db, mock = newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()
	svc = NewServiceService(db, &mockServiceRepo{
		findByUUIDFn: func(any, ...string) (*Service, error) {
			return &Service{ServiceID: 1, ServiceUUID: svcUUID, TenantID: tenantID}, nil
		},
	}, &mockAPIRepo{}, &mockServicePolicyRepo{
		findByServiceAndPolicyFn: func(int64, int64) (*ServicePolicy, error) {
			return &ServicePolicy{ServicePolicyID: 3}, nil
		},
		deleteByServiceAndPolicyFn: func(int64, int64) error { return nil },
	}, &mockPolicyRepo{
		findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Policy, error) {
			return &Policy{PolicyID: 2, PolicyUUID: polUUID}, nil
		},
	}, events)
	require.NoError(t, svc.RemovePolicy(context.Background(), svcUUID, polUUID, tenantID))
	require.Len(t, events.events, 2)
	require.Equal(t, authevent.AuthEventTypeIAMServicePolicyRemoved, events.events[1].EventType)
}

func TestPolicyService_Update_EmitsPolicyUpdatedEvent(t *testing.T) {
	events := &recordingAuthEventService{}
	policyUUID := uuid.New()
	policy := &Policy{PolicyID: 1, PolicyUUID: policyUUID, TenantID: tenantID, Name: "old", Version: "v1", Status: "active", Document: datatypes.JSON(`{"version":"v1","statement":[]}`)}
	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	svc := NewPolicyService(db, &mockPolicyRepo{
		findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Policy, error) { return policy, nil },
		findByNameAndVersionFn:  func(string, string, int64) (*Policy, error) { return nil, nil },
		updateByUUIDFn: func(any, any) (*Policy, error) {
			return policy, nil
		},
	}, &mockServiceRepo{}, &mockAPIRepo{}, nil, events)
	_, err := svc.Update(context.Background(), policyUUID, tenantID, "new", nil, policy.Document, "v1", "active", PolicyChangeContext{})
	require.NoError(t, err)
	require.Len(t, events.events, 1)
	require.Equal(t, authevent.AuthEventTypeIAMPolicyUpdated, events.events[0].EventType)
}

// SetStatusByUUID was the only policy mutation emitting neither an integration
// event nor an auth event, though Create/Update/Delete all do — and deactivation is
// the revocation path, so the change downstream bundle consumers most need to hear
// about was the one nobody was told about.
func TestPolicyService_SetStatusByUUID_EmitsEvents(t *testing.T) {
	policyUUID := uuid.New()
	active := &Policy{PolicyID: 1, PolicyUUID: policyUUID, TenantID: tenantID, Name: "read-only", Version: "v1", Status: "active"}

	t.Run("deactivation emits both events", func(t *testing.T) {
		events := &recordingAuthEventService{}
		integration := &recordingEventService{}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		calls := 0

		svc := NewPolicyService(db, &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Policy, error) {
				calls++
				if calls == 1 {
					return active, nil
				}
				inactive := *active
				inactive.Status = "inactive"
				return &inactive, nil
			},
		}, &mockServiceRepo{}, &mockAPIRepo{}, integration, events)

		_, err := svc.SetStatusByUUID(context.Background(), policyUUID, tenantID, "inactive")

		require.NoError(t, err)
		require.Len(t, events.events, 1)
		require.Equal(t, authevent.AuthEventTypeIAMPolicyUpdated, events.events[0].EventType)
		require.Len(t, integration.emitted, 1)
		require.Equal(t, event.EventTypeIAMPolicyUpdated, integration.emitted[0].EventType)
	})

	// A no-op write is not a state change, so it raises no integration event —
	// matching Update, which emits only when a field actually changed.
	t.Run("re-setting the same status emits no integration event", func(t *testing.T) {
		integration := &recordingEventService{}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		svc := NewPolicyService(db, &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Policy, error) { return active, nil },
		}, &mockServiceRepo{}, &mockAPIRepo{}, integration)

		_, err := svc.SetStatusByUUID(context.Background(), policyUUID, tenantID, "active")

		require.NoError(t, err)
		require.Empty(t, integration.emitted)
	})
}
