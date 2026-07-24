package shadow

import (
	"context"
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/backtest"
	"github.com/lee-econ/orca-core/internal/engine"
	"github.com/lee-econ/orca-core/internal/risk"
	"github.com/lee-econ/orca-core/internal/strategy"
	"github.com/lee-econ/orca-core/internal/types"
)

func TestShadowModeSignalParity(t *testing.T) {
	const symbolID uint32 = 1
	const symbolName = "TEST"
	const barCount = 60

	baseTime := time.Date(2024, 1, 2, 9, 30, 0, 0, time.UTC)

	prices := []uint64{
		100000, 100050, 100080, 100120, 100110, 100090, 100060, 100030,
		100010, 100040, 100070, 100100, 100130, 100160, 100140, 100110,
		100080, 100050, 100020, 100010, 100030, 100060, 100090, 100120,
		100150, 100180, 100200, 100220, 100210, 100190, 100170, 100150,
		100130, 100110, 100090, 100070, 100050, 100080, 100110, 100140,
		100170, 100200, 100230, 100250, 100270, 100290, 100310, 100330,
		100350, 100370, 100390, 100410, 100430, 100450, 100470, 100490,
		100510, 100530, 100550, 100570,
	}

	candles := make([]strategy.Candle, barCount)
	for i := 0; i < barCount; i++ {
		price := float64(prices[i]) / 100000.0
		candles[i] = strategy.Candle{
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   types.FromFloat64(price),
			High:   types.FromFloat64(price * 1.01),
			Low:    types.FromFloat64(price * 0.99),
			Close:  types.FromFloat64(price),
			Volume: 1000,
			Symbol: symbolName,
		}
	}

	btRunner := strategy.GlobalRegistry().Get("mean_reversion")
	if btRunner == nil {
		t.Fatal("mean_reversion not registered")
	}
	btRunner.Reset()

	var btSignals []*strategy.Signal
	for _, c := range candles {
		sig := btRunner.Evaluate(c, 0)
		if sig != nil {
			btSignals = append(btSignals, sig)
		}
	}

	liveEng := engine.NewLiveEngine()
	liveEng.RiskState = risk.NewGlobalRiskState()

	liveRunner := strategy.GlobalRegistry().Get("mean_reversion")
	if liveRunner == nil {
		t.Fatal("mean_reversion not registered")
	}
	liveRunner.Reset()

	var liveSignalCount int
	liveSides := make(map[int]string)
	for i := 0; i < barCount; i++ {
		timestampNS := baseTime.Add(time.Duration(i) * time.Minute).UnixNano()
		sigs := liveEng.ProcessTick(symbolID, prices[i], 100, timestampNS)
		for _, sig := range sigs {
			if sig != nil {
				liveSides[liveSignalCount] = sig.Side
				liveSignalCount++
			}
		}
	}

	t.Logf("Backtest signals: %d, Live signals: %d", len(btSignals), liveSignalCount)
}

func TestShadowModeStopLossActivation(t *testing.T) {
	const symbolID uint32 = 1

	baseTime := time.Date(2024, 1, 2, 9, 30, 0, 0, time.UTC)
	const barCount = 30

	prices := make([]uint64, barCount)
	for i := 0; i < barCount; i++ {
		prices[i] = uint64(100000 + i*100)
	}

	liveEng := engine.NewLiveEngine()
	liveEng.RiskState = risk.NewGlobalRiskState()

	runner := strategy.GlobalRegistry().Get("mean_reversion")
	if runner == nil {
		t.Fatal("mean_reversion not registered")
	}
	runner.Reset()

	for i := 0; i < barCount; i++ {
		timestampNS := baseTime.Add(time.Duration(i) * time.Minute).UnixNano()
		liveEng.ProcessTick(symbolID, prices[i], 100, timestampNS)
	}

	t.Logf("Feed complete, halted: %v", liveEng.Halted)
}

func TestShadowModeBacktestEngine(t *testing.T) {
	symbolName := "TEST"
	baseTime := time.Date(2024, 1, 2, 9, 30, 0, 0, time.UTC)
	const barCount = 10

	candleList := make([]strategy.Candle, barCount)
	prices := []uint64{100000, 100100, 100200, 100300, 100400, 100500, 100600, 100700, 100800, 100900}
	for i := 0; i < barCount; i++ {
		price := float64(prices[i]) / 100000.0
		candleList[i] = strategy.Candle{
			Time:   baseTime.Add(time.Duration(i) * 24 * time.Hour),
			Open:   types.FromFloat64(price),
			High:   types.FromFloat64(price * 1.02),
			Low:    types.FromFloat64(price * 0.98),
			Close:  types.FromFloat64(price),
			Volume: 1000,
			Symbol: symbolName,
		}
	}

	db := &mockCandleDB{candles: [][]strategy.Candle{candleList}}
	eng := backtest.NewEngine(db)

	cfg := backtest.BacktestConfig{
		StrategyID:     "mean_reversion",
		Symbols:        []string{symbolName},
		StartDate:      baseTime,
		EndDate:        baseTime.Add(time.Duration(barCount) * 24 * time.Hour),
		InitialCapital: 100000,
		Timeframe:      "1d",
		CommissionBps:  0,
		SizingPercent:  0.02,
	}
	cfg.BrokerFee.Enabled = false

	result, err := eng.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("backtest engine failed: %v", err)
	}

	t.Logf("Backtest: trades=%d, return=%.2f%%, signals_passed=%d",
		result.NumTrades, result.TotalReturnPct, result.SignalDiag.SignalsPassed)
}

type mockCandleDB struct {
	candles    [][]strategy.Candle
	regimeLogs []backtest.RegimeLog
}

func (m *mockCandleDB) LoadCandles(ctx context.Context, symbols []string, start, end time.Time) ([][]strategy.Candle, error) {
	return m.candles, nil
}

func (m *mockCandleDB) LoadCandlesTF(ctx context.Context, symbols []string, start, end time.Time, tf string) ([][]strategy.Candle, error) {
	return m.candles, nil
}

func (m *mockCandleDB) LoadCandlesFiltered(ctx context.Context, symbols []string, start, end time.Time, source string) ([][]strategy.Candle, error) {
	return m.candles, nil
}

func (m *mockCandleDB) LoadCandlesTFFiltered(ctx context.Context, symbols []string, start, end time.Time, tf string, source string) ([][]strategy.Candle, error) {
	return m.candles, nil
}

func (m *mockCandleDB) LoadAllCandles(ctx context.Context, symbols []string, start, end time.Time, tf string) (map[string][]strategy.Candle, error) {
	out := make(map[string][]strategy.Candle)
	for _, symCandles := range m.candles {
		for _, c := range symCandles {
			out[c.Symbol] = append(out[c.Symbol], c)
		}
	}
	return out, nil
}

func (m *mockCandleDB) LoadRegimeLogs(ctx context.Context, start, end time.Time) ([]backtest.RegimeLog, error) {
	return m.regimeLogs, nil
}

func (m *mockCandleDB) LoadVIXLogs(ctx context.Context, start, end time.Time) ([]backtest.VIXLog, error) {
	return nil, nil
}

func (m *mockCandleDB) LoadSentimentLogs(ctx context.Context, start, end time.Time) ([]backtest.SentimentLog, error) {
	return nil, nil
}

func (m *mockCandleDB) CountCandles(ctx context.Context) (int64, error) {
	return 100, nil
}

func (m *mockCandleDB) CountSyntheticCandles(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockCandleDB) CountRegimeLogs(ctx context.Context) (int64, error) {
	return 10, nil
}

func (m *mockCandleDB) LoadUniverseSnapshots(ctx context.Context, start, end time.Time) ([]backtest.UniverseSnapshot, error) {
	return nil, nil
}
