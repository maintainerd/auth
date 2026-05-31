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
	repo WebhookEndpointRepository
	wg   sync.WaitGroup
}

// NewDispatcher creates a new Dispatcher backed by the given
func NewDispatcher(repo WebhookEndpointRepository) *Dispatcher {
	return &Dispatcher{repo: repo}
}

// Dispatch finds all active webhook endpoints for the event's tenant and
// delivers the event to those that subscribe to the event type.
// Each delivery runs in its own goroutine; errors are logged but never propagated.
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
		d.wg.Add(1)
		go func(ep WebhookEndpoint) {
			defer d.wg.Done()
			d.deliver(ctx, ep, event)
		}(ep)
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
	d.wg.Wait()
}
