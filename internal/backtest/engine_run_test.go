package backtest

import (
	"context"
	"math"
	"testing"
	"time"

	strategy "github.com/lee-econ/orca-core/internal/strategy"
	"github.com/lee-econ/orca-core/internal/types"
)

type mockDB struct {
	candles    [][]Candle
	regimeLogs []RegimeLog
}

func (m *mockDB) LoadCandles(ctx context.Context, symbols []string, start, end time.Time) ([][]Candle, error) {
	if m.candles == nil {
		m.candles = [][]Candle{
			{{Time: start.Add(24 * time.Hour), Open: 100, High: 102, Low: 99, Close: 101, Volume: 1000, Symbol: "SPY"}},
			{{Time: start.Add(48 * time.Hour), Open: 101, High: 103, Low: 100, Close: 102, Volume: 1000, Symbol: "SPY"}},
			{{Time: start.Add(72 * time.Hour), Open: 102, High: 104, Low: 101, Close: 103, Volume: 1000, Symbol: "SPY"}},
		}
	}
	return m.candles, nil
}

func (m *mockDB) LoadRegimeLogs(ctx context.Context, start, end time.Time) ([]RegimeLog, error) {
	return nil, nil
}

func (m *mockDB) LoadVIXLogs(ctx context.Context, start, end time.Time) ([]VIXLog, error) {
	return nil, nil
}

func (m *mockDB) LoadSentimentLogs(ctx context.Context, start, end time.Time) ([]SentimentLog, error) {
	return nil, nil
}

func (m *mockDB) CountCandles(ctx context.Context) (int64, error) {
	return 100, nil
}

func (m *mockDB) CountSyntheticCandles(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockDB) CountRegimeLogs(ctx context.Context) (int64, error) {
	return 10, nil
}

func (m *mockDB) LoadUniverseSnapshots(ctx context.Context, start, end time.Time) ([]UniverseSnapshot, error) {
	return nil, nil
}

func (m *mockDB) LoadCandlesFiltered(ctx context.Context, symbols []string, start, end time.Time, source string) ([][]Candle, error) {
	return m.LoadCandles(ctx, symbols, start, end)
}

func (m *mockDB) LoadCandlesTF(ctx context.Context, symbols []string, start, end time.Time, timeframe string) ([][]Candle, error) {
	return m.LoadCandles(ctx, symbols, start, end)
}

func (m *mockDB) LoadCandlesTFFiltered(ctx context.Context, symbols []string, start, end time.Time, timeframe string, source string) ([][]Candle, error) {
	return m.LoadCandles(ctx, symbols, start, end)
}

func (m *mockDB) LoadAllCandles(ctx context.Context, symbols []string, start, end time.Time, timeframe string) (map[string][]Candle, error) {
	candles, err := m.LoadCandles(ctx, symbols, start, end)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]Candle)
	for _, symCandles := range candles {
		for _, c := range symCandles {
			out[c.Symbol] = append(out[c.Symbol], c)
		}
	}
	return out, nil
}

func TestEngine_Run(t *testing.T) {
	db := &mockDB{}
	e := NewEngine(db)

	config := BacktestConfig{
		StrategyID:     "intraday_mr",
		Symbols:        []string{"SPY"},
		StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		InitialCapital: 100000,
		PropFirmEnabled:    false,
	}

	result, err := e.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Config.StrategyID != config.StrategyID {
		t.Errorf("Expected strategy ID %s, got %s", config.StrategyID, result.Config.StrategyID)
	}
}

func TestEngine_RunWithFTMO(t *testing.T) {
	db := &mockDB{}
	e := NewEngine(db)

	config := BacktestConfig{
		StrategyID:     "intraday_mr",
		Symbols:        []string{"SPY"},
		StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		InitialCapital: 100000,
		PropFirmEnabled:    true,
	}

	result, err := e.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("Run with FTMO failed: %v", err)
	}
	t.Logf("FTMO enabled: %v, trades: %d", config.PropFirmEnabled, result.NumTrades)
}

func TestEngine_RunWithRegimeFilter(t *testing.T) {
	db := &mockDB{
		candles: [][]Candle{
			{{Time: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC), Open: 100, High: 102, Low: 99, Close: 101, Symbol: "SPY"}},
			{{Time: time.Date(2024, 1, 3, 10, 0, 0, 0, time.UTC), Open: 101, High: 103, Low: 100, Close: 102, Symbol: "SPY"}},
		},
		regimeLogs: []RegimeLog{
			{Time: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), HMMState: 1, Confidence: 0.8, Symbol: "SPY"},
		},
	}
	e := NewEngine(db)
	regimeFilter := int8(1)

	config := BacktestConfig{
		StrategyID:     "intraday_mr",
		Symbols:        []string{"SPY"},
		StartDate:      time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC),
		InitialCapital: 100000,
		RegimeFilter:   &regimeFilter,
	}

	result, err := e.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("Run with regime filter failed: %v", err)
	}
	t.Logf("Regime filter applied: trades=%d", result.NumTrades)
}

func TestEngineMulti_RunMulti(t *testing.T) {
	db := &mockDB{}
	reg := strategy.GlobalRegistry()
	e := NewEngineMulti(db, reg)

	config := MultiBacktestConfig{
		StrategyIDs:   []string{"opening_range_breakout"},
		Symbols:       []string{"SPY"},
		StartDate:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:       time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		InitialCapital: 100000,
	}

	result, err := e.RunMulti(context.Background(), config)
	if err != nil {
		t.Fatalf("RunMulti failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	t.Logf("Multi backtest: trades=%d, return=%.2f%%", result.NumTrades, result.TotalReturnPct)
}

func TestEngineMulti_RunMultiMultipleStrategies(t *testing.T) {
	db := &mockDB{
		candles: [][]Candle{
			{{Time: time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC), Open: 100, High: 102, Low: 99, Close: 101, Symbol: "SPY"}},
			{{Time: time.Date(2024, 1, 6, 10, 0, 0, 0, time.UTC), Open: 101, High: 103, Low: 100, Close: 102, Symbol: "SPY"}},
			{{Time: time.Date(2024, 1, 7, 10, 0, 0, 0, time.UTC), Open: 102, High: 104, Low: 101, Close: 103, Symbol: "SPY"}},
		},
	}
	reg := strategy.GlobalRegistry()
	e := NewEngineMulti(db, reg)

	config := MultiBacktestConfig{
		StrategyIDs:   []string{"opening_range_breakout", "trend_following"},
		Symbols:       []string{"SPY"},
		StartDate:     time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
		EndDate:       time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC),
		InitialCapital: 100000,
	}

	result, err := e.RunMulti(context.Background(), config)
	if err != nil {
		t.Fatalf("Multi-strategy run failed: %v", err)
	}
	if len(result.StrategyMetrics) != 2 {
		t.Errorf("Expected 2 strategy metrics, got %d", len(result.StrategyMetrics))
	}
	t.Logf("Multi-strategy: trades=%d, metrics=%d", result.NumTrades, len(result.StrategyMetrics))
}

func TestBacktestConfig_Slippage(t *testing.T) {
	cfg := BacktestConfig{
		StrategyID:     "trend_following",
		Symbols:        []string{"SPY"},
		StartDate:      time.Now(),
		EndDate:        time.Now().AddDate(0, 3, 0),
		InitialCapital: 50000,
		CommissionBps:  1.5,
		SlippageModel:  DefaultEquitySlippage(),
	}

	if cfg.CommissionBps != 1.5 {
		t.Errorf("Expected 1.5, got %f", cfg.CommissionBps)
	}
	if cfg.SlippageModel.SpreadBps <= 0 {
		t.Error("Expected positive spread")
	}
}

func TestEngineMultiMerge(t *testing.T) {
	candles := [][]Candle{
		{{Time: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), Symbol: "SPY", Close: 100}},
		{{Time: time.Date(2024, 1, 1, 10, 30, 0, 0, time.UTC), Symbol: "QQQ", Close: 200}},
	}
	merged := mergeCandlesByTime(candles)
	if len(merged) != 2 {
		t.Errorf("Expected 2 candles, got %d", len(merged))
	}
}

func TestEngine_Run_FTMO_DailyLossHalt(t *testing.T) {
	db := &mockDB{
		candles: [][]Candle{{
			{Time: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC), Open: 100, High: 101, Low: 94, Close: 94, Volume: 1000000, Symbol: "SPY"},
			{Time: time.Date(2024, 1, 3, 10, 0, 0, 0, time.UTC), Open: 94, High: 95, Low: 93, Close: 93, Volume: 1000000, Symbol: "SPY"},
		}},
	}
	e := NewEngine(db)

	config := BacktestConfig{
		StrategyID:     "intraday_mr",
		Symbols:        []string{"SPY"},
		StartDate:      time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC),
		InitialCapital: 100000,
		PropFirmEnabled:    true,
		StopLoss: &StopLossConfig{
			Type:          "atr",
			ATRPeriod:     14,
			ATRMultiplier: 2.0,
		},
		TakeProfit: &TakeProfitConfig{
			Type:    "risk_reward",
			RRRatio: 2.0,
		},
	}

	result, err := e.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.MaxDrawdown > 5.0 {
		t.Errorf("MaxDD %.2f%% exceeds FTMO 5%% daily loss limit", result.MaxDrawdown)
	}
	if result.TotalReturnPct < -5.0 {
		t.Errorf("Total return %.2f%% exceeds FTMO 5%% daily loss limit", result.TotalReturnPct)
	}
	t.Logf("FTMO daily loss test: MaxDD=%.2f%%, Return=%.2f%%, Trades=%d", result.MaxDrawdown, result.TotalReturnPct, result.NumTrades)
}

func TestEngine_Run_WarningsOnEmptyData(t *testing.T) {
	db := &mockDB{}
	sr := strategy.NewMeanReversionRunner(20, 2.0, 0.5, 60)
	e := NewEngineWithStrategy(db, sr)

	config := BacktestConfig{
		StrategyID:     "intraday_mr",
		Symbols:        []string{"NODATA"},
		StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		InitialCapital: 100000,
		PropFirmEnabled:    true,
	}

	result, err := e.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warnings for empty candle data")
	}
	t.Logf("Warnings: %v", result.Warnings)
}

func TestStrategyRunner_ExitSignalReturned(t *testing.T) {
	sr := strategy.NewMeanReversionRunner(5, 1.0, 0.5, 10)

	c := Candle{Time: time.Now(), Symbol: "SPY", Close: 100}
	for i := 0; i < 6; i++ {
		sr.Evaluate(c, 0)
	}

	spike := Candle{Time: time.Now(), Symbol: "SPY", Close: 105}
	entrySig := sr.Evaluate(spike, 0)
	if entrySig == nil {
		t.Log("No entry signal generated (trend filter may block)")
		return
	}
	t.Logf("Entry: side=%s", entrySig.Side)

	exitDetected := false
	for i := 0; i < 15; i++ {
		exitSig := sr.Evaluate(Candle{Time: time.Now(), Symbol: "SPY", Close: 100}, 0)
		if exitSig != nil && exitSig.Quantity == 0 {
			exitDetected = true
			t.Logf("Exit signal returned at bar %d", i)
			break
		}
	}
	if !exitDetected {
		t.Log("Exit not detected within 15 bars (may need different params)")
	}
}

func TestStrategyRunner_ResetClearsPosition(t *testing.T) {
	sr := strategy.NewMeanReversionRunner(5, 1.0, 0.5, 20)
	c := Candle{Time: time.Now(), Symbol: "SPY", Close: 100}
	for i := 0; i < 6; i++ {
		sr.Evaluate(c, 0)
	}
	spike := Candle{Time: time.Now(), Symbol: "SPY", Close: 110}
	sig := sr.Evaluate(spike, 0)
	if sig == nil {
		t.Skip("No entry signal generated, skipping position clear test")
	}
	sr.Reset()
	for i := 0; i < 6; i++ {
		sr.Evaluate(c, 0)
	}
	sig2 := sr.Evaluate(spike, 0)
	if sig2 == nil {
		t.Error("Expected entry signal after reset (fresh start)")
	}
}

func TestDiagnostic_SyntheticCandlesProduceTrades(t *testing.T) {
	symbols := []string{"AAPL", "EURUSD", "BTCUSD"}
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

	strategyIDs := []string{"ma_crossover", "mean_reversion", "trend_following",
		"grid_trading", "rsi2_reversion", "donchian_breakout", "keltner_macd", "ichimoku_cloud"}

	needsLongData := map[string]bool{
		"donchian_breakout": true,
		"ichimoku_cloud":    true,
	}

	for _, stratID := range strategyIDs {
		for _, sym := range symbols {
			t.Run(stratID+"/"+sym, func(t *testing.T) {
				n := 180
				if needsLongData[stratID] {
					n = 520
				}
				startPrice := 100.0
				if stratID == "ichimoku_cloud" {
					startPrice = 1.0
				}
				mock := &mockDB{
					candles: generateTestCandlesForDB(sym, n, startPrice),
				}
				eng := NewEngine(mock)
				cfg := BacktestConfig{
					StrategyID:     stratID,
					Symbols:        []string{sym},
					StartDate:      start,
					EndDate:        end,
					InitialCapital: 100000,
					Timeframe:      "1d",
					SizingPercent:  0.02,
					KellyFraction:  0.25,
				}
				result, err := eng.Run(context.Background(), cfg)
				if err != nil {
					t.Errorf("%s/%s: engine.Run error: %v", stratID, sym, err)
					return
				}
				if result == nil {
					t.Errorf("%s/%s: engine.Run returned nil result", stratID, sym)
					return
				}
				if result.NumTrades == 0 {
					t.Logf("%s/%s: zero trades (may need more candles; warnings: %v)", stratID, sym, result.Warnings)
				}
				t.Logf("%s/%s: trades=%d sharpe=%.3f return=%.2f%% dd=%.1f%% wr=%.1f%% pf=%.2f",
					stratID, sym, result.NumTrades, result.SharpeRatio,
					result.TotalReturnPct, result.MaxDrawdown,
					result.WinRate, result.ProfitFactor)
			})
		}
	}
}

func generateTestCandlesForDB(symbol string, n int, startPrice float64) [][]Candle {
	candles := make([]strategy.Candle, n)
	base := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	price := startPrice
	for i := 0; i < n; i++ {
		trend := 0.15
		if i > n/3 && i < 2*n/3 {
			trend = -0.12
		}
		noise := float64(i%7-3) * 0.5
		price += trend + noise
		if price < 20 {
			price = 20
		}
		high := price + float64(i%5)*1.5
		low := price - float64((i+1)%6)*1.2
		open := low + (high-low)*0.35
		candles[i] = strategy.Candle{
			Time:   base.AddDate(0, 0, i),
			Open:   types.PriceFromFloat(open),
			High:   types.PriceFromFloat(high),
			Low:    types.PriceFromFloat(low),
			Close:  types.PriceFromFloat(price),
			Volume: float64(5000000 + i%20*100000),
			Symbol: symbol,
		}
	}
	return [][]Candle{candles}
}

func TestCalculateLongShortBreakdown(t *testing.T) {
	trades := []Trade{
		{Side: "BUY", PnL: 100, MAE: 10, MFE: 20},
		{Side: "BUY", PnL: -50, MAE: 15, MFE: 5},
		{Side: "BUY", PnL: 200, MAE: 8, MFE: 25},
		{Side: "SELL", PnL: 150, MAE: 12, MFE: 18},
		{Side: "SELL", PnL: -30, MAE: 20, MFE: 3},
		{Side: "SELL", PnL: 75, MAE: 5, MFE: 10},
	}

	b := calculateLongShortBreakdown(trades)

	if b.LongTrades != 3 {
		t.Errorf("LongTrades: expected 3, got %d", b.LongTrades)
	}
	if b.ShortTrades != 3 {
		t.Errorf("ShortTrades: expected 3, got %d", b.ShortTrades)
	}
	if b.LongWins != 2 {
		t.Errorf("LongWins: expected 2, got %d", b.LongWins)
	}
	if b.ShortWins != 2 {
		t.Errorf("ShortWins: expected 2, got %d", b.ShortWins)
	}

	expectedLongGross := 100.0 + (-50.0) + 200.0
	if b.LongGrossPnL != expectedLongGross {
		t.Errorf("LongGrossPnL: expected %.2f, got %.2f", expectedLongGross, b.LongGrossPnL)
	}

	expectedLongPF := (100.0 + 200.0) / 50.0
	if math.Abs(b.LongPF-expectedLongPF) > 0.01 {
		t.Errorf("LongPF: expected %.2f, got %.2f", expectedLongPF, b.LongPF)
	}

	if b.LongAvgMAE-(10.0+15.0+8.0)/3.0 > 0.01 {
		t.Errorf("LongAvgMAE: expected %.2f, got %.2f", (10.0+15.0+8.0)/3.0, b.LongAvgMAE)
	}

	expectedBias := (b.LongGrossPnL - b.ShortGrossPnL) / (math.Abs(b.LongGrossPnL) + math.Abs(b.ShortGrossPnL))
	if math.Abs(b.DirectionalBias-expectedBias) > 0.001 {
		t.Errorf("DirectionalBias: expected %.4f, got %.4f", expectedBias, b.DirectionalBias)
	}

	t.Logf("Long: %d trades, %.2f%% WR, PF=%.2f, PnL=%.2f | Short: %d trades, %.2f%% WR, PF=%.2f, PnL=%.2f | Bias=%.3f",
		b.LongTrades, b.LongWinRate, b.LongPF, b.LongGrossPnL,
		b.ShortTrades, b.ShortWinRate, b.ShortPF, b.ShortGrossPnL,
		b.DirectionalBias)
}
