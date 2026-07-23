package strategy

import "testing"

func TestStopLossChecker_Buy(t *testing.T) {
	sc := StopLossChecker{}
	if !sc.IsStopLossHit(95, 100, "BUY") {
		t.Error("BUY stop should trigger when price <= stop")
	}
	if !sc.IsStopLossHit(100, 100, "BUY") {
		t.Error("BUY stop should trigger at exact stop (price <= stopLoss)")
	}
	if sc.IsStopLossHit(105, 100, "BUY") {
		t.Error("BUY stop should not trigger above stop")
	}
}

func TestStopLossChecker_Sell(t *testing.T) {
	sc := StopLossChecker{}
	if !sc.IsStopLossHit(105, 100, "SELL") {
		t.Error("SELL stop should trigger when price >= stop")
	}
	if !sc.IsStopLossHit(100, 100, "SELL") {
		t.Error("SELL stop should trigger at exact stop (price >= stopLoss)")
	}
	if sc.IsStopLossHit(95, 100, "SELL") {
		t.Error("SELL stop should not trigger below stop")
	}
}

func TestTakeProfitChecker_Buy(t *testing.T) {
	tc := TakeProfitChecker{}
	if !tc.IsTakeProfitHit(110, 100, "BUY") {
		t.Error("BUY TP should trigger when price >= target")
	}
	if !tc.IsTakeProfitHit(100, 100, "BUY") {
		t.Error("BUY TP should trigger at exact target")
	}
	if tc.IsTakeProfitHit(90, 100, "BUY") {
		t.Error("BUY TP should not trigger below target")
	}
}

func TestTakeProfitChecker_Sell(t *testing.T) {
	tc := TakeProfitChecker{}
	if !tc.IsTakeProfitHit(90, 100, "SELL") {
		t.Error("SELL TP should trigger when price <= target")
	}
	if !tc.IsTakeProfitHit(100, 100, "SELL") {
		t.Error("SELL TP should trigger at exact target")
	}
	if tc.IsTakeProfitHit(110, 100, "SELL") {
		t.Error("SELL TP should not trigger above target")
	}
}
