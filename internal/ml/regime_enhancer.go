package ml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os/exec"
	"sync"
	"time"
)

// RegimeState represents a fine-grained market regime (6 states).
type RegimeState int8

const (
	RegimeCalm         RegimeState = 0
	RegimeAccumulation RegimeState = 1
	RegimeTrending     RegimeState = 2
	RegimeDistribution RegimeState = 3
	RegimeHighVol      RegimeState = 4
	RegimeCrisis       RegimeState = 5
)

var regimeLabels = [6]string{
	"calm", "accumulation", "trending", "distribution", "high_vol", "crisis",
}

func (r RegimeState) String() string {
	if int(r) < len(regimeLabels) {
		return regimeLabels[r]
	}
	return "unknown"
}

// RegimeScoreWeights map each regime state to a Kelly multiplier weight.
// calm=1.0 (trade more), accumulation=0.9, trending=0.8, distribution=0.7,
// high_vol=0.4, crisis=0.0 (no trading).
var RegimeScoreWeights = [6]float64{1.0, 0.9, 0.8, 0.7, 0.4, 0.0}

// RegimeEnhancer takes HMM alpha vector + market features and produces a
// continuous regime score via a trained XGBoost classifier (subprocess).
type RegimeEnhancer struct {
	config  RegimeEnhancerConfig
	version string
	mu      sync.RWMutex
	healthy bool
	closed  bool
	logger  *slog.Logger
}

// RegimeEnhancerConfig configures the regime enhancement model.
type RegimeEnhancerConfig struct {
	ModelPath       string        `json:"model_path" yaml:"model_path"`
	PythonPath      string        `json:"python_path" yaml:"python_path"`
	InferenceScript string        `json:"inference_script" yaml:"inference_script"`
	Timeout         time.Duration `json:"timeout" yaml:"timeout"`
}

// DefaultRegimeEnhancerConfig returns safe defaults.
func DefaultRegimeEnhancerConfig() RegimeEnhancerConfig {
	return RegimeEnhancerConfig{
		ModelPath:       "models/regime_classifier.json",
		PythonPath:      "python",
		InferenceScript: "orca/ml/regime_inference.py",
		Timeout:         5 * time.Second,
	}
}

// NewRegimeEnhancer creates a new regime enhancer.
func NewRegimeEnhancer(cfg RegimeEnhancerConfig) *RegimeEnhancer {
	return &RegimeEnhancer{
		config:  cfg,
		version: "v1",
		healthy: true,
		logger:  slog.Default().With("component", "regime_enhancer"),
	}
}

// IsHealthy returns true if the enhancer is operational.
func (re *RegimeEnhancer) IsHealthy() bool {
	re.mu.RLock()
	defer re.mu.RUnlock()
	return re.healthy && !re.closed
}

// Disable marks the enhancer as unhealthy (kill-switch).
func (re *RegimeEnhancer) Disable() {
	re.mu.Lock()
	defer re.mu.Unlock()
	re.healthy = false
	re.logger.Warn("regime enhancer disabled")
}

// Predict evaluates the current regime and returns a continuous score [0, 1].
//
// Input: HMM alpha vector (4-dim), VIX, sentiment, CVD trend, vol structure, hour.
// Output: continuous regime score where 1.0 = ideal trading, 0.0 = stop trading.
func (re *RegimeEnhancer) Predict(
	hmmAlpha [4]float64,
	vix float64,
	sentiment int,
	cvdTrend float64,
	volStructure float64,
	hour float64,
) (float64, error) {
	re.mu.RLock()
	if re.closed {
		re.mu.RUnlock()
		return 0.5, fmt.Errorf("regime enhancer is closed")
	}
	re.mu.RUnlock()

	// Feature vector: 14-dims matching build_regime_features in Python
	features := [14]float64{
		hmmAlpha[0], hmmAlpha[1], hmmAlpha[2], hmmAlpha[3],
		vix / 100.0,
		float64(sentiment) / 100.0,
		cvdTrend,
		volStructure,
		math.Sin(2 * math.Pi * hour / 24.0),
		math.Cos(2 * math.Pi * hour / 24.0),
		math.Sin(2 * math.Pi * hour / 24.0 * 2),
		math.Cos(2 * math.Pi * hour / 24.0 * 2),
		boolToFloat(vix > 25),
		boolToFloat(vix > 35),
	}

	ctx, cancel := context.WithTimeout(context.Background(), re.config.Timeout)
	defer cancel()

	input := map[string]interface{}{
		"model_path": re.config.ModelPath,
		"features":   features,
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return 0.5, fmt.Errorf("marshal regime input: %w", err)
	}

	cmd := exec.CommandContext(ctx, re.config.PythonPath, re.config.InferenceScript)
	cmd.Stdin = bytes.NewReader(inputJSON)
	output, err := cmd.CombinedOutput()
	if err != nil {
		re.logger.Error("regime inference failed", "error", err)
		return 0.5, fmt.Errorf("regime subprocess: %w", err)
	}

	var result struct {
		Score   float64 `json:"regime_score"`
		State   int     `json:"regime_state"`
		Probs   []float64 `json:"probs"`
		Version string  `json:"version"`
		Error   string  `json:"error,omitempty"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return 0.5, fmt.Errorf("parse regime output: %w", err)
	}
	if result.Error != "" {
		return 0.5, fmt.Errorf("regime inference error: %s", result.Error)
	}
	if result.Score < 0 || result.Score > 1.0 {
		return 0.5, fmt.Errorf("invalid regime score: %f", result.Score)
	}

	return result.Score, nil
}

// Evaluate returns regime score and Kelly multiplier in one call.
func (re *RegimeEnhancer) Evaluate(
	hmmAlpha [4]float64,
	vix float64,
	sentiment int,
	cvdTrend float64,
	volStructure float64,
	hour float64,
) (regimeScore float64, kellyMult float64) {
	if !re.IsHealthy() {
		// Fallback: use HMM state directly as old step-function
		return 1.0, 1.0
	}

	score, err := re.Predict(hmmAlpha, vix, sentiment, cvdTrend, volStructure, hour)
	if err != nil {
		return 1.0, 1.0
	}

	// Continuous Kelly multiplier from score
	mult := scoreToKellyMultiplier(score)
	return score, mult
}

// FallbackScore computes a regime score from raw HMM state only (no ML).
// This is used when the enhancer is disabled or unavailable.
func FallbackScore(hmmState int8) float64 {
	switch hmmState {
	case 0:
		return 1.0 // calm → 1.0
	case 1:
		return 0.8 // trending → 0.8
	case 2:
		return 0.5 // high_vol → 0.5
	case 3:
		return 0.0 // crisis → 0.0
	default:
		return 0.8
	}
}

// scoreToKellyMultiplier maps regime score to Kelly multiplier.
func scoreToKellyMultiplier(score float64) float64 {
	if score < 0 {
		return 0
	}
	return math.Min(1.5*score, 1.5)
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
