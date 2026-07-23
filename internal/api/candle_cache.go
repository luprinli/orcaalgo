package api

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/lee-econ/orca-core/internal/backtest"
	"golang.org/x/sync/semaphore"
)

// globalMatrixSem bounds the TOTAL number of concurrent backtests across ALL
// matrix batches (not per-batch). Without this, each submitted batch spawned its
// own worker pool, so N concurrent batches ran N×workers backtests and could OOM
// the host (observed as "runtime: cannot allocate memory"). A single shared pool
// makes total concurrency — and therefore peak memory — independent of how many
// batches are in flight (execution plan §4.1: single dispatcher/worker pool).
var (
	globalMatrixSem     *semaphore.Weighted
	globalMatrixSemOnce sync.Once
)

func matrixSemaphore() *semaphore.Weighted {
	globalMatrixSemOnce.Do(func() {
		globalMatrixSem = semaphore.NewWeighted(int64(backtest.MatrixWorkers(20)))
	})
	return globalMatrixSem
}

// cachingRepoAdapter wraps a backtest.Database with a bounded LRU cache of candle
// loads. Combined with cheapest-first, cache-grouped combo ordering, this keeps
// candle memory bounded to a small number of (symbols,range,tf,source) sets
// regardless of matrix size — the "bounded peak" property from the execution plan
// (§2.3/§3.4, Risk B/C). Cached slices are read-only for the engine.
type cachingRepoAdapter struct {
	backtest.Database
	mu    sync.Mutex
	cap   int
	ll    *list.List               // MRU at front
	items map[string]*list.Element // key -> element
}

type cacheEntry struct {
	key    string
	candle [][]backtest.Candle
}

func newCachingRepoAdapter(base backtest.Database) *cachingRepoAdapter {
	return &cachingRepoAdapter{
		Database: base,
		cap:      backtest.CandleCacheCap(),
		ll:       list.New(),
		items:    make(map[string]*list.Element),
	}
}

func candleKey(symbols []string, start, end time.Time, timeframe, source string) string {
	key := timeframe + "|" + source + "|" +
		start.Format("2006-01-02") + "|" + end.Format("2006-01-02") + "|"
	for _, s := range symbols {
		key += s + ","
	}
	return key
}

func (a *cachingRepoAdapter) get(key string) ([][]backtest.Candle, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if el, ok := a.items[key]; ok {
		a.ll.MoveToFront(el)
		return el.Value.(*cacheEntry).candle, true
	}
	return nil, false
}

func (a *cachingRepoAdapter) put(key string, v [][]backtest.Candle) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if el, ok := a.items[key]; ok {
		a.ll.MoveToFront(el)
		el.Value.(*cacheEntry).candle = v
		return
	}
	el := a.ll.PushFront(&cacheEntry{key: key, candle: v})
	a.items[key] = el
	for a.ll.Len() > a.cap {
		back := a.ll.Back()
		if back == nil {
			break
		}
		a.ll.Remove(back)
		delete(a.items, back.Value.(*cacheEntry).key)
	}
}

func (a *cachingRepoAdapter) LoadCandlesFiltered(ctx context.Context, symbols []string, start, end time.Time, source string) ([][]backtest.Candle, error) {
	key := candleKey(symbols, start, end, "", source)
	if v, ok := a.get(key); ok {
		return v, nil
	}
	v, err := a.Database.LoadCandlesFiltered(ctx, symbols, start, end, source)
	if err != nil {
		return nil, err
	}
	a.put(key, v)
	return v, nil
}

func (a *cachingRepoAdapter) LoadCandlesTFFiltered(ctx context.Context, symbols []string, start, end time.Time, timeframe, source string) ([][]backtest.Candle, error) {
	key := candleKey(symbols, start, end, timeframe, source)
	if v, ok := a.get(key); ok {
		return v, nil
	}
	v, err := a.Database.LoadCandlesTFFiltered(ctx, symbols, start, end, timeframe, source)
	if err != nil {
		return nil, err
	}
	a.put(key, v)
	return v, nil
}
