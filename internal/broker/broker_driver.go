package broker

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
)

type adapterEntry struct {
	adapter  CapableAdapter
	manifest AdapterManifest
	healthy  bool
	lastErr  error
	lastSeen time.Time
}

// BrokerDriverRegistry implements capability-based adapter routing with
// priority-ordered fallback, inspired by Opptrix's provider plugin system.
//
// Adapters register with a manifest declaring their capabilities and
// priority. Requests are routed to the highest-priority healthy adapter
// that supports the requested capability. If the primary adapter is
// unhealthy or returns an error, the registry automatically falls back
// to the next adapter in priority order.
type BrokerDriverRegistry struct {
	mu       sync.RWMutex
	entries  []*adapterEntry
	byID     map[string]*adapterEntry
}

func NewBrokerDriverRegistry() *BrokerDriverRegistry {
	return &BrokerDriverRegistry{
		byID: make(map[string]*adapterEntry),
	}
}

// Register adds a capable adapter to the registry. Adapters with the
// same ID overwrite previous registrations.
func (r *BrokerDriverRegistry) Register(adapter CapableAdapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	m := adapter.Manifest()
	if m.ID == "" {
		return fmt.Errorf("adapter manifest missing ID")
	}

	entry := &adapterEntry{
		adapter:  adapter,
		manifest: m,
		healthy:  false, // will be set by health check
		lastSeen: time.Now(),
	}

	if old, exists := r.byID[m.ID]; exists {
		r.removeEntry(old)
	}
	r.byID[m.ID] = entry
	r.entries = append(r.entries, entry)
	r.sortByPriority()
	return nil
}

// Unregister removes an adapter by ID.
func (r *BrokerDriverRegistry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry, ok := r.byID[id]; ok {
		r.removeEntry(entry)
		delete(r.byID, id)
	}
}

func (r *BrokerDriverRegistry) removeEntry(entry *adapterEntry) {
	for i, e := range r.entries {
		if e == entry {
			r.entries = append(r.entries[:i], r.entries[i+1:]...)
			return
		}
	}
}

func (r *BrokerDriverRegistry) sortByPriority() {
	sort.SliceStable(r.entries, func(i, j int) bool {
		return r.entries[i].manifest.Priority < r.entries[j].manifest.Priority
	})
}

// candidates returns all registered adapters that support the given
// capability, sorted by priority (lowest = most preferred). Caller
// must hold r.mu.
func (r *BrokerDriverRegistry) candidates(cap Capability) []*adapterEntry {
	var out []*adapterEntry
	for _, e := range r.entries {
		if e.supports(cap) {
			out = append(out, e)
		}
	}
	return out
}

func (e *adapterEntry) supports(cap Capability) bool {
	for _, c := range e.manifest.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// Resolve returns the best healthy adapter for the given capability.
// Tries adapters in priority order; returns an error if no adapter
// supports the capability or all supporting adapters are unhealthy.
func (r *BrokerDriverRegistry) Resolve(ctx context.Context, cap Capability) (CapableAdapter, error) {
	r.mu.RLock()
	cands := r.candidates(cap)
	r.mu.RUnlock()

	if len(cands) == 0 {
		return nil, fmt.Errorf("no adapter registered for capability %s", cap)
	}

	for _, e := range cands {
		r.mu.RLock()
		healthy := e.healthy
		r.mu.RUnlock()

		if healthy {
			return e.adapter, nil
		}
	}

	return nil, fmt.Errorf("all %d adapter(s) for capability %s are unhealthy", len(cands), cap)
}

// ResolveWithFallback returns the best healthy adapter and executes fn.
// If fn returns an error, the registry marks the adapter unhealthy and
// retries with the next adapter. Returns the result of the first
// successful fn, or the last error if all adapters fail.
func (r *BrokerDriverRegistry) ResolveWithFallback(
	ctx context.Context,
	cap Capability,
	fn func(Adapter) error,
) error {
	for attempt := 0; attempt < 3; attempt++ {
		r.mu.RLock()
		cands := r.candidates(cap)
		r.mu.RUnlock()

		for _, e := range cands {
			r.mu.RLock()
			healthy := e.healthy
			r.mu.RUnlock()

			if !healthy {
				continue
			}

			if err := fn(e.adapter); err != nil {
				r.mu.Lock()
				e.healthy = false
				e.lastErr = err
				r.mu.Unlock()
				slog.Error("operation failed, marking unhealthy", "broker_id", e.manifest.ID, "error", err, "component", "broker")
				break // try next adapter (or re-resolve if health changed)
			} else {
				return nil
			}
		}

		// No healthy adapter found this round — run health checks to
		// potentially recover adapters, then retry.
		if attempt < 2 {
			r.RunHealthChecks(ctx)
		}
	}

	return fmt.Errorf("all adapters for capability %s failed after retries", cap)
}

// RunHealthChecks pings every registered adapter. Healthy adapters
// are promoted; unhealthy ones are demoted. This is intended to be
// called periodically by the scheduler and on-demand after failures.
func (r *BrokerDriverRegistry) RunHealthChecks(ctx context.Context) {
	r.mu.RLock()
	entries := make([]*adapterEntry, len(r.entries))
	copy(entries, r.entries)
	r.mu.RUnlock()

	for _, e := range entries {
		hcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := e.adapter.HealthCheck(hcCtx)
		cancel()

		r.mu.Lock()
		e.lastSeen = time.Now()
		if err != nil {
			e.healthy = false
			e.lastErr = err
		} else {
			e.healthy = true
			e.lastErr = nil
		}
		r.mu.Unlock()
	}
}

// Get returns an adapter by ID (for direct access when capability
// routing is not needed). Returns the adapter regardless of health.
func (r *BrokerDriverRegistry) Get(id string) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.byID[id]
	if !ok {
		return nil, false
	}
	return e.adapter, ok
}

// List returns all registered adapter IDs with their broker type,
// priority, and health status.
func (r *BrokerDriverRegistry) List() map[string]AdapterStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]AdapterStatus, len(r.entries))
	for _, e := range r.entries {
		out[e.manifest.ID] = AdapterStatus{
			BrokerType: e.manifest.BrokerType,
			Priority:   e.manifest.Priority,
			Healthy:    e.healthy,
			LastError:  e.lastErr,
			LastSeen:   e.lastSeen,
		}
	}
	return out
}

type AdapterStatus struct {
	BrokerType string
	Priority   int
	Healthy    bool
	LastError  error
	LastSeen   time.Time
}

// CancelAllOrders calls CancelAllOrders on every registered adapter.
func (r *BrokerDriverRegistry) CancelAllOrders(ctx context.Context) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, e := range r.entries {
		if err := e.adapter.CancelAllOrders(ctx); err != nil {
			slog.Error("CancelAllOrders failed", "broker_id", e.manifest.ID, "error", err, "component", "broker")
		}
	}
}

// CloseAllPositions calls CloseAllPositions on every registered adapter.
func (r *BrokerDriverRegistry) CloseAllPositions(ctx context.Context) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, e := range r.entries {
		if err := e.adapter.CloseAllPositions(ctx); err != nil {
			slog.Error("CloseAllPositions failed", "broker_id", e.manifest.ID, "error", err, "component", "broker")
		}
	}
}
