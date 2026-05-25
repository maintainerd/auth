package webhook

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
)

// Dispatcher delivers auth events to subscribed webhook endpoints.
type Dispatcher struct {
	repo repository.WebhookEndpointRepository
}

// NewDispatcher creates a new Dispatcher backed by the given repository.
func NewDispatcher(repo repository.WebhookEndpointRepository) *Dispatcher {
	return &Dispatcher{repo: repo}
}

// Dispatch finds all active webhook endpoints for the event's tenant and
// delivers the event to those that subscribe to the event type.
// Each delivery runs in its own goroutine; errors are logged but never propagated.
func (d *Dispatcher) Dispatch(ctx context.Context, event *model.AuthEvent) {
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
		go d.deliver(ctx, ep, event)
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
