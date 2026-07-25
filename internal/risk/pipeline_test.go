package risk

import (
	"context"
	"testing"
)

// mockCapitalGate is a test double that records RequestCapital calls.
type mockCapitalGate struct {
	approvedSize float64
	reason       string
	halted       bool
	haltReason   string
	balance      float64
	fillCount    int
	resetCount   int
}

func (m *mockCapitalGate) RequestCapital(ctx context.Context, req CapitalRequest) CapitalResult {
	return CapitalResult{ApprovedSize: m.approvedSize, Reason: m.reason}
}

func (m *mockCapitalGate) RecordFill(strategyID, symbol, side string, pnl, quantity float64) {
	m.fillCount++
	m.balance += pnl
}

func (m *mockCapitalGate) ResetDaily() { m.resetCount++ }
func (m *mockCapitalGate) Halted() bool    { return m.halted }
func (m *mockCapitalGate) HaltReason() string { return m.haltReason }
func (m *mockCapitalGate) TotalBalance() float64 { return m.balance }

// mockPropFirmGate is a test double for PropFirmGate.
type mockPropFirmGate struct {
	halted          bool
	haltReason      string
	currentPhase    int
	profitTargetMet bool
	positionSize    float64
	violated        string
}

func (m *mockPropFirmGate) CheckDailyLimits() (bool, string) {
	if m.halted {
		return false, m.haltReason
	}
	return true, ""
}

func (m *mockPropFirmGate) OnFill(pnl, balance float64) {}
func (m *mockPropFirmGate) OnNewDay()                     {}
func (m *mockPropFirmGate) IsHalted() bool                { return m.halted }
func (m *mockPropFirmGate) HaltReason() string            { return m.haltReason }
func (m *mockPropFirmGate) MarkViolated(reason string)    { m.violated = reason; m.halted = true }
func (m *mockPropFirmGate) CurrentPhase() int             { return m.currentPhase }
func (m *mockPropFirmGate) ProfitTargetMet() bool         { return m.profitTargetMet }
func (m *mockPropFirmGate) GetPositionSize(baseQty float64) float64 {
	if m.positionSize > 0 {
		return m.positionSize
	}
	return baseQty
}

// mockSignalGate is a test double for SignalGate.
type mockSignalGate struct {
	validated     bool
	validReason   string
	sized         float64
	exposed       bool
	exposureOk    bool
	exposureReason string
	recorded      []string
	removed       []string
}

func (m *mockSignalGate) ValidateSignal(runningCapital float64) (bool, string) {
	return m.validated, m.validReason
}

func (m *mockSignalGate) ApplySizing(baseSize, runningCapital, confidence float64) float64 {
	if m.sized > 0 {
		return m.sized
	}
	return baseSize * 0.25
}

func (m *mockSignalGate) CheckExposure(symbol, side string, notional float64) (bool, string) {
	return m.exposureOk, m.exposureReason
}

func (m *mockSignalGate) RecordExposure(symbol, side string, notional float64) {
	m.recorded = append(m.recorded, symbol+":"+side)
}

func (m *mockSignalGate) RemoveExposure(symbol, side string, notional float64) {
	m.removed = append(m.removed, symbol+":"+side)
}

func TestRiskPipeline_ProcessSignal_Approved(t *testing.T) {
	capital := &mockCapitalGate{approvedSize: 5000, reason: "ok", balance: 100000}
	signal := &mockSignalGate{validated: true, sized: 2500, exposureOk: true}
	propfirm := &mockPropFirmGate{}

	pipeline := &RiskPipeline{
		SignalGate: signal,
		Capital:    capital,
		PropFirm:   propfirm,
		KellyMult:  0.25,
	}

	result := pipeline.ProcessSignal(context.Background(), ProcessSignalRequest{
		StrategyID:       "strat-1",
		Symbol:           "SPY",
		Side:             "BUY",
		Price:            500.0,
		Confidence:       0.8,
		BaseSize:         10000,
		ExistingPosition: 0,
		RunningCapital:   100000,
	})

	if !result.Approved {
		t.Errorf("Expected approved, got rejected: %s", result.Reason)
	}
	if result.Size <= 0 {
		t.Errorf("Expected positive size, got %f", result.Size)
	}
	if len(signal.recorded) != 1 {
		t.Errorf("Expected 1 exposure recording, got %d", len(signal.recorded))
	}
}

func TestRiskPipeline_ProcessSignal_RejectedByPoolHalted(t *testing.T) {
	capital := &mockCapitalGate{halted: true, haltReason: "max_drawdown"}
	pipeline := &RiskPipeline{Capital: capital, KellyMult: 0.25}

	result := pipeline.ProcessSignal(context.Background(), ProcessSignalRequest{
		StrategyID: "s1", Symbol: "SPY", Side: "BUY",
		Price: 500, Confidence: 0.8, BaseSize: 10000, RunningCapital: 100000,
	})

	if result.Approved {
		t.Error("Expected rejected when pool is halted")
	}
	if result.Reason != "pool_halted:max_drawdown" {
		t.Errorf("Expected reason 'pool_halted:max_drawdown', got '%s'", result.Reason)
	}
}

func TestRiskPipeline_ProcessSignal_RejectedByPropFirmHalted(t *testing.T) {
	propfirm := &mockPropFirmGate{halted: true, haltReason: "daily_loss"}
	pipeline := &RiskPipeline{PropFirm: propfirm, KellyMult: 0.25}

	result := pipeline.ProcessSignal(context.Background(), ProcessSignalRequest{
		StrategyID: "s1", Symbol: "SPY", Side: "BUY",
		Price: 500, Confidence: 0.8, BaseSize: 10000, RunningCapital: 100000,
	})

	if result.Approved {
		t.Error("Expected rejected when prop-firm gate is halted")
	}
}

func TestRiskPipeline_ProcessSignal_RejectedByExposure(t *testing.T) {
	signal := &mockSignalGate{validated: true, sized: 5000, exposureOk: false, exposureReason: "max_leverage"}
	pipeline := &RiskPipeline{SignalGate: signal, KellyMult: 0.25}

	result := pipeline.ProcessSignal(context.Background(), ProcessSignalRequest{
		StrategyID: "s1", Symbol: "SPY", Side: "BUY",
		Price: 500, Confidence: 0.8, BaseSize: 10000, RunningCapital: 100000,
	})

	if result.Approved {
		t.Error("Expected rejected by exposure check")
	}
	if result.Reason != "exposure:max_leverage" {
		t.Errorf("Expected reason 'exposure:max_leverage', got '%s'", result.Reason)
	}
}

func TestRiskPipeline_ProcessSignal_RejectedByCapital(t *testing.T) {
	capital := &mockCapitalGate{approvedSize: 0, reason: "max_open_positions"}
	signal := &mockSignalGate{validated: true, sized: 5000, exposureOk: true}
	pipeline := &RiskPipeline{SignalGate: signal, Capital: capital, KellyMult: 0.25}

	result := pipeline.ProcessSignal(context.Background(), ProcessSignalRequest{
		StrategyID: "s1", Symbol: "SPY", Side: "BUY",
		Price: 500, Confidence: 0.8, BaseSize: 10000, RunningCapital: 100000,
	})

	if result.Approved {
		t.Error("Expected rejected by capital gate")
	}
}

func TestRiskPipeline_ReconcileFill(t *testing.T) {
	capital := &mockCapitalGate{balance: 100000}
	propfirm := &mockPropFirmGate{}
	signal := &mockSignalGate{}
	pipeline := &RiskPipeline{SignalGate: signal, Capital: capital, PropFirm: propfirm}

	pipeline.ReconcileFill("s1", "SPY", "BUY", 500, 100, 500)

	if capital.fillCount != 1 {
		t.Errorf("Expected 1 fill recorded, got %d", capital.fillCount)
	}
	if capital.balance != 100500 {
		t.Errorf("Expected balance 100500 after +500 PnL, got %f", capital.balance)
	}
	if len(signal.removed) != 1 {
		t.Errorf("Expected 1 exposure removal, got %d", len(signal.removed))
	}
}

func TestRiskPipeline_ReconcileFill_PropFirmViolation(t *testing.T) {
	capital := &mockCapitalGate{balance: 100000}
	propfirm := &mockPropFirmGate{halted: false}
	pipeline := &RiskPipeline{Capital: capital, PropFirm: propfirm}

	// Set up a condition where CheckDailyLimits will fail
	propfirm.halted = true
	propfirm.haltReason = "daily_loss_limit"
	pipeline.ReconcileFill("s1", "SPY", "BUY", -6000, 100, 500)

	if propfirm.violated != "daily_loss_limit" {
		t.Errorf("Expected prop-firm to be marked violated with 'daily_loss_limit', got '%s'", propfirm.violated)
	}
}

func TestRiskPipeline_ReconcileFillWithoutPropFirm(t *testing.T) {
	capital := &mockCapitalGate{balance: 100000}
	signal := &mockSignalGate{}
	pipeline := &RiskPipeline{SignalGate: signal, Capital: capital}

	pipeline.ReconcileFillWithoutPropFirm("s1", "SPY", "BUY", 500, 100, 500)

	if capital.fillCount != 1 {
		t.Errorf("Expected 1 fill recorded, got %d", capital.fillCount)
	}
}

func TestRiskPipeline_ZeroSizeRejected(t *testing.T) {
	pipeline := &RiskPipeline{KellyMult: 0.25}

	result := pipeline.ProcessSignal(context.Background(), ProcessSignalRequest{
		StrategyID: "s1", Symbol: "SPY", Side: "BUY",
		Price: 500, Confidence: 0.8, BaseSize: 0, RunningCapital: 100000,
	})

	if result.Approved {
		t.Error("Expected rejected for zero base size")
	}
}

func TestRiskPipeline_NullComponentsTolerated(t *testing.T) {
	pipeline := &RiskPipeline{KellyMult: 0.25}

	result := pipeline.ProcessSignal(context.Background(), ProcessSignalRequest{
		StrategyID: "s1", Symbol: "SPY", Side: "BUY",
		Price: 500, Confidence: 0.8, BaseSize: 1000, RunningCapital: 100000,
	})

	if !result.Approved {
		t.Errorf("Expected approved with null components, got rejected: %s", result.Reason)
	}
}
