package strategy

import (
	"testing"

	"github.com/lee-econ/orca-core/internal/types"
)

// Tests for the circular-buffer linearization (Rule 6): the indicator library
// receives chronological windows, never a scrambled circular slice.

func TestLinearWindow_CircularOrder(t *testing.T) {
	buf := make([]float64, 4)
	idx, count := 0, 0
	for _, v := range []float64{1, 2, 3, 4, 5} {
		buf[idx%4] = v
		idx++
		if count < 4 {
			count++
		}
	}
	got := linearWindow(buf, count, idx, 4, 4)
	want := []float64{2, 3, 4, 5}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("linearWindow[%d] = %v, want %v (full=%v)", i, got[i], want[i], got)
		}
	}
}

func TestLinearWindow_ClampsToCount(t *testing.T) {
	buf := []float64{10, 20, 30, 40}
	got := linearWindow(buf, 3, 3, 4, 10) // request 10, only 3 valid
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
}

func TestHistoryBuffer_LastPrices(t *testing.T) {
	h := NewHistoryBuffer(4)
	for _, v := range []float64{1, 2, 3, 4, 5} {
		h.Push(types.PriceFromFloat(v), types.PriceFromFloat(v), types.PriceFromFloat(v), 0)
	}
	if h.Count() != 4 {
		t.Fatalf("Count = %d, want 4", h.Count())
	}
	got := h.LastPrices(4)
	want := []float64{2, 3, 4, 5}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("LastPrices[%d] = %v, want %v (full=%v)", i, got[i], want[i], got)
		}
	}
}

func TestBaseRunner_LinearPrices(t *testing.T) {
	b := NewBaseRunner(4)
	for _, v := range []float64{1, 2, 3, 4, 5} {
		b.PushPrice(types.PriceFromFloat(v), types.PriceFromFloat(v), types.PriceFromFloat(v), 0)
	}
	got := b.LinearPrices(4)
	want := []float64{2, 3, 4, 5}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("LinearPrices[%d] = %v, want %v (full=%v)", i, got[i], want[i], got)
		}
	}
}
