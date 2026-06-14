package event

import (
	"context"
	"fmt"
	"log/slog"
	"os"

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
func ConnectAMQP(cfg *AMQPConfig) (func(ctx context.Context, exchange, routingKey string, body []byte, messageID, eventType string) error, func(), error) {
	if cfg == nil {
		return nil, nil, nil
	}

	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, nil, fmt.Errorf("amqp: dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("amqp: open channel: %w", err)
	}

	if err := ch.ExchangeDeclare(
		"maintainerd-auth.events",
		"topic",  // topic exchange allows wildcard routing keys
		true,     // durable
		false,    // auto-delete
		false,    // internal
		false,    // no-wait
		nil,      // args
	); err != nil {
		ch.Close()
		conn.Close()
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
		ch.Close()
		conn.Close()
		slog.Info("amqp: disconnected from RabbitMQ")
	}

	return publish, closeFn, nil
}
