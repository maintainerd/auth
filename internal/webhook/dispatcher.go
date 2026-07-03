package webhook

import (
	"context"
	"log/slog"
	"sync"

	"github.com/maintainerd/maintainerd-auth/internal/authevent"
)

// Dispatcher delivers auth events to subscribed webhook endpoints.
// Deprecated: The integration event plane now uses the outbox + relay pattern.
// This dispatcher is kept for backward compatibility with audit event webhook delivery.
type Dispatcher struct {
	repo   WebhookEndpointRepository
	jobs   chan webhookDelivery
	wg     sync.WaitGroup
	mu     sync.Mutex
	closed bool
}

type webhookDelivery struct {
	ctx   context.Context
	event *authevent.AuthEvent
	ep    WebhookEndpoint
}

const (
	defaultDispatchWorkers = 4
	defaultDispatchQueue   = 128
)

// NewDispatcher creates a new Dispatcher backed by the given endpoint repository.
func NewDispatcher(repo WebhookEndpointRepository) *Dispatcher {
	d := &Dispatcher{
		repo: repo,
		jobs: make(chan webhookDelivery, defaultDispatchQueue),
	}
	for i := 0; i < defaultDispatchWorkers; i++ {
		d.wg.Add(1)
		go d.worker()
	}
	return d
}

// Dispatch finds all active webhook endpoints for the event's tenant and
// delivers the event to those that subscribe to the event type.
func (d *Dispatcher) Dispatch(ctx context.Context, event *authevent.AuthEvent) {
	endpoints, err := d.repo.FindActiveByTenantID(event.TenantID)
	if err != nil {
		slog.Error("webhook dispatch: find endpoints", "tenant_id", event.TenantID, "err", err)
		return
	}

	for _, ep := range endpoints {
		ep := ep
		if !matchesEventType(ep, event.EventType) {
			continue
		}
		d.enqueue(webhookDelivery{ctx: ctx, event: event, ep: ep})
	}
}

func (d *Dispatcher) enqueue(job webhookDelivery) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return
	}
	select {
	case d.jobs <- job:
	default:
		slog.Warn("webhook dispatch: queue full, dropping delivery",
			"tenant_id", job.event.TenantID,
			"event_type", job.event.EventType,
			"webhook_endpoint_id", job.ep.WebhookEndpointID,
		)
	}
}

func (d *Dispatcher) worker() {
	defer d.wg.Done()
	for job := range d.jobs {
		d.deliver(job.ctx, job.ep, job.event)
	}
}

// matchesEventType returns true when the endpoint subscribes to the event type.
// subscribe_all = true means all events are delivered.
func matchesEventType(ep WebhookEndpoint, _ string) bool {
	return ep.SubscribeAll
}

// Shutdown waits for all in-flight webhook deliveries to complete.
func (d *Dispatcher) Shutdown() {
	d.mu.Lock()
	if !d.closed {
		d.closed = true
		close(d.jobs)
	}
	d.mu.Unlock()
	d.wg.Wait()
}
