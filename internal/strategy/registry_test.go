package strategy_test

import (
	"testing"

	strategy "github.com/lee-econ/orca-core/internal/strategy"
	"github.com/lee-econ/orca-core/internal/types"
)

func TestGlobalRegistry_HasAllStrategies(t *testing.T) {
	reg := strategy.GlobalRegistry()
	expected := []string{
		"opening_range_breakout", "breakout",
		"grid", "grid_trading",
		"trend_following", "trend",
		"session_scalp", "scalp",
	}
	for _, name := range expected {
		s := reg.Get(name)
		if s == nil {
			t.Errorf("GlobalRegistry missing: %s", name)
		}
	}
}

func TestRegistry_GetUnknownReturnsNil(t *testing.T) {
	reg := strategy.GlobalRegistry()
	if reg.Get("nonexistent_strategy") != nil {
		t.Error("Expected nil for unknown strategy")
	}
}

func TestOrbRunner_ImplementsStrategy(t *testing.T) {
	var s strategy.Strategy = strategy.NewOrbRunner()
	if s.Name() != "opening_range_breakout" {
		t.Errorf("OrbRunner.Name = %s", s.Name())
	}
	if s.Type() != "breakout" {
		t.Errorf("OrbRunner.Type = %s", s.Type())
	}
}

func TestTrendRunner_ImplementsStrategy(t *testing.T) {
	var s strategy.Strategy = strategy.NewTrendRunner()
	if s.Name() != "trend_following" {
		t.Errorf("TrendRunner.Name = %s", s.Name())
	}
}

func TestGridRunner_ImplementsStrategy(t *testing.T) {
	var s strategy.Strategy = strategy.NewGridRunner()
	if s.Name() != "grid_trading" {
		t.Errorf("GridRunner.Name = %s", s.Name())
	}
}

func TestSessionScalpRunner_ImplementsStrategy(t *testing.T) {
	var s strategy.Strategy = strategy.NewSessionScalpRunner()
	if s.Name() != "session_scalp" {
		t.Errorf("SessionScalpRunner.Name = %s", s.Name())
	}
}

func TestMeanReversionRunner_ImplementsStrategy(t *testing.T) {
	var s strategy.Strategy = strategy.NewMeanReversionRunner(14, 2.0, 0.5, 60)
	if s.Name() != "mean_reversion" {
		t.Errorf("MeanReversionRunner.Name = %s", s.Name())
	}
	if s.Type() != "mr" {
		t.Errorf("MeanReversionRunner.Type = %s", s.Type())
	}
}

func TestMeanReversionRunner_Reset(t *testing.T) {
	s := strategy.NewMeanReversionRunner(14, 2.0, 0.5, 60)
	candle := strategy.Candle{Close: types.PriceFromFloat(100.0), Symbol: "TEST"}
	s.Evaluate(candle, 0)
	s.Reset()
	params := s.Params()
	if params["lookback"] != 14 {
		t.Errorf("Reset should preserve params: lookback = %v", params["lookback"])
	}
}

func TestAllStrategies_HaveParamDefs(t *testing.T) {
	reg := strategy.GlobalRegistry()
	names := []string{"opening_range_breakout", "grid", "trend_following", "session_scalp"}
	for _, name := range names {
		s := reg.Get(name)
		if s == nil {
			t.Errorf("strategy not found: %s", name)
			continue
		}
		defs := s.ParamDefs()
		if len(defs) == 0 {
			t.Errorf("%s has no ParamDefs", name)
		}
	}
	mr := strategy.NewMeanReversionRunner(14, 2.0, 0.5, 60)
	if len(mr.ParamDefs()) == 0 {
		t.Error("MeanReversionRunner has no ParamDefs")
	}
}

func TestAllStrategies_ParamsReturnsNonEmpty(t *testing.T) {
	reg := strategy.GlobalRegistry()
	names := []string{"opening_range_breakout", "grid", "trend_following", "session_scalp"}
	for _, name := range names {
		s := reg.Get(name)
		if s == nil {
			continue
		}
		p := s.Params()
		if len(p) == 0 {
			t.Errorf("%s has no params", name)
		}
	}
	mr := strategy.NewMeanReversionRunner(14, 2.0, 0.5, 60)
	if len(mr.Params()) == 0 {
		t.Error("MeanReversionRunner has no params")
	}
}
