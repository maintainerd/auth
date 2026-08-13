package event

import (
	"context"
	"fmt"
	"log/slog"
)

// RabbitMQPublisher publishes integration events to RabbitMQ exchanges.
// It implements the BrokerDeliveryFunc signature.
type RabbitMQPublisher struct {
	// publishFunc is the underlying AMQP publish call, set by the wire layer.
	publishFunc func(ctx context.Context, exchange, routingKey string, body []byte, messageID, eventType string) error
	enabled     bool
}

// NewRabbitMQPublisher creates a new RabbitMQPublisher.
// The publishFunc is the AMQP channel.Publish call, injected from the app layer.
// Pass nil to disable the broker.
func NewRabbitMQPublisher(publishFunc func(ctx context.Context, exchange, routingKey string, body []byte, messageID, eventType string) error) *RabbitMQPublisher {
	if publishFunc == nil {
		slog.Info("RabbitMQ: no publisher configured, disabling broker")
		return &RabbitMQPublisher{enabled: false}
	}
	return &RabbitMQPublisher{
		publishFunc: publishFunc,
		enabled:     true,
	}
}

// IsEnabled returns whether the RabbitMQ publisher is active.
func (p *RabbitMQPublisher) IsEnabled() bool {
	return p != nil && p.enabled
}

// Publish delivers an outbox event to the configured RabbitMQ exchange.
// This satisfies the BrokerDeliveryFunc signature.
func (p *RabbitMQPublisher) Publish(ctx context.Context, outbox *Outbox, payload []byte) error {
	if !p.enabled {
		return nil
	}

	if len(payload) == 0 {
		return fmt.Errorf("empty outbox payload")
	}

	return p.publishFunc(
		ctx,
		"maintainerd-auth.events",
		outbox.EventType,
		payload,
		outbox.EventID.String(),
		outbox.EventType,
	)
}
