package broker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
	status   map[string]string
}

func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]Adapter),
		status:   make(map[string]string),
	}
}

func (r *Registry) Register(id string, adapter Adapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.adapters[id]; exists {
		return fmt.Errorf("broker %s already registered", id)
	}
	r.adapters[id] = adapter
	r.status[id] = "disconnected"
	return nil
}

func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.adapters, id)
	delete(r.status, id)
}

func (r *Registry) Get(id string) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.adapters[id]
	return a, ok
}

func (r *Registry) SetStatus(id, status string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status[id] = status
}

func (r *Registry) GetStatus(id string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.status[id]
}

func (r *Registry) List() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]string, len(r.adapters))
	for id, st := range r.status {
		result[id] = st
	}
	return result
}

func (r *Registry) CancelAllOrders(ctx context.Context) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for id, adapter := range r.adapters {
		if err := adapter.CancelAllOrders(ctx); err != nil {
			slog.Error("CancelAllOrders failed", "broker_id", id, "error", err, "component", "broker")
		}
	}
}

func (r *Registry) CloseAllPositions(ctx context.Context) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for id, adapter := range r.adapters {
		if err := adapter.CloseAllPositions(ctx); err != nil {
			slog.Error("CloseAllPositions failed", "broker_id", id, "error", err, "component", "broker")
		}
	}
}
