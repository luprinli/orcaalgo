//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lee-econ/orca-core/internal/api"
	"github.com/lee-econ/orca-core/internal/broker/paper"
	"github.com/lee-econ/orca-core/internal/monitor"
	"github.com/lee-econ/orca-core/internal/risk"
)

func newTestEnv(t *testing.T) (*api.Server, *paper.PaperAdapter, *monitor.WSHub, *risk.KillSwitch) {
	t.Helper()

	os.Setenv("ORCA_JWT_SECRET", "test-jwt-secret-for-e2e-tests-only")
	os.Setenv("ORCA_ADMIN_PASSWORD", "test-admin-password")
	vault := &risk.EnvVault{}
	adapter := paper.NewAdapter(100000.0)
	ks := risk.NewKillSwitch(adapter)
	hub := monitor.NewWSHub()

	server := api.NewServer(vault, adapter, ks, hub, nil, nil)

	return server, adapter, hub, ks
}

func request(method, path, body string, headers map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
	if body != "" && headers == nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func TestE2E_HealthCheck(t *testing.T) {
	server, _, _, _ := newTestEnv(t)

	req := request("GET", "/api/v1/risk/status", "", nil)
	w := httptest.NewRecorder()
	server.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestE2E_KillSwitch(t *testing.T) {
	server, _, _, ks := newTestEnv(t)

	req := request("POST", "/api/v1/emergency/stop", "", map[string]string{"X-2FA-Token": "123456"})
	w := httptest.NewRecorder()
	server.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !ks.IsHalted() {
		t.Error("expected kill switch halted after stop")
	}

	req2 := request("POST", "/api/v1/emergency/resume", "", map[string]string{"X-2FA-Token": "123456"})
	w2 := httptest.NewRecorder()
	server.Engine().ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w2.Code)
	}
	if ks.IsHalted() {
		t.Error("expected kill switch resumed")
	}
}

func TestE2E_SymbolsCRUD(t *testing.T) {
	server, _, _, _ := newTestEnv(t)

	req := request("GET", "/api/v1/symbols", "", nil)
	w := httptest.NewRecorder()
	server.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if _, ok := body["symbols"]; !ok {
		t.Error("expected symbols array")
	}

	createBody := `{"ticker":"TEST","exchange":"NYSE","asset_type":"equity","tick_size":0.01}`
	req2 := request("POST", "/api/v1/symbols", createBody, nil)
	w2 := httptest.NewRecorder()
	server.Engine().ServeHTTP(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestE2E_ProvidersCRUD(t *testing.T) {
	server, _, _, _ := newTestEnv(t)

	req := request("GET", "/api/v1/providers", "", nil)
	w := httptest.NewRecorder()
	server.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	createBody := `{"name":"test-provider","type":"broker","driver":"alpaca"}`
	req2 := request("POST", "/api/v1/providers", createBody, nil)
	w2 := httptest.NewRecorder()
	server.Engine().ServeHTTP(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestE2E_OrderLifecycle(t *testing.T) {
	server, adapter, _, _ := newTestEnv(t)

	orderBody := `{"symbol":"SPY","side":"BUY","type":"LIMIT","quantity":100,"limit_price":450.00,"time_in_force":"DAY"}`
	req := request("POST", "/api/v1/orders", orderBody, nil)
	w := httptest.NewRecorder()
	server.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var orderResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &orderResp)
	orderID, _ := orderResp["id"].(string)
	if orderID == "" {
		t.Error("expected order ID")
	}

	positions, _ := adapter.GetPositions(context.Background())
	if len(positions) != 1 {
		t.Errorf("expected 1 position, got %d", len(positions))
	}
	if len(positions) > 0 && positions[0].Symbol != "SPY" {
		t.Errorf("expected SPY, got %s", positions[0].Symbol)
	}

	cancelReq := httptest.NewRequest("DELETE", "/api/v1/orders/"+orderID, nil)
	cancelW := httptest.NewRecorder()
	server.Engine().ServeHTTP(cancelW, cancelReq)

	if cancelW.Code != http.StatusOK {
		t.Errorf("expected 200 from cancel, got %d", cancelW.Code)
	}
}

func TestE2E_WebSocketConnect(t *testing.T) {
	_, _, hub, _ := newTestEnv(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWS(w, r)
	}))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws connect failed: %v", err)
	}
	defer conn.Close()

	hub.Broadcast("test", map[string]interface{}{"msg": "hello"})

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ws read failed: %v", err)
	}

	var wsMsg struct {
		Channel string                 `json:"channel"`
		Data    map[string]interface{} `json:"data"`
	}
	json.Unmarshal(msg, &wsMsg)

	if wsMsg.Channel != "test" {
		t.Errorf("expected channel 'test', got '%s'", wsMsg.Channel)
	}
}

func TestE2E_RiskStatusBroadcast(t *testing.T) {
	_, _, hub, _ := newTestEnv(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWS(w, r)
	}))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws connect failed: %v", err)
	}
	defer conn.Close()

	hub.Broadcast("risk", map[string]interface{}{
		"halted":           false,
		"daily_loss_used":  1.5,
		"drawdown_used":    3.2,
		"daily_limit_pct":  5.0,
		"max_dd_pct":       10.0,
	})

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ws read failed: %v", err)
	}

	var wsMsg struct {
		Channel string                 `json:"channel"`
		Data    map[string]interface{} `json:"data"`
	}
	json.Unmarshal(msg, &wsMsg)

	if wsMsg.Channel != "risk" {
		t.Errorf("expected channel 'risk', got '%s'", wsMsg.Channel)
	}
}

func TestE2E_StrategiesEndpoint(t *testing.T) {
	server, _, _, _ := newTestEnv(t)

	req := request("GET", "/api/v1/strategies", "", nil)
	w := httptest.NewRecorder()
	server.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestE2E_BacktestSubmission(t *testing.T) {
	server, _, _, _ := newTestEnv(t)

	backtestBody := `{"strategy_id":"1","symbols":["SPY"],"start_date":"2024-01-01","end_date":"2024-01-31","initial_capital":100000}`
	req := request("POST", "/api/v1/backtests", backtestBody, nil)
	w := httptest.NewRecorder()
	server.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}
}

func TestE2E_CredentialFlow(t *testing.T) {
	server, _, _, _ := newTestEnv(t)

	req := request("GET", "/api/v1/credentials", "", nil)
	w := httptest.NewRecorder()
	server.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	storeBody := `{"provider_id":"test-id","key_label":"paper_key","api_key":"test_key_12345","api_secret":"test_secret_67890"}`
	req2 := request("POST", "/api/v1/credentials", storeBody, nil)
	w2 := httptest.NewRecorder()
	server.Engine().ServeHTTP(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestE2E_PaperTradeEndToEnd(t *testing.T) {
	server, adapter, _, _ := newTestEnv(t)

	orderBody := `{"symbol":"AAPL","side":"BUY","type":"MARKET","quantity":50}`
	req := request("POST", "/api/v1/orders", orderBody, nil)
	w := httptest.NewRecorder()
	server.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("place order: %d %s", w.Code, w.Body.String())
	}

	positions, err := adapter.GetPositions(context.Background())
	if err != nil {
		t.Fatalf("get positions: %v", err)
	}
	if len(positions) == 0 {
		t.Fatal("expected positions after order")
	}
	if positions[0].Symbol != "AAPL" {
		t.Errorf("expected AAPL, got %s", positions[0].Symbol)
	}
	if positions[0].Quantity != 50 {
		t.Errorf("expected qty 50, got %f", positions[0].Quantity)
	}
}
