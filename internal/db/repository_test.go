package db

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

func TestConfigDefaults(t *testing.T) {
	t.Setenv("ORCA_DB_HOST", "")
	t.Setenv("ORCA_DB_PORT", "")
	t.Setenv("ORCA_DB_USER", "")
	t.Setenv("ORCA_DB_PASSWORD", "")
	t.Setenv("ORCA_DB_NAME", "")
	t.Setenv("ORCA_DB_SSLMODE", "")

	cfg := DefaultConfig()

	if cfg.Host != "localhost" {
		t.Errorf("default Host = %q, want localhost", cfg.Host)
	}
	if cfg.Port != 5432 {
		t.Errorf("default Port = %d, want 5432", cfg.Port)
	}
	if cfg.Database != "orca_core" {
		t.Errorf("default Database = %q, want orca_core", cfg.Database)
	}
	if cfg.SSLMode != "disable" {
		t.Errorf("default SSLMode = %q, want disable", cfg.SSLMode)
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("ORCA_DB_HOST", "db.example.com")
	t.Setenv("ORCA_DB_PORT", "5433")
	t.Setenv("ORCA_DB_NAME", "orca_prod")

	cfg := DefaultConfig()

	if cfg.Host != "db.example.com" {
		t.Errorf("Host = %q, want db.example.com", cfg.Host)
	}
	if cfg.Port != 5433 {
		t.Errorf("Port = %d, want 5433", cfg.Port)
	}
	if cfg.Database != "orca_prod" {
		t.Errorf("Database = %q, want orca_prod", cfg.Database)
	}
}

func TestPriceScaleConstant(t *testing.T) {
	if PRICE_SCALE_F != 100_000 {
		t.Errorf("PRICE_SCALE_F = %d, want 100_000", PRICE_SCALE_F)
	}
}

func TestCandleTypePriceConversion(t *testing.T) {
	c := Candle{
		Time:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Open:   types.PriceFromFloat(100.0),
		High:   types.PriceFromFloat(101.0),
		Low:    types.PriceFromFloat(99.0),
		Close:  types.PriceFromFloat(100.5),
		Volume: 1000,
		Symbol: "TEST",
	}
	if c.Open.Float64() != 100.0 {
		t.Fatalf("Open field broken: got %f, want 100.0", c.Open.Float64())
	}
	if c.Close.Float64() != 100.5 {
		t.Fatalf("Close field broken: got %f, want 100.5", c.Close.Float64())
	}
}

func TestCandlePriceRoundTrip(t *testing.T) {
	tests := []float64{0.01, 1.0, 100.0, 9999.99, 50000.0}
	for _, price := range tests {
		p := types.PriceFromFloat(price)
		got := p.Float64()
		diff := got - price
		if diff < -0.001 || diff > 0.001 {
			t.Errorf("Price round-trip error: input=%f, output=%f, diff=%f", price, got, diff)
		}
	}
}

func TestCandlePriceScalePrecision(t *testing.T) {
	p := types.PriceFromFloat(123.45678)
	got := p.Float64()
	if got < 123.45 || got > 123.46 {
		t.Errorf("Expected ~123.456, got %f", got)
	}
	if p.Int64() != 12345678 {
		t.Errorf("Raw Int64 = %d, want 12345678", p.Int64())
	}
}

func TestTradeExecutionTypeFields(t *testing.T) {
	te := TradeExecution{
		ID:                    "trade-1",
		StrategyID:            "ma_crossover",
		Symbol:                "SPY",
		Side:                  "BUY",
		Quantity:              100,
		Price:                 types.PriceFromFloat(450.0),
		HMMRegime:             0,
		RiskApproved:          true,
		ConsistencyMultiplier: 0.8,
		RejectedReason:        "",
		ExecutedAt:            time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
		BrokerOrderID:         "broker-abc-123",
	}

	if te.Symbol != "SPY" {
		t.Errorf("Symbol = %q, want SPY", te.Symbol)
	}
	if te.Quantity != 100 {
		t.Errorf("Quantity = %f, want 100", te.Quantity)
	}
	if !te.RiskApproved {
		t.Error("RiskApproved should be true")
	}
	if te.Price.Float64() != 450.0 {
		t.Errorf("Price = %f, want 450.0", te.Price.Float64())
	}
}

func TestStrategyTypeJSONRoundTrip(t *testing.T) {
	s := Strategy{
		ID:      "strat-001",
		Name:    "MA Crossover",
		Type:    "trend_following",
		Parameters: map[string]interface{}{
			"fast_period":  10.0,
			"slow_period":  50.0,
			"kelly_fraction": 0.25,
		},
		Enabled:   true,
		CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded Strategy
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.ID != s.ID {
		t.Errorf("ID mismatch: %q vs %q", decoded.ID, s.ID)
	}
	if decoded.Name != s.Name {
		t.Errorf("Name mismatch: %q vs %q", decoded.Name, s.Name)
	}
	if !decoded.Enabled {
		t.Error("Enabled should be true after decode")
	}
}

func TestCandleJSONRoundTrip(t *testing.T) {
	c := Candle{
		Time:   time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC),
		Open:   types.PriceFromFloat(450.0),
		High:   types.PriceFromFloat(452.5),
		Low:    types.PriceFromFloat(448.0),
		Close:  types.PriceFromFloat(451.25),
		Volume: 500000,
		Symbol: "SPY",
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded Candle
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.Symbol != "SPY" {
		t.Errorf("Symbol = %q, want SPY", decoded.Symbol)
	}
	if decoded.Volume != 500000 {
		t.Errorf("Volume = %f, want 500000", decoded.Volume)
	}
	if decoded.Close.Float64() != 451.25 {
		t.Errorf("Close = %f, want 451.25", decoded.Close.Float64())
	}
}

func TestSymbolTypeDefaults(t *testing.T) {
	sym := Symbol{
		ID:        1,
		Ticker:    "SPY",
		Exchange:  "NYSE",
		AssetType: "equity",
		TickSize:  0.01,
		LotSize:   1,
		IsActive:  true,
	}

	if sym.AssetType != "equity" {
		t.Errorf("AssetType = %q, want equity", sym.AssetType)
	}
	if sym.TickSize != 0.01 {
		t.Errorf("TickSize = %f, want 0.01", sym.TickSize)
	}
}

func TestRegimeLogType(t *testing.T) {
	rl := RegimeLog{
		Time:       time.Date(2025, 7, 1, 10, 0, 0, 0, time.UTC),
		HMMState:   2,
		Confidence: 0.85,
		Symbol:     "SPY",
	}

	if rl.HMMState != 2 {
		t.Errorf("HMMState = %d, want 2", rl.HMMState)
	}
	if rl.Confidence < 0.8 || rl.Confidence > 0.9 {
		t.Errorf("Confidence = %f, want ~0.85", rl.Confidence)
	}
}

func TestMatrixProgressRecordDefaults(t *testing.T) {
	m := MatrixProgressRecord{
		BatchID:   "batch-001",
		Mode:      "single",
		Total:     100,
		Completed: 50,
		Status:    "running",
	}

	if m.Mode != "single" {
		t.Errorf("Mode = %q, want single", m.Mode)
	}
	if m.Completed != 50 {
		t.Errorf("Completed = %d, want 50", m.Completed)
	}
}

func TestRepositoryMethodsExist(t *testing.T) {
	r := &Repository{}
	if r == nil {
		t.Fatal("Repository should not be nil")
	}
	_ = r.LoadCandles
	_ = r.LoadCandlesByTimeframe
	_ = r.ListStrategies
	_ = r.ListSymbols
	_ = r.Ping
	_ = r.Close
	_ = r.Pool
	_ = r.IsConnected
}
