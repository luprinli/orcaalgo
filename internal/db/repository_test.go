package db

import (
	"testing"
	"time"
)

// TestCandleStructFields verifies the Candle type fields.
func TestCandleStructFields(t *testing.T) {
	c := Candle{
		Time:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Open:   100.0,
		High:   101.0,
		Low:    99.0,
		Close:  100.5,
		Volume: 1000,
		Symbol: "TEST",
	}
	if c.Open != 100.0 {
		t.Fatalf("Open field broken: got %f, want 100.0", c.Open)
	}
	if c.Symbol != "TEST" {
		t.Fatalf("Symbol field broken: got %s, want TEST", c.Symbol)
	}

	// TODO: RegimeLabel field to be added for synthetic regime label propagation
	// TODO: Migrate Open/High/Low/Close to fixed.Price (github.com/lee-econ/orca-core/internal/fixed)
}

// TestRepositoryMethodsExist verifies core repository methods exist.
func TestRepositoryMethodsExist(t *testing.T) {
	r := &Repository{}
	if r == nil {
		t.Fatal("Repository should not be nil")
	}
	// Compile-time check: these methods exist on *Repository
	_ = r.LoadCandles
	_ = r.LoadCandlesByTimeframe
	// TODO: LoadCandlesFiltered to be added for regime-filtered candle loading
}
