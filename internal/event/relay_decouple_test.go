package event

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// relayMockOutboxRepo records the per-arm mark calls so the decoupling behaviour
// can be asserted without a database.
type relayMockOutboxRepo struct {
	markWebhook   int
	markBroker    int
	markPublished int
}

func (m *relayMockOutboxRepo) Create(o *Outbox) (*Outbox, error)            { return o, nil }
func (m *relayMockOutboxRepo) CreateOrUpdate(o *Outbox) (*Outbox, error)    { return o, nil }
func (m *relayMockOutboxRepo) FindUnpublished(int) ([]Outbox, error)        { return nil, nil }
func (m *relayMockOutboxRepo) ClaimUnpublished(int) ([]Outbox, error)       { return nil, nil }
func (m *relayMockOutboxRepo) FindByTenantID(int64) ([]Outbox, error)       { return nil, nil }
func (m *relayMockOutboxRepo) FindByEventID(uuid.UUID) (*Outbox, error)     { return nil, nil }
func (m *relayMockOutboxRepo) MarkWebhookDelivered(int64) error             { m.markWebhook++; return nil }
func (m *relayMockOutboxRepo) MarkBrokerPublished(int64) error              { m.markBroker++; return nil }
func (m *relayMockOutboxRepo) MarkPublished(int64) error                    { m.markPublished++; return nil }
func (m *relayMockOutboxRepo) DeleteOlderThan(time.Time) (int64, error)     { return 0, nil }
func (m *relayMockOutboxRepo) DeleteBySubjectUUID(uuid.UUID) (int64, error) { return 0, nil }
func (m *relayMockOutboxRepo) WithTx(*gorm.DB) OutboxRepository             { return m }

// TestRelay_DeliverOne_ArmsAreDecoupled verifies the F6 fix: when the broker arm
// fails but the webhook arm succeeds, the webhook arm is marked done and the row
// is NOT published; on re-claim the already-delivered webhook arm is SKIPPED
// (no duplicate fan-out) while only the broker arm re-runs.
func TestRelay_DeliverOne_ArmsAreDecoupled(t *testing.T) {
	repo := &relayMockOutboxRepo{}
	webhookCalls, brokerCalls := 0, 0
	r := &Relay{
		outboxRepo:     repo,
		deliverWebhook: func(context.Context, *Outbox) error { webhookCalls++; return nil },
		deliverBroker:  func(context.Context, *Outbox) error { brokerCalls++; return errors.New("broker down") },
		stopCh:         make(chan struct{}),
	}

	// First claim: broker down.
	r.deliverOne(context.Background(), &Outbox{OutboxID: 1})
	assert.Equal(t, 1, webhookCalls, "webhook arm runs once")
	assert.Equal(t, 1, brokerCalls, "broker arm runs once")
	assert.Equal(t, 1, repo.markWebhook, "webhook arm marked delivered")
	assert.Equal(t, 0, repo.markBroker, "failed broker arm not marked")
	assert.Equal(t, 0, repo.markPublished, "row not fully published while broker pending")

	// Re-claim: webhook already delivered (column set). The webhook arm must NOT
	// run again — this is what prevents duplicate fan-out during a broker outage.
	now := time.Now().UTC()
	r.deliverOne(context.Background(), &Outbox{OutboxID: 1, WebhookDeliveredAt: &now})
	assert.Equal(t, 1, webhookCalls, "webhook arm is NOT re-run on re-claim")
	assert.Equal(t, 2, brokerCalls, "only the broker arm re-runs")
	assert.Equal(t, 0, repo.markPublished, "still not published — broker still failing")

	// Broker recovers on the next re-claim.
	r.deliverBroker = func(context.Context, *Outbox) error { brokerCalls++; return nil }
	r.deliverOne(context.Background(), &Outbox{OutboxID: 1, WebhookDeliveredAt: &now})
	assert.Equal(t, 1, webhookCalls, "webhook arm still not re-run")
	assert.Equal(t, 1, repo.markBroker, "broker arm marked once it succeeds")
	assert.Equal(t, 1, repo.markPublished, "row fully published once BOTH arms are done")
}
