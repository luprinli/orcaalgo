package backtest

import (
	"context"
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

// pairMockDB returns cointegrated primary/secondary candles with a
// mean-reverting log-spread so the pairs runner can generate entries.
type pairMockDB struct {
	candles [][]Candle
}

func (m *pairMockDB) LoadCandles(ctx context.Context, symbols []string, start, end time.Time) ([][]Candle, error) {
	return m.candles, nil
}
func (m *pairMockDB) LoadCandlesFiltered(ctx context.Context, symbols []string, start, end time.Time, source string) ([][]Candle, error) {
	return m.candles, nil
}
func (m *pairMockDB) LoadCandlesTF(ctx context.Context, symbols []string, start, end time.Time, timeframe string) ([][]Candle, error) {
	return m.candles, nil
}
func (m *pairMockDB) LoadCandlesTFFiltered(ctx context.Context, symbols []string, start, end time.Time, timeframe, source string) ([][]Candle, error) {
	return m.candles, nil
}
func (m *pairMockDB) LoadAllCandles(ctx context.Context, symbols []string, start, end time.Time, timeframe string) (map[string][]Candle, error) {
	out := map[string][]Candle{}
	for _, row := range m.candles {
		for _, c := range row {
			out[c.Symbol] = append(out[c.Symbol], c)
		}
	}
	return out, nil
}
func (m *pairMockDB) LoadRegimeLogs(ctx context.Context, start, end time.Time) ([]RegimeLog, error) {
	return nil, nil
}
func (m *pairMockDB) LoadVIXLogs(ctx context.Context, start, end time.Time) ([]VIXLog, error) {
	return nil, nil
}
func (m *pairMockDB) LoadSentimentLogs(ctx context.Context, start, end time.Time) ([]SentimentLog, error) {
	return nil, nil
}
func (m *pairMockDB) CountCandles(ctx context.Context) (int64, error)          { return 0, nil }
func (m *pairMockDB) CountSyntheticCandles(ctx context.Context) (int64, error) { return 0, nil }
func (m *pairMockDB) CountRegimeLogs(ctx context.Context) (int64, error)       { return 0, nil }
func (m *pairMockDB) LoadUniverseSnapshots(ctx context.Context, start, end time.Time) ([]UniverseSnapshot, error) {
	return nil, nil
}

func TestPairsTrading_EngineWiring(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	const n = 150
	rng := rand.New(rand.NewPCG(42, 1))
	eur := make([]Candle, n)
	gbp := make([]Candle, n)
	// Mean-reverting log-spread with occasional divergence spikes so the
	// rolling z-score crosses the entry threshold (±2). The secondary price is
	// held constant to isolate the engine's secondary-symbol wiring (the
	// correlation gate is disabled via min_pair_correlation=0 below).
	spread := 0.0
	for i := 0; i < n; i++ {
		d := start.Add(time.Duration(i) * 24 * time.Hour)
		spread = 0.7*spread + (rng.Float64()-0.5)*0.004
		if i%30 == 15 {
			spread += 0.015 // divergence shock
		}
		if i%30 == 29 {
			spread -= 0.015 // divergence shock (other direction)
		}
		g := 1.27
		e := g * math.Exp(spread)
		eur[i] = Candle{Time: d, Symbol: "EURUSD", Open: types.PriceFromFloat(e), High: types.PriceFromFloat(e * 1.001), Low: types.PriceFromFloat(e * 0.999), Close: types.PriceFromFloat(e), Volume: 1000}
		gbp[i] = Candle{Time: d, Symbol: "GBPUSD", Open: types.PriceFromFloat(g), High: types.PriceFromFloat(g * 1.001), Low: types.PriceFromFloat(g * 0.999), Close: types.PriceFromFloat(g), Volume: 1000}
	}
	db := &pairMockDB{candles: [][]Candle{eur, gbp}}

	cfg := BacktestConfig{
		StrategyID:       "pairs_trading",
		Symbols:          []string{"EURUSD", "GBPUSD"},
		SecondarySymbols: map[string]string{"EURUSD": "GBPUSD"},
		StartDate:        start,
		EndDate:          start.Add(time.Duration(n) * 24 * time.Hour),
		InitialCapital:   100000,
		PropFirmEnabled:  false,
		// This test exercises the engine's secondary-symbol wiring only; the
		// cointegration correlation gate is covered separately by
		// TestPairsRunnerPairLogCorrelation.
		StrategyParams: map[string]float64{"min_pair_correlation": 0},
	}

	e := NewEngine(db)
	result, err := e.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	t.Logf("pairs_trading: trades=%d candles=%d warnings=%v", result.NumTrades, result.CandleCount, result.Warnings)
	t.Logf("signalDiag: %+v", result.SignalDiag)
	if result.NumTrades == 0 {
		t.Error("pairs_trading produced 0 trades with cointegrated EURUSD/GBPUSD data — engine secondary-symbol wiring is broken")
	}
}
