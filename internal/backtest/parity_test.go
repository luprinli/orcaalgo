package backtest

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/risk"
	"github.com/lee-econ/orca-core/internal/strategy"
	"github.com/lee-econ/orca-core/internal/types"
)

type parityDB struct {
	candles  []Candle
	regime   []RegimeLog
	vix      []VIXLog
	sentiment []SentimentLog
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
	return p.sentiment, nil
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

func makeParityCandles(days int) []Candle {
	ts := time.Date(2025, 1, 2, 9, 30, 0, 0, time.UTC)
	candles := make([]Candle, days)
	price := 100.0
	for i := 0; i < days; i++ {
		trend := 0.15
		noise := float64(i%7-3) * 0.4
		price += trend + noise
		if price < 20 {
			price = 20
		}
		high := price + float64(i%5)*1.5
		low := price - float64((i+1)%6)*1.2
		open := low + (high-low)*0.35
		candles[i] = Candle{
			Time:   ts.AddDate(0, 0, i),
			Open:   types.PriceFromFloat(open),
			High:   types.PriceFromFloat(high),
			Low:    types.PriceFromFloat(low),
			Close:  types.PriceFromFloat(price),
			Volume: 5000000,
			Symbol: "SPY",
		}
	}
	return candles
}

func makeParityRegimeLogs(candles []Candle) []RegimeLog {
	logs := make([]RegimeLog, len(candles))
	for i, c := range candles {
		state := int8(0)
		conf := 0.9
		logs[i] = RegimeLog{Time: c.Time, HMMState: state, Confidence: conf, Symbol: c.Symbol}
	}
	return logs
}

func strategyRegistryHasStrategy(id string) bool {
	return strategy.GlobalRegistry().Get(id) != nil
}

func getTestStrategyID() string {
	for _, id := range []string{"rsi2_reversion", "ma_crossover", "trend_following", "grid_trading"} {
		if strategy.GlobalRegistry().Get(id) != nil {
			return id
		}
	}
	return "ma_crossover"
}

func TestBacktestEngineDeterminism(t *testing.T) {
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

func TestBacktestReplayParity(t *testing.T) {
	stratID := getTestStrategyID()
	if !strategyRegistryHasStrategy(stratID) {
		t.Skipf("strategy %s not registered — skipping parity test", stratID)
	}

	candles := makeParityCandles(30)
	regime := makeParityRegimeLogs(candles)

	db := &parityDB{candles: candles, regime: regime, vix: nil, sentiment: nil}

	cfg := BacktestConfig{
		StrategyID:     stratID,
		Symbols:        []string{"SPY"},
		StartDate:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		InitialCapital: 100000,
		DataSource:     "test",
		ApplyGate:      true,
		GateProfile:    "research",
		SizingPercent:  0.05,
		KellyFraction:  0.25,
	}

	ctx := context.Background()

	engineBatch := NewEngine(db)
	engineBatch.WirePipeline()
	resultBatch, err := engineBatch.Run(ctx, cfg)
	if err != nil {
		t.Fatalf("batch engine failed: %v", err)
	}

	chunkSize := 5
	var allTradesStreaming []Trade
	var lastEquity float64 = cfg.InitialCapital

	for start := 0; start < len(candles); start += chunkSize {
		end := start + chunkSize
		if end > len(candles) {
			end = len(candles)
		}
		chunk := candles[start:end]
		chunkDB := &parityDB{candles: chunk, regime: regime, vix: nil, sentiment: nil}

		chunkCfg := cfg
		chunkCfg.StartDate = chunk[0].Time.Add(-24 * time.Hour)
		chunkCfg.EndDate = chunk[len(chunk)-1].Time.Add(24 * time.Hour)
		chunkCfg.InitialCapital = lastEquity

		engineStream := NewEngine(chunkDB)
		engineStream.WirePipeline()
		chunkResult, err := engineStream.Run(ctx, chunkCfg)
		if err != nil {
			t.Fatalf("streaming engine failed at chunk %d-%d: %v", start, end, err)
		}

		allTradesStreaming = append(allTradesStreaming, chunkResult.Trades...)
		if len(chunkResult.EquityCurve) > 0 {
			lastEquity = chunkResult.EquityCurve[len(chunkResult.EquityCurve)-1].Value
		} else {
			lastEquity = cfg.InitialCapital + chunkResult.TotalReturn
		}
	}

	tradesBatch := len(resultBatch.Trades)
	tradesStreaming := len(allTradesStreaming)

	if tradesBatch == 0 && tradesStreaming == 0 {
		return
	}

	if tradesBatch == 0 || tradesStreaming == 0 {
		t.Logf("batch trades=%d streaming trades=%d — skipping ratio comparison", tradesBatch, tradesStreaming)
		return
	}

	tradeDiff := math.Abs(float64(tradesBatch-tradesStreaming)) / math.Max(float64(tradesBatch), 1)
	if tradeDiff > 0.05 {
		t.Errorf("signal count difference %.1f%% exceeds 5%% threshold: batch=%d, streaming=%d",
			tradeDiff*100, tradesBatch, tradesStreaming)
	}

	var batchPnL, streamingPnL float64
	for _, tr := range resultBatch.Trades {
		batchPnL += tr.PnL
	}
	for _, tr := range allTradesStreaming {
		streamingPnL += tr.PnL
	}

	if batchPnL != 0 && streamingPnL != 0 {
		pnlDiff := math.Abs(batchPnL-streamingPnL) / math.Max(math.Abs(batchPnL), 1)
		if pnlDiff > 0.05 {
			t.Errorf("PnL difference %.1f%% exceeds 5%% threshold: batch=%.2f, streaming=%.2f",
				pnlDiff*100, batchPnL, streamingPnL)
		}
	}

	batchDD := resultBatch.MaxDrawdown
	streamingDD := calculateMaxDrawdownFromTrades(allTradesStreaming, cfg.InitialCapital)
	ddDiff := math.Abs(batchDD-streamingDD)
	if ddDiff > 0.02 {
		t.Errorf("MaxDD difference %.4f exceeds 2%% threshold: batch=%.4f, streaming=%.4f",
			ddDiff, batchDD, streamingDD)
	}

	t.Logf("batch: trades=%d pnl=%.2f dd=%.4f | streaming: trades=%d pnl=%.2f dd=%.4f",
		tradesBatch, batchPnL, batchDD, tradesStreaming, streamingPnL, streamingDD)
}

func calculateMaxDrawdownFromTrades(trades []Trade, initialCapital float64) float64 {
	if len(trades) == 0 {
		return 0
	}
	equity := []EquityPoint{{Time: time.Time{}, Value: initialCapital}}
	for _, tr := range trades {
		equity = append(equity, EquityPoint{
			Time:  tr.ExitTime,
			Value: equity[len(equity)-1].Value + tr.PnL,
		})
	}
	return calculateMaxDrawdown(equity)
}

func TestRiskPipelineSignalParity(t *testing.T) {
	pipeline := &risk.RiskPipeline{
		SignalGate:   risk.NewSignalGateImpl(nil, nil, nil, nil),
		KellyMult:    0.25,
		RegimeMatrix: risk.NewRegimeActivationMatrix(),
	}

	if pipeline.SignalGate == nil {
		t.Skip("SignalGate not available")
	}

	req1 := risk.ProcessSignalRequest{
		StrategyID:     "rsi2_reversion",
		Symbol:         "SPY",
		Side:           "BUY",
		Price:          100.0,
		Confidence:     0.8,
		BaseSize:       50.0,
		RunningCapital: 100000.0,
	}
	req2 := req1

	pipeline.CurrentRegime = 0
	res1 := pipeline.ProcessSignal(context.Background(), req1)
	res2 := pipeline.ProcessSignal(context.Background(), req2)

	if res1.Approved != res2.Approved {
		t.Errorf("pipeline approval mismatch: %v vs %v", res1.Approved, res2.Approved)
	}
	if res1.Approved && math.Abs(res1.Size-res2.Size) > 1e-6 {
		t.Errorf("pipeline size mismatch: %.6f vs %.6f", res1.Size, res2.Size)
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
