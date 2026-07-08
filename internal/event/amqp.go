package event

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/retry"
	amqp "github.com/rabbitmq/amqp091-go"
)

// AMQPConfig holds the RabbitMQ connection parameters.
type AMQPConfig struct {
	URL string
}

// NewAMQPConfigFromEnv reads AMQP configuration from environment variables.
// Returns nil when RABBITMQ_URL is empty (broker disabled).
func NewAMQPConfigFromEnv() *AMQPConfig {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		return nil
	}
	return &AMQPConfig{URL: url}
}

// ConnectAMQP dials RabbitMQ, opens a channel, and declares the exchange.
// Returns a publish function and a close function for cleanup.
// Returns (nil, nil, nil) when config is nil (broker disabled).
//
// The dial is retried with exponential backoff using the shared retry helper,
// inheriting the caller's context so startup can be cancelled cleanly.
func ConnectAMQP(ctx context.Context, cfg *AMQPConfig) (func(ctx context.Context, exchange, routingKey string, body []byte, messageID, eventType string) error, func(), error) {
	if cfg == nil {
		return nil, nil, nil
	}

	var conn *amqp.Connection
	if err := retry.WithBackoff(ctx, "amqp", 10, 2*time.Second, func() error {
		var dialErr error
		conn, dialErr = amqp.Dial(cfg.URL)
		return dialErr
	}); err != nil {
		return nil, nil, fmt.Errorf("amqp: dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("amqp: open channel: %w", err)
	}

	if err := ch.ExchangeDeclare(
		"maintainerd-auth.events",
		"topic", // topic exchange allows wildcard routing keys
		true,    // durable
		false,   // auto-delete
		false,   // internal
		false,   // no-wait
		nil,     // args
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, nil, fmt.Errorf("amqp: declare exchange: %w", err)
	}

	slog.Info("amqp: connected to RabbitMQ", "url", cfg.URL)

	publish := func(ctx context.Context, exchange, routingKey string, body []byte, messageID, eventType string) error {
		return ch.PublishWithContext(ctx,
			exchange,
			routingKey,
			false, // mandatory
			false, // immediate
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				MessageId:    messageID,
				Type:         eventType,
				Body:         body,
			},
		)
	}

	closeFn := func() {
		_ = ch.Close()
		_ = conn.Close()
		slog.Info("amqp: disconnected from RabbitMQ")
	}

	return publish, closeFn, nil
}
