package alpaca

import "testing"

func TestNewAdapterWithCredentials_UsesPassedValues(t *testing.T) {
	a := NewAdapterWithCredentials("sk-key", "sk-secret", "https://paper-api.alpaca.markets")
	if a.apiKey != "sk-key" {
		t.Errorf("apiKey = %q, want sk-key", a.apiKey)
	}
	if a.apiSecret != "sk-secret" {
		t.Errorf("apiSecret = %q, want sk-secret", a.apiSecret)
	}
	if a.baseURL != "https://paper-api.alpaca.markets" {
		t.Errorf("baseURL = %q, want paper URL", a.baseURL)
	}
	if !a.paper {
		t.Error("paper should be true for paper URL")
	}
}

func TestNewAdapterWithCredentials_LiveDefault(t *testing.T) {
	a := NewAdapterWithCredentials("k", "s", "https://api.alpaca.markets")
	if a.paper {
		t.Error("paper should be false for live URL")
	}
}

func TestNewAdapterWithCredentials_EmptyBaseURLDefaultsLive(t *testing.T) {
	a := NewAdapterWithCredentials("k", "s", "")
	if a.baseURL != "https://api.alpaca.markets" {
		t.Errorf("baseURL = %q, want live URL", a.baseURL)
	}
	if a.paper {
		t.Error("paper should be false for default live URL")
	}
}
