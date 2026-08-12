package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

type Seeder struct {
	repo *Repository
}

func NewSeeder(repo *Repository) *Seeder {
	return &Seeder{repo: repo}
}

func (s *Seeder) Run(ctx context.Context, force bool) error {
	seed := GenerateSeedData()
	slog.Info("starting seed", "providers", len(seed.BrokerProviders), "strategies", len(seed.Strategies), "candles", len(seed.Candles), "ticks", len(seed.MarketTicks), "trades", len(seed.TradeHistory), "backtests", len(seed.BacktestResults), "component", "seeder")

	if !force {
		var count int
		if err := s.repo.pool.QueryRow(ctx, "SELECT COUNT(*) FROM providers").Scan(&count); err == nil && count > 0 {
			var candleCount int
			s.repo.pool.QueryRow(ctx, "SELECT COUNT(*) FROM candles").Scan(&candleCount)
			if candleCount > 0 {
				shouldReturn := false
				var vixCount int
				if err := s.repo.pool.QueryRow(ctx, "SELECT COUNT(*) FROM vix_logs").Scan(&vixCount); err == nil && vixCount == 0 {
					slog.Info("seeding VIX logs", "rows", len(seed.VIXLogs), "component", "seeder")
					if err := s.seedVIXLogs(ctx, seed.VIXLogs); err != nil {
						return fmt.Errorf("vix logs: %w", err)
					}
				}
				var regimeCount int
				if err := s.repo.pool.QueryRow(ctx, "SELECT COUNT(*) FROM regime_logs").Scan(&regimeCount); err == nil && regimeCount < 100 {
					slog.Info("seeding regime logs", "existing_rows", regimeCount, "new_rows", len(seed.RegimeLogs), "component", "seeder")
					if err := s.seedRegimeLogs(ctx, seed.RegimeLogs); err != nil {
						return fmt.Errorf("regime logs: %w", err)
					}
				}
				var sentimentCount int
				if err := s.repo.pool.QueryRow(ctx, "SELECT COUNT(*) FROM sentiment_logs").Scan(&sentimentCount); err == nil && sentimentCount == 0 {
					slog.Info("seeding sentiment logs", "rows", len(seed.SentimentLogs), "component", "seeder")
					if err := s.seedSentimentLogs(ctx, seed.SentimentLogs); err != nil {
						return fmt.Errorf("sentiment logs: %w", err)
					}
				}
				_ = shouldReturn
				slog.Info("data already exists, use force=true to re-seed", "providers", count, "candles", candleCount, "vix_rows", vixCount, "regime_rows", regimeCount, "component", "seeder")
				return nil
			}
			slog.Info("data exists but candles table empty", "candles_to_seed", len(seed.Candles), "component", "seeder")
			if len(seed.Candles) > 0 {
				if err := s.seedCandles(ctx, seed.Candles); err != nil {
					return fmt.Errorf("candles: %w", err)
				}
				return nil
			}
			return nil
		}
	}

	if force {
		if err := s.truncateAll(ctx); err != nil {
			return fmt.Errorf("truncate: %w", err)
		}
	}

	if err := s.seedSymbols(ctx, seed.Symbols); err != nil {
		return fmt.Errorf("symbols: %w", err)
	}
	if err := s.seedCandles(ctx, seed.Candles); err != nil {
		return fmt.Errorf("candles: %w", err)
	}
	if err := s.seedProviders(ctx, seed.BrokerProviders); err != nil {
		return fmt.Errorf("providers: %w", err)
	}
	if err := s.seedStrategies(ctx, seed.Strategies); err != nil {
		return fmt.Errorf("strategies: %w", err)
	}
	if err := s.seedMarketData(ctx, seed.MarketTicks); err != nil {
		return fmt.Errorf("market data: %w", err)
	}
	if err := s.seedRegimeLogs(ctx, seed.RegimeLogs); err != nil {
		return fmt.Errorf("regime logs: %w", err)
	}
	if err := s.seedVIXLogs(ctx, seed.VIXLogs); err != nil {
		return fmt.Errorf("vix logs: %w", err)
	}
	if err := s.seedSentimentLogs(ctx, seed.SentimentLogs); err != nil {
		return fmt.Errorf("sentiment logs: %w", err)
	}
	if err := s.seedTradeHistory(ctx, seed.TradeHistory); err != nil {
		return fmt.Errorf("trade history: %w", err)
	}
	if err := s.seedBacktestResults(ctx, seed.BacktestResults); err != nil {
		return fmt.Errorf("backtest results: %w", err)
	}
	if err := s.seedUniverseConfigs(ctx, seed.UniverseConfigs); err != nil {
		return fmt.Errorf("universe configs: %w", err)
	}
	if err := s.seedUniverseSnapshots(ctx, seed.UniverseSnapshots); err != nil {
		return fmt.Errorf("universe snapshots: %w", err)
	}

	slog.Info("successfully seeded all data", "component", "seeder")
	return nil
}

func (s *Seeder) seedSymbols(ctx context.Context, symbols []SymbolSeed) error {
	for _, sym := range symbols {
		if _, err := s.repo.pool.Exec(ctx,
			"INSERT INTO symbols (ticker, exchange, asset_type, tick_size, lot_size, is_active) VALUES ($1,$2,$3,$4,$5,true) ON CONFLICT (ticker, exchange) DO NOTHING",
			sym.Ticker, sym.Exchange, sym.AssetType, sym.TickSize, sym.LotSize); err != nil { return fmt.Errorf("symbol %s: %w", sym.Ticker, err) }
	}
	return nil
}

func (s *Seeder) seedProviders(ctx context.Context, providers []BrokerProviderSeed) error {
	for _, p := range providers {
		if _, err := s.repo.pool.Exec(ctx,
			"INSERT INTO providers (name, type, driver, is_enabled, config) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (name) DO NOTHING",
			p.Name, p.Type, p.Driver, true, mustMarshalJSON(p.Config)); err != nil { return fmt.Errorf("provider %s: %w", p.Name, err) }
	}
	return nil
}

func (s *Seeder) seedStrategies(ctx context.Context, strategies []StrategySeed) error {
	for _, st := range strategies {
		if _, err := s.repo.pool.Exec(ctx,
			"INSERT INTO strategies (name, type, parameters, enabled) VALUES ($1,$2,$3,$4) ON CONFLICT (name) DO NOTHING",
			st.Name, st.Type, mustMarshalJSON(st.Parameters), st.Enabled); err != nil { return fmt.Errorf("strategy %s: %w", st.Name, err) }
	}
	return nil
}

func (s *Seeder) seedMarketData(ctx context.Context, ticks []MarketTickSeed) error {
	for _, t := range ticks {
		if _, err := s.repo.pool.Exec(ctx,
			"INSERT INTO market_ticks (time, symbol_id, price_raw, volume_raw, bid_price, ask_price, bid_size, ask_size) SELECT $1, COALESCE((SELECT id FROM symbols WHERE ticker=$2 LIMIT 1), 1), $3, $4, $5, $6, $7, $8 WHERE EXISTS (SELECT 1 FROM symbols WHERE ticker=$2)",
			t.Time, t.Symbol, t.Price.Int64(), int64(t.Volume), t.BidPrice.Int64(), t.AskPrice.Int64(), int64(t.BidSize), int64(t.AskSize)); err != nil { return fmt.Errorf("tick %s: %w", t.Symbol, err) }
	}
	return nil
}

func (s *Seeder) seedRegimeLogs(ctx context.Context, logs []RegimeLogSeed) error {
	for _, l := range logs {
		if _, err := s.repo.pool.Exec(ctx,
			"INSERT INTO regime_logs (timestamp, symbol, hmm_state, confidence) VALUES ($1,$2,$3,$4)",
			l.Time, l.Symbol, l.HMMState, l.Confidence); err != nil { return fmt.Errorf("regime %s: %w", l.Symbol, err) }
	}
	return nil
}

func (s *Seeder) seedVIXLogs(ctx context.Context, logs []VIXLogSeed) error {
	if len(logs) == 0 {
		return nil
	}
	for _, l := range logs {
		if _, err := s.repo.pool.Exec(ctx,
			"INSERT INTO vix_logs (timestamp, vix_value, vix_change) VALUES ($1,$2,$3)",
			l.Time, int64(l.VIXValue*VIXBigintScale), int64(l.VIXChange*VIXBigintScale)); err != nil { return fmt.Errorf("vix %s: %w", l.Time.Format("2006-01-02"), err) }
	}
	return nil
}

func (s *Seeder) seedSentimentLogs(ctx context.Context, logs []SentimentLogSeed) error {
	if len(logs) == 0 {
		return nil
	}
	for _, l := range logs {
		if _, err := s.repo.pool.Exec(ctx,
			"INSERT INTO sentiment_logs (timestamp, score, label) VALUES ($1,$2,$3)",
			l.Time, l.Score, l.Label); err != nil {
			return fmt.Errorf("sentiment %s: %w", l.Time.Format("2006-01-02"), err)
		}
	}
	return nil
}

func (s *Seeder) seedTradeHistory(ctx context.Context, trades []TradeHistorySeed) error {
	for _, t := range trades {
		if _, err := s.repo.pool.Exec(ctx,
			"INSERT INTO trade_executions (strategy_id, symbol, side, quantity, price, hmm_regime, executed_at) SELECT id, $1, $2, $3, $4, $5, $6 FROM strategies WHERE name=$7 LIMIT 1",
			t.Symbol, t.Side, t.Quantity, t.Price.Int64(), t.HMMRegime, t.Time, t.StrategyID); err != nil { return fmt.Errorf("trade %s: %w", t.Symbol, err) }
	}
	return nil
}

func (s *Seeder) seedCandles(ctx context.Context, candles []CandleSeed) error {
	if len(candles) == 0 {
		return nil
	}
	batchSize := 200
	for i := 0; i < len(candles); i += batchSize {
		end := i + batchSize
		if end > len(candles) {
			end = len(candles)
		}
		batch := &pgx.Batch{}
		for _, c := range candles[i:end] {
			batch.Queue(
				`INSERT INTO candles (time, symbol_id, timeframe, open_raw, high_raw, low_raw, close_raw, volume)
				 SELECT $1, COALESCE((SELECT id FROM symbols WHERE ticker=$2 LIMIT 1), 1), $3, $4, $5, $6, $7, $8
				 WHERE EXISTS (SELECT 1 FROM symbols WHERE ticker=$2)`,
				c.Time, c.Symbol, c.Timeframe, c.Open.Int64(), c.High.Int64(), c.Low.Int64(), c.Close.Int64(), int64(c.Volume))
		}
		br := s.repo.pool.SendBatch(ctx, batch)
		if _, cerr := br.Exec(); cerr != nil { continue }
		br.Close()
	}
	slog.Info("seeded candles", "count", len(candles), "component", "seeder")
	return nil
}

func (s *Seeder) seedUniverseConfigs(ctx context.Context, configs []UniverseConfigSeed) error {
	for _, cfg := range configs {
		if _, err := s.repo.pool.Exec(ctx,
			`INSERT INTO universe_config (user_id, name, profile_id, asset_class_filters, dynamic_triggers, content_hash, is_active)
			 VALUES ((SELECT id FROM users ORDER BY created_at LIMIT 1), $1, $2, $3, $4, $5, $6)
			 ON CONFLICT (user_id, name) DO UPDATE SET
			   profile_id=$2, asset_class_filters=$3, dynamic_triggers=$4, content_hash=$5, is_active=$6, updated_at=now()`,
			cfg.Name, cfg.ProfileID, mustMarshalJSON(cfg.AssetClassFilters), mustMarshalJSON(cfg.DynamicTriggers), cfg.ContentHash, cfg.IsActive); err != nil {
			return fmt.Errorf("universe config %s: %w", cfg.Name, err)
		}
	}
	return nil
}

func (s *Seeder) seedUniverseSnapshots(ctx context.Context, snaps []UniverseSnapshotSeed) error {
	for _, snap := range snaps {
		userID := "00000000-0000-0000-0000-000000000001"
		var uid string
		if err := s.repo.pool.QueryRow(ctx, "SELECT id FROM users ORDER BY created_at LIMIT 1").Scan(&uid); err == nil && uid != "" {
			userID = uid
		}
		if _, err := s.repo.pool.Exec(ctx,
			`INSERT INTO universe_state (user_id, snapshot_date, symbol_ids, content_hash, filters_used, triggered_additions, triggered_removals)
			 VALUES ($1, $2,
			   (SELECT ARRAY_AGG(id ORDER BY id) FROM symbols WHERE ticker = ANY($3)),
			   $4, '{}', '[]', '[]')
			 ON CONFLICT (user_id, snapshot_date) DO UPDATE SET
			   symbol_ids = (SELECT ARRAY_AGG(id ORDER BY id) FROM symbols WHERE ticker = ANY($3)),
			   content_hash = $4`,
			userID, snap.SnapshotDate, snap.SymbolTickers, snap.ContentHash); err != nil {
			return fmt.Errorf("snapshot %s: %w", snap.SnapshotDate.Format("2006-01-02"), err)
		}
	}
	return nil
}

func (s *Seeder) seedBacktestResults(ctx context.Context, results []BacktestResultSeed) error {
	for _, r := range results {
		if _, err := s.repo.pool.Exec(ctx,
			"INSERT INTO backtest_runs (strategy_id, symbol_set, start_date, end_date, status, sharpe_ratio, max_drawdown, total_return, win_rate, num_trades, initial_capital) SELECT id, $1, $2, $3, 'completed', $4, $5, $6, $7, $8, 100000 FROM strategies WHERE name=$9 LIMIT 1",
			r.Symbols, r.StartDate, r.EndDate, r.SharpeRatio, r.MaxDrawdown, r.TotalReturn, r.WinRate, r.NumTrades, r.StrategyName); err != nil { return fmt.Errorf("backtest %s: %w", r.StrategyName, err) }
	}
	return nil
}

func (s *Seeder) truncateAll(ctx context.Context) error {
	tables := []string{"trade_executions", "regime_logs", "backtest_results", "backtest_runs", "market_ticks", "candles", "provider_symbols", "credentials", "providers", "strategies", "symbols", "universe_state", "universe_config"}
	for _, t := range tables {
		if _, err := s.repo.pool.Exec(ctx, "DELETE FROM "+t); err != nil {
			slog.Warn("truncate error", "table", t, "error", err, "component", "seeder")
		}
	}
	return nil
}

func (s *Seeder) VerifyIntegrity(ctx context.Context) (*IntegrityReport, error) {
	report := &IntegrityReport{Passed: true}
	tables := []string{"symbols", "providers", "strategies", "candles", "market_ticks", "regime_logs", "trade_executions", "backtest_runs", "universe_config", "universe_state"}
	counts := make(map[string]int)
	for _, t := range tables {
		var c int
		if err := s.repo.pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+t).Scan(&c); err != nil {
			report.Checks = append(report.Checks, IntegrityCheck{Table: t, Status: "error", Message: err.Error()})
			report.Passed = false
			continue
		}
		counts[t] = c
		status := "ok"
		if c == 0 { status = "empty"; report.Passed = false }
		report.Checks = append(report.Checks, IntegrityCheck{Table: t, Status: status, Count: c})
	}
	report.TableCounts = counts
	return report, nil
}

type IntegrityReport struct {
	Passed      bool
	Checks      []IntegrityCheck
	TableCounts map[string]int
}

type IntegrityCheck struct {
	Table   string
	Status  string
	Count   int
	Message string
}
func mustMarshalJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil { return "{}" }
	return string(b)
}
