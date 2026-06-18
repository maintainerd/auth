package event

import (
	"context"
	"encoding/json"
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

// OutboxPayload extracts the outbox event fields as a generic map for delivery.
func OutboxPayload(outbox *Outbox) map[string]any {
	return map[string]any{
		"event_id":       outbox.EventID.String(),
		"event_type":     outbox.EventType,
		"event_version":  outbox.EventVersion,
		"tenant_id":      outbox.TenantID,
		"actor_user_id":  outbox.ActorUserID,
		"subject_uuid":   outbox.SubjectUUID,
		"subject_type":   outbox.SubjectType,
		"changed_fields": unmarshalStringSliceRaw(outbox.ChangedFields),
		"payload":        json.RawMessage(outbox.Payload),
		"occurred_at":    outbox.OccurredAt,
		"trace_id":       outbox.TraceID,
		"request_id":     outbox.RequestID,
	}
}
