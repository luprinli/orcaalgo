package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/lee-econ/orca-core/internal/types"
	"github.com/lee-econ/orca-core/pkg/temporal"
)

func (r *Repository) LoadCandles(ctx context.Context, symbols []string, start, end time.Time) ([][]Candle, error) {
	return r.loadCandles(ctx, symbols, start, end, "1d", nil, "LoadCandles")
}

// LoadCandlesFiltered loads 1d candles, optionally restricted to the given
// logical data source. An empty source matches all sources.
func (r *Repository) LoadCandlesFiltered(ctx context.Context, symbols []string, start, end time.Time, source string) ([][]Candle, error) {
	return r.loadCandles(ctx, symbols, start, end, "1d", SourceValues(source), "LoadCandlesFiltered")
}

func (r *Repository) LoadCandlesByTimeframe(ctx context.Context, symbols []string, start, end time.Time, timeframe string) ([][]Candle, error) {
	return r.loadCandles(ctx, symbols, start, end, timeframe, nil, "LoadCandlesByTimeframe")
}

// LoadCandlesByTimeframeFiltered loads candles for a timeframe, optionally
// restricted to the given logical data source. An empty source matches all
// sources. timeframe "" or "1d" selects daily bars.
func (r *Repository) LoadCandlesByTimeframeFiltered(ctx context.Context, symbols []string, start, end time.Time, timeframe, source string) ([][]Candle, error) {
	return r.loadCandles(ctx, symbols, start, end, timeframe, SourceValues(source), "LoadCandlesByTimeframeFiltered")
}

// SourceValues maps a logical data-source selector to the set of candle
// `source` values to include. Returns nil (no filter) for empty/all/any.
//
// The "stooq" selector represents the real-data pipeline exclusively:
// stooq (real raw bars, including daily), stooq-resampled (derived from real
// stooq), and stooq-calibrated (synthetic gap-fill calibrated from stooq σ/μ).
//
// The legacy "seed" development fixture and the "yahoo" provider are NOT part
// of the stooq selector. Merging them with real stooq bars produced ~7-10x
// price-scale discontinuities (NVDA, ^_US) and absurd backtest results. The
// loadCandles query additionally applies a source-priority DISTINCT ON so the
// highest-priority source wins per bar even when no filter is supplied.
func SourceValues(source string) []string {
	switch source {
	case "", "all", "any":
		return nil
	case "stooq":
		return []string{"stooq", "stooq-resampled", "stooq-calibrated"}
	case "yahoo":
		return []string{"yahoo"}
	case "seed":
		return []string{"seed"}
	default:
		return []string{source}
	}
}

// loadCandles is the shared loader backing LoadCandles, LoadCandlesFiltered,
// LoadCandlesByTimeframe, and LoadCandlesByTimeframeFiltered. sourceValues nil
// means no source restriction; timeframe "" or "1d" selects daily bars.
func (r *Repository) loadCandles(ctx context.Context, symbols []string, start, end time.Time, timeframe string, sourceValues []string, caller string) ([][]Candle, error) {
	if timeframe == "" || timeframe == "1d" {
		timeframe = "1d"
	}
	var result [][]Candle
	var scanErrors []string
	for _, sym := range symbols {
		// DISTINCT ON (c.time) + source-priority ordering loads the
		// highest-priority source per bar instead of merging every source.
		// Priority: stooq (real) > stooq-resampled > stooq-calibrated >
		// yahoo > seed. This prevents incompatible price scales from
		// coexisting in a single series.
		query := `SELECT d.time, d.open_raw, d.high_raw, d.low_raw, d.close_raw, d.volume, d.source, d.generation_id
			 FROM (
				SELECT DISTINCT ON (c.time)
					c.time, c.open_raw, c.high_raw, c.low_raw, c.close_raw, c.volume,
					c.source, c.generation_id
				FROM candles c
				JOIN symbols s ON c.symbol_id = s.id
				WHERE s.ticker = $1 AND c.time >= $2 AND c.time <= $3 AND c.timeframe = $4`
		args := []interface{}{sym, start, end, timeframe}
		if len(sourceValues) > 0 {
			query += ` AND c.source = ANY($5)`
			args = append(args, sourceValues)
		}
		query += ` ORDER BY c.time ASC,
				CASE c.source
					WHEN 'stooq' THEN 0
					WHEN 'stooq-resampled' THEN 1
					WHEN 'stooq-calibrated' THEN 2
					WHEN 'yahoo' THEN 3
					WHEN 'seed' THEN 4
					ELSE 5
				END ASC, c.source ASC
			 ) d
			 ORDER BY d.time ASC`

		// Transient pool/connection failures under matrix concurrency can drop a
		// read-only candle query; retry with exponential backoff before treating it
		// as a real data gap.
		var rows pgx.Rows
		var err error
		const maxCandleQueryRetries = 5
		for attempt := 0; attempt < maxCandleQueryRetries; attempt++ {
			rows, err = r.pool.Query(ctx, query, args...)
			if err == nil {
				break
			}
			if attempt >= maxCandleQueryRetries-1 || ctx.Err() != nil {
				break
			}
			time.Sleep(time.Duration(1<<attempt) * 100 * time.Millisecond)
		}
		if err != nil {
			scanErrors = append(scanErrors, fmt.Sprintf("%s/%s: query err (%v)", sym, timeframe, err))
			continue
		}
		var candles []Candle
		rowErrors := 0
		for rows.Next() {
			var c Candle
			var openRaw, highRaw, lowRaw, closeRaw, vol int64
			var source string
			var generationID *string
			if err := rows.Scan(&c.Time, &openRaw, &highRaw, &lowRaw, &closeRaw, &vol, &source, &generationID); err != nil {
				rowErrors++
				continue
			}
			c.Open = types.PriceFromFloat(float64(openRaw) / PRICE_SCALE_F)
			c.High = types.PriceFromFloat(float64(highRaw) / PRICE_SCALE_F)
			c.Low = types.PriceFromFloat(float64(lowRaw) / PRICE_SCALE_F)
			c.Close = types.PriceFromFloat(float64(closeRaw) / PRICE_SCALE_F)
			c.Volume = float64(vol)
			c.Symbol = sym
			c.Source = source
			if generationID != nil {
				c.GenerationID = *generationID
			}
			// Default identity factor; corporate-action splits are applied on
			// load below (see corporate_actions.go).
			c.AdjustmentFactor = 1.0
			candles = append(candles, c)
		}
		rows.Close()
		if rowErrors > 0 {
			scanErrors = append(scanErrors, fmt.Sprintf("%s/%s: %d row scan errors", sym, timeframe, rowErrors))
		}
		// Apply corporate-action adjustment factors (splits/dividends) on load,
		// replacing the identity factor once corporate-action data is ingested.
		if actions, err := r.LoadCorporateActions(ctx, sym); err == nil {
			candles = ApplyCorporateActions(candles, actions)
		}
		result = append(result, candles)
	}
	if len(symbols) > 0 && len(result) == 0 {
		return nil, fmt.Errorf("%s: all %d symbols failed to load data for timeframe %s (%v)", caller, len(symbols), timeframe, scanErrors)
	}
	return result, nil
}

func (r *Repository) LoadCandlesUpTo(ctx context.Context, symbols []string, start, end time.Time) ([][]Candle, error) {
	if maxTime, ok := temporal.GetMaxTime(ctx); ok && maxTime.Before(end) {
		end = maxTime
	}
	return r.LoadCandles(ctx, symbols, start, end)
}

func (r *Repository) LoadAllCandles(ctx context.Context, symbols []string, start, end time.Time, timeframe string) (map[string][]Candle, error) {
	query := `SELECT d.time, d.open_raw, d.high_raw, d.low_raw, d.close_raw, d.volume, d.ticker, d.source, d.generation_id
		 FROM (
			SELECT DISTINCT ON (c.symbol_id, c.time)
				c.time, c.open_raw, c.high_raw, c.low_raw, c.close_raw, c.volume, s.ticker, c.source, c.generation_id, c.symbol_id
			FROM candles c JOIN symbols s ON c.symbol_id = s.id
			WHERE s.ticker = ANY($1) AND c.time BETWEEN $2 AND $3`
	args := []interface{}{symbols, start, end}
	if timeframe != "" && timeframe != "1d" {
		query += ` AND c.timeframe = $4`
		args = append(args, timeframe)
	}
	query += ` ORDER BY c.symbol_id ASC, c.time ASC,
			CASE c.source
				WHEN 'stooq' THEN 0
				WHEN 'stooq-resampled' THEN 1
				WHEN 'stooq-calibrated' THEN 2
				WHEN 'yahoo' THEN 3
				WHEN 'seed' THEN 4
				ELSE 5
			END ASC, c.source ASC
		 ) d
		 ORDER BY d.ticker ASC, d.time ASC`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]Candle)
	for rows.Next() {
		var c Candle
		var ticker string
		var openRaw, highRaw, lowRaw, closeRaw, vol int64
		var source string
		var generationID *string
		if err := rows.Scan(&c.Time, &openRaw, &highRaw, &lowRaw, &closeRaw, &vol, &ticker, &source, &generationID); err != nil {
			continue
		}
		c.Open = types.PriceFromFloat(float64(openRaw) / PRICE_SCALE_F)
		c.High = types.PriceFromFloat(float64(highRaw) / PRICE_SCALE_F)
		c.Low = types.PriceFromFloat(float64(lowRaw) / PRICE_SCALE_F)
		c.Close = types.PriceFromFloat(float64(closeRaw) / PRICE_SCALE_F)
		c.Volume = float64(vol)
		c.Symbol = ticker
		c.Source = source
		if generationID != nil {
			c.GenerationID = *generationID
		}
		result[ticker] = append(result[ticker], c)
	}
	return result, nil
}

func (r *Repository) CountCandles(ctx context.Context) (int64, error) {
	if r.pool == nil {
		return 0, fmt.Errorf("no pool")
	}
	var count int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM candles WHERE source != 'synthetic'").Scan(&count)
	return count, err
}

func (r *Repository) CountSyntheticCandles(ctx context.Context) (int64, error) {
	if r.pool == nil {
		return 0, fmt.Errorf("no pool")
	}
	var count int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM candles WHERE source = 'synthetic'").Scan(&count)
	return count, err
}

func (r *Repository) CountRegimeLogs(ctx context.Context) (int64, error) {
	if r.pool == nil {
		return 0, fmt.Errorf("no pool")
	}
	var count int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM regime_logs").Scan(&count)
	return count, err
}

func (r *Repository) CountTable(ctx context.Context, tableName string) (int64, error) {
	if r.pool == nil {
		return 0, fmt.Errorf("no pool")
	}
	var count int64
	err := r.pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", pgx.Identifier{tableName}.Sanitize())).Scan(&count)
	return count, err
}

func (r *Repository) BatchInsertTicks(ctx context.Context, ticks [][]interface{}) error {
	_, err := r.pool.CopyFrom(
		ctx,
		pgx.Identifier{"market_ticks"},
		[]string{"time", "symbol_id", "price_raw", "volume_raw", "bid_price", "ask_price", "bid_size", "ask_size"},
		pgx.CopyFromRows(ticks),
	)
	return err
}

func (r *Repository) InsertTradeExecution(ctx context.Context, t *TradeExecution) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO trade_executions (strategy_id, symbol, side, quantity, price, hmm_regime, risk_approved, consistency_multiplier, rejected_reason, broker_order_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		t.StrategyID, t.Symbol, t.Side, t.Quantity, t.Price.Int64(),
		t.HMMRegime, t.RiskApproved, t.ConsistencyMultiplier, t.RejectedReason, t.BrokerOrderID,
	)
	return err
}
