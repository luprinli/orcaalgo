package api

import (
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/types"
)

func TestNormalizeBenchmark_Base100(t *testing.T) {
	candles := []db.Candle{
		{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Close: types.PriceFromFloat(200)},
		{Time: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Close: types.PriceFromFloat(210)},
		{Time: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), Close: types.PriceFromFloat(190)},
	}
	out := normalizeBenchmark(candles)
	if len(out) != 3 {
		t.Fatalf("expected 3 points, got %d", len(out))
	}
	if out[0].Value != 100 {
		t.Errorf("first point should be 100, got %f", out[0].Value)
	}
	// 210/200 = 1.05 -> 105
	if out[1].Value < 104.9 || out[1].Value > 105.1 {
		t.Errorf("second point should be ~105, got %f", out[1].Value)
	}
	// 190/200 = 0.95 -> 95
	if out[2].Value < 94.9 || out[2].Value > 95.1 {
		t.Errorf("third point should be ~95, got %f", out[2].Value)
	}
}

func TestNormalizeBenchmark_Empty(t *testing.T) {
	out := normalizeBenchmark(nil)
	if len(out) != 0 {
		t.Errorf("empty input should return empty, got %d", len(out))
	}
}
