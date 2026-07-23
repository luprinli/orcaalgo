package ingest

import (
	"fmt"
	"sync"
)

type DataFeed interface {
	Connect() error
	Close() error
	IsConnected() bool
}

type Registry struct {
	mu    sync.RWMutex
	feeds map[string]DataFeed
}

func NewRegistry() *Registry {
	return &Registry{
		feeds: make(map[string]DataFeed),
	}
}

func (r *Registry) Register(id string, feed DataFeed) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.feeds[id]; exists {
		return fmt.Errorf("data feed %s already registered", id)
	}
	r.feeds[id] = feed
	return nil
}

func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if feed, ok := r.feeds[id]; ok {
		feed.Close()
		delete(r.feeds, id)
	}
}

func (r *Registry) Get(id string) (DataFeed, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.feeds[id]
	return f, ok
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.feeds))
	for id := range r.feeds {
		ids = append(ids, id)
	}
	return ids
}

func (r *Registry) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, feed := range r.feeds {
		feed.Close()
		delete(r.feeds, id)
	}
}
