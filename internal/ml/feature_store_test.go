package ml

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/strategy"
	"github.com/lee-econ/orca-core/internal/types"
)

func TestNewFeatureStore(t *testing.T) {
	closes := make([]float64, 50)
	highs := make([]float64, 50)
	lows := make([]float64, 50)
	volumes := make([]float64, 50)
	for i := 0; i < 50; i++ {
		closes[i] = 100.0 + float64(i)*0.1
		highs[i] = closes[i] + 0.5
		lows[i] = closes[i] - 0.5
		volumes[i] = 1000.0
	}
	fs := NewFeatureStore(closes, highs, lows, volumes)
	if fs.count != 50 {
		t.Errorf("expected count=50, got %d", fs.count)
	}
}

// TestFeatureStorePointInTime verifies the feature store is causal (lookahead-safe):
// the feature vector computed at bar t must be identical whether the store was fed
// incrementally up to t, or seeded directly with only bars[0..t]. A divergence would
// indicate the feature computation reads future bars (lookahead leakage).
func TestFeatureStorePointInTime(t *testing.T) {
	n := 200
	rawCloses := make([]float64, n)
	rawHighs := make([]float64, n)
	rawLows := make([]float64, n)
	rawVolumes := make([]float64, n)
	for i := 0; i < n; i++ {
		rawCloses[i] = 100.0 + float64(i)*0.05 + math.Sin(float64(i)*0.1)*2.0
		rawHighs[i] = rawCloses[i] + 0.5
		rawLows[i] = rawCloses[i] - 0.5
		rawVolumes[i] = 1000.0 + float64(i%10)*100.0
	}

	// Round-trip every price through types.Price (fixed-point), so both the
	// incremental and batch stores operate on the exact same values.
	candles := make([]strategy.Candle, n)
	closes := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	volumes := make([]float64, n)
	for i := 0; i < n; i++ {
		candles[i] = strategy.Candle{
			Close:  types.PriceFromFloat(rawCloses[i]),
			High:   types.PriceFromFloat(rawHighs[i]),
			Low:    types.PriceFromFloat(rawLows[i]),
			Volume: rawVolumes[i],
			Time:   time.Unix(int64(i)*3600, 0),
		}
		closes[i] = candles[i].Close.Float64()
		highs[i] = candles[i].High.Float64()
		lows[i] = candles[i].Low.Float64()
		volumes[i] = candles[i].Volume
	}

	// Incremental store fed one bar at a time.
	inc := NewFeatureStore(nil, nil, nil, nil)
	for _, c := range candles {
		inc.Push(c)
	}

	// At every step t in [40, n), the incremental feature must equal the
	// batch-seeded feature from bars[0..t] — proving no future bars leak in.
	for bar := 40; bar < n; bar++ {
		ts := time.Unix(int64(bar)*3600, 0)
		fvInc, err := inc.ComputeAt(bar, ts, [4]float64{0.1, 0.2, 0.3, 0.4}, 0.5, 1, 0.7, 0.0, 0.01)
		if err != nil {
			t.Fatalf("incremental compute at %d: %v", bar, err)
		}
		batch := NewFeatureStore(closes[:bar+1], highs[:bar+1], lows[:bar+1], volumes[:bar+1])
		fvBatch, err := batch.Compute(ts, [4]float64{0.1, 0.2, 0.3, 0.4}, 0.5, 1, 0.7, 0.0, 0.01)
		if err != nil {
			t.Fatalf("batch compute at %d: %v", bar, err)
		}
		for i := 0; i < FeatureDim; i++ {
			if math.Abs(float64(fvInc[i]-fvBatch[i])) > 1e-6 {
				t.Fatalf("point-in-time violation at bar %d feature %d: inc=%v batch=%v", bar, i, fvInc[i], fvBatch[i])
			}
		}
	}
}

func TestFeatureStoreCompute(t *testing.T) {
	// Create 60 bars of synthetic data
	n := 60
	closes := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	volumes := make([]float64, n)

	base := 100.0
	for i := 0; i < n; i++ {
		closes[i] = base + float64(i)*0.05 + math.Sin(float64(i)*0.1)*0.5
		highs[i] = closes[i] + 0.3
		lows[i] = closes[i] - 0.3
		volumes[i] = 1000.0 + float64(i%5)*200.0
	}

	fs := NewFeatureStore(closes, highs, lows, volumes)

	ts := time.Date(2026, 7, 5, 14, 30, 0, 0, time.UTC)
	hmmAlpha := [4]float64{0.7, 0.2, 0.08, 0.02}

	fv, err := fs.Compute(ts, hmmAlpha, 0.85, 1, 0.75, 0.0, 0.001)
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	if !fv.Validate() {
		t.Error("feature vector contains NaN or Inf")
	}

	// Basic sanity checks
	if fv[0] == 0.0 && closes[n-1] != closes[n-2] {
		t.Error("ret1 should be non-zero for price changes")
	}

	if fv[5] < 0 || fv[5] > 100 {
		t.Errorf("rsi14 out of range: %f", fv[5])
	}

	if fv[12] != 0.7 {
		t.Errorf("hmm_state_0 expected 0.7, got %f", fv[12])
	}

	// Time features
	expectedHourSin := float32(math.Sin(2 * math.Pi * 14.5 / 24.0))
	if math.Abs(float64(fv[19]-expectedHourSin)) > 1e-6 {
		t.Errorf("hour_sin expected %f, got %f", expectedHourSin, fv[19])
	}
}

func TestFeatureStoreInsufficientData(t *testing.T) {
	closes := make([]float64, 10)
	fs := NewFeatureStore(closes, closes, closes, closes)
	_, err := fs.Compute(time.Now(), [4]float64{}, 0, 0, 0, 0, 0)
	if err == nil {
		t.Error("expected error for insufficient data")
	}
}

func TestFeatureStorePush(t *testing.T) {
	closes := make([]float64, 50)
	for i := 0; i < 50; i++ {
		closes[i] = 100.0
	}
	fs := NewFeatureStore(closes, closes, closes, closes)
	if fs.count != 50 {
		t.Fatalf("expected count=50, got %d", fs.count)
	}

	// Push new data
	fs.Push(strategy.Candle{
		Close:  types.PriceFromFloat(100.5),
		High:   types.PriceFromFloat(100.8),
		Low:    types.PriceFromFloat(100.2),
		Volume: 1200,
		Symbol: "TEST",
		Time:   time.Now(),
	})

	if fs.count != 51 {
		t.Errorf("expected count=51 after push, got %d", fs.count)
	}
}

func TestComputeEWMAVolatility(t *testing.T) {
	prices := make([]float64, 30)
	for i := 0; i < 30; i++ {
		prices[i] = 100.0 + float64(i)*0.1 + math.Sin(float64(i)*0.2)*0.5
	}
	vol := computeEWMAVolatility(prices, 20)
	if vol <= 0 {
		t.Error("volatility should be positive")
	}
	if math.IsNaN(vol) {
		t.Error("volatility is NaN")
	}
}

func TestComputeRSI(t *testing.T) {
	// Prices trending up
	closes := make([]float64, 20)
	for i := 0; i < 20; i++ {
		closes[i] = 100.0 + float64(i)*0.5
	}
	rsi := computeRSI(closes, 14)
	if rsi < 50 {
		t.Errorf("RSI should be > 50 for uptrend, got %f", rsi)
	}

	// Prices trending down
	for i := 0; i < 20; i++ {
		closes[i] = 100.0 - float64(i)*0.5
	}
	rsi = computeRSI(closes, 14)
	if rsi > 50 {
		t.Errorf("RSI should be < 50 for downtrend, got %f", rsi)
	}
}

func TestFeatureVectorValidate(t *testing.T) {
	fv := FeatureVector{}
	// Zero vector should be valid
	if !fv.Validate() {
		t.Error("zero feature vector should validate")
	}

	// NaN should invalidate
	fv[0] = float32(math.NaN())
	if fv.Validate() {
		t.Error("NaN in feature vector should fail validation")
	}

	// Inf should invalidate
	fv[0] = float32(math.Inf(1))
	if fv.Validate() {
		t.Error("Inf in feature vector should fail validation")
	}
}

func TestFeatureVectorToSlice(t *testing.T) {
	fv := FeatureVector{}
	fv[0] = 1.0
	fv[20] = 2.0
	s := fv.ToSlice()
	if len(s) != FeatureDim {
		t.Errorf("slice length %d != %d", len(s), FeatureDim)
	}
	if s[0] != 1.0 || s[20] != 2.0 {
		t.Error("slice values don't match")
	}
}

func TestDefaultMLConfig(t *testing.T) {
	cfg := DefaultMLConfig()
	if cfg.MetaWinThreshold <= 0 || cfg.MetaWinThreshold >= 1.0 {
		t.Errorf("invalid meta_win_threshold: %f", cfg.MetaWinThreshold)
	}
	if cfg.InferenceTimeoutUs <= 0 {
		t.Errorf("invalid inference_timeout_us: %d", cfg.InferenceTimeoutUs)
	}
}

func TestFeatureStorePersistLogic(t *testing.T) {
	closes := make([]float64, 100)
	highs := make([]float64, 100)
	lows := make([]float64, 100)
	volumes := make([]float64, 100)
	for i := 0; i < 100; i++ {
		v := 100.0 + float64(i)*0.5
		closes[i] = v
		highs[i] = v + 1.0
		lows[i] = v - 1.0
		volumes[i] = 1000.0
	}

	fs := NewFeatureStore(closes, highs, lows, volumes)
	if fs.count != 100 {
		t.Fatalf("expected count 100, got %d", fs.count)
	}

	pgStr := floatSliceToPostgres(closes)
	if len(pgStr) < 10 {
		t.Fatal("postgres array should be non-empty")
	}

	parsed := parseFloatSlice(pgStr)
	if len(parsed) != 100 {
		t.Fatalf("round-trip: expected 100 elements, got %d", len(parsed))
	}
	for i := 0; i < 100; i++ {
		if parsed[i] != closes[i] {
			t.Errorf("round-trip mismatch at index %d: %.8f != %.8f", i, parsed[i], closes[i])
			break
		}
	}
	t.Logf("feature store persist round-trip: 100 elements verified")

	ctx := context.Background()
	err := fs.Persist(ctx, nil, "SPY", "")
	if err != nil {
		t.Fatalf("Persist with nil pool should return nil: %v", err)
	}

	_, _, err = LoadFeatureStore(ctx, nil, "SPY")
	if err == nil {
		t.Fatal("LoadFeatureStore with nil pool should return error")
	}
}
