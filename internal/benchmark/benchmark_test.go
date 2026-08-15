package benchmark

import "testing"

func TestSpecHash_DeterministicAndDistinct(t *testing.T) {
	a := SpecHash("equity_index", "SPY")
	b := SpecHash("equity_index", "SPY")
	if a != b {
		t.Fatalf("hash must be deterministic: %s != %s", a, b)
	}
	if a == SpecHash("equity_index", "QQQ") {
		t.Fatal("different tickers must produce different hashes")
	}
	if a == SpecHash("risk_free", "SPY") {
		t.Fatal("different kinds must produce different hashes")
	}
	if len(a) != 64 {
		t.Fatalf("expected sha256 hex (64 chars), got %d", len(a))
	}
}
