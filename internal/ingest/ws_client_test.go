package ingest

import (
	"testing"
)

func TestSymbolToID(t *testing.T) {
	tests := []struct {
		symbol string
	}{
		{"AAPL"}, {"MSFT"}, {"SPY"}, {"QQQ"},
		{"TSLA"}, {"NVDA"}, {"BTCUSD"},
	}
	for _, tt := range tests {
		id := hashSymbol(tt.symbol)
		if id == 0 && tt.symbol != "" {
			t.Errorf("hashSymbol(%q) returned 0", tt.symbol)
		}
		id2 := hashSymbol(tt.symbol)
		if id != id2 {
			t.Errorf("hashSymbol(%q) not deterministic: %d vs %d", tt.symbol, id, id2)
		}
		t.Logf("%s → %d", tt.symbol, id)
	}
}

func TestNextPowerOfTwo(t *testing.T) {
	tests := []struct {
		in, want int
	}{
		{0, 1}, {1, 1}, {2, 2}, {3, 4}, {4, 4},
		{5, 8}, {7, 8}, {8, 8}, {9, 16}, {15, 16},
		{16, 16}, {17, 32}, {31, 32}, {32, 32},
		{100, 128}, {255, 256}, {256, 256},
		{1000, 1024}, {4095, 4096}, {4096, 4096},
		{5000, 8192},
	}
	for _, tt := range tests {
		got := nextPowerOfTwo(tt.in)
		if got != tt.want {
			t.Errorf("nextPowerOfTwo(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
