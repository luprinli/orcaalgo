package api

import (
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/types"
)

func makeExec(id string, symbol string, side string, qty, price float64, t time.Time) db.TradeExecution {
	return db.TradeExecution{ID: id, Symbol: symbol, Side: side, Quantity: qty, Price: price, ExecutedAt: t}
}

func makeFill(symbol string, side broker.OrderSide, qty float64, price types.Price, t time.Time) broker.TradeFill {
	return broker.TradeFill{Symbol: symbol, Side: side, Quantity: qty, FillPrice: price, FillTime: t}
}

func TestReconciliation_AllMatched(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	internal := []db.TradeExecution{
		makeExec("1", "SPY", "BUY", 100, 580.0, now),
		makeExec("2", "QQQ", "SELL", 50, 420.0, now.Add(500*time.Millisecond)),
	}
	brokerFills := []broker.TradeFill{
		makeFill("SPY", broker.Buy, 100, types.FromFloat64(580.0), now),
		makeFill("QQQ", broker.Sell, 50, types.FromFloat64(420.0), now.Add(500*time.Millisecond)),
	}

	result := MatchReconciliation(internal, brokerFills)
	if result.Matched != 2 {
		t.Errorf("Matched = %d, want 2", result.Matched)
	}
	if result.InternalCount != 2 {
		t.Errorf("InternalCount = %d, want 2", result.InternalCount)
	}
	if result.Extra != 0 {
		t.Errorf("Extra = %d, want 0", result.Extra)
	}
	if result.Missing != 0 {
		t.Errorf("Missing = %d, want 0", result.Missing)
	}
}

func TestReconciliation_MissingFill(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	internal := []db.TradeExecution{
		makeExec("1", "SPY", "BUY", 100, 580.0, now),
	}
	brokerFills := []broker.TradeFill{
		makeFill("SPY", broker.Buy, 100, types.FromFloat64(580.0), now),
		makeFill("QQQ", broker.Sell, 50, types.FromFloat64(420.0), now),
	}

	result := MatchReconciliation(internal, brokerFills)
	if result.Matched != 1 {
		t.Errorf("Matched = %d, want 1", result.Matched)
	}
	if result.Missing != 1 {
		t.Errorf("Missing = %d, want 1", result.Missing)
	}
}

func TestReconciliation_ExtraExecution(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	internal := []db.TradeExecution{
		makeExec("1", "SPY", "BUY", 100, 580.0, now),
		makeExec("2", "QQQ", "SELL", 50, 420.0, now),
	}
	brokerFills := []broker.TradeFill{
		makeFill("SPY", broker.Buy, 100, types.FromFloat64(580.0), now),
	}

	result := MatchReconciliation(internal, brokerFills)
	if result.Matched != 1 {
		t.Errorf("Matched = %d, want 1", result.Matched)
	}
	if result.Extra != 1 {
		t.Errorf("Extra = %d, want 1", result.Extra)
	}
}

func TestReconciliation_PriceDiscrepancy(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	internal := []db.TradeExecution{
		makeExec("1", "SPY", "BUY", 100, 580.0, now),
	}
	brokerFills := []broker.TradeFill{
		makeFill("SPY", broker.Buy, 100, types.FromFloat64(582.0), now),
	}

	result := MatchReconciliation(internal, brokerFills)
	if result.PriceDiscrepancies != 1 {
		t.Errorf("PriceDiscrepancies = %d, want 1", result.PriceDiscrepancies)
	}
}

func TestReconciliation_WithinTolerance(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	internal := []db.TradeExecution{
		makeExec("1", "SPY", "BUY", 100, 580.0, now),
	}
	brokerFills := []broker.TradeFill{
		makeFill("SPY", broker.Buy, 100, types.FromFloat64(580.02), now),
	}

	result := MatchReconciliation(internal, brokerFills)
	if result.Matched != 1 {
		t.Errorf("Matched = %d, want 1 (should be within 5bps tolerance)", result.Matched)
	}
	if result.PriceDiscrepancies != 0 {
		t.Errorf("PriceDiscrepancies = %d, want 0 (within tolerance)", result.PriceDiscrepancies)
	}
}

func TestReconciliation_EmptyBrokerFills(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	internal := []db.TradeExecution{
		makeExec("1", "SPY", "BUY", 100, 580.0, now),
	}

	result := MatchReconciliation(internal, nil)
	if result.Extra != 1 {
		t.Errorf("Extra = %d, want 1 (internal-only unmatched)", result.Extra)
	}
	if result.InternalCount != 1 {
		t.Errorf("InternalCount = %d, want 1", result.InternalCount)
	}
}
