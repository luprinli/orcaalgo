package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/lee-econ/orca-core/internal/types"
)

func (r *Repository) UpsertStrategy(ctx context.Context, s *Strategy) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO strategies (id, name, type, parameters, enabled, created_at)
		 VALUES ($1, $2, $3, $4, $5, now())
		 ON CONFLICT (id) DO UPDATE SET name=$2, type=$3, parameters=$4, enabled=$5`,
		s.ID, s.Name, s.Type, s.Parameters, s.Enabled,
	)
	return err
}

func (r *Repository) GetStrategy(ctx context.Context, id string) (*Strategy, error) {
	var s Strategy
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, type, parameters, enabled, created_at
		 FROM strategies WHERE id=$1`, id,
	).Scan(&s.ID, &s.Name, &s.Type, &s.Parameters, &s.Enabled, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) ListStrategies(ctx context.Context) ([]Strategy, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, type, parameters, enabled, created_at
		 FROM strategies ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var strategies []Strategy
	for rows.Next() {
		var s Strategy
		if err := rows.Scan(&s.ID, &s.Name, &s.Type, &s.Parameters, &s.Enabled, &s.CreatedAt); err != nil {
			continue
		}
		strategies = append(strategies, s)
	}
	return strategies, nil
}

func (r *Repository) DeleteStrategy(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM strategies WHERE id=$1`, id)
	return err
}

func (r *Repository) SaveBacktestResult(ctx context.Context, result *BacktestResult) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE backtest_runs SET status='completed',
		 sharpe_ratio=$1, max_drawdown=$2, total_return=$3, win_rate=$4, num_trades=$5
		 WHERE id=$6`,
		result.SharpeRatio, result.MaxDrawdown,
		result.TotalReturnPct, result.WinRate, result.NumTrades,
		result.ID,
	)
	return err
}

func (r *Repository) RunMigrations(ctx context.Context) error {
	if r.pool == nil {
		return fmt.Errorf("no database pool")
	}

	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename   TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	// Single source of truth for schema changes: the Go-managed migration
	// runner applies every pending `*.up.sql` file (sorted by filename prefix)
	// and records it in schema_migrations. Replaces the former golang-migrate
	// `scripts/migrate.ps1` (which used a conflicting `version`/`dirty` schema).
	dir := os.Getenv("ORCA_MIGRATIONS_DIR")
	if dir == "" {
		dir = "internal/db/migrations"
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %s: %w", dir, err)
	}
	var files []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)

	for _, name := range files {
		if err := r.applyMigrationIfPending(ctx, name, dir); err != nil {
			return err
		}
	}
	return nil
}

// applyMigrationIfPending applies a single migration file when its filename is
// not yet recorded. The SQL and the filename recording run in one transaction,
// so a failed migration is never marked applied.
func (r *Repository) applyMigrationIfPending(ctx context.Context, name, dir string) error {
	var exists bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename=$1)`, name,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check migration %s: %w", name, err)
	}
	if exists {
		return nil
	}

	sqlBytes, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return fmt.Errorf("read migration %s: %w", name, err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("apply migration %s: %w", name, err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT (filename) DO NOTHING`, name,
	); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}
	return nil
}

func (r *Repository) ListPendingMigrations(ctx context.Context, migrationsDir string) ([]string, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("no database pool")
	}

	rows, err := r.pool.Query(ctx, "SELECT filename FROM schema_migrations ORDER BY filename")
	if err != nil {
		return nil, fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			continue
		}
		applied[filename] = true
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %s: %w", migrationsDir, err)
	}

	var pending []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		if !applied[name] {
			pending = append(pending, name)
		}
	}
	sort.Strings(pending)
	return pending, nil
}

func (r *Repository) SeedSymbols(ctx context.Context) error {
	for _, s := range DefaultSymbols {
		if _, err := r.InsertSymbol(ctx, &s); err != nil {
			return fmt.Errorf("seed symbol %s: %w", s.Ticker, err)
		}
	}
	// Deactivate any active symbol whose (ticker, exchange) pair is not part of
	// the canonical universe. This removes both stale tickers (e.g. legacy
	// GOOGL/AMZN/ES/NQ/CL or misnamed BTCUSD/ETHUSD) and duplicate rows created
	// by an earlier seed that used a different exchange label for the same ticker.
	values := make([]string, 0, len(DefaultSymbols))
	args := make([]interface{}, 0, len(DefaultSymbols)*2)
	for i, s := range DefaultSymbols {
		values = append(values, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		args = append(args, s.Ticker, s.Exchange)
	}
	query := fmt.Sprintf(
		`UPDATE symbols SET is_active=false
		 WHERE is_active=true
		   AND (ticker, exchange) NOT IN (%s)`,
		strings.Join(values, ", "),
	)
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("deactivate stale symbols: %w", err)
	}
	return nil
}

func (r *Repository) InsertSymbol(ctx context.Context, s *Symbol) (int32, error) {
	var id int32
	err := r.pool.QueryRow(ctx,
		`INSERT INTO symbols (ticker, exchange, asset_type, tick_size, lot_size, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (ticker, exchange) DO UPDATE SET is_active=true
		 RETURNING id`,
		s.Ticker, s.Exchange, s.AssetType, s.TickSize, s.LotSize, s.IsActive,
	).Scan(&id)
	return id, err
}

// UpsertSymbolFromAsset inserts a broker-discovered asset as an inactive
// symbol, leaving any existing symbol untouched (single source of truth for
// the canonical universe). Returns true when a new row was inserted.
func (r *Repository) UpsertSymbolFromAsset(ctx context.Context, ticker, exchange, assetType string) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`INSERT INTO symbols (ticker, exchange, asset_type, tick_size, lot_size, is_active)
		 VALUES ($1, $2, $3, 0.01, 1, false)
		 ON CONFLICT (ticker, exchange) DO NOTHING`,
		ticker, exchange, assetType)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) ListSymbols(ctx context.Context) ([]Symbol, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, ticker, exchange, asset_type, tick_size, lot_size, is_active, created_at
		 FROM symbols WHERE is_active=true ORDER BY ticker`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []Symbol
	for rows.Next() {
		var s Symbol
		if err := rows.Scan(&s.ID, &s.Ticker, &s.Exchange, &s.AssetType, &s.TickSize, &s.LotSize, &s.IsActive, &s.CreatedAt); err != nil {
			continue
		}
		symbols = append(symbols, s)
	}
	return symbols, nil
}

func (r *Repository) DeleteSymbol(ctx context.Context, id int32) error {
	_, err := r.pool.Exec(ctx, `UPDATE symbols SET is_active=false WHERE id=$1`, id)
	return err
}

func (r *Repository) LoadActiveSymbols(ctx context.Context) ([]Symbol, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, ticker, exchange, asset_type, tick_size, lot_size, is_active, created_at,
		        COALESCE(market_cap, 0), COALESCE(last_volume, 0),
		        COALESCE(last_atr_pct, 0), COALESCE(last_rsi, 0), metrics_updated
		 FROM symbols WHERE is_active=true ORDER BY ticker`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var symbols []Symbol
	for rows.Next() {
		var s Symbol
		if err := rows.Scan(&s.ID, &s.Ticker, &s.Exchange, &s.AssetType, &s.TickSize, &s.LotSize,
			&s.IsActive, &s.CreatedAt, &s.MarketCap, &s.LastVolume, &s.LastATRPct, &s.LastRSI, &s.MetricsUpdated); err != nil {
			continue
		}
		symbols = append(symbols, s)
	}
	return symbols, nil
}

func (r *Repository) UpdateSymbolMetrics(ctx context.Context, symbolID int32, lastPrice types.Price, marketCap int64, lastVolume int64, lastATRPct float64, lastRSI float64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE symbols SET market_cap=$1, last_volume=$2, last_atr_pct=$3, last_rsi=$4, metrics_updated=now()
		 WHERE id=$5`,
		marketCap, lastVolume, lastATRPct, lastRSI, symbolID,
	)
	_ = lastPrice
	return err
}

func (r *Repository) GetSymbolByTicker(ctx context.Context, ticker string) (*Symbol, error) {
	var s Symbol
	err := r.pool.QueryRow(ctx,
		`SELECT id, ticker, exchange, asset_type, tick_size, lot_size, is_active, created_at,
		        COALESCE(market_cap, 0), COALESCE(last_volume, 0),
		        COALESCE(last_atr_pct, 0), COALESCE(last_rsi, 0), metrics_updated
		 FROM symbols WHERE ticker=$1`, ticker,
	).Scan(&s.ID, &s.Ticker, &s.Exchange, &s.AssetType, &s.TickSize, &s.LotSize,
		&s.IsActive, &s.CreatedAt, &s.MarketCap, &s.LastVolume, &s.LastATRPct, &s.LastRSI, &s.MetricsUpdated)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) ResolveSnapshotSymbols(ctx context.Context, symbolIDs []int32) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ticker FROM symbols WHERE id = ANY($1) ORDER BY ticker`, symbolIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tickers []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			continue
		}
		tickers = append(tickers, t)
	}
	return tickers, nil
}

func (r *Repository) InsertProvider(ctx context.Context, p *Provider) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO providers (id, name, type, driver, is_enabled, config)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		p.ID, p.Name, p.Type, p.Driver, p.IsEnabled, p.Config,
	)
	return err
}

func (r *Repository) ListProviders(ctx context.Context) ([]Provider, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, type, driver, is_enabled, config, created_at, updated_at
		 FROM providers ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []Provider
	for rows.Next() {
		var p Provider
		if err := rows.Scan(&p.ID, &p.Name, &p.Type, &p.Driver, &p.IsEnabled, &p.Config, &p.CreatedAt, &p.UpdatedAt); err != nil {
			continue
		}
		providers = append(providers, p)
	}
	return providers, nil
}

func (r *Repository) DeleteProvider(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM providers WHERE id=$1`, id)
	return err
}

func (r *Repository) InsertProviderSymbol(ctx context.Context, ps *ProviderSymbol) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO provider_symbols (provider_id, symbol_id, feed_type, priority, is_enabled)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (provider_id, symbol_id, feed_type) DO UPDATE SET priority=$4, is_enabled=$5`,
		ps.ProviderID, ps.SymbolID, ps.FeedType, ps.Priority, ps.IsEnabled,
	)
	return err
}

func (r *Repository) ListProviderSymbols(ctx context.Context, symbolID int32) ([]ProviderSymbol, error) {
	var cond string
	var args []interface{}
	if symbolID > 0 {
		cond = "WHERE ps.symbol_id=$1"
		args = append(args, symbolID)
	}
	rows, err := r.pool.Query(ctx,
		`SELECT ps.provider_id, ps.symbol_id, ps.feed_type, ps.priority, ps.is_enabled
		 FROM provider_symbols ps `+cond+` ORDER BY ps.priority`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var symbols []ProviderSymbol
	for rows.Next() {
		var ps ProviderSymbol
		if err := rows.Scan(&ps.ProviderID, &ps.SymbolID, &ps.FeedType, &ps.Priority, &ps.IsEnabled); err != nil {
			continue
		}
		symbols = append(symbols, ps)
	}
	return symbols, nil
}

func (r *Repository) DeleteProviderSymbol(ctx context.Context, providerID string, symbolID int32, feedType string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM provider_symbols WHERE provider_id=$1 AND symbol_id=$2 AND feed_type=$3`,
		providerID, symbolID, feedType,
	)
	return err
}

func (r *Repository) InsertUniverseConfig(ctx context.Context, cfg *UniverseConfig) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO universe_config (id, user_id, name, profile_id, asset_class_filters, dynamic_triggers, content_hash, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (user_id, name) DO UPDATE SET
		   profile_id=$4, asset_class_filters=$5, dynamic_triggers=$6, content_hash=$7, is_active=$8, updated_at=now()`,
		cfg.ID, cfg.UserID, cfg.Name, cfg.ProfileID, cfg.AssetClassFilters, cfg.DynamicTriggers, cfg.ContentHash, cfg.IsActive,
	)
	return err
}

func (r *Repository) GetActiveUniverseConfig(ctx context.Context, userID string) (*UniverseConfig, error) {
	var cfg UniverseConfig
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, profile_id, asset_class_filters, dynamic_triggers, content_hash, is_active, created_at, updated_at
		 FROM universe_config WHERE user_id=$1 AND is_active=true LIMIT 1`, userID,
	).Scan(&cfg.ID, &cfg.UserID, &cfg.Name, &cfg.ProfileID, &cfg.AssetClassFilters, &cfg.DynamicTriggers,
		&cfg.ContentHash, &cfg.IsActive, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *Repository) ListUniverseConfigs(ctx context.Context, userID string) ([]UniverseConfig, error) {
	var rows pgx.Rows
	var err error
	if userID == "" {
		rows, err = r.pool.Query(ctx,
			`SELECT id, user_id, name, profile_id, asset_class_filters, dynamic_triggers, content_hash, is_active, created_at, updated_at
			 FROM universe_config ORDER BY name`)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT id, user_id, name, profile_id, asset_class_filters, dynamic_triggers, content_hash, is_active, created_at, updated_at
			 FROM universe_config WHERE user_id=$1 ORDER BY name`, userID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var configs []UniverseConfig
	for rows.Next() {
		var cfg UniverseConfig
		if err := rows.Scan(&cfg.ID, &cfg.UserID, &cfg.Name, &cfg.ProfileID, &cfg.AssetClassFilters,
			&cfg.DynamicTriggers, &cfg.ContentHash, &cfg.IsActive, &cfg.CreatedAt, &cfg.UpdatedAt); err != nil {
			continue
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}

func (r *Repository) SetActiveUniverseConfig(ctx context.Context, configID, userID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE universe_config SET is_active=false WHERE user_id=$1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE universe_config SET is_active=true, updated_at=now() WHERE id=$1 AND user_id=$2`, configID, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) InsertUniverseSnapshot(ctx context.Context, snap *UniverseSnapshot) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO universe_state (id, user_id, snapshot_date, symbol_ids, content_hash, filters_used, triggered_additions, triggered_removals)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (user_id, snapshot_date) DO UPDATE SET
		   symbol_ids=$4, content_hash=$5, filters_used=$6, triggered_additions=$7, triggered_removals=$8`,
		snap.ID, snap.UserID, snap.SnapshotDate, snap.SymbolIDs, snap.ContentHash,
		snap.FiltersUsed, snap.TriggeredAdditions, snap.TriggeredRemovals,
	)
	return err
}

func (r *Repository) GetUniverseSnapshot(ctx context.Context, userID string, date time.Time) (*UniverseSnapshot, error) {
	var snap UniverseSnapshot
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, snapshot_date, symbol_ids, content_hash, filters_used, triggered_additions, triggered_removals, created_at
		 FROM universe_state WHERE user_id=$1 AND snapshot_date=$2`, userID, date,
	).Scan(&snap.ID, &snap.UserID, &snap.SnapshotDate, &snap.SymbolIDs, &snap.ContentHash,
		&snap.FiltersUsed, &snap.TriggeredAdditions, &snap.TriggeredRemovals, &snap.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

func (r *Repository) ListUniverseSnapshots(ctx context.Context, userID string, start, end time.Time) ([]UniverseSnapshot, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, snapshot_date, symbol_ids, content_hash, filters_used, triggered_additions, triggered_removals, created_at
		 FROM universe_state WHERE user_id=$1 AND snapshot_date >= $2 AND snapshot_date <= $3
		 ORDER BY snapshot_date ASC`, userID, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var snaps []UniverseSnapshot
	for rows.Next() {
		var snap UniverseSnapshot
		if err := rows.Scan(&snap.ID, &snap.UserID, &snap.SnapshotDate, &snap.SymbolIDs, &snap.ContentHash,
			&snap.FiltersUsed, &snap.TriggeredAdditions, &snap.TriggeredRemovals, &snap.CreatedAt); err != nil {
			continue
		}
		snaps = append(snaps, snap)
	}
	return snaps, nil
}

func (r *Repository) UpsertMatrixProgress(ctx context.Context, mp *MatrixProgressRecord) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO matrix_progress (batch_id, mode, total, completed, failed, running, passed, status, start_time, updated_at, combos_json, results_json, best_sharpe, best_strategy, best_symbol, total_trades)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		 ON CONFLICT (batch_id) DO UPDATE SET
		   total=$3, completed=$4, failed=$5, running=$6, passed=$7, status=$8,
		   updated_at=$10, combos_json=$11, results_json=$12,
		   best_sharpe=$13, best_strategy=$14, best_symbol=$15, total_trades=$16`,
		mp.BatchID, mp.Mode, mp.Total, mp.Completed, mp.Failed, mp.Running,
		mp.Passed, mp.Status, mp.StartTime, mp.UpdatedAt,
		mp.CombosJSON, mp.ResultsJSON, mp.BestSharpe, mp.BestStrategy, mp.BestSymbol, mp.TotalTrades,
	)
	return err
}

func (r *Repository) GetMatrixProgress(ctx context.Context, batchID string) (*MatrixProgressRecord, error) {
	var mp MatrixProgressRecord
	err := r.pool.QueryRow(ctx,
		`SELECT batch_id, mode, total, completed, failed, running, passed, status, start_time, updated_at,
		        combos_json, results_json, best_sharpe, best_strategy, best_symbol, total_trades
		 FROM matrix_progress WHERE batch_id=$1`, batchID,
	).Scan(&mp.BatchID, &mp.Mode, &mp.Total, &mp.Completed, &mp.Failed, &mp.Running,
		&mp.Passed, &mp.Status, &mp.StartTime, &mp.UpdatedAt,
		&mp.CombosJSON, &mp.ResultsJSON, &mp.BestSharpe, &mp.BestStrategy, &mp.BestSymbol, &mp.TotalTrades,
	)
	if err != nil {
		return nil, err
	}
	return &mp, nil
}

func (r *Repository) ListActiveMatrices(ctx context.Context) ([]*MatrixProgressRecord, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT batch_id, mode, total, completed, failed, running, passed, status, start_time, updated_at,
		        combos_json, results_json, best_sharpe, best_strategy, best_symbol, total_trades
		 FROM matrix_progress WHERE status='running' ORDER BY start_time DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*MatrixProgressRecord
	for rows.Next() {
		var mp MatrixProgressRecord
		if err := rows.Scan(&mp.BatchID, &mp.Mode, &mp.Total, &mp.Completed, &mp.Failed, &mp.Running,
			&mp.Passed, &mp.Status, &mp.StartTime, &mp.UpdatedAt,
			&mp.CombosJSON, &mp.ResultsJSON, &mp.BestSharpe, &mp.BestStrategy, &mp.BestSymbol, &mp.TotalTrades); err != nil {
			continue
		}
		results = append(results, &mp)
	}
	return results, nil
}

func (r *Repository) DeleteMatrixBatch(ctx context.Context, batchID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM matrix_progress WHERE batch_id=$1`, batchID)
	return err
}

func (r *Repository) CleanupOldMatrices(ctx context.Context) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM matrix_progress WHERE status IN ('completed','failed','cancelled') AND updated_at < NOW() - INTERVAL '24 hours'`,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
