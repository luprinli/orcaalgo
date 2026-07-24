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

	"github.com/lee-econ/orca-core/internal/types"
)

// ExitOrchestrator manages ML-based dynamic exit for all strategy runners.
// It computes exit urgency from trade context features and outputs a
// dynamic stop multiplier that can tighten or widen stops in real-time.
type ExitOrchestrator struct {
	config  ExitOrchestratorConfig
	mu      sync.RWMutex
	healthy bool
	closed  bool
	logger  *slog.Logger
}

// ExitOrchestratorConfig configures the exit optimization model.
type ExitOrchestratorConfig struct {
	ModelPath        string        `json:"model_path" yaml:"model_path"`
	PythonPath       string        `json:"python_path" yaml:"python_path"`
	InferenceScript  string        `json:"inference_script" yaml:"inference_script"`
	BaseMultiplier   float64       `json:"base_multiplier" yaml:"base_multiplier"`
	AdjustmentFactor float64       `json:"adjustment_factor" yaml:"adjustment_factor"`
	MinMultiplier    float64       `json:"min_multiplier" yaml:"min_multiplier"`
	MaxMultiplier    float64       `json:"max_multiplier" yaml:"max_multiplier"`
	Timeout          time.Duration `json:"timeout" yaml:"timeout"`
}

// NewExitOrchestrator creates a new exit orchestrator.
func NewExitOrchestrator(cfg ExitOrchestratorConfig) *ExitOrchestrator {
	return &ExitOrchestrator{
		config:  cfg,
		healthy: true,
		logger:  slog.Default().With("component", "exit_orchestrator"),
	}
}

// IsHealthy returns true if the orchestrator is operational.
func (eo *ExitOrchestrator) IsHealthy() bool {
	eo.mu.RLock()
	defer eo.mu.RUnlock()
	return eo.healthy && !eo.closed
}

// Disable marks the orchestrator as unhealthy.
func (eo *ExitOrchestrator) Disable() {
	eo.mu.Lock()
	defer eo.mu.Unlock()
	eo.healthy = false
	eo.logger.Warn("exit orchestrator disabled")
}

// Evaluate computes the dynamic stop multiplier for a trade.
//
// Input: trade context (entry, current, stop, duration, vol, regime, etc.)
// Output: dynamic stop multiplier clamped to [min, max].
// If healthy=false or inference fails, returns base multiplier (no-op).
func (eo *ExitOrchestrator) Evaluate(ctx ExitContext) (urgency float64, multiplier float64) {
	if !eo.IsHealthy() {
		return 0.0, eo.config.BaseMultiplier
	}

	features := BuildExitFeatures(ctx)
	urgency, err := eo.predict(features)
	if err != nil {
		eo.logger.Debug("exit prediction failed, using base multiplier", "error", err)
		return 0.0, eo.config.BaseMultiplier
	}

	urgency = clamp(urgency, 0.0, 1.0)
	multiplier = UrgencyToStopMultiplier(urgency, eo.config.BaseMultiplier, eo.config.AdjustmentFactor)
	multiplier = clamp(multiplier, eo.config.MinMultiplier, eo.config.MaxMultiplier)

	return urgency, multiplier
}

// ComputeNewStop calculates a new stop price based on ML urgency.
// For long positions: newStop = price - multiplier * atr
// The multiplier is the ML-adjusted dynamic stop multiplier.
func (eo *ExitOrchestrator) ComputeNewStop(
	side string,
	entryPrice types.Price,
	currentPrice types.Price,
	atr float64,
	ctx ExitContext,
) float64 {
	_, mult := eo.Evaluate(ctx)
	entryP := entryPrice.Float64()
	currP := currentPrice.Float64()
	if side == "BUY" {
		return math.Max(entryP*0.8, currP-mult*atr)
	}
	return math.Min(entryP*1.2, currP+mult*atr)
}

// predict calls the Python inference subprocess.
func (eo *ExitOrchestrator) predict(features [ExitFeaturesDim]float64) (float64, error) {
	eo.mu.RLock()
	if eo.closed {
		eo.mu.RUnlock()
		return 0.5, fmt.Errorf("exit orchestrator closed")
	}
	eo.mu.RUnlock()

	input := map[string]interface{}{
		"model_path": eo.config.ModelPath,
		"features":   features,
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return 0.5, fmt.Errorf("marshal input: %w", err)
	}

	pCtx, cancel := context.WithTimeout(context.Background(), eo.config.Timeout)
	defer cancel()

	cmd := exec.CommandContext(pCtx, eo.config.PythonPath, eo.config.InferenceScript)
	cmd.Stdin = bytes.NewReader(inputJSON)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0.5, fmt.Errorf("exit subprocess: %w", err)
	}

	var result ExitPrediction
	if err := json.Unmarshal(output, &result); err != nil {
		return 0.5, fmt.Errorf("parse exit output: %w", err)
	}
	if result.Error != "" {
		return 0.5, fmt.Errorf("exit inference: %s", result.Error)
	}

	return result.Urgency, nil
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
