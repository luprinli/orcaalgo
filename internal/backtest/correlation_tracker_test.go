package backtest

import (
	"testing"
)

func TestCorrelationTracker_PerfectPositive(t *testing.T) {
	ct := NewCorrelationTracker(30, 0.6)
	for i := 0; i < 20; i++ {
		ct.RecordEquity("a", float64(100+i))
		ct.RecordEquity("b", float64(100+i))
	}
	corrs, _ := ct.CheckCorrelations()
	if len(corrs) != 1 {
		t.Fatalf("expected 1 correlation, got %d", len(corrs))
	}
	if corrs[0].Correlation < 0.99 {
		t.Errorf("expected >0.99 for identical equity, got %f", corrs[0].Correlation)
	}
	if !corrs[0].BrakeActive {
		t.Error("brake should be active for ρ > 0.6")
	}
}

func TestCorrelationTracker_Uncorrelated(t *testing.T) {
	ct := NewCorrelationTracker(30, 0.6)
	for i := 0; i < 50; i++ {
		a := 100.0 + float64(i)
		b := 100.0
		if i%2 == 0 { b += 2 } else { b -= 2 }
		ct.RecordEquity("a", a)
		ct.RecordEquity("b", b)
	}
	corrs, _ := ct.CheckCorrelations()
	if len(corrs) != 1 {
		t.Fatalf("expected 1 correlation, got %d", len(corrs))
	}
	if corrs[0].Correlation > 0.3 || corrs[0].Correlation < -0.3 {
		t.Errorf("expected low correlation, got %f (should be near zero)", corrs[0].Correlation)
	}
}

func TestCorrelationTracker_InsufficientData(t *testing.T) {
	ct := NewCorrelationTracker(30, 0.6)
	ct.RecordEquity("a", 100)
	ct.RecordEquity("b", 200)
	corrs, _ := ct.CheckCorrelations()
	if len(corrs) > 0 {
		t.Errorf("expected 0 correlations with <2 points each, got %d", len(corrs))
	}
}

func TestCorrelationTracker_BrakeDiscount(t *testing.T) {
	ct := NewCorrelationTracker(30, 0.6)
	for i := 0; i < 20; i++ {
		ct.RecordEquity("a", float64(100+i))
		ct.RecordEquity("b", float64(100+i))
	}
	ct.CheckCorrelations()
	d := ct.BrakeDiscount("a", "b")
	if d != 0.70 {
		t.Errorf("expected brake discount 0.70, got %f", d)
	}
	d2 := ct.BrakeDiscount("b", "a")
	if d2 != 0.70 {
		t.Errorf("brake discount should be symmetric, got %f", d2)
	}
}

func TestCorrelationTracker_SingleStrategy(t *testing.T) {
	ct := NewCorrelationTracker(30, 0.6)
	for i := 0; i < 10; i++ {
		ct.RecordEquity("a", float64(100+i))
	}
	corrs, _ := ct.CheckCorrelations()
	if len(corrs) != 0 {
		t.Errorf("expected 0 correlations with 1 strategy, got %d", len(corrs))
	}
}

func TestCorrelationTracker_PairMatrix(t *testing.T) {
	ct := NewCorrelationTracker(30, 0.6)
	for i := 0; i < 20; i++ {
		ct.RecordEquity("a", float64(100+i))
		ct.RecordEquity("b", float64(100+i))
		ct.RecordEquity("c", float64(200-i))
	}
	ct.CheckCorrelations()
	matrix := ct.PairMatrix()
	if len(matrix) != 3 {
		t.Errorf("expected 3 strategies in pair matrix, got %d", len(matrix))
	}
	if _, ok := matrix["a"]["b"]; !ok {
		t.Error("missing a-b correlation")
	}
}
