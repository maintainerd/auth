package event

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	writeGateCacheTTL         = 30 * time.Second
	writeGateInvalidationChan = "event:gate:invalidate"
	writeGateRefreshInterval  = 25 * time.Second
)

// WriteGate decides whether an integration event should be written to the outbox.
// Resolved from an in-memory cache before opening the transaction.
type WriteGate struct {
	mu                 sync.RWMutex
	activeTypes        map[string]bool
	tenantDisabledKeys map[int64]map[string]bool
	tenantHasListener  map[int64]bool
	lastRefresh        time.Time

	eventTypeRepo       EventTypeRepository
	tenantEventTypeRepo TenantEventTypeRepository
	listenerChecker     ListenerChecker
	eventRouteRepo      EventRouteRepository

	rdb    *redis.Client
	sub    *redis.PubSub
	stopCh chan struct{}

	ttlOnly bool
}

// ListenerChecker checks whether a tenant has any active listener (webhook or broker).
type ListenerChecker interface {
	HasAnyActiveListener(tenantID int64) (bool, error)
}

// NewWriteGate creates a new WriteGate. Pass nil rdb when Redis is unavailable.
func NewWriteGate(
	eventTypeRepo EventTypeRepository,
	tenantEventTypeRepo TenantEventTypeRepository,
	listenerChecker ListenerChecker,
	eventRouteRepo EventRouteRepository,
	rdb *redis.Client,
) *WriteGate {
	wg := &WriteGate{
		activeTypes:         make(map[string]bool),
		tenantDisabledKeys:  make(map[int64]map[string]bool),
		tenantHasListener:   make(map[int64]bool),
		eventTypeRepo:       eventTypeRepo,
		tenantEventTypeRepo: tenantEventTypeRepo,
		listenerChecker:     listenerChecker,
		eventRouteRepo:      eventRouteRepo,
		rdb:                 rdb,
		stopCh:              make(chan struct{}),
		ttlOnly:             rdb == nil,
	}

	if rdb != nil {
		wg.sub = rdb.Subscribe(context.Background(), writeGateInvalidationChan)
		go wg.listenInvalidation()
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("recovered from panic in background goroutine", "panic", r)
			}
		}()
		wg.startBackgroundRefresh()
	}()
	return wg
}

func (wg *WriteGate) startBackgroundRefresh() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("write gate: background refresh panicked, stopping", "panic", r)
		}
	}()

	ticker := time.NewTicker(writeGateRefreshInterval)
	defer ticker.Stop()

	wg.refreshActiveTypes(context.Background())

	for {
		select {
		case <-wg.stopCh:
			return
		case <-ticker.C:
			wg.refreshActiveTypes(context.Background())
			if wg.ttlOnly {
				wg.mu.Lock()
				wg.tenantDisabledKeys = make(map[int64]map[string]bool)
				wg.tenantHasListener = make(map[int64]bool)
				wg.mu.Unlock()
			}
		}
	}
}

func (wg *WriteGate) listenInvalidation() {
	ch := wg.sub.Channel()
	for {
		select {
		case <-wg.stopCh:
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			slog.Debug("write gate invalidation received", "channel", msg.Channel)
			wg.mu.Lock()
			wg.tenantDisabledKeys = make(map[int64]map[string]bool)
			wg.tenantHasListener = make(map[int64]bool)
			wg.mu.Unlock()
		}
	}
}

// ShouldEmit checks whether an event of the given type for the given tenant
// should be written to the outbox.
func (wg *WriteGate) ShouldEmit(ctx context.Context, eventType string, tenantID int64) bool {
	if wg.needsRefresh() {
		wg.refreshActiveTypes(ctx)
	}

	wg.ensureTenantState(ctx, tenantID)

	wg.mu.RLock()
	defer wg.mu.RUnlock()

	if !wg.activeTypes[eventType] {
		return false
	}

	if disabled, ok := wg.tenantDisabledKeys[tenantID]; ok {
		if disabled[eventType] {
			return false
		}
	}

	has, ok := wg.tenantHasListener[tenantID]
	if !ok {
		return true
	}
	return has
}

func (wg *WriteGate) ensureTenantState(ctx context.Context, tenantID int64) {
	wg.mu.RLock()
	_, hasListener := wg.tenantHasListener[tenantID]
	_, hasDisabled := wg.tenantDisabledKeys[tenantID]
	wg.mu.RUnlock()

	if hasListener && hasDisabled {
		return
	}

	wg.mu.Lock()
	defer wg.mu.Unlock()

	if _, ok := wg.tenantHasListener[tenantID]; !ok {
		has, err := wg.listenerChecker.HasAnyActiveListener(tenantID)
		if err != nil {
			slog.Error("write gate: check tenant listeners failed", "tenant_id", tenantID, "err", err)
			wg.tenantHasListener[tenantID] = true
		} else {
			wg.tenantHasListener[tenantID] = has
		}
	}

	if _, ok := wg.tenantDisabledKeys[tenantID]; !ok {
		disabledKeys, err := wg.tenantEventTypeRepo.FindDisabledKeysByTenantID(tenantID)
		if err != nil {
			slog.Error("write gate: load disabled keys failed", "tenant_id", tenantID, "err", err)
			wg.tenantDisabledKeys[tenantID] = make(map[string]bool)
		} else {
			disabledMap := make(map[string]bool, len(disabledKeys))
			for _, k := range disabledKeys {
				disabledMap[k] = true
			}
			wg.tenantDisabledKeys[tenantID] = disabledMap
		}
	}
}

// needsRefresh reports whether the TTL-only cache is stale. Reads lastRefresh
// under the read lock to avoid a data race with refreshActiveTypes.
func (wg *WriteGate) needsRefresh() bool {
	if !wg.ttlOnly {
		return false
	}
	wg.mu.RLock()
	defer wg.mu.RUnlock()
	return time.Since(wg.lastRefresh) > writeGateCacheTTL
}

func (wg *WriteGate) refreshActiveTypes(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Warn("write gate: refresh active types panicked", "panic", r)
		}
	}()

	active, err := wg.eventTypeRepo.FindAllActive()
	if err != nil {
		slog.Error("write gate: load active event types failed", "err", err)
		return
	}

	activeMap := make(map[string]bool, len(active))
	for _, et := range active {
		activeMap[et.Key] = true
	}

	wg.mu.Lock()
	wg.activeTypes = activeMap
	wg.lastRefresh = time.Now()
	wg.mu.Unlock()
}

// InvalidateTenant removes cached state for a specific tenant.
func (wg *WriteGate) InvalidateTenant(ctx context.Context, tenantID int64) {
	wg.mu.Lock()
	delete(wg.tenantDisabledKeys, tenantID)
	delete(wg.tenantHasListener, tenantID)
	wg.mu.Unlock()

	if wg.rdb != nil {
		// Best-effort cross-replica invalidation; log failures so a dropped
		// publish (Redis blip) that leaves other replicas stale is visible.
		if err := wg.rdb.Publish(ctx, writeGateInvalidationChan, tenantID).Err(); err != nil {
			slog.Error("write gate: publish invalidation failed", "tenant_id", tenantID, "err", err)
		}
	}
}

// Shutdown stops background goroutines.
func (wg *WriteGate) Shutdown() {
	close(wg.stopCh)
	if wg.sub != nil {
		_ = wg.sub.Close()
	}
}
