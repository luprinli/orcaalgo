package backtest

import (
	"context"
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/risk"
	"github.com/lee-econ/orca-core/internal/types"
)

type parityDB struct {
	candles []Candle
	regime  []RegimeLog
	vix     []VIXLog
}

func (p *parityDB) LoadCandles(ctx context.Context, symbols []string, start, end time.Time) ([][]Candle, error) {
	return [][]Candle{p.candles}, nil
}
func (p *parityDB) LoadCandlesFiltered(ctx context.Context, symbols []string, start, end time.Time, source string) ([][]Candle, error) {
	return [][]Candle{p.candles}, nil
}
func (p *parityDB) LoadCandlesTF(ctx context.Context, symbols []string, start, end time.Time, timeframe string) ([][]Candle, error) {
	return [][]Candle{p.candles}, nil
}
func (p *parityDB) LoadCandlesTFFiltered(ctx context.Context, symbols []string, start, end time.Time, timeframe string, source string) ([][]Candle, error) {
	return [][]Candle{p.candles}, nil
}
func (p *parityDB) LoadAllCandles(ctx context.Context, symbols []string, start, end time.Time, timeframe string) (map[string][]Candle, error) {
	m := make(map[string][]Candle)
	for _, s := range symbols {
		m[s] = p.candles
	}
	return m, nil
}
func (p *parityDB) LoadRegimeLogs(ctx context.Context, start, end time.Time) ([]RegimeLog, error) {
	return p.regime, nil
}
func (p *parityDB) LoadVIXLogs(ctx context.Context, start, end time.Time) ([]VIXLog, error) {
	return p.vix, nil
}
func (p *parityDB) LoadSentimentLogs(ctx context.Context, start, end time.Time) ([]SentimentLog, error) {
	return nil, nil
}
func (p *parityDB) CountCandles(ctx context.Context) (int64, error) {
	return int64(len(p.candles)), nil
}
func (p *parityDB) CountSyntheticCandles(ctx context.Context) (int64, error) { return 0, nil }
func (p *parityDB) CountRegimeLogs(ctx context.Context) (int64, error)      { return int64(len(p.regime)), nil }
func (p *parityDB) LoadUniverseSnapshots(ctx context.Context, start, end time.Time) ([]UniverseSnapshot, error) {
	return nil, nil
}

func makeTestCandles(count int) []Candle {
	ts := time.Date(2025, 1, 1, 9, 30, 0, 0, time.UTC)
	candles := make([]Candle, count)
	basePrice := 100.0
	for i := 0; i < count; i++ {
		candles[i] = Candle{
			Time:   ts.Add(time.Duration(i) * time.Minute),
			Symbol: "TEST",
			Open:   types.PriceFromFloat(basePrice + float64(i)*0.01),
			High:   types.PriceFromFloat(basePrice + float64(i)*0.02 + 0.5),
			Low:    types.PriceFromFloat(basePrice + float64(i)*0.01 - 0.5),
			Close:  types.PriceFromFloat(basePrice + float64(i+1)*0.01),
			Volume: 10000,
		}
	}
	return candles
}

func TestBacktestLiveParity(t *testing.T) {
	candles := makeTestCandles(25)

	regime := []RegimeLog{
		{Time: time.Date(2025, 1, 1, 9, 30, 0, 0, time.UTC), HMMState: 0, Confidence: 0.9, Symbol: "TEST"},
	}

	db := &parityDB{candles: candles, regime: regime, vix: nil}

	engineA := NewEngine(db)
	engineB := NewEngine(db)

	cfg := BacktestConfig{
		StrategyID:     "rsi2_reversion",
		Symbols:        []string{"TEST"},
		StartDate:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		InitialCapital: 100000,
		DataSource:     "test",
	}

	ctx := context.Background()

	resultA, errA := engineA.Run(ctx, cfg)
	if errA != nil {
		t.Fatalf("engineA failed: %v", errA)
	}

	resultB, errB := engineB.Run(ctx, cfg)
	if errB != nil {
		t.Fatalf("engineB failed: %v", errB)
	}

	if len(resultA.Trades) != len(resultB.Trades) {
		t.Errorf("trade count mismatch: A=%d, B=%d", len(resultA.Trades), len(resultB.Trades))
	}

	if len(resultA.Trades) > 0 && len(resultB.Trades) > 0 {
		if resultA.TotalReturn != resultB.TotalReturn {
			t.Errorf("TotalReturn mismatch: A=%f, B=%f", resultA.TotalReturn, resultB.TotalReturn)
		}
		if resultA.SharpeRatio != resultB.SharpeRatio {
			t.Errorf("SharpeRatio mismatch: A=%f, B=%f", resultA.SharpeRatio, resultB.SharpeRatio)
		}
		if resultA.MaxDrawdown != resultB.MaxDrawdown {
			t.Errorf("MaxDrawdown mismatch: A=%f, B=%f", resultA.MaxDrawdown, resultB.MaxDrawdown)
		}
	}

	if resultA.TotalReturnPct != resultB.TotalReturnPct {
		t.Errorf("TotalReturnPct mismatch: A=%f, B=%f", resultA.TotalReturnPct, resultB.TotalReturnPct)
	}
}

func TestSortinoGuard_D1(t *testing.T) {
	equity := []EquityPoint{
		{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Value: 100000},
		{Time: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Value: 100000},
	}
	ratio := calculateSortino(equity, 1.0)
	if ratio != 0 {
		t.Errorf("expected Sortino=0 for flat equity (no downside deviation), got %f", ratio)
	}
}

func TestSortinoGuard_D1_NearZeroDownside(t *testing.T) {
	equity := []EquityPoint{
		{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Value: 100000},
		{Time: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Value: 100000.000001},
	}
	ratio := calculateSortino(equity, 1.0)
	if ratio != 0 {
		t.Errorf("expected Sortino=0 for near-zero downside deviation, got %f", ratio)
	}
}

func TestRegimeActivationMatrix_rsi2(t *testing.T) {
	matrix := risk.NewRegimeActivationMatrix()

	if matrix.IsAllowed("rsi2_reversion", 0) != true {
		t.Error("rsi2_reversion should be allowed in Calm regime")
	}
	if matrix.IsAllowed("rsi2_reversion", 1) != true {
		t.Error("rsi2_reversion should be allowed in Trending regime")
	}
	if matrix.IsAllowed("rsi2_reversion", 2) != false {
		t.Error("rsi2_reversion should NOT be allowed in HighVol regime")
	}
	if matrix.IsAllowed("rsi2_reversion", 3) != false {
		t.Error("rsi2_reversion should NOT be allowed in Crisis regime")
	}
}
