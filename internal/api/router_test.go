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

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["symbol"] != "SPY" {
		t.Errorf("unexpected symbol: %v", resp["symbol"])
	}
	if resp["range"] != "1D" {
		t.Errorf("unexpected range: %v", resp["range"])
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

		if w.Code != http.StatusOK {
			t.Errorf("range %s: expected 200, got %d", r, w.Code)
		}
	}
}

func TestGetCandles_DefaultSymbol(t *testing.T) {
	srv := mockServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/candles", nil)
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["symbol"] != "SPY" {
		t.Errorf("expected default symbol SPY, got %v", resp["symbol"])
	}
}

func TestCandles_Synthetic_ReturnsProperStructure(t *testing.T) {
	// nil repo → synthetic candles path. Verifies JSON structure, not hardcoded values.
	srv := mockServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/candles?symbol=SPY&range=1D", nil)
	req.Header.Set("Authorization", authHeader())
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Symbol  string `json:"symbol"`
		Range   string `json:"range"`
		Candles []struct {
			Time   string  `json:"time"`
			Open   float64 `json:"open"`
			High   float64 `json:"high"`
			Low    float64 `json:"low"`
			Close  float64 `json:"close"`
			Volume float64 `json:"volume"`
		} `json:"candles"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Symbol != "SPY" {
		t.Errorf("symbol = %s, want SPY", resp.Symbol)
	}
	if len(resp.Candles) == 0 {
		t.Fatal("synthetic candles empty — should generate demo data")
	}
	if len(resp.Candles) > 400 {
		t.Errorf("got %d candles for 1D range, expected ≤400", len(resp.Candles))
	}
	// Verify every candle has valid structure
	for i, c := range resp.Candles {
		if c.Time == "" {
			t.Errorf("candle[%d].time is empty", i)
		}
		if c.High < c.Low {
			t.Errorf("candle[%d]: high (%f) < low (%f)", i, c.High, c.Low)
		}
		if c.Close <= 0 {
			t.Errorf("candle[%d]: close is zero or negative", i)
		}
		if c.Volume <= 0 {
			t.Errorf("candle[%d]: volume is zero or negative", i)
		}
		// Time should be valid RFC3339
		if _, err := time.Parse(time.RFC3339, c.Time); err != nil {
			t.Errorf("candle[%d]: invalid time format %q", i, c.Time)
		}
	}
}

func TestCandles_DifferentSymbols_DifferentPrices(t *testing.T) {
	srv := mockServer()
	symbols := []string{"SPY", "AAPL", "TSLA", "MSFT", "GOOGL", "AMZN"}
	prices := make(map[string]float64)
	for _, sym := range symbols {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/candles?symbol="+sym+"&range=1D", nil)
		req.Header.Set("Authorization", authHeader())
		srv.router.ServeHTTP(w, req)

		var resp struct{ Candles []struct{ Close float64 } `json:"candles"` }
		json.Unmarshal(w.Body.Bytes(), &resp)
		if len(resp.Candles) == 0 {
			t.Errorf("%s: no synthetic candles generated", sym)
			continue
		}
		mid := resp.Candles[len(resp.Candles)/2].Close
		prices[sym] = mid
	}
	// All prices must be in plausible range
	for sym, p := range prices {
		if p < 10 || p > 10000 {
			t.Errorf("%s: synthetic price %.2f outside plausible range", sym, p)
		}
	}
	// Different symbols must have different price ranges (deterministic hash)
	unique := make(map[float64]bool)
	for _, p := range prices {
		// Round to nearest 10 to allow for small jitter within same hash range
		unique[math.Round(p/10)*10] = true
	}
	if len(unique) < 3 {
		t.Errorf("only %d distinct price ranges across %d symbols — all too similar", len(unique), len(symbols))
	}
	t.Logf("Price midpoints: SPY=%.2f AAPL=%.2f TSLA=%.2f MSFT=%.2f GOOGL=%.2f AMZN=%.2f",
		prices["SPY"], prices["AAPL"], prices["TSLA"], prices["MSFT"], prices["GOOGL"], prices["AMZN"])
}

func TestCandles_AllRanges_ReturnProportionalData(t *testing.T) {
	srv := mockServer()
	ranges := []struct{ label string; minExpected int }{
		{"1D", 50},
		{"1W", 200},
		{"1M", 500},
	}
	for _, r := range ranges {
		t.Run(r.label, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/candles?symbol=SPY&range="+r.label, nil)
			req.Header.Set("Authorization", authHeader())
			srv.router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", w.Code)
			}
			var resp struct{ Candles []struct{} `json:"candles"` }
			json.Unmarshal(w.Body.Bytes(), &resp)
			if len(resp.Candles) < r.minExpected {
				t.Errorf("range %s: got %d candles, expected ≥%d", r.label, len(resp.Candles), r.minExpected)
			}
		})
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
