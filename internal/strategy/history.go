package strategy

import "github.com/lee-econ/orca-core/internal/types"

// linearWindow returns the last `n` values of a circular buffer in chronological
// order (oldest→newest). The indicator library (cinar) expects chronological
// slices, but the runners store circularly — passing the circular buffer
// directly (values[:count]) scrambles the window once it wraps. This helper is
// the single linearization (Rule 6).
func linearWindow(buf []float64, count, idx, size, n int) []float64 {
	if n <= 0 || count <= 0 || size <= 0 || len(buf) == 0 {
		return nil
	}
	if n > count {
		n = count
	}
	out := make([]float64, n)
	start := (idx - n) % size
	if start < 0 {
		start += size
	}
	for i := 0; i < n; i++ {
		out[i] = buf[(start+i)%size]
	}
	return out
}

// HistoryBuffer is a circular ring buffer with linearized access. It is the
// future home of the runners' price/high/low/volume history (currently still
// held as raw slices on BaseRunner); LastX(n) returns chronological windows so
// the circular-vs-linear indicator bug can never recur (Rule 6).
type HistoryBuffer struct {
	prices, highs, lows, vols []float64
	size, count, idx          int
}

func NewHistoryBuffer(size int) *HistoryBuffer {
	return &HistoryBuffer{
		prices: make([]float64, size),
		highs:  make([]float64, size),
		lows:   make([]float64, size),
		vols:   make([]float64, size),
		size:   size,
	}
}

func (h *HistoryBuffer) Push(p, high, low types.Price, vol float64) {
	if !finite(p.Float64()) {
		return
	}
	i := h.idx % h.size
	h.prices[i] = p.Float64()
	h.highs[i] = high.Float64()
	h.lows[i] = low.Float64()
	h.vols[i] = vol
	h.idx++
	if h.count < h.size {
		h.count++
	}
}

func (h *HistoryBuffer) Count() int { return h.count }

func (h *HistoryBuffer) LastPrices(n int) []float64 {
	return linearWindow(h.prices, h.count, h.idx, h.size, n)
}
func (h *HistoryBuffer) LastHighs(n int) []float64 {
	return linearWindow(h.highs, h.count, h.idx, h.size, n)
}
func (h *HistoryBuffer) LastLows(n int) []float64 {
	return linearWindow(h.lows, h.count, h.idx, h.size, n)
}
func (h *HistoryBuffer) LastVolumes(n int) []float64 {
	return linearWindow(h.vols, h.count, h.idx, h.size, n)
}

func (h *HistoryBuffer) Reset() {
	h.count, h.idx = 0, 0
	for i := range h.prices {
		h.prices[i], h.highs[i], h.lows[i], h.vols[i] = 0, 0, 0, 0
	}
}
