package strategy_test

import (
	"testing"
	"time"

	strategy "github.com/lee-econ/orca-core/internal/strategy"
	"github.com/lee-econ/orca-core/internal/types"
)

func TestBaseRunner_PushPrice(t *testing.T) {
	b := strategy.NewBaseRunner(64)
	b.PushPrice(types.PriceFromFloat(100.0), types.PriceFromFloat(101.0), types.PriceFromFloat(99.0), 1000.0)
	if b.HistCount != 1 {
		t.Errorf("HistCount = %d, want 1", b.HistCount)
	}
	if b.PriceHistory[0] != 100.0 {
		t.Errorf("PriceHistory[0] = %v, want 100.0", b.PriceHistory[0])
	}
	if b.HighHistory[0] != 101.0 {
		t.Errorf("HighHistory[0] = %v, want 101.0", b.HighHistory[0])
	}
	if b.LowHistory[0] != 99.0 {
		t.Errorf("LowHistory[0] = %v, want 99.0", b.LowHistory[0])
	}
	if b.VolumeHistory[0] != 1000.0 {
		t.Errorf("VolumeHistory[0] = %v, want 1000.0", b.VolumeHistory[0])
	}
}

func TestBaseRunner_RingBufferWrap(t *testing.T) {
	b := strategy.NewBaseRunner(4)
	for i := 0; i < 8; i++ {
		b.PushPrice(types.PriceFromFloat(float64(i)), 0, 0, 0)
	}
	if b.HistCount != 4 {
		t.Errorf("HistCount = %d, want 4", b.HistCount)
	}
	if b.HistIdx != 8 {
		t.Errorf("HistIdx = %d, want 8", b.HistIdx)
	}
	if b.PriceHistory[0] != 4.0 {
		t.Errorf("PriceHistory[0] = %v, want 4.0", b.PriceHistory[0])
	}
}

func TestBaseRunner_Reset(t *testing.T) {
	b := strategy.NewBaseRunner(64)
	b.PushPrice(types.PriceFromFloat(100.0), types.PriceFromFloat(101.0), types.PriceFromFloat(99.0), 1000.0)
	b.Reset()
	if b.HistCount != 0 {
		t.Errorf("HistCount after reset = %d, want 0", b.HistCount)
	}
	if b.HistIdx != 0 {
		t.Errorf("HistIdx after reset = %d, want 0", b.HistIdx)
	}
}

func TestBaseRunner_OpenClosePosition(t *testing.T) {
	b := strategy.NewBaseRunner(64)
	now := time.Now()
	b.OpenPosition("BUY", types.PriceFromFloat(100.0), types.PriceFromFloat(95.0), types.PriceFromFloat(110.0), now)
	if !b.PositionOpen {
		t.Fatal("Position should be open")
	}
	if b.CurrentSide != "BUY" {
		t.Errorf("Side = %v, want BUY", b.CurrentSide)
	}
	if b.EntryPrice.Float64() != 100.0 {
		t.Errorf("EntryPrice = %v, want 100.0", b.EntryPrice.Float64())
	}

	b.ClosePosition()
	if b.PositionOpen {
		t.Error("Position should be closed")
	}
}

func TestBaseRunner_StopLossHit_Long(t *testing.T) {
	b := strategy.NewBaseRunner(64)
	now := time.Now()
	b.OpenPosition("BUY", types.PriceFromFloat(100.0), types.PriceFromFloat(95.0), types.PriceFromFloat(110.0), now)
	if !b.IsStopLossHit(types.PriceFromFloat(94.0)) {
		t.Error("Stop loss should be hit at 94.0")
	}
	if b.IsStopLossHit(types.PriceFromFloat(96.0)) {
		t.Error("Stop loss should NOT be hit at 96.0")
	}
}

func TestBaseRunner_StopLossHit_Short(t *testing.T) {
	b := strategy.NewBaseRunner(64)
	now := time.Now()
	b.OpenPosition("SELL", types.PriceFromFloat(100.0), types.PriceFromFloat(105.0), types.PriceFromFloat(90.0), now)
	if !b.IsStopLossHit(types.PriceFromFloat(106.0)) {
		t.Error("Stop loss should be hit at 106.0")
	}
	if b.IsStopLossHit(types.PriceFromFloat(104.0)) {
		t.Error("Stop loss should NOT be hit at 104.0")
	}
}

func TestBaseRunner_TakeProfitHit_Long(t *testing.T) {
	b := strategy.NewBaseRunner(64)
	now := time.Now()
	b.OpenPosition("BUY", types.PriceFromFloat(100.0), types.PriceFromFloat(95.0), types.PriceFromFloat(110.0), now)
	if !b.IsTakeProfitHit(types.PriceFromFloat(111.0)) {
		t.Error("Take profit should be hit at 111.0")
	}
}

func TestBaseRunner_TakeProfitHit_Short(t *testing.T) {
	b := strategy.NewBaseRunner(64)
	now := time.Now()
	b.OpenPosition("SELL", types.PriceFromFloat(100.0), types.PriceFromFloat(105.0), types.PriceFromFloat(90.0), now)
	if !b.IsTakeProfitHit(types.PriceFromFloat(89.0)) {
		t.Error("Take profit should be hit at 89.0")
	}
}

func TestBaseRunner_TimeExit(t *testing.T) {
	b := strategy.NewBaseRunner(64)
	entry := time.Now().Add(-10 * time.Minute)
	b.OpenPosition("BUY", types.PriceFromFloat(100.0), types.PriceFromFloat(95.0), types.PriceFromFloat(110.0), entry)
	if !b.IsTimeExit(5.0, time.Now()) {
		t.Error("Time exit should trigger after 10 min with 5 min limit")
	}
}

func TestBaseRunner_NoExitWhenClosed(t *testing.T) {
	b := strategy.NewBaseRunner(64)
	if b.IsStopLossHit(types.PriceFromFloat(1.0)) {
		t.Error("No exit when no position")
	}
	if b.IsTakeProfitHit(types.PriceFromFloat(1.0)) {
		t.Error("No take profit when no position")
	}
	if b.IsTimeExit(1.0, time.Now()) {
		t.Error("No time exit when no position")
	}
}

func TestBaseRunner_PushPriceOnly(t *testing.T) {
	b := strategy.NewBaseRunner(64)
	b.PushPriceOnly(types.PriceFromFloat(50.0))
	if b.PriceHistory[0] != 50.0 {
		t.Errorf("Price = %v, want 50.0", b.PriceHistory[0])
	}
	if b.HighHistory[0] != 0 {
		t.Errorf("High should be 0 for PushPriceOnly, got %v", b.HighHistory[0])
	}
}
