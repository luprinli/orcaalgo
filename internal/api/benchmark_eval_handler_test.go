package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/metrics"
)

func TestBenchmarkConfigRoundTrip(t *testing.T) {
	raw := benchmarkConfigJSON("equity_index", "SPY")
	if len(raw) == 0 {
		t.Fatal("benchmark config JSON must be non-empty")
	}
	kind, symbol := benchmarkFromConfig(raw)
	if kind != "equity_index" || symbol != "SPY" {
		t.Fatalf("round-trip mismatch: kind=%s symbol=%s", kind, symbol)
	}
}

func TestBenchmarkConfigNilWhenEmpty(t *testing.T) {
	if benchmarkConfigJSON("", "") != nil {
		t.Fatal("empty kind+symbol must produce nil config")
	}
}

func TestBenchmarkFromConfigDefaults(t *testing.T) {
	kind, symbol := benchmarkFromConfig(nil)
	if kind != "equity_index" || symbol != "SPY" {
		t.Fatalf("defaults wrong: kind=%s symbol=%s", kind, symbol)
	}
	garbage := json.RawMessage(`{"not":"benchmark"}`)
	kind, symbol = benchmarkFromConfig(garbage)
	if kind != "equity_index" || symbol != "SPY" {
		t.Fatalf("non-benchmark config must fall back to defaults")
	}
}

func TestAlignDailyReturns_IntersectsByDate(t *testing.T) {
	day := func(y, m, d int) time.Time { return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC) }

	daily := []metrics.DailyReturn{
		{Date: day(2024, 1, 2), ReturnPct: 1.0}, // 0.01
		{Date: day(2024, 1, 3), ReturnPct: -2.0},
		{Date: day(2024, 1, 4), ReturnPct: 3.0},
		{Date: day(2024, 1, 5), ReturnPct: 4.0}, // no benchmark bar -> dropped
	}
	bench := map[string]float64{
		"2024-01-02": 0.005,
		"2024-01-03": -0.01,
		"2024-01-04": 0.02,
		"2024-01-06": 0.03, // no strategy bar -> ignored
	}

	strat, benchOut := alignDailyReturns(daily, bench)
	if len(strat) != 3 || len(benchOut) != 3 {
		t.Fatalf("expected 3 aligned pairs, got %d/%d", len(strat), len(benchOut))
	}
	if strat[0] != 0.01 {
		t.Fatalf("ReturnPct should be converted to decimal: got %v", strat[0])
	}
	if benchOut[1] != -0.01 {
		t.Fatalf("benchmark alignment wrong: got %v", benchOut[1])
	}
}

func TestAlignDailyReturns_EmptyIntersection(t *testing.T) {
	daily := []metrics.DailyReturn{{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), ReturnPct: 1.0}}
	strat, bench := alignDailyReturns(daily, map[string]float64{})
	if len(strat) != 0 || len(bench) != 0 {
		t.Fatalf("expected empty alignment, got %d/%d", len(strat), len(bench))
	}
}
