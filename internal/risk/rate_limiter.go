package risk

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
)

type DynamicRateLimiter struct {
	client       *redis.Client
	engineState  EngineStateReader
	mu           sync.Mutex
	failCount    int64
	lastSuccess  int64
	circuitOpen  int32
}

type EngineStateReader interface {
	GetCurrentRegime() int8
}

func NewDynamicRateLimiter(client *redis.Client, engine EngineStateReader) *DynamicRateLimiter {
	return &DynamicRateLimiter{
		client:      client,
		engineState: engine,
	}
}

func (l *DynamicRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	if atomic.LoadInt32(&l.circuitOpen) == 1 {
		return false, fmt.Errorf("rate limiter circuit open: Redis unavailable")
	}

	regime := l.engineState.GetCurrentRegime()

	var limit int
	switch {
	case regime >= 3:
		limit = 5
	case regime == 2:
		limit = 20
	default:
		limit = 100
	}

	pipe := l.client.Pipeline()
	now := time.Now().Unix()
	windowStart := now - 60

	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
	cardCmd := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, &redis.Z{Score: float64(now), Member: fmt.Sprintf("%d", now)})
	pipe.Expire(ctx, key, 2*time.Minute)

	if _, err := pipe.Exec(ctx); err != nil {
		fails := atomic.AddInt64(&l.failCount, 1)
		if fails > 5 {
			atomic.StoreInt32(&l.circuitOpen, 1)
		}
		return false, fmt.Errorf("rate limiter: Redis pipeline failed: %w", err)
	}

	atomic.StoreInt64(&l.failCount, 0)
	atomic.StoreInt32(&l.circuitOpen, 0)
	atomic.StoreInt64(&l.lastSuccess, time.Now().Unix())

	return cardCmd.Val() < int64(limit), nil
}

func (l *DynamicRateLimiter) IsCircuitOpen() bool {
	if atomic.LoadInt32(&l.circuitOpen) == 1 {
		if time.Now().Unix()-atomic.LoadInt64(&l.lastSuccess) > 60 {
			atomic.StoreInt32(&l.circuitOpen, 0)
			atomic.StoreInt64(&l.failCount, 0)
			return false
		}
		return true
	}
	return false
}

func (l *DynamicRateLimiter) ResetCircuit() {
	atomic.StoreInt32(&l.circuitOpen, 0)
	atomic.StoreInt64(&l.failCount, 0)
}
