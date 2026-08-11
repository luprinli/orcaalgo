package db

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lee-econ/orca-core/internal/types"
	"github.com/lee-econ/orca-core/pkg/temporal"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
	PoolMax  int
	PoolMin  int
}

func DefaultConfig() Config {
	port, _ := strconv.Atoi(envOrDefault("ORCA_DB_PORT", "5432"))
	return Config{
		Host:     envOrDefault("ORCA_DB_HOST", "localhost"),
		Port:     port,
		User:     envOrDefault("ORCA_DB_USER", "orca"),
		Password: envOrDefault("ORCA_DB_PASSWORD", "orca"),
		Database: envOrDefault("ORCA_DB_NAME", "orca_core"),
		SSLMode:  envOrDefault("ORCA_DB_SSLMODE", "disable"),
		PoolMax:  20,
		PoolMin:  2,
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(cfg Config) (*Repository, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&pool_max_conns=%d&pool_min_conns=%d",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode, cfg.PoolMax, cfg.PoolMin,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Repository{pool: pool}, nil
}

func (r *Repository) Ping(ctx context.Context) error {
	if r.pool == nil {
		return fmt.Errorf("repository pool not initialized")
	}
	return r.pool.Ping(ctx)
}

func (r *Repository) Close() {
	if r.pool != nil {
		r.pool.Close()
	}
}

func (r *Repository) Pool() *pgxpool.Pool {
	return r.pool
}

func (r *Repository) IsConnected() bool {
	if r.pool == nil {
		return false
	}
	return r.pool.Ping(context.Background()) == nil
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

func (r *Repository) InsertRegimeLog(ctx context.Context, symbol string, state int8, confidence float64) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO regime_logs (symbol, hmm_state, confidence) VALUES ($1, $2, $3)`,
		symbol, state, confidence,
	)
	return err
}

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

func (r *Repository) LoadRegimeLogs(ctx context.Context, start, end time.Time) ([]RegimeLog, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT timestamp, hmm_state, confidence, symbol
		 FROM regime_logs WHERE timestamp >= $1 AND timestamp <= $2
		 ORDER BY timestamp ASC`, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []RegimeLog
	for rows.Next() {
		var l RegimeLog
		if err := rows.Scan(&l.Time, &l.HMMState, &l.Confidence, &l.Symbol); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func (r *Repository) LoadVIXLogs(ctx context.Context, start, end time.Time) ([]VIXLog, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT timestamp, vix_value, vix_change
		 FROM vix_logs WHERE timestamp >= $1 AND timestamp <= $2
		 ORDER BY timestamp ASC`, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []VIXLog
	for rows.Next() {
		var l VIXLog
		if err := rows.Scan(&l.Time, &l.VIXValue, &l.VIXChange); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	return logs, nil
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

func (r *Repository) SeedSymbols(ctx context.Context) error {
	for _, s := range DefaultSymbols {
		if _, err := r.InsertSymbol(ctx, &s); err != nil {
			return fmt.Errorf("seed symbol %s: %w", s.Ticker, err)
		}
	}
	return nil
}

func (r *Repository) RunMigrations(ctx context.Context) error {
	if r.pool == nil {
		return fmt.Errorf("no database pool")
	}

	// Ensure migrations tracking table exists
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version  INTEGER PRIMARY KEY,
			name     TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	// Check if strategies table exists as a proxy for migrations done
	var count int
	err = r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_name='strategies'").Scan(&count)
	if err != nil {
		return fmt.Errorf("check migrations: %w", err)
	}
	if count > 0 {
		return nil // migrations already applied
	}
	return fmt.Errorf("migrations not applied: run scripts/migrate.ps1 or docker compose up the postgres with migration volume")
}

const PRICE_SCALE_F = 100_000

type Strategy struct {
	ID         string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
	Enabled    bool                   `json:"enabled"`
	CreatedAt time.Time              `json:"created_at"`
}

type TradeExecution struct {
	ID                    string
	StrategyID            string
	Symbol                string
	Side                  string
	Quantity              float64
	Price                 types.Price
	HMMRegime             int8
	RiskApproved          bool
	ConsistencyMultiplier float64
	RejectedReason        string
	ExecutedAt            time.Time
	BrokerOrderID         string
}

type ConsistencyLog struct {
	Date        time.Time
	DailyPnLPct float64
	IsOutlier   bool
	ActionTaken string
}

type AdversarialNews struct {
	ID              int64
	DetectedAt      time.Time
	Headline        string
	Source          string
	SentimentScore  float64
	Confidence      float64
	WasCorroborated bool
	SymbolsAffected []string
}

type Symbol struct {
	ID             int32     `json:"id"`
	Ticker         string    `json:"ticker"`
	Exchange       string    `json:"exchange"`
	AssetType      string    `json:"asset_type"`
	TickSize       float64   `json:"tick_size"`
	LotSize        float64   `json:"lot_size"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	LastPrice      types.Price `json:"last_price,omitempty"`
	MarketCap      int64     `json:"market_cap,omitempty"`
	LastVolume     int64     `json:"last_volume,omitempty"`
	LastATRPct     float64   `json:"last_atr_pct,omitempty"`
	LastRSI        float64   `json:"last_rsi,omitempty"`
	MetricsUpdated *time.Time `json:"metrics_updated,omitempty"`
}

type UniverseConfig struct {
	ID                string
	UserID            string
	Name              string
	ProfileID         string
	AssetClassFilters map[string]interface{} `json:"asset_class_filters"`
	DynamicTriggers   map[string]interface{} `json:"dynamic_triggers"`
	ContentHash       string
	IsActive          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type UniverseSnapshot struct {
	ID                 string
	UserID             string
	SnapshotDate       time.Time
	SymbolIDs          []int32
	ContentHash        string
	FiltersUsed        map[string]interface{} `json:"filters_used"`
	TriggeredAdditions []map[string]interface{} `json:"triggered_additions"`
	TriggeredRemovals  []map[string]interface{} `json:"triggered_removals"`
	CreatedAt          time.Time
}

type Provider struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Driver    string                 `json:"driver"`
	IsEnabled bool                   `json:"is_enabled"`
	Config    map[string]interface{} `json:"config"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type ProviderSymbol struct {
	ProviderID string
	SymbolID   int32
	FeedType   string
	Priority   int16
	IsEnabled  bool
}

type Credential struct {
	ID            string
	ProviderID    string
	KeyLabel      string
	VaultPath     string
	IsActive      bool
	LastValidated *time.Time
	CreatedAt     time.Time
}

type Candle struct {
	Time   time.Time
	Open   types.Price
	High   types.Price
	Low    types.Price
	Close  types.Price
	Volume float64
	Symbol string
}

type RegimeLog struct {
	Time       time.Time
	HMMState   int8
	Confidence float64
	Symbol     string
}

type VIXLog struct {
	Time      time.Time
	VIXValue  float64
	VIXChange float64
}

type BacktestResult struct {
	ID             string
	StrategyID     string
	Config         interface{}
	SharpeRatio    float64
	MaxDrawdown    float64
	TotalReturnPct float64
	WinRate        float64
	NumTrades      int
}

type MatrixProgressRecord struct {
	BatchID      string    `json:"batch_id"`
	Mode         string    `json:"mode"`
	Total        int       `json:"total"`
	Completed    int       `json:"completed"`
	Failed       int       `json:"failed"`
	Running      int       `json:"running"`
	Passed       int       `json:"passed"`
	Status       string    `json:"status"`
	StartTime    time.Time `json:"start_time"`
	UpdatedAt    time.Time `json:"updated_at"`
	CombosJSON   []byte    `json:"-"`
	ResultsJSON  []byte    `json:"-"`
	BestSharpe   float64   `json:"best_sharpe"`
	BestStrategy string    `json:"best_strategy"`
	BestSymbol   string    `json:"best_symbol"`
	TotalTrades  int       `json:"total_trades"`
}

var DefaultSymbols = []Symbol{
	{Ticker: "AAPL", Exchange: "NASDAQ", AssetType: "equity", TickSize: 0.01, LotSize: 1, IsActive: true},
	{Ticker: "MSFT", Exchange: "NASDAQ", AssetType: "equity", TickSize: 0.01, LotSize: 1, IsActive: true},
	{Ticker: "GOOGL", Exchange: "NASDAQ", AssetType: "equity", TickSize: 0.01, LotSize: 1, IsActive: true},
	{Ticker: "AMZN", Exchange: "NASDAQ", AssetType: "equity", TickSize: 0.01, LotSize: 1, IsActive: true},
	{Ticker: "NVDA", Exchange: "NASDAQ", AssetType: "equity", TickSize: 0.01, LotSize: 1, IsActive: true},
	{Ticker: "TSLA", Exchange: "NASDAQ", AssetType: "equity", TickSize: 0.01, LotSize: 1, IsActive: true},
	{Ticker: "META", Exchange: "NASDAQ", AssetType: "equity", TickSize: 0.01, LotSize: 1, IsActive: true},
	{Ticker: "SPY", Exchange: "ARCA", AssetType: "etf", TickSize: 0.01, LotSize: 1, IsActive: true},
	{Ticker: "QQQ", Exchange: "NASDAQ", AssetType: "etf", TickSize: 0.01, LotSize: 1, IsActive: true},
	{Ticker: "IWM", Exchange: "ARCA", AssetType: "etf", TickSize: 0.01, LotSize: 1, IsActive: true},
	{Ticker: "DIA", Exchange: "ARCA", AssetType: "etf", TickSize: 0.01, LotSize: 1, IsActive: true},
	{Ticker: "VOO", Exchange: "ARCA", AssetType: "etf", TickSize: 0.01, LotSize: 1, IsActive: true},
	{Ticker: "TLT", Exchange: "NASDAQ", AssetType: "etf", TickSize: 0.01, LotSize: 1, IsActive: true},
	{Ticker: "GLD", Exchange: "ARCA", AssetType: "etf", TickSize: 0.01, LotSize: 1, IsActive: true},
	{Ticker: "USO", Exchange: "ARCA", AssetType: "etf", TickSize: 0.01, LotSize: 1, IsActive: true},
	{Ticker: "BTCUSD", Exchange: "CRYPTO", AssetType: "crypto", TickSize: 0.01, LotSize: 1, IsActive: true},
	{Ticker: "ETHUSD", Exchange: "CRYPTO", AssetType: "crypto", TickSize: 0.01, LotSize: 1, IsActive: true},
	{Ticker: "ES", Exchange: "CME", AssetType: "futures", TickSize: 0.25, LotSize: 50, IsActive: true},
	{Ticker: "NQ", Exchange: "CME", AssetType: "futures", TickSize: 0.25, LotSize: 20, IsActive: true},
	{Ticker: "CL", Exchange: "NYMEX", AssetType: "futures", TickSize: 0.01, LotSize: 1000, IsActive: true},
}

func NewRepositoryFromPool(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) GetSetting(ctx context.Context, key string) (map[string]interface{}, error) {
	var val map[string]interface{}
	err := r.pool.QueryRow(ctx,
		`SELECT value FROM settings WHERE key=$1`, key,
	).Scan(&val)
	return val, err
}

func (r *Repository) UpsertSetting(ctx context.Context, key string, value map[string]interface{}) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, now())
		 ON CONFLICT (key) DO UPDATE SET value=$2, updated_at=now()`,
		key, value,
	)
	return err
}

func (r *Repository) ListSettings(ctx context.Context) (map[string]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx, `SELECT key, value FROM settings ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	settings := make(map[string]map[string]interface{})
	for rows.Next() {
		var k string
		var v map[string]interface{}
		if err := rows.Scan(&k, &v); err != nil {
			continue
		}
		settings[k] = v
	}
	return settings, nil
}

func (r *Repository) InsertAuditLog(ctx context.Context, level, component, message string, metadata map[string]interface{}) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO audit_log (level, component, message, metadata) VALUES ($1, $2, $3, $4)`,
		level, component, message, metadata,
	)
	return err
}

func (r *Repository) ListAuditLogs(ctx context.Context, component string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, timestamp, level, component, message, metadata FROM audit_log`
	args := []interface{}{}
	if component != "" {
		query += ` WHERE component=$1`
		args = append(args, component)
	}
	query += ` ORDER BY timestamp DESC LIMIT $` + fmt.Sprintf("%d", len(args)+1)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []map[string]interface{}
	for rows.Next() {
		var id int64
		var ts time.Time
		var lvl, comp, msg string
		var meta map[string]interface{}
		if err := rows.Scan(&id, &ts, &lvl, &comp, &msg, &meta); err != nil {
			continue
		}
		logs = append(logs, map[string]interface{}{
			"id": id, "timestamp": ts, "level": lvl, "component": comp, "message": msg, "metadata": meta,
		})
	}
	return logs, nil
}

func (r *Repository) InsertKillSwitchEvent(ctx context.Context, reason, source string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO kill_switch_history (reason, source) VALUES ($1, $2)`,
		reason, source,
	)
	return err
}

func (r *Repository) ResolveKillSwitchEvent(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE kill_switch_history SET resolved_at=now() WHERE id=$1`, id,
	)
	return err
}

func (r *Repository) ListKillSwitchHistory(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, triggered_at, reason, source, resolved_at FROM kill_switch_history ORDER BY triggered_at DESC LIMIT $1`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var history []map[string]interface{}
	for rows.Next() {
		var id int64
		var triggered time.Time
		var reason, source string
		var resolved *time.Time
		if err := rows.Scan(&id, &triggered, &reason, &source, &resolved); err != nil {
			continue
		}
		entry := map[string]interface{}{
			"id": id, "triggered_at": triggered, "reason": reason, "source": source,
		}
		if resolved != nil {
			entry["resolved_at"] = *resolved
		}
		history = append(history, entry)
	}
	return history, nil
}

func (r *Repository) DeleteStrategy(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM strategies WHERE id=$1`, id)
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

func (r *Repository) BatchInsertTicks(ctx context.Context, ticks [][]interface{}) error {
	_, err := r.pool.CopyFrom(
		ctx,
		pgx.Identifier{"market_ticks"},
		[]string{"time", "symbol_id", "price_raw", "volume_raw", "bid_price", "ask_price", "bid_size", "ask_size"},
		pgx.CopyFromRows(ticks),
	)
	return err
}

func (r *Repository) CountTable(ctx context.Context, tableName string) (int64, error) {
	if r.pool == nil {
		return 0, fmt.Errorf("no pool")
	}
	var count int64
	err := r.pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
	return count, err
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
