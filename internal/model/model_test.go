package model

import (
	"testing"
	"time"
)

func TestConstantLatency(t *testing.T) {
	m := ConstantLatency{Entry: 50 * time.Millisecond, Response: 30 * time.Millisecond}
	e := m.EntryLatency(time.Now(), 100000, 1000, "BUY")
	r := m.ResponseLatency(time.Now(), 100000, "BUY")
	if e != 50*time.Millisecond {
		t.Errorf("expected 50ms entry, got %v", e)
	}
	if r != 30*time.Millisecond {
		t.Errorf("expected 30ms response, got %v", r)
	}
}

func TestZeroLatency(t *testing.T) {
	m := ZeroLatency{}
	if m.EntryLatency(time.Now(), 100000, 1000, "BUY") != 0 {
		t.Error("ZeroLatency should return 0")
	}
	if m.ResponseLatency(time.Now(), 100000, "BUY") != 0 {
		t.Error("ZeroLatency should return 0")
	}
}

func TestMidPriceFill(t *testing.T) {
	m := MidPriceFill{}
	if m.FillProbability(100000, 100000, 100, 5000, 0.01) != 1.0 {
		t.Error("MidPriceFill always fills")
	}
	if m.FillPrice(101000, 100000, "BUY", 0.01) != 100000 {
		t.Error("MidPriceFill always fills at mid")
	}
}

func TestProbabilisticFill_Basic(t *testing.T) {
	p := NewProbabilisticFill(0.8, 2.5, 100.0)
	prob := p.FillProbability(100000, 100000, 100, 5000, 10.0)
	if prob < 0.6 || prob > 1.0 {
		t.Errorf("at mid with some volume should be reasonable, got %.4f", prob)
	}

	prob2 := p.FillProbability(100200, 100000, 100, 5000, 10.0)
	if prob2 > prob {
		t.Error("far from mid should have lower fill probability")
	}
}

func TestProbabilisticFill_Edge(t *testing.T) {
	p := NewProbabilisticFill(0.8, 2.5, 100.0)
	prob := p.FillProbability(100000, 100000, 0, 5000, 0.01)
	if prob < 0.7 || prob > 0.9 {
		t.Errorf("zero spread: expected ~0.8, got %.4f", prob)
	}

	prob2 := p.FillProbability(100000, 100000, 100, 0, 0.01)
	if prob2 <= 0 {
		t.Error("zero volume should still produce non-zero probability")
	}
}

func TestProbabilisticFill_Clamped(t *testing.T) {
	p := NewProbabilisticFill(0.8, 100.0, 100.0)
	prob := p.FillProbability(100000, 100000, 10, 1000000, 0.01)
	if prob > 1.0 {
		t.Errorf("probability should be clamped to 1.0, got %.4f", prob)
	}
}

func TestFixedFee(t *testing.T) {
	f := FixedFee{MakerBps: 2.0, TakerBps: 5.0}
	maker := f.MakerFee(100000)
	taker := f.TakerFee(100000)
	if maker != 20.0 {
		t.Errorf("maker fee: expected 20, got %.2f", maker)
	}
	if taker != 50.0 {
		t.Errorf("taker fee: expected 50, got %.2f", taker)
	}
}

func TestZeroFee(t *testing.T) {
	z := ZeroFee{}
	if z.MakerFee(100000) != 0 || z.TakerFee(100000) != 0 {
		t.Error("ZeroFee should return 0")
	}
}

func TestTradingState(t *testing.T) {
	ts := TradingState{
		Timestamp:    time.Now(),
		Balance:      100000,
		Position:     100,
		MidPrice:     50000,
		Fee:          5.0,
		TradingVolume: 1000,
		TradingValue:  50000,
		NumTrades:    10,
	}
	if ts.Balance != 100000 || ts.NumTrades != 10 {
		t.Error("TradingState fields should match")
	}
}

func TestOrderTimestamp(t *testing.T) {
	now := time.Now()
	ot := OrderTimestamp{
		RequestedAt:    now,
		SentAt:         now.Add(1 * time.Millisecond),
		AcknowledgedAt: now.Add(55 * time.Millisecond),
		FilledAt:       now.Add(55 * time.Millisecond),
	}
	if ot.FilledAt.Before(ot.RequestedAt) {
		t.Error("FilledAt should be after RequestedAt")
	}
}
