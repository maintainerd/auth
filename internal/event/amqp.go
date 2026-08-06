package event

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/retry"
	amqp "github.com/rabbitmq/amqp091-go"
)

// AMQPConfig holds the RabbitMQ connection parameters.
type AMQPConfig struct {
	URL string
}

// redactAMQPURL strips the userinfo from an AMQP URL so it can be logged.
//
// RABBITMQ_URL carries the broker password inline
// ("amqp://user:password@host:5672/"), and the startup line logged it whole — so
// every boot wrote a live credential into the application log, where it is then
// shipped to whatever aggregates logs and outlives any rotation of the secret
// itself. Only the host and vhost are useful for diagnosing a connection anyway.
// A URL that will not parse is reported as a constant rather than echoed, since
// an unparseable string may still contain the password.
func redactAMQPURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "amqp://[unparseable]"
	}
	if u.User != nil {
		// A bare word, not "[redacted]": url.String() percent-encodes the userinfo,
		// so brackets would come back as %5Bredacted%5D.
		u.User = url.User("redacted")
	}
	return u.String()
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

	// Put the channel in publisher-confirm mode so a publish only reports success
	// once the broker has durably accepted the message. Without this, a publish
	// returns as soon as the frame hits the socket buffer, so a broker crash
	// before persistence would be silently lost while the relay marks the outbox
	// row published.
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, nil, fmt.Errorf("amqp: enable publisher confirms: %w", err)
	}

	// Surface unroutable messages. With mandatory=true the broker returns a
	// message that matched no queue; log it loudly (it means a tenant has an
	// enabled event route but no consumer queue bound — a deployment gap).
	returns := ch.NotifyReturn(make(chan amqp.Return, 16))
	go func() {
		for r := range returns {
			slog.Error("amqp: message returned as unroutable",
				"exchange", r.Exchange, "routing_key", r.RoutingKey,
				"message_id", r.MessageId, "reply_text", r.ReplyText)
		}
	}()

	slog.Info("amqp: connected to RabbitMQ", "broker", redactAMQPURL(cfg.URL))

	publish := func(ctx context.Context, exchange, routingKey string, body []byte, messageID, eventType string) error {
		dConf, err := ch.PublishWithDeferredConfirmWithContext(ctx,
			exchange,
			routingKey,
			true,  // mandatory — return (and log) unroutable messages
			false, // immediate
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				MessageId:    messageID,
				Type:         eventType,
				Body:         body,
			},
		)
		if err != nil {
			return err
		}
		// Block until the broker confirms (or the context/connection ends). A
		// non-ack (broker nack / channel drop) is a real failure: return an error
		// so the relay leaves the outbox row unpublished for re-claim.
		acked, err := dConf.WaitContext(ctx)
		if err != nil {
			return fmt.Errorf("amqp: await publish confirm: %w", err)
		}
		if !acked {
			return fmt.Errorf("amqp: publish nacked by broker (message_id=%s)", messageID)
		}
		return nil
	}

	closeFn := func() {
		_ = ch.Close()
		_ = conn.Close()
		slog.Info("amqp: disconnected from RabbitMQ")
	}

	return publish, closeFn, nil
}
