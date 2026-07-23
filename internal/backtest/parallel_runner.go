package backtest

import (
	"container/heap"
	"context"
	"sync"

	strategy "github.com/lee-econ/orca-core/internal/strategy"
)

func (e *Engine) RunParallel(ctx context.Context, configs []BacktestConfig) map[string]*BacktestResult {
	results := make(map[string]*BacktestResult)
	if e.db == nil {
		return results
	}
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, config := range configs {
		wg.Add(1)
		go func(cfg BacktestConfig) {
			defer wg.Done()
			r, err := e.Run(ctx, cfg)
			if err == nil {
			mu.Lock()
			for _, sym := range cfg.Symbols {
				results[sym] = r
			}
			mu.Unlock()
			}
		}(config)
	}

	wg.Wait()
	return results
}

type TickQueue struct {
	items []*TickItem
}

type TickItem struct {
	Timestamp int64
	Symbol    string
	Candle    strategy.Candle
	index     int
}

func (tq *TickQueue) Len() int           { return len(tq.items) }
func (tq *TickQueue) Less(i, j int) bool { return tq.items[i].Timestamp < tq.items[j].Timestamp }
func (tq *TickQueue) Swap(i, j int) {
	tq.items[i], tq.items[j] = tq.items[j], tq.items[i]
	tq.items[i].index = i
	tq.items[j].index = j
}

func (tq *TickQueue) Push(x interface{}) {
	n := len(tq.items)
	item := x.(*TickItem)
	item.index = n
	tq.items = append(tq.items, item)
}

func (tq *TickQueue) Pop() interface{} {
	old := tq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	tq.items = old[0 : n-1]
	return item
}

func (tq *TickQueue) PushTick(symbol string, candle strategy.Candle) {
	heap.Push(tq, &TickItem{Timestamp: candle.Time.UnixNano(), Symbol: symbol, Candle: candle})
}

func (tq *TickQueue) PopTick() (string, strategy.Candle) {
	if tq.Len() == 0 {
		return "", strategy.Candle{}
	}
	item := heap.Pop(tq).(*TickItem)
	return item.Symbol, item.Candle
}

func (tq *TickQueue) PushCandles(symbol string, candles []strategy.Candle) {
	for i := range candles {
		tq.PushTick(symbol, candles[i])
	}
}
