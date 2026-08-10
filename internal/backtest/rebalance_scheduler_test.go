package backtest

import (
	"testing"

	"github.com/lee-econ/orca-core/internal/risk"
)

func TestRebalanceScheduler_IsFullRebalanceDue_Cadence(t *testing.T) {
	s := NewRebalanceScheduler(5, risk.NewRegimeActivationMatrix())
	for i := 1; i <= 4; i++ {
		if s.IsFullRebalanceDue() {
			t.Errorf("call %d should not trigger rebalance", i)
		}
	}
	if !s.IsFullRebalanceDue() {
		t.Error("5th call should trigger rebalance")
	}
	if s.BarCount != 0 {
		t.Errorf("BarCount should reset after rebalance, got %d", s.BarCount)
	}
}

func TestRebalanceScheduler_ComputeWeights_EqualKelly(t *testing.T) {
	s := NewRebalanceScheduler(20, risk.NewRegimeActivationMatrix())
	active := []EligibilityResult{
		{StrategyID: "a", Kelly: 0.25, Sharpe: 1.0},
		{StrategyID: "b", Kelly: 0.25, Sharpe: 1.0},
		{StrategyID: "c", Kelly: 0.25, Sharpe: 1.0},
	}
	weights := s.ComputeWeights(active)
	if len(weights) != 3 {
		t.Fatalf("expected 3 weights, got %d", len(weights))
	}
	for _, w := range weights {
		if w < 0.33 || w > 0.34 {
			t.Errorf("expected weight ~0.333, got %f", w)
		}
	}
}

func TestRebalanceScheduler_ComputeWeights_SingleActive(t *testing.T) {
	s := NewRebalanceScheduler(20, risk.NewRegimeActivationMatrix())
	active := []EligibilityResult{
		{StrategyID: "a", Kelly: 0.25, Sharpe: 1.5},
	}
	weights := s.ComputeWeights(active)
	if v := weights["a"]; v != 1.0 {
		t.Errorf("single active strategy should have weight 1.0, got %f", v)
	}
	if v := s.ActiveWeight(weights, "absent"); v != 0 {
		t.Errorf("absent strategy should have weight 0, got %f", v)
	}
}

func TestRebalanceScheduler_ComputeWeights_Empty(t *testing.T) {
	s := NewRebalanceScheduler(20, risk.NewRegimeActivationMatrix())
	weights := s.ComputeWeights(nil)
	if weights != nil {
		t.Errorf("expected nil from empty active list, got %v", weights)
	}
	weights = s.ComputeWeights([]EligibilityResult{})
	if weights != nil {
		t.Errorf("expected nil from empty active slice, got %v", weights)
	}
}

func TestRebalanceScheduler_ComputeWeights_ZeroSharpe(t *testing.T) {
	s := NewRebalanceScheduler(20, risk.NewRegimeActivationMatrix())
	active := []EligibilityResult{
		{StrategyID: "a", Kelly: 0.25, Sharpe: 0},
		{StrategyID: "b", Kelly: 0.25, Sharpe: 0},
	}
	weights := s.ComputeWeights(active)
	if len(weights) != 2 {
		t.Fatalf("expected 2 weights, got %d", len(weights))
	}
	for _, w := range weights {
		if w < 0.49 || w > 0.51 {
			t.Errorf("expected equal fallback weight ~0.5, got %f", w)
		}
	}
}

func TestRebalanceScheduler_EvaluateEligibility_RegimeBlocked(t *testing.T) {
	matrix := risk.NewRegimeActivationMatrix()
	s := NewRebalanceScheduler(20, matrix)
	result := s.EvaluateEligibility("grid_trading", 1, true, 0.25)
	if result.Eligible {
		t.Error("grid_trading in Trending should be blocked")
	}
	if result.Reason != "regime_blocked" {
		t.Errorf("wrong reason: %s", result.Reason)
	}
}

func TestRebalanceScheduler_EvaluateEligibility_Active(t *testing.T) {
	matrix := risk.NewRegimeActivationMatrix()
	s := NewRebalanceScheduler(20, matrix)
	result := s.EvaluateEligibility("grid_trading", 0, true, 0.25)
	if !result.Eligible {
		t.Error("grid_trading in Calm should be active")
	}
	if result.Reason != "active" {
		t.Errorf("wrong reason: %s", result.Reason)
	}
}

func TestRebalanceScheduler_CadenceForTimeframe(t *testing.T) {
	s := NewRebalanceScheduler(20, risk.NewRegimeActivationMatrix())
	tests := map[string]int{"1d": 20, "4h": 40, "1h": 80, "30m": 120, "15m": 160, "unknown": 40}
	for tf, expected := range tests {
		if got := s.CadenceForTimeframe(tf); got != expected {
			t.Errorf("CadenceForTimeframe(%s): got %d, want %d", tf, got, expected)
		}
	}
}
