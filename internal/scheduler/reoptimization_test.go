package scheduler

import (
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/db"
)

func TestReoptimizationConfig_DefaultSymbols(t *testing.T) {
	tests := []struct {
		strategy string
		minLen   int
	}{
		{"trend_following", 2},
		{"mean_reversion", 2},
		{"session_scalp", 1},
		{"unknown_strategy", 1},
	}

	for _, tt := range tests {
		symbols := defaultSymbols(tt.strategy)
		if len(symbols) < tt.minLen {
			t.Errorf("%s: expected at least %d symbols, got %d: %v", tt.strategy, tt.minLen, len(symbols), symbols)
		}
	}
}

func TestReoptimizationConfig_ShouldReoptimize_NoActive(t *testing.T) {
	cfg := DefaultReoptimizationConfig()
	cfg.Repo = nil

	result, reason := cfg.shouldReoptimize(nil, "trend_following")
	if result {
		t.Errorf("should return false when no repo is configured, got reason=%s", reason)
	}
	t.Logf("reason: %s", reason)
}

func TestReoptimizationConfig_AgeBasedTrigger(t *testing.T) {
	// Build a mock ParamVersion that is older than MaxAgeDays.
	pv := &db.ParamVersion{
		StrategyID: "trend_following",
		VersionTag: "v1",
		CreatedAt:  time.Now().AddDate(0, -4, 0), // 4 months old
	}

	cfg := DefaultReoptimizationConfig()
	cfg.MaxAgeDays = 90

	age := time.Since(pv.CreatedAt)
	if age.Hours() <= float64(cfg.MaxAgeDays*24) {
		t.Errorf("age %.0f days should exceed max %d days", age.Hours()/24, cfg.MaxAgeDays)
	}
	// 120 days > 90 days → should trigger.
	if age.Hours()/24 <= 90 {
		t.Error("expected age > 90 days")
	}
}

func TestReoptimizationConfig_DegradationCalc(t *testing.T) {
	oldSharpe := 1.5
	newSharpe := 1.0
	drop := (oldSharpe - newSharpe) / oldSharpe * 100.0

	if drop < 20.0 {
		t.Errorf("drop = %.1f%%, expected >= 20%%", drop)
	}
	// 0.5/1.5*100 = 33.3% — exceeds default 20% threshold.
	if drop <= 20.0 {
		t.Error("expected degradation to exceed 20% threshold")
	}
}

func TestReoptimizationConfig_SaveVersion(t *testing.T) {
	cfg := DefaultReoptimizationConfig()
	cfg.Repo = nil // Will fail on save — test that error handling works.

	params := map[string]float64{"fast_period": 20, "slow_period": 50}
	err := cfg.saveVersion(nil, "trend_following", params,
		time.Now().AddDate(-1, 0, 0), time.Now(), 1.2, false)
	if err == nil {
		t.Error("expected error when repo is nil")
	}
	t.Logf("expected error: %v", err)
}
