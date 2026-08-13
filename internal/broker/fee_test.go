package broker

import (
	"testing"
)

func TestCalculateFee_Basic(t *testing.T) {
	f := DefaultBrokerageFee()
	fee := f.CalculateFee(100, 500.0)
	if fee < f.MinFee {
		t.Errorf("Fee should be >= MinFee: got %f, min %f", fee, f.MinFee)
	}
}

func TestCalculateFee_Disabled(t *testing.T) {
	f := BrokerageFeeConfig{Enabled: false}
	if fee := f.CalculateFee(100, 500.0); fee != 0 {
		t.Errorf("Disabled config should return 0, got %f", fee)
	}
}

func TestCalculateFee_MinFeeClamp(t *testing.T) {
	f := BrokerageFeeConfig{PerTradeFixed: 0.1, PerShare: 0.01, MinFee: 1.0, TakerFeeBps: 0.1, Enabled: true}
	fee := f.CalculateFee(1, 10.0)
	if fee < f.MinFee {
		t.Errorf("Should clamp to min fee: got %f, min %f", fee, f.MinFee)
	}
}

func TestCalculateFee_LargeOrder(t *testing.T) {
	f := DefaultBrokerageFee()
	fee := f.CalculateFee(10000, 500.0)
	if fee < f.MinFee {
		t.Errorf("Large order fee should be >= MinFee: got %f", fee)
	}
	expectedFloor := 10000 * 500.0 * f.SECTxnFee
	if fee < expectedFloor*0.9 {
		t.Errorf("Large order fee should reflect SEC txn fee: got %f, expected >= %f", fee, expectedFloor*0.9)
	}
}

func TestCalculateMakerFee_CheaperThanTaker(t *testing.T) {
	f := BrokerageFeeConfig{
		PerTradeFixed: 0.35, PerShare: 0.0035, MinFee: 0.5,
		MakerFeeBps: 0.01, TakerFeeBps: 0.5, Enabled: true,
	}
	makerFee := f.CalculateMakerFee(100, 500.0)
	takerFee := f.CalculateFee(100, 500.0)
	if makerFee >= takerFee {
		t.Errorf("Maker fee should be cheaper: maker=%f, taker=%f", makerFee, takerFee)
	}
}

func TestCalculateFee_ZeroQuantity(t *testing.T) {
	f := DefaultBrokerageFee()
	fee := f.CalculateFee(0, 500.0)
	if fee < f.MinFee {
		t.Errorf("Zero-qty should still pay min fee: got %f, min %f", fee, f.MinFee)
	}
}

func TestCalculateMakerFee_Disabled(t *testing.T) {
	f := BrokerageFeeConfig{Enabled: false}
	if fee := f.CalculateMakerFee(100, 500.0); fee != 0 {
		t.Errorf("Disabled config should return 0 for maker fee, got %f", fee)
	}
}

func TestDefaultBrokerageFee_HasAllFields(t *testing.T) {
	f := DefaultBrokerageFee()
	if f.PerTradeFixed <= 0 {
		t.Error("PerTradeFixed should be positive")
	}
	if f.Enabled != true {
		t.Error("Default should be enabled")
	}
	if f.TakerFeeBps <= 0 {
		t.Error("TakerFeeBps should be positive")
	}
}

func TestCalculateHoldingFee_Basic(t *testing.T) {
	f := DefaultBrokerageFee()
	// $10,000 notional held 1 year at 0.08% expense ratio = $8.
	fee := f.CalculateHoldingFee(10000.0, 0.0008, 1.0)
	if fee < 7.9 || fee > 8.1 {
		t.Errorf("Holding fee should be ~8.0, got %f", fee)
	}
}

func TestCalculateHoldingFee_ZeroAndNegative(t *testing.T) {
	f := DefaultBrokerageFee()
	if fee := f.CalculateHoldingFee(0, 0.001, 1.0); fee != 0 {
		t.Errorf("Zero notional should return 0, got %f", fee)
	}
	if fee := f.CalculateHoldingFee(10000, 0.001, 0); fee != 0 {
		t.Errorf("Zero yearsHeld should return 0, got %f", fee)
	}
	if fee := f.CalculateHoldingFee(10000, -0.001, 1.0); fee != 0 {
		t.Errorf("Negative ratio should return 0, got %f", fee)
	}
	if fee := f.CalculateHoldingFee(-10000, 0.001, 1.0); fee != 0 {
		t.Errorf("Negative notional should return 0, got %f", fee)
	}
}

func TestCalculateHoldingFee_Disabled(t *testing.T) {
	f := BrokerageFeeConfig{Enabled: false}
	if fee := f.CalculateHoldingFee(10000, 0.001, 1.0); fee != 0 {
		t.Errorf("Disabled config should return 0, got %f", fee)
	}
}
