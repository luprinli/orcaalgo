package ingest

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TickerWriter struct {
	pool  *pgxpool.Pool
	batch []*GoMarketTick
	size  int
	timer *time.Ticker
}

func NewTickerWriter(pool *pgxpool.Pool, batchSize int, flushInterval time.Duration) *TickerWriter {
	return &TickerWriter{
		pool:  pool,
		batch: make([]*GoMarketTick, 0, batchSize),
		size:  batchSize,
		timer: time.NewTicker(flushInterval),
	}
}

func (w *TickerWriter) Write(ctx context.Context, tick *GoMarketTick) {
	w.batch = append(w.batch, tick)
	if len(w.batch) >= w.size {
		w.flush(ctx)
	}
}

func (w *TickerWriter) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.flush(ctx)
			return
		case <-w.timer.C:
			w.flush(ctx)
		}
	}
}

func (w *TickerWriter) flush(ctx context.Context) {
	if len(w.batch) == 0 {
		return
	}

	batch := w.batch
	w.batch = make([]*GoMarketTick, 0, w.size)

	_, err := w.pool.CopyFrom(
		ctx,
		pgx.Identifier{"market_ticks"},
		[]string{"time", "symbol_id", "price_raw", "volume_raw", "bid_price", "ask_price", "bid_size", "ask_size"},
		pgx.CopyFromSlice(len(batch), func(i int) ([]interface{}, error) {
			tick := batch[i]
			return []interface{}{
				time.Unix(0, tick.Timestamp),
				tick.SymbolID,
				tick.PriceRaw,
				tick.VolumeRaw,
				tick.BidPrice,
				tick.AskPrice,
				tick.BidSize,
				tick.AskSize,
			}, nil
		}),
	)
	if err != nil {
		log.Printf("tick writing error: %v", err)
	}
}
