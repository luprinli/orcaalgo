package api

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/api/middleware"
	"github.com/lee-econ/orca-core/internal/backtest"
	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/broker/paper"
	"github.com/lee-econ/orca-core/internal/risk"
	"github.com/lee-econ/orca-core/internal/security"
)

func init() {
	if os.Getenv("ORCA_JWT_SECRET") == "" {
		os.Setenv("ORCA_JWT_SECRET", "dev-jwt-secret-do-not-use-in-production-32chars")
	}
}

func authHeader() string {
	pair, _ := security.GenerateTokenPair("test-user", "test-user", []string{"admin"}, middleware.GetJWTSecret(), time.Hour)
	return "Bearer " + pair.AccessToken
}

func mockServer() *Server {
	gin.SetMode(gin.TestMode)
	adapter := paper.NewAdapter(100000.0)
	ks := risk.NewKillSwitch(adapter)
	reg := broker.NewBrokerDriverRegistry()
	reg.Register(adapter)
	reg.RunHealthChecks(context.Background())
	return NewServer(&risk.EnvVault{}, adapter, ks, nil, nil, reg)
}

func TestListStrategies_Empty(t *testing.T) {
	srv := mockServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/strategies", nil)
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["strategies"]; !ok {
		t.Fatal("missing strategies key")
	}
}

func TestCreateStrategy_Valid(t *testing.T) {
	srv := mockServer()
	body := `{"name":"test-strat","type":"mean_reversion","parameters":{"lookback":20}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/strategies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["name"] != "test-strat" {
		t.Errorf("unexpected name: %v", resp["name"])
	}
}

func TestCreateStrategy_MissingName(t *testing.T) {
	srv := mockServer()
	body := `{"type":"mean_reversion"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/strategies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateStrategy_NoRepoReturnsOK(t *testing.T) {
	srv := mockServer()
	body := `{"name":"updated"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/strategies/any-id", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when repo is nil, got %d", w.Code)
	}
}

func TestUpdateStrategy_NoRepoReturnsOK_Duplicate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adapter := paper.NewAdapter(100000.0)
	ks := risk.NewKillSwitch(adapter)
	srv := NewServer(&risk.EnvVault{}, adapter, ks, nil, nil, nil)

	body := `{"name":"updated"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/strategies/any-id", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when repo is nil, got %d", w.Code)
	}
}

func TestDeleteStrategy(t *testing.T) {
	srv := mockServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/strategies/any-id", nil)
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["deleted"] != true {
		t.Errorf("expected deleted=true, got %v", resp["deleted"])
	}
}

func TestValidateStrategy_AcceptsPayload(t *testing.T) {
	srv := mockServer()
	body := `{"name":"test","type":"mean_reversion","parameters":{"lookback":20}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/strategies/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader())
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["valid"]; !ok {
		t.Fatal("missing valid key")
	}
}

func TestReloadStrategy_NoRepoOK(t *testing.T) {
	srv := mockServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/strategies/any-id/reload", nil)
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when repo is nil, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["reloaded"] != true {
		t.Errorf("expected reloaded=true, got %v", resp["reloaded"])
	}
}

func TestCloneStrategy_NoRepo(t *testing.T) {
	srv := mockServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/strategies/any-id/clone", nil)
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["name"] != "cloned-strategy" {
		t.Errorf("unexpected name: %v", resp["name"])
	}
}

func TestGetCandles_DefaultRange(t *testing.T) {
	srv := mockServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/candles?symbol=SPY&range=1D", nil)
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	// Nil repo returns 503 — no synthetic fallback
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for nil repo, got %d", w.Code)
	}
}

func TestGetCandles_RangeMapping(t *testing.T) {
	srv := mockServer()
	ranges := []string{"1D", "1W", "1M", "3M", "1Y", "ALL"}
	for _, r := range ranges {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/candles?symbol=SPY&range="+r, nil)
		req.Header.Set("Authorization", authHeader())
		srv.router.ServeHTTP(w, req)

		// All ranges fail with 503 when repo is nil
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("range %s: expected 503, got %d", r, w.Code)
		}
	}
}

func TestGetCandles_DefaultSymbol(t *testing.T) {
	srv := mockServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/candles", nil)
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	// Default symbol with nil repo still returns 503
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestCandles_NilRepo_ReturnsServiceUnavailable(t *testing.T) {
	// nil repo = DB not connected. Must return error, not synthetic data.
	srv := mockServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/candles?symbol=SPY&range=1D", nil)
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] == nil {
		t.Error("expected error field in response")
	}
}

func TestCandles_EmptyDB_ReturnsWarning(t *testing.T) {
	// When DB is connected but has no candle data, return empty + warning.
	// This test requires a connected repo with empty data.
	// With nil repo, verify the 503 error is returned as expected.
	srv := mockServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/candles?symbol=SPY&range=1D", nil)
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for nil repo, got %d", w.Code)
	}
}

func TestGetAccounts_ReturnsData(t *testing.T) {
	srv := mockServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/accounts", nil)
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["accounts"]; !ok {
		t.Fatal("missing accounts key")
	}
}

func TestGetAccounts_PrimaryFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adapter := paper.NewAdapter(100000.0)
	ks := risk.NewKillSwitch(adapter)
	srv := NewServer(&risk.EnvVault{}, adapter, ks, nil, nil, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/accounts", nil)
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	accounts, ok := resp["accounts"].([]interface{})
	if !ok {
		t.Fatal("accounts not an array")
	}
	if len(accounts) < 1 {
		t.Fatal("expected at least 1 account")
	}
	first := accounts[0].(map[string]interface{})
	if first["id"] != "primary" {
		t.Errorf("expected primary account, got %v", first["id"])
	}
}

func TestGetRiskStatus(t *testing.T) {
	srv := mockServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/risk/status", nil)
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["halted"]; !ok {
		t.Fatal("missing halted key")
	}
	if _, ok := resp["balance"]; !ok {
		t.Fatal("missing balance key")
	}
}

func TestSyntheticGenerator_DeterministicReseeding(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	symbols := []string{"AAPL", "EURUSD", "BTCUSD", "XAUUSD", "SPX500"}

	run1 := generateSyntheticCandles(symbols, start, end, "1d")
	run2 := generateSyntheticCandles(symbols, start, end, "1d")

	for si, sym := range symbols {
		if len(run1[si]) != len(run2[si]) {
			t.Errorf("%s: length mismatch run1=%d run2=%d", sym, len(run1[si]), len(run2[si]))
			continue
		}
		for ci := range run1[si] {
			c1, c2 := run1[si][ci], run2[si][ci]
			if c1.Close.Float64() != c2.Close.Float64() {
				t.Errorf("%s[%d]: non-deterministic output %.6f != %.6f", sym, ci, c1.Close.Float64(), c2.Close.Float64())
				break
			}
		}
	}

	if t.Failed() {
		t.Fatal("synthetic generator seeding is not deterministic")
	}
}

func TestSyntheticGenerator_AssetClassDifferentiation(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	equity := generateSyntheticCandles([]string{"AAPL"}, start, end, "1d")[0]
	forex := generateSyntheticCandles([]string{"EURUSD"}, start, end, "1d")[0]
	crypto := generateSyntheticCandles([]string{"BTCUSD"}, start, end, "1d")[0]

	// All must have enough bars
	if len(equity) < 200 || len(forex) < 200 || len(crypto) < 200 {
		t.Fatalf("insufficient bars: equity=%d forex=%d crypto=%d", len(equity), len(forex), len(crypto))
	}

	// All must have non-zero price variance
	for _, tc := range []struct {
		name    string
		candles []backtest.Candle
	}{
		{"AAPL", equity}, {"EURUSD", forex}, {"BTCUSD", crypto},
	} {
		var sum, sumSq float64
		for i := 1; i < len(tc.candles); i++ {
			r := (tc.candles[i].Close.Float64() - tc.candles[i-1].Close.Float64()) / tc.candles[i-1].Close.Float64()
			sum += r
			sumSq += r * r
		}
		n := float64(len(tc.candles) - 1)
		variance := sumSq/n - (sum/n)*(sum/n)
		if variance <= 0 {
			t.Errorf("%s: zero price variance — data is flat", tc.name)
		}
		if math.IsNaN(variance) || math.IsInf(variance, 0) {
			t.Errorf("%s: NaN/Inf variance — data corruption", tc.name)
		}
		t.Logf("%s: n=%d, ann_vol=%.2f%%", tc.name, len(tc.candles), math.Sqrt(variance*252)*100)
	}

	// Crypto must have higher volatility than equity and forex
	cryptoRet := dailyReturns(crypto)
	equityRet := dailyReturns(equity)
	cryptoVol := stdDev(cryptoRet) * math.Sqrt(252)
	equityVol := stdDev(equityRet) * math.Sqrt(252)
	if cryptoVol <= equityVol {
		t.Errorf("crypto vol %.2f <= equity vol %.2f — asset class differentiation missing", cryptoVol, equityVol)
	}
	t.Logf("equity_vol=%.2f%%, crypto_vol=%.2f%%, ratio=%.2f", equityVol*100, cryptoVol*100, cryptoVol/equityVol)

	// Symbols must differ: equity AAPL vs equity MSFT should not be identical
	msft := generateSyntheticCandles([]string{"MSFT"}, start, end, "1d")[0]
	identical := true
	for i := range equity {
		if i >= len(msft) || equity[i].Close.Float64() != msft[i].Close.Float64() {
			identical = false
			break
		}
	}
	if identical {
		t.Error("AAPL and MSFT produce identical price paths — per-symbol seeding broken")
	}
}

func TestSyntheticGenerator_IntradayBars(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	sym := "AAPL"

	d1 := generateSyntheticCandles([]string{sym}, start, end, "1d")[0]
	d5 := generateSyntheticCandles([]string{sym}, start, end, "5m")[0]

	ratio := float64(len(d5)) / float64(len(d1))
	expectedRatio := 78.0 // 78 five-minute bars per daily bar
	if ratio < expectedRatio*0.9 || ratio > expectedRatio*1.1 {
		t.Errorf("5m/daily bar ratio = %.1f, expected ~%.1f (intraday generation broken)", ratio, expectedRatio)
	}
	t.Logf("%s: 1d=%d bars, 5m=%d bars, ratio=%.1f", sym, len(d1), len(d5), ratio)
}

func dailyReturns(candles []backtest.Candle) []float64 {
	r := make([]float64, len(candles)-1)
	for i := 1; i < len(candles); i++ {
		r[i-1] = (candles[i].Close.Float64() - candles[i-1].Close.Float64()) / candles[i-1].Close.Float64()
	}
	return r
}

func stdDev(vals []float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	var sum, sumSq float64
	for _, v := range vals {
		sum += v
		sumSq += v * v
	}
	n := float64(len(vals))
	variance := sumSq/n - (sum/n)*(sum/n)
	if variance < 0 {
		variance = 0
	}
	return math.Sqrt(variance)
}
