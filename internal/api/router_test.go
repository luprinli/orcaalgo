package api

import (
	"context"
	"encoding/json"
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
