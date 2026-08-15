package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// DeliveryAdapter provides the delivery functions called by the relay.
// It is wired in app.go with the actual webhook and broker implementations.
type DeliveryAdapter struct {
	webhookDeliver func(ctx context.Context, outbox *Outbox) error
	brokerDeliver  func(ctx context.Context, outbox *Outbox) error
}

// NewDeliveryAdapter creates a DeliveryAdapter with the given delivery functions.
func NewDeliveryAdapter(
	webhookDeliver func(ctx context.Context, outbox *Outbox) error,
	brokerDeliver func(ctx context.Context, outbox *Outbox) error,
) *DeliveryAdapter {
	return &DeliveryAdapter{
		webhookDeliver: webhookDeliver,
		brokerDeliver:  brokerDeliver,
	}
}

// DeliverWebhook satisfies the WebhookDeliveryFunc signature.
func (a *DeliveryAdapter) DeliverWebhook(ctx context.Context, outbox *Outbox) error {
	if a.webhookDeliver == nil {
		return nil
	}
	return a.webhookDeliver(ctx, outbox)
}

// DeliverBroker satisfies the BrokerDeliveryFunc signature.
func (a *DeliveryAdapter) DeliverBroker(ctx context.Context, outbox *Outbox) error {
	if a.brokerDeliver == nil {
		return nil
	}
	return a.brokerDeliver(ctx, outbox)
}

// PublicIDResolver maps internal database primary keys to external UUIDs for
// integration-event delivery. Webhook and broker payloads must never expose
// internal integer identifiers.
type PublicIDResolver interface {
	TenantUUIDByID(ctx context.Context, id int64) (uuid.UUID, bool)
	UserUUIDByID(ctx context.Context, id int64) (uuid.UUID, bool)
}

// OutboxPayload extracts the outbox event fields as the external delivery
// envelope. Public *_id fields intentionally contain UUID strings, matching the
// public API convention while keeping integer primary keys internal.
func OutboxPayload(ctx context.Context, outbox *Outbox, resolver PublicIDResolver) (map[string]any, error) {
	tenantUUID, ok := uuid.Nil, false
	if resolver != nil {
		tenantUUID, ok = resolver.TenantUUIDByID(ctx, outbox.TenantID)
	}
	if !ok && outbox.SubjectType == "tenant" && outbox.SubjectUUID != nil {
		tenantUUID = *outbox.SubjectUUID
		ok = true
	}
	if !ok || tenantUUID == uuid.Nil {
		return nil, fmt.Errorf("resolve tenant uuid for outbox event %s", outbox.EventID)
	}

	var actorUserID any
	if outbox.ActorUserID != nil && resolver != nil {
		if actorUUID, ok := resolver.UserUUIDByID(ctx, *outbox.ActorUserID); ok && actorUUID != uuid.Nil {
			actorUserID = actorUUID.String()
		}
	}

	payload := datatypes.JSON([]byte("{}"))
	if len(outbox.Payload) > 0 {
		payload = outbox.Payload
	}

	return map[string]any{
		"event_id":       outbox.EventID.String(),
		"event_type":     outbox.EventType,
		"event_version":  outbox.EventVersion,
		"tenant_id":      tenantUUID.String(),
		"actor_user_id":  actorUserID,
		"subject_id":     outbox.SubjectUUID,
		"subject_type":   outbox.SubjectType,
		"changed_fields": unmarshalStringSliceRaw(outbox.ChangedFields),
		"payload":        json.RawMessage(payload),
		"occurred_at":    outbox.OccurredAt,
		"trace_id":       outbox.TraceID,
		"request_id":     outbox.RequestID,
	}, nil
}
