package db

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lee-econ/orca-core/internal/metrics"
	"github.com/lee-econ/orca-core/internal/types"
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
	// Default port 5432; Docker compose maps 5433→5432 externally. Set ORCA_DB_PORT to override.
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
	done chan struct{}
}

func NewRepository(cfg Config) (*Repository, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&pool_max_conns=%d&pool_min_conns=%d&pool_max_conn_lifetime=1h&pool_max_conn_idle_time=30m&pool_health_check_period=1m",
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

	repo := &Repository{pool: pool, done: make(chan struct{})}
	go repo.logPoolStats()
	return repo, nil
}

func (r *Repository) Ping(ctx context.Context) error {
	if r.pool == nil {
		return fmt.Errorf("repository pool not initialized")
	}
	return r.pool.Ping(ctx)
}

func (r *Repository) Close() {
	if r.done != nil {
		close(r.done)
	}
	if r.pool != nil {
		r.pool.Close()
	}
}

func (r *Repository) logPoolStats() {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-r.done:
			return
		case <-t.C:
			stat := r.pool.Stat()
			metrics.SetDBPoolInUse(int32(stat.TotalConns()))
			slog.Info("db pool stats", "total", stat.TotalConns(), "idle", stat.IdleConns(), "acquired", stat.AcquireCount())
		}
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

func NewRepositoryFromPool(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const PRICE_SCALE_F = 100_000

type Strategy struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
	Enabled    bool                   `json:"enabled"`
	CreatedAt  time.Time              `json:"created_at"`
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
	ID             int32      `json:"id"`
	Ticker         string     `json:"ticker"`
	Exchange       string     `json:"exchange"`
	AssetType      string     `json:"asset_type"`
	TickSize       float64    `json:"tick_size"`
	LotSize        float64    `json:"lot_size"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	LastPrice      types.Price `json:"last_price,omitempty"`
	MarketCap      int64      `json:"market_cap,omitempty"`
	LastVolume     int64      `json:"last_volume,omitempty"`
	LastATRPct     float64    `json:"last_atr_pct,omitempty"`
	LastRSI        float64    `json:"last_rsi,omitempty"`
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
	FiltersUsed        map[string]interface{}   `json:"filters_used"`
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
	Time             time.Time
	Open             types.Price
	High             types.Price
	Low              types.Price
	Close            types.Price
	Volume           float64
	Symbol           string
	AdjustmentFactor float64
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

const VIXBigintScale = 10000.0

type SentimentLog struct {
	Time  time.Time
	Score int
	Label string
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
