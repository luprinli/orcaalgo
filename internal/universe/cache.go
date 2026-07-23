package universe

import (
	"sync"
	"time"

	"github.com/lee-econ/orca-core/internal/db"
)

type UniverseCache struct {
	mu         sync.RWMutex
	symbols    []db.Symbol
	expiresAt  time.Time
	ttl        time.Duration
	configHash string
}

func NewUniverseCache(ttl time.Duration) *UniverseCache {
	return &UniverseCache{
		ttl: ttl,
	}
}

func (c *UniverseCache) Get() ([]db.Symbol, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.symbols == nil || time.Now().After(c.expiresAt) {
		return nil, false
	}
	result := make([]db.Symbol, len(c.symbols))
	copy(result, c.symbols)
	return result, true
}

func (c *UniverseCache) Set(symbols []db.Symbol, configHash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.symbols = make([]db.Symbol, len(symbols))
	copy(c.symbols, symbols)
	c.configHash = configHash
	c.expiresAt = time.Now().Add(c.ttl)
}

func (c *UniverseCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.symbols = nil
	c.expiresAt = time.Time{}
}
