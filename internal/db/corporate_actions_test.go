package db

import (
	"testing"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

func TestApplyCorporateActions_IdentityWhenNoActions(t *testing.T) {
	candles := []Candle{
		{Time: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: types.PriceFromFloat(100.0)},
		{Time: time.Date(2024, 6, 3, 0, 0, 0, 0, time.UTC), Close: types.PriceFromFloat(200.0)},
	}
	out := ApplyCorporateActions(candles, nil)
	for i := range out {
		if out[i].AdjustmentFactor != 1.0 {
			t.Errorf("bar %d: expected identity factor 1.0, got %f", i, out[i].AdjustmentFactor)
		}
	}
}

func TestApplyCorporateActions_CumulativeSplit(t *testing.T) {
	// A 10:1 split (split_ratio = 0.1) on 2024-06-01: bars before that date get
	// multiplied by 0.1 to be comparable with post-split bars.
	actions := []CorporateAction{
		{SymbolID: 1, ActionDate: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), SplitRatio: 0.1},
	}
	candles := []Candle{
		{Time: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: types.PriceFromFloat(1000.0)}, // pre-split
		{Time: time.Date(2024, 6, 3, 0, 0, 0, 0, time.UTC), Close: types.PriceFromFloat(100.0)},  // post-split
	}
	out := ApplyCorporateActions(candles, actions)

	if out[0].AdjustmentFactor != 0.1 {
		t.Errorf("pre-split bar: expected factor 0.1, got %f", out[0].AdjustmentFactor)
	}
	if out[1].AdjustmentFactor != 1.0 {
		t.Errorf("post-split bar: expected factor 1.0, got %f", out[1].AdjustmentFactor)
	}
}

func TestApplyCorporateActions_MultipleSplits(t *testing.T) {
	actions := []CorporateAction{
		{SymbolID: 1, ActionDate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), SplitRatio: 0.5},
		{SymbolID: 1, ActionDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), SplitRatio: 0.25},
	}
	candles := []Candle{
		{Time: time.Date(2019, 6, 1, 0, 0, 0, 0, time.UTC)}, // before both splits
		{Time: time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)}, // between splits
		{Time: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)}, // after both splits
	}
	out := ApplyCorporateActions(candles, actions)

	if out[0].AdjustmentFactor != 0.5*0.25 {
		t.Errorf("bar before both splits: expected 0.125, got %f", out[0].AdjustmentFactor)
	}
	if out[1].AdjustmentFactor != 0.25 {
		t.Errorf("bar between splits: expected 0.25, got %f", out[1].AdjustmentFactor)
	}
	if out[2].AdjustmentFactor != 1.0 {
		t.Errorf("bar after both splits: expected 1.0, got %f", out[2].AdjustmentFactor)
	}
}
