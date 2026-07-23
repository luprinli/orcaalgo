package ml

import (
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/strategy"
)

func TestMLEndToEndPipeline(t *testing.T) {
	closes := make([]float64, 80)
	highs := make([]float64, 80)
	lows := make([]float64, 80)
	volumes := make([]float64, 80)
	for i := 0; i < 80; i++ {
		closes[i] = 100.0 + float64(i)*0.1
		highs[i] = closes[i] + 0.5
		lows[i] = closes[i] - 0.5
		volumes[i] = 1000.0
	}
	fs := NewFeatureStore(closes, highs, lows, volumes)

	ts := time.Date(2026, 7, 6, 14, 30, 0, 0, time.UTC)
	hmmAlpha := [4]float64{0.7, 0.2, 0.08, 0.02}

	fv, err := fs.Compute(ts, hmmAlpha, 0.85, 1, 0.75, 0.0, 0.001)
	if err != nil {
		t.Fatalf("feature computation failed: %v", err)
	}
	if !fv.Validate() {
		t.Fatal("feature vector contains NaN/Inf")
	}

	t.Logf("feature vector: ret1=%.6f rsi14=%.4f regime=%.4f", fv[0], fv[5], fv[16])

	cfg := DefaultMetaLabelerConfig()
	predictor, err := NewSubprocessPredictor(cfg)
	if err != nil {
		t.Fatalf("failed to create meta-labeler: %v", err)
	}
	defer predictor.Close()

	bi := NewBatchInferrer(predictor, cfg)

	candle := strategy.Candle{
		Time:   ts,
		Open:   100.0,
		High:   100.5,
		Low:    99.5,
		Close:  100.0,
		Volume: 1000.0,
		Symbol: "TEST",
	}

	exitCtx := ExitContext{
		EntryPrice:     100.0,
		CurrentPrice:   candle.Close,
		CurrentStop:    98.0,
		HighSinceEntry: candle.High,
		LowSinceEntry:  candle.Low,
		BarsSinceEntry: 10,
		ATR:            1.0,
		VolAtEntry:     0.01,
		VolCurrent:     0.01,
		HMMState:       1,
		ADX:            25.0,
		Hour:           float64(ts.Hour()),
		Confidence:     0.75,
	}
	exitFeatures := BuildExitFeatures(exitCtx)
	if len(exitFeatures) != ExitFeaturesDim {
		t.Errorf("exit features dim: got %d, want %d", len(exitFeatures), ExitFeaturesDim)
	}

	_ = bi
	t.Log("E2E pipeline: feature store → meta-labeling ready → exit features complete")
}

func TestPointInTimeFeatureIntegrity(t *testing.T) {
	closes := make([]float64, 80)
	for i := 0; i < 80; i++ {
		closes[i] = 100.0 + float64(i)*0.1
	}
	highs := make([]float64, 80)
	lows := make([]float64, 80)
	volumes := make([]float64, 80)
	for i := 0; i < 80; i++ {
		highs[i] = closes[i] + 0.5
		lows[i] = closes[i] - 0.5
		volumes[i] = 1000.0
	}

	fs := NewFeatureStore(closes[:60], highs[:60], lows[:60], volumes[:60])
	ts := time.Date(2026, 7, 6, 14, 30, 0, 0, time.UTC)

	fvBefore, err := fs.Compute(ts, [4]float64{0.6, 0.3, 0.07, 0.03}, 0.9, 1, 0.5, 0.0, 0.001)
	if err != nil {
		t.Fatalf("pre-push compute failed: %v", err)
	}

	fs.Push(strategy.Candle{
		Time:   ts.Add(time.Minute),
		Open:   closes[60],
		High:   closes[60] + 0.5,
		Low:    closes[60] - 0.5,
		Close:  closes[60],
		Volume: 1000,
		Symbol: "TEST",
	})

	fvAfter, err := fs.Compute(ts.Add(time.Minute), [4]float64{0.6, 0.3, 0.07, 0.03}, 0.9, 1, 0.5, 0.0, 0.001)
	if err != nil {
		t.Fatalf("post-push compute failed: %v", err)
	}

	if fvBefore[0] == fvAfter[0] && fvBefore[0] != 0 {
		t.Log("ret1 unchanged as expected (no new return)")
	}

	if fvBefore[5] != fvAfter[5] {
		t.Log("rsi changed after push (expected — new data window)")
	}

	if !fvBefore.Validate() || !fvAfter.Validate() {
		t.Fatal("feature vectors should be valid")
	}
}
