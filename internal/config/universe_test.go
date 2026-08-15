package config

import "testing"

func TestSecondaryTickerPairs(t *testing.T) {
	cases := []struct {
		primary string
		want    string
	}{
		{"SPY", "QQQ"},
		{"QQQ", "SPY"},
		{"AAPL", "MSFT"},
		{"EURUSD", "GBPUSD"},
		{"GBPUSD", "EURUSD"},
		{"BTC-USD", "ETH-USD"},
		{"ETH-USD", "BTC-USD"},
		{"^DAX", "SPY"},
	}
	for _, c := range cases {
		if got := SecondaryTicker(c.primary); got != c.want {
			t.Errorf("SecondaryTicker(%q) = %q, want %q", c.primary, got, c.want)
		}
	}
}

func TestTickersUniverse(t *testing.T) {
	tickers, err := Tickers()
	if err != nil {
		t.Fatalf("Tickers() failed: %v", err)
	}
	if len(tickers) != 17 {
		t.Errorf("expected 17 tickers in universe, got %d", len(tickers))
	}
	seen := map[string]bool{}
	for _, tk := range tickers {
		seen[tk] = true
	}
	for _, want := range []string{"SPY", "EURUSD", "BTC-USD", "^DAX"} {
		if !seen[want] {
			t.Errorf("universe missing ticker %q", want)
		}
	}
}
