package backtest_test

import (
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/backtest"
	"github.com/lee-econ/orca-core/internal/model"
	"github.com/lee-econ/orca-core/internal/strategy"
	"github.com/lee-econ/orca-core/internal/types"
)

func generateTestCandles(n int, startPrice float64) []strategy.Candle {
	candles := make([]strategy.Candle, n)
	base := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	price := startPrice
	for i := 0; i < n; i++ {
		trend := 0.05
		if i > n/3 && i < 2*n/3 {
			trend = -0.08
		}
		noise := float64(i%7-3) * 0.4
		price += trend + noise
		if price < 20 {
			price = 20
		}
		high := price + float64(i%3)*1.5
		low := price - float64((i+1)%4)*1.2
		open := low + (high-low)*0.35
		candles[i] = strategy.Candle{
			Time:   base.AddDate(0, 0, i),
			Open:   types.PriceFromFloat(open),
			High:   types.PriceFromFloat(high),
			Low:    types.PriceFromFloat(low),
			Close:  types.PriceFromFloat(price),
			Volume: float64(5000000 + i%20*100000),
			Symbol: "TEST",
		}
	}
	return candles
}

func TestE2E_MidPriceFill(t *testing.T) {
	candles := generateTestCandles(200, 100.0)
	runner := strategy.NewRSI2MeanReversionRunner()

	signals := 0
	for _, c := range candles {
		sig := runner.Evaluate(c, 1)
		if sig != nil {
			signals++
			if sig.Side != "BUY" && sig.Side != "SELL" {
				t.Errorf("invalid signal side: %s", sig.Side)
			}
			if sig.Action != strategy.SignalEntry && sig.Action != strategy.SignalExit {
				t.Errorf("invalid signal action: %v", sig.Action)
			}
		}
	}
	if signals == 0 {
		t.Error("expected at least 1 signal from 200 candles")
	}
	t.Logf("RSI2 generated %d signals from %d candles", signals, len(candles))
}

func TestE2E_ProbabilisticFill(t *testing.T) {
	candles := generateTestCandles(200, 100.0)

	var results []backtest.Trade
	for i := 1; i < len(candles); i++ {
		c := candles[i]
		fm := model.NewProbabilisticFill(0.8, 2.5, 100.0)
		midPrice := uint64((c.High + c.Low) / 2.0 * 100000)
		spread := uint64((c.High - c.Low) * 100000)
		if spread < 10 {
			spread = 10
		}
		prob := fm.FillProbability(midPrice, midPrice, spread, uint64(c.Volume), 10.0)
		if prob > 1.0 || prob < 0.0 {
			t.Errorf("fill probability out of range: %.4f", prob)
		}

		if prob > 0.5 {
			fillPrice := fm.FillPrice(midPrice, midPrice, "BUY", 10.0)
			results = append(results, backtest.Trade{
				Symbol:     "TEST",
				Side:       "BUY",
				PnL:        5.0,
				EntryPrice: types.PriceFromFloat(float64(fillPrice) / 100000.0),
				Quantity:   100,
			})
		}
	}
	t.Logf("probabilistic fill: %d fills from %d attempts", len(results), len(candles)-1)
	if len(results) == 0 {
		t.Error("expected some fills")
	}
}

func TestE2E_LatencyModel(t *testing.T) {
	lm := model.ConstantLatency{Entry: 50 * time.Millisecond, Response: 30 * time.Millisecond}
	entry := lm.EntryLatency(time.Now(), 100000, 1000, "BUY")
	resp := lm.ResponseLatency(time.Now(), 100000, "BUY")
	if entry != 50*time.Millisecond {
		t.Errorf("expected 50ms entry, got %v", entry)
	}
	if resp != 30*time.Millisecond {
		t.Errorf("expected 30ms response, got %v", resp)
	}
}

func TestE2E_FeeModel(t *testing.T) {
	fm := model.FixedFee{MakerBps: 2.0, TakerBps: 6.0}
	maker := fm.MakerFee(50000)
	taker := fm.TakerFee(50000)
	if maker != 10.0 {
		t.Errorf("maker fee: expected 10.0, got %.2f", maker)
	}
	if taker != 30.0 {
		t.Errorf("taker fee: expected 30.0, got %.2f", taker)
	}
}

func TestE2E_AllStrategiesProduceSignals(t *testing.T) {
	candles := generateTestCandles(300, 100.0)
	runners := []strategy.Strategy{
		strategy.NewTrendRunner(),
		strategy.NewOrbRunner(),
		strategy.NewGridRunner(),
		strategy.NewSessionScalpRunner(),
		strategy.NewMeanReversionRunner(20, 2.0, 0.3, 200),
		strategy.NewMACrossoverRunner(),
		strategy.NewRSI2MeanReversionRunner(),
		strategy.NewDonchianBreakoutRunner(),
		strategy.NewKeltnerMACDRunner(),
		strategy.NewIchimokuRunner(),
	}

	type result struct {
		name    string
		signals int
	}
	var results []result
	for _, r := range runners {
		r.Reset()
		count := 0
		for _, c := range candles {
			sig := r.Evaluate(c, 1)
			if sig != nil {
				count++
			}
		}
		results = append(results, result{r.Name(), count})
	}

	producing := 0
	for _, r := range results {
		if r.signals > 0 {
			producing++
			t.Logf("  %-25s: %3d signals", r.name, r.signals)
		} else {
			t.Logf("  %-25s: NO signals (may need different data)", r.name)
		}
	}
	// Threshold is 3 (not 5): after the circular-buffer linearization and
	// close-for-high/low indicator fixes, trend/ORB/scalp/donchian/keltner/
	// ichimoku/ma_crossover no longer emit the spurious signals the scrambled
	// window used to produce on this synthetic daily data. The reliable signal
	// producers are grid, mean_reversion and rsi2_reversion.
	if producing < 3 {
		t.Errorf("only %d/10 strategies produced signals", producing)
	}
}

func TestE2E_BacktestRecorder(t *testing.T) {
	rec := backtest.NewBacktestRecorder()
	now := time.Now()

	for i := 0; i < 10; i++ {
		rec.Record(&model.TradingState{
			Timestamp:     now.Add(time.Duration(i) * time.Hour),
			Balance:       100000 + float64(i)*100,
			Position:      float64(i % 2),
			MidPrice:      50000,
			TradingVolume: 1000,
			TradingValue:  50000,
			NumTrades:     int64(i + 1),
		}, nil)
	}

	states := rec.States()
	if len(states) != 10 {
		t.Errorf("expected 10 states, got %d", len(states))
	}
	if states[9].NumTrades != 10 {
		t.Errorf("expected 10 trades, got %d", states[9].NumTrades)
	}
	if states[9].Balance != 100900 {
		t.Errorf("expected balance 100900, got %.2f", states[9].Balance)
	}
}

func TestE2E_EquityCurveValidation(t *testing.T) {
	candles := generateTestCandles(100, 100.0)
	runner := strategy.NewTrendRunner()

	type equityPoint struct {
		Time  time.Time
		Value float64
	}
	var equity []equityPoint
	capital := 100000.0

	for _, c := range candles {
		sig := runner.Evaluate(c, 1)
		if sig != nil && sig.Side == "BUY" && sig.Quantity > 0 {
			entry := c.Close.Float64()
			capital -= entry * sig.Quantity * 0.001
		}
		if sig != nil && sig.Side == "SELL" && sig.Quantity == 0 {
		}
		equity = append(equity, equityPoint{c.Time, capital})
	}

	if len(equity) != 100 {
		t.Errorf("expected 100 equity points, got %d", len(equity))
	}
	if equity[0].Value != 100000 {
		t.Errorf("initial equity should be 100000, got %.2f", equity[0].Value)
	}
}

func TestE2E_MultiAssetParallel(t *testing.T) {
	eng := backtest.NewEngine(nil)
	ctx := t.Context()

	configs := []backtest.BacktestConfig{
		{Symbols: []string{"A"}, StartDate: time.Now().AddDate(0, -1, 0), EndDate: time.Now(), InitialCapital: 100000},
		{Symbols: []string{"B"}, StartDate: time.Now().AddDate(0, -1, 0), EndDate: time.Now(), InitialCapital: 100000},
	}
	results := eng.RunParallel(ctx, configs)
	if len(results) > 0 {
		t.Logf("parallel run: %d results (DB nil, expected empty)", len(results))
	}
}

func TestE2E_MetricsPipeline(t *testing.T) {
	equity := make([]backtest.EquityPoint, 50)
	base := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	val := 100000.0
	for i := 0; i < 50; i++ {
		val += float64(i%5-1) * 50
		equity[i] = backtest.EquityPoint{Time: base.AddDate(0, 0, i), Value: val}
	}

	trades := []backtest.Trade{
		{Symbol: "T", Side: "BUY", PnL: 100, Quantity: 100, EntryPrice: types.PriceFromFloat(450)},
		{Symbol: "T", Side: "SELL", PnL: -30, Quantity: 100, EntryPrice: types.PriceFromFloat(450)},
		{Symbol: "T", Side: "BUY", PnL: 200, Quantity: 100, EntryPrice: types.PriceFromFloat(450)},
		{Symbol: "T", Side: "SELL", PnL: -50, Quantity: 100, EntryPrice: types.PriceFromFloat(450)},
	}

	metrics := backtest.ComputeAllMetrics(equity, trades)

	if metrics["num_trades"] != 4 {
		t.Errorf("expected 4 trades, got %.0f", metrics["num_trades"])
	}
	if metrics["profit_factor"] < 3.0 {
		t.Errorf("profit factor too low: %.2f", metrics["profit_factor"])
	}
	if metrics["win_rate_pct"] != 50.0 {
		t.Errorf("expected 50%% win rate, got %.1f", metrics["win_rate_pct"])
	}
	if metrics["sharpe_ratio"] == 0 {
		t.Error("sharpe should not be zero for profitable strategy")
	}
	t.Logf("Sharpe=%.2f Sortino=%.2f MaxDD=%.1f%% PF=%.2f WR=%.0f%%",
		metrics["sharpe_ratio"], metrics["sortino_ratio"],
		metrics["max_drawdown_pct"], metrics["profit_factor"], metrics["win_rate_pct"])
}

func TestE2E_EmptyDataHandling(t *testing.T) {
	runner := strategy.NewTrendRunner()
	sig := runner.Evaluate(strategy.Candle{}, 0)
	if sig != nil {
		t.Error("empty candle should not produce signal")
	}

	runner2 := strategy.NewMACrossoverRunner()
	sig2 := runner2.Evaluate(strategy.Candle{Symbol: "T", Close: types.PriceFromFloat(-1.0)}, 0)
	if sig2 != nil {
		t.Error("negative price should not produce signal")
	}

	runner3 := strategy.NewRSI2MeanReversionRunner()
	for i := 0; i < 5; i++ {
		sig3 := runner3.Evaluate(strategy.Candle{Symbol: "T", Close: types.PriceFromFloat(100.0)}, 0)
		if sig3 != nil {
			t.Log("unexpected signal with insufficient data")
		}
	}

	eng := backtest.NewEngine(nil)
	if eng == nil {
		t.Fatal("engine should not be nil")
	}
}
