package ml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"sync"
	"time"
)

// MetaLabeler predicts the win probability of a trade signal using a trained
// XGBoost model. Inference is performed via subprocess call to Python
// (consistent with the existing orca validate / monte carlo patterns).
//
// The Predictor interface allows future replacement with ONNX runtime
// without changing the engine integration code.
type Predictor interface {
	Predict(features []float32) (float64, error)
	IsHealthy() bool
	ModelVersion() string
	Close() error
}

// MetaLabelerConfig configures the meta-labeling model.
type MetaLabelerConfig struct {
	ModelPath      string  `json:"model_path" yaml:"model_path"`
	WinThreshold   float64 `json:"win_threshold" yaml:"win_threshold"`
	ExtremeLow     float64 `json:"extreme_low" yaml:"extreme_low"`
	ExtremeHigh    float64 `json:"extreme_high" yaml:"extreme_high"`
	PositionScale  float64 `json:"position_scale_cap" yaml:"position_scale_cap"`
	PythonPath     string  `json:"python_path" yaml:"python_path"`
	InferenceScript string `json:"inference_script" yaml:"inference_script"`
	Timeout        time.Duration `json:"timeout" yaml:"timeout"`
}

// DefaultMetaLabelerConfig returns safe production defaults.
func DefaultMetaLabelerConfig() MetaLabelerConfig {
	return MetaLabelerConfig{
		ModelPath:       "models/meta_labeling.json",
		WinThreshold:    0.55,
		ExtremeLow:      0.05,
		ExtremeHigh:     0.95,
		PositionScale:   1.50,
		PythonPath:      "python",
		InferenceScript: "orca/ml/inference.py",
		Timeout:         5 * time.Second,
	}
}

// SubprocessPredictor implements Predictor via Python subprocess.
type SubprocessPredictor struct {
	config   MetaLabelerConfig
	version  string
	mu       sync.RWMutex
	healthy  bool
	closed   bool
	logger   *slog.Logger
}

// NewSubprocessPredictor creates a new subprocess-based predictor.
func NewSubprocessPredictor(config MetaLabelerConfig) (*SubprocessPredictor, error) {
	sp := &SubprocessPredictor{
		config:  config,
		version: "v1",
		healthy: true,
		logger:  slog.Default().With("component", "meta_labeler"),
	}
	sp.logger.Info("meta-labeler initialized", "model", config.ModelPath, "version", sp.version)
	return sp, nil
}

// Predict runs inference via Python subprocess.
// The Python script receives features as JSON on stdin and returns
// {"p_win": 0.72, "version": "v1"} on stdout.
func (sp *SubprocessPredictor) Predict(features []float32) (float64, error) {
	sp.mu.RLock()
	if sp.closed {
		sp.mu.RUnlock()
		return 0, fmt.Errorf("predictor is closed")
	}
	sp.mu.RUnlock()

	if len(features) != FeatureDim {
		return 0, fmt.Errorf("expected %d features, got %d", FeatureDim, len(features))
	}

	ctx, cancel := context.WithTimeout(context.Background(), sp.config.Timeout)
	defer cancel()

	input := map[string]interface{}{
		"model_path": sp.config.ModelPath,
		"features":   features,
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return 0, fmt.Errorf("marshal input: %w", err)
	}

	cmd := exec.CommandContext(ctx, sp.config.PythonPath, sp.config.InferenceScript)
	cmd.Stdin = bytes.NewReader(inputJSON)

	output, err := cmd.CombinedOutput()
	if err != nil {
		sp.logger.Error("inference subprocess failed", "error", err, "output", string(output))
		return 0, fmt.Errorf("inference subprocess: %w (output: %s)", err, string(output))
	}

	var result struct {
		PWin    float64 `json:"p_win"`
		Version string  `json:"version"`
		Error   string  `json:"error,omitempty"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return 0, fmt.Errorf("parse output: %w (output: %s)", err, string(output))
	}

	if result.Error != "" {
		return 0, fmt.Errorf("inference error: %s", result.Error)
	}

	if result.PWin < 0 || result.PWin > 1.0 {
		return 0, fmt.Errorf("invalid p_win: %f", result.PWin)
	}

	return result.PWin, nil
}

// IsHealthy returns true if the predictor is operational.
func (sp *SubprocessPredictor) IsHealthy() bool {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.healthy && !sp.closed
}

// ModelVersion returns the current model version string.
func (sp *SubprocessPredictor) ModelVersion() string {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.version
}

// Close shuts down the predictor.
func (sp *SubprocessPredictor) Close() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.closed = true
	sp.healthy = false
	return nil
}

// Disable marks the predictor as unhealthy (used by kill-switch).
func (sp *SubprocessPredictor) Disable() {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.healthy = false
	sp.logger.Warn("meta-labeler disabled")
}

// Reload attempts to load a new model version.
func (sp *SubprocessPredictor) Reload(modelPath string, version string) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.config.ModelPath = modelPath
	sp.version = version
	sp.healthy = true
	sp.logger.Info("meta-labeler reloaded", "model", modelPath, "version", version)
	return nil
}

// EvaluateSignal applies the meta-labeling gate to a raw signal.
// Returns the result with p_win, acceptance decision, and reason.
func (sp *SubprocessPredictor) EvaluateSignal(features []float32) MetaLabelingResult {
	if !sp.IsHealthy() {
		return MetaLabelingResult{
			PWin:      1.0,
			Threshold: sp.config.WinThreshold,
			Accepted:  true,
			Reason:    "predictor_unhealthy",
		}
	}

	// Check feature validity
	fv := FeatureVector{}
	copy(fv[:], features)
	if !fv.Validate() {
		return MetaLabelingResult{
			PWin:      1.0,
			Threshold: sp.config.WinThreshold,
			Accepted:  true,
			Reason:    "invalid_features",
		}
	}

	pWin, err := sp.Predict(features)
	if err != nil {
		sp.logger.Error("prediction failed, accepting signal by default", "error", err)
		return MetaLabelingResult{
			PWin:      1.0,
			Threshold: sp.config.WinThreshold,
			Accepted:  true,
			Reason:    fmt.Sprintf("prediction_error: %v", err),
		}
	}

	accepted := pWin >= sp.config.WinThreshold
	reason := fmt.Sprintf("p_win=%.3f threshold=%.3f", pWin, sp.config.WinThreshold)

	return MetaLabelingResult{
		PWin:      pWin,
		Threshold: sp.config.WinThreshold,
		Accepted:  accepted,
		Reason:    reason,
	}
}
