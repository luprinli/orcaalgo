package ml

import (
	"math"
	"testing"
)

func TestRegimeScoreToKellyMultiplier(t *testing.T) {
	tests := []struct {
		score    float64
		expected float64
	}{
		{1.0, 1.5},
		{0.0, 0.0},
		{0.5, 0.75},
		{0.8, 1.2},
		{2.0, 1.5},
		{-1.0, 0.0},
	}
	for _, tt := range tests {
		got := scoreToKellyMultiplier(tt.score)
		if math.Abs(got-tt.expected) > 1e-9 {
			t.Errorf("scoreToKellyMultiplier(%f) = %f, want %f", tt.score, got, tt.expected)
		}
	}
}

func TestFallbackScore(t *testing.T) {
	if s := FallbackScore(0); s != 1.0 {
		t.Errorf("calm fallback: got %f, want 1.0", s)
	}
	if s := FallbackScore(1); s != 0.8 {
		t.Errorf("trending fallback: got %f, want 0.8", s)
	}
	if s := FallbackScore(2); s != 0.5 {
		t.Errorf("high_vol fallback: got %f, want 0.5", s)
	}
	if s := FallbackScore(3); s != 0.0 {
		t.Errorf("crisis fallback: got %f, want 0.0", s)
	}
}

func TestRegimeEnhancerDefaults(t *testing.T) {
	cfg := DefaultRegimeEnhancerConfig()
	re := NewRegimeEnhancer(cfg)
	if !re.IsHealthy() {
		t.Error("new regime enhancer should be healthy")
	}

	re.Disable()
	if re.IsHealthy() {
		t.Error("disabled regime enhancer should not be healthy")
	}

	// Fallback when disabled
	score, mult := re.Evaluate([4]float64{}, 20, 50, 0, 0.5, 12)
	if score != 1.0 || mult != 1.0 {
		t.Errorf("disabled enhancer should return fallback: got score=%f mult=%f", score, mult)
	}
}

func TestRegimeStateLabels(t *testing.T) {
	labels := map[RegimeState]string{
		RegimeCalm:         "calm",
		RegimeAccumulation: "accumulation",
		RegimeTrending:     "trending",
		RegimeDistribution: "distribution",
		RegimeHighVol:      "high_vol",
		RegimeCrisis:       "crisis",
	}
	for state, expected := range labels {
		if state.String() != expected {
			t.Errorf("state %d: got %s, want %s", state, state.String(), expected)
		}
	}
}

func TestRegimeScoreWeights(t *testing.T) {
	if RegimeScoreWeights[0] != 1.0 {
		t.Error("calm weight should be 1.0")
	}
	if RegimeScoreWeights[5] != 0.0 {
		t.Error("crisis weight should be 0.0")
	}
	for i := 0; i < 6; i++ {
		if i > 0 && RegimeScoreWeights[i] >= RegimeScoreWeights[i-1] {
			t.Errorf("weights should be decreasing: w[%d]=%f >= w[%d]=%f", i, RegimeScoreWeights[i], i-1, RegimeScoreWeights[i-1])
		}
	}
}

func TestBoolToFloat(t *testing.T) {
	if boolToFloat(true) != 1.0 {
		t.Error("boolToFloat(true) should be 1.0")
	}
	if boolToFloat(false) != 0.0 {
		t.Error("boolToFloat(false) should be 0.0")
	}
}
