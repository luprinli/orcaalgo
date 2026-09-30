package api

import (
	"testing"

	"github.com/lee-econ/orca-core/internal/backtest"
)

func TestProgressStoreOptimizationDoesNotConsumeCombo(t *testing.T) {
	ps := NewProgressStore()
	ps.Create("batch-1", 2, []ComboProgress{
		{StrategyID: "trend_following", Symbol: "SPX500", Timeframe: "1d", Status: "pending"},
		{StrategyID: "mean_reversion", Symbol: "SPX500", Timeframe: "1d", Status: "pending"},
	})

	ps.UpdateCombo("batch-1", 1, "optimized", "", &backtest.ComboResult{StrategyID: "mean_reversion"})
	mp := ps.Get("batch-1")
	if mp.Combos[1].Status != "pending" {
		t.Fatalf("optimization update consumed combo status: %q", mp.Combos[1].Status)
	}

	summary := ps.GetSummary("batch-1")
	if summary["phase"] != "optimizing" {
		t.Fatalf("expected optimizing phase, got %v", summary["phase"])
	}
	current, ok := summary["current"].(*MatrixCurrent)
	if !ok || current.Strategy != "mean_reversion" {
		t.Fatalf("expected optimizing strategy, got %#v", summary["current"])
	}
}

func TestProgressStoreSkippedCountsTowardCompletion(t *testing.T) {
	ps := NewProgressStore()
	ps.Create("batch-1", 1, []ComboProgress{
		{StrategyID: "session_scalp", Symbol: "SPX500", Timeframe: "5m", Status: "pending"},
	})

	ps.UpdateCombo("batch-1", 0, "skipped", "requires real intraday market data", nil)
	summary := ps.GetSummary("batch-1")
	if summary["completed"] != 1 || summary["skipped"] != 1 {
		t.Fatalf("skipped combo was not counted: %#v", summary)
	}
	if summary["failed"] != 0 {
		t.Fatalf("skip reason was counted as a failure: %#v", summary)
	}
	if summary["status"] != "completed" {
		t.Fatalf("expected terminal status, got %v", summary["status"])
	}
}

func TestProgressStoreListsOnlyActiveBatches(t *testing.T) {
	ps := NewProgressStore()
	ps.Create("active", 2, nil)
	ps.Create("done", 1, []ComboProgress{{Status: "pending"}})
	ps.UpdateCombo("done", 0, "completed", "", &backtest.ComboResult{StrategyID: "s", Symbol: "x", Timeframe: "1d"})

	active := ps.ListActiveSummaries()
	if len(active) != 1 || active[0]["batch_run_id"] != "active" {
		t.Fatalf("unexpected active batches: %#v", active)
	}
}
