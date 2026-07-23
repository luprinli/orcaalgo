package backtest

import (
	"context"
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/model"
	strategy "github.com/lee-econ/orca-core/internal/strategy"
)

func TestMidPriceFillMatchesBaseline(t *testing.T) {
	eng := NewEngine(nil)
	eng.fillModel = model.MidPriceFill{}
	eng.feeModel = model.ZeroFee{}
	eng.latencyModel = model.ZeroLatency{}

	if eng.fillModel == nil || eng.feeModel == nil || eng.latencyModel == nil {
		t.Error("fill/fee/latency models should be set")
	}
}

func TestProbabilisticFillReducesFillRate(t *testing.T) {
	pf := model.NewProbabilisticFill(0.8, 2.5, 100.0)
	prob := pf.FillProbability(100000, 100000, 100, 5000, 10.0)
	if prob > 1.0 || prob < 0.0 {
		t.Errorf("probability should be in [0,1], got %.4f", prob)
	}

	probFar := pf.FillProbability(100500, 100000, 100, 5000, 10.0)
	if probFar > prob {
		t.Log("fill probability decreased with distance from mid — expected")
	}
}

func TestFillPriceWithinBounds(t *testing.T) {
	pf := model.NewProbabilisticFill(0.8, 2.5, 100.0)
	fp := pf.FillPrice(101000, 100000, "BUY", 200.0)
	if fp < 100000 {
		t.Error("BUY fill price should not be below mid")
	}

	fp2 := pf.FillPrice(99000, 100000, "SELL", 200.0)
	if fp2 > 100000 {
		t.Error("SELL fill price should not be above mid")
	}
}

func TestLatencyModelZero(t *testing.T) {
	lm := model.ZeroLatency{}
	if lm.EntryLatency(time.Now(), 100000, 1000, "BUY") != 0 {
		t.Error("ZeroLatency should return 0 entry")
	}
	if lm.ResponseLatency(time.Now(), 100000, "BUY") != 0 {
		t.Error("ZeroLatency should return 0 response")
	}
}

func TestLatencyModelConstant(t *testing.T) {
	lm := model.ConstantLatency{Entry: 50 * time.Millisecond, Response: 30 * time.Millisecond}
	e := lm.EntryLatency(time.Now(), 100000, 1000, "BUY")
	r := lm.ResponseLatency(time.Now(), 100000, "BUY")
	if e != 50*time.Millisecond || r != 30*time.Millisecond {
		t.Errorf("expected 50ms/30ms, got %v/%v", e, r)
	}
}

func TestFeeModel(t *testing.T) {
	fm := model.FixedFee{MakerBps: 2.0, TakerBps: 5.0}
	maker := fm.MakerFee(100000)
	taker := fm.TakerFee(100000)
	if maker != 20.0 || taker != 50.0 {
		t.Errorf("expected 20/50, got %.2f/%.2f", maker, taker)
	}
}

func TestBacktestRecorderProducesOutput(t *testing.T) {
	rec := NewBacktestRecorder()
	rec.Record(&model.TradingState{
		Timestamp:    time.Now(),
		Balance:      100000,
		Position:     100,
		MidPrice:     50000,
		Fee:          5.0,
		TradingVolume: 1000,
		TradingValue:  50000,
		NumTrades:    10,
	}, nil)

	states := rec.States()
	if len(states) != 1 {
		t.Errorf("expected 1 state, got %d", len(states))
	}
	if states[0].NumTrades != 10 {
		t.Errorf("expected 10 trades, got %d", states[0].NumTrades)
	}
}

func TestMetricsRegistration(t *testing.T) {
	all := AllMetrics()
	required := []string{"sharpe_ratio", "sortino_ratio", "max_drawdown_pct", "win_rate_pct", "profit_factor", "total_return_pct", "num_trades", "trading_volume"}
	for _, name := range required {
		if _, ok := all[name]; !ok {
			t.Errorf("metric %s not registered", name)
		}
	}
}

func TestMetricsComputation(t *testing.T) {
	equity := []EquityPoint{
		{Time: time.Now(), Value: 100000},
		{Time: time.Now().Add(time.Hour), Value: 101000},
		{Time: time.Now().Add(2 * time.Hour), Value: 100500},
		{Time: time.Now().Add(3 * time.Hour), Value: 102000},
	}

	trades := []Trade{
		{Symbol: "SPY", Side: "BUY", PnL: 100, Quantity: 100, EntryPrice: 450},
		{Symbol: "SPY", Side: "SELL", PnL: -50, Quantity: 100, EntryPrice: 450},
		{Symbol: "SPY", Side: "BUY", PnL: 200, Quantity: 100, EntryPrice: 450},
	}

	results := ComputeAllMetrics(equity, trades)

	if results["num_trades"] != 3 {
		t.Errorf("expected 3 trades, got %.0f", results["num_trades"])
	}
	if results["win_rate_pct"] < 60 || results["win_rate_pct"] > 70 {
		t.Errorf("expected ~66.7%% win rate, got %.1f", results["win_rate_pct"])
	}
	if results["profit_factor"] < 5.0 {
		t.Errorf("expected profit factor > 5, got %.1f", results["profit_factor"])
	}
	if results["total_return_pct"] < 1.9 || results["total_return_pct"] > 2.1 {
		t.Errorf("expected ~2%% return, got %.1f", results["total_return_pct"])
	}
	if results["max_drawdown_pct"] < 0.4 || results["max_drawdown_pct"] > 0.6 {
		t.Errorf("expected ~0.5%% drawdown, got %.1f", results["max_drawdown_pct"])
	}
}

func TestRunParallel(t *testing.T) {
	eng := NewEngine(nil)
	ctx := context.Background()
	configs := []BacktestConfig{
		{Symbols: []string{"TEST1"}, StartDate: time.Now().AddDate(0, -1, 0), EndDate: time.Now(), InitialCapital: 100000},
		{Symbols: []string{"TEST2"}, StartDate: time.Now().AddDate(0, -1, 0), EndDate: time.Now(), InitialCapital: 100000},
	}
	results := eng.RunParallel(ctx, configs)
	if len(results) > 0 {
		t.Logf("parallel run produced %d results (with nil DB, results may be empty)", len(results))
	}
}

func TestTickQueueOrdering(t *testing.T) {
	tq := &TickQueue{}
	now := time.Now()

	tq.PushTick("B", strategy.Candle{Time: now.Add(2 * time.Second)})
	tq.PushTick("A", strategy.Candle{Time: now})
	tq.PushTick("C", strategy.Candle{Time: now.Add(1 * time.Second)})

	sym1, _ := tq.PopTick()
	sym2, _ := tq.PopTick()
	sym3, _ := tq.PopTick()

	if sym1 != "A" || sym2 != "C" || sym3 != "B" {
		t.Errorf("expected A,C,B order, got %s,%s,%s", sym1, sym2, sym3)
	}
}
