package webhook

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/maintainerd/auth/internal/authevent"
)

// Dispatcher delivers auth events to subscribed webhook endpoints.
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
// delivers the event to those that subscribe to the event type. Matching
// deliveries are queued into a bounded worker pool; overflow is logged and
// dropped so webhook pressure cannot create unbounded goroutines.
func (d *Dispatcher) Dispatch(ctx context.Context, event *authevent.AuthEvent) {
	endpoints, err := d.repo.FindActiveByTenantID(event.TenantID)
	if err != nil {
		slog.Error("webhook dispatch: find endpoints", "tenant_id", event.TenantID, "err", err)
		return
	}

	for _, ep := range endpoints {
		ep := ep
		if !matchesEvent(ep.Events, event.EventType) {
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

// matchesEvent returns true when the endpoint's event list contains the given
// event type, "*" (wildcard), or is empty (matches all).
func matchesEvent(events []byte, eventType string) bool {
	if len(events) == 0 {
		return true
	}
	var list []string
	if err := json.Unmarshal(events, &list); err != nil {
		return false
	}
	if len(list) == 0 {
		return true
	}
	for _, e := range list {
		if e == "*" || e == eventType {
			return true
		}
	}
	return false
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
