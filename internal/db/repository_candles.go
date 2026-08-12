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
	var result [][]Candle
	var scanErrors []string
	for _, sym := range symbols {
		rows, err := r.pool.Query(ctx,
			`SELECT c.time, c.open_raw, c.high_raw, c.low_raw, c.close_raw, c.volume
			 FROM candles c
			 JOIN symbols s ON c.symbol_id = s.id
			 WHERE s.ticker = $1 AND c.time >= $2 AND c.time <= $3
			 ORDER BY c.time ASC`, sym, start, end,
		)
		if err != nil {
			scanErrors = append(scanErrors, fmt.Sprintf("%s: query err (%v)", sym, err))
			continue
		}
		var candles []Candle
		rowErrors := 0
		for rows.Next() {
			var c Candle
			var openRaw, highRaw, lowRaw, closeRaw, vol int64
			if err := rows.Scan(&c.Time, &openRaw, &highRaw, &lowRaw, &closeRaw, &vol); err != nil {
				rowErrors++
				continue
			}
			c.Open = types.PriceFromFloat(float64(openRaw) / PRICE_SCALE_F)
			c.High = types.PriceFromFloat(float64(highRaw) / PRICE_SCALE_F)
			c.Low = types.PriceFromFloat(float64(lowRaw) / PRICE_SCALE_F)
			c.Close = types.PriceFromFloat(float64(closeRaw) / PRICE_SCALE_F)
			c.Volume = float64(vol)
			c.Symbol = sym
			candles = append(candles, c)
		}
		rows.Close()
		if rowErrors > 0 {
			scanErrors = append(scanErrors, fmt.Sprintf("%s: %d row scan errors", sym, rowErrors))
		}
		result = append(result, candles)
	}
	if len(symbols) > 0 && len(result) == 0 {
		return nil, fmt.Errorf("LoadCandles: all %d symbols failed to load data (%v)", len(symbols), scanErrors)
	}
	return result, nil
}

func (r *Repository) LoadCandlesByTimeframe(ctx context.Context, symbols []string, start, end time.Time, timeframe string) ([][]Candle, error) {
	if timeframe == "" || timeframe == "1d" {
		return r.LoadCandles(ctx, symbols, start, end)
	}
	var result [][]Candle
	var scanErrors []string
	for _, sym := range symbols {
		rows, err := r.pool.Query(ctx,
			`SELECT c.time, c.open_raw, c.high_raw, c.low_raw, c.close_raw, c.volume
			 FROM candles c
			 JOIN symbols s ON c.symbol_id = s.id
			 WHERE s.ticker = $1 AND c.time >= $2 AND c.time <= $3 AND c.timeframe = $4
			 ORDER BY c.time ASC`, sym, start, end, timeframe,
		)
		if err != nil {
			scanErrors = append(scanErrors, fmt.Sprintf("%s/%s: query err (%v)", sym, timeframe, err))
			continue
		}
		var candles []Candle
		rowErrors := 0
		for rows.Next() {
			var c Candle
			var openRaw, highRaw, lowRaw, closeRaw, vol int64
			if err := rows.Scan(&c.Time, &openRaw, &highRaw, &lowRaw, &closeRaw, &vol); err != nil {
				rowErrors++
				continue
			}
			c.Open = types.PriceFromFloat(float64(openRaw) / PRICE_SCALE_F)
			c.High = types.PriceFromFloat(float64(highRaw) / PRICE_SCALE_F)
			c.Low = types.PriceFromFloat(float64(lowRaw) / PRICE_SCALE_F)
			c.Close = types.PriceFromFloat(float64(closeRaw) / PRICE_SCALE_F)
			c.Volume = float64(vol)
			c.Symbol = sym
			candles = append(candles, c)
		}
		rows.Close()
		if rowErrors > 0 {
			scanErrors = append(scanErrors, fmt.Sprintf("%s/%s: %d row scan errors", sym, timeframe, rowErrors))
		}
		result = append(result, candles)
	}
	if len(symbols) > 0 && len(result) == 0 {
		return nil, fmt.Errorf("LoadCandlesByTimeframe: all %d symbols failed to load data for timeframe %s (%v)", len(symbols), timeframe, scanErrors)
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
	query := `SELECT c.time, c.open_raw, c.high_raw, c.low_raw, c.close_raw, c.volume, s.ticker
		 FROM candles c JOIN symbols s ON c.symbol_id = s.id
		 WHERE s.ticker = ANY($1) AND c.time BETWEEN $2 AND $3`
	args := []interface{}{symbols, start, end}
	if timeframe != "" && timeframe != "1d" {
		query += ` AND c.timeframe = $4 ORDER BY s.ticker, c.time ASC`
		args = append(args, timeframe)
	} else {
		query += ` ORDER BY s.ticker, c.time ASC`
	}
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
		if err := rows.Scan(&c.Time, &openRaw, &highRaw, &lowRaw, &closeRaw, &vol, &ticker); err != nil {
			continue
		}
		c.Open = types.PriceFromFloat(float64(openRaw) / PRICE_SCALE_F)
		c.High = types.PriceFromFloat(float64(highRaw) / PRICE_SCALE_F)
		c.Low = types.PriceFromFloat(float64(lowRaw) / PRICE_SCALE_F)
		c.Close = types.PriceFromFloat(float64(closeRaw) / PRICE_SCALE_F)
		c.Volume = float64(vol)
		c.Symbol = ticker
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
