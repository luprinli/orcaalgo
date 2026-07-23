package ibkr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/types"
)

func newMockIBServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path

		switch {
		case strings.Contains(path, "/iserver/auth/ssodh/init"):
			json.NewEncoder(w).Encode(map[string]interface{}{"authenticated": true})

		case strings.Contains(path, "/iserver/account/positions"):
			json.NewEncoder(w).Encode([]ibkrPosition{
				{AcctID: "DU000000", ContractDesc: "SPY", Position: 100, MktPrice: 580.0, MktValue: 58000, AvgCost: 575.0, UnrealizedPnl: 500},
				{AcctID: "DU000000", ContractDesc: "QQQ", Position: -50, MktPrice: 420.0, MktValue: -21000, AvgCost: 415.0, UnrealizedPnl: -250},
			})

		case strings.Contains(path, "/iserver/account/pnl/partitioned"):
			json.NewEncoder(w).Encode(ibkrPnlResponse{
				Total: struct {
					ROW []struct {
						DailyPnL       float64 `json:"dailyPnL"`
						UnrealizedPnL  float64 `json:"unrealizedPnL"`
						RealizedPnL    float64 `json:"realizedPnL"`
						NetLiquidation float64 `json:"netLiquidation"`
					} `json:"ROW"`
				}{ROW: []struct {
					DailyPnL       float64 `json:"dailyPnL"`
					UnrealizedPnL  float64 `json:"unrealizedPnL"`
					RealizedPnL    float64 `json:"realizedPnL"`
					NetLiquidation float64 `json:"netLiquidation"`
				}{{DailyPnL: 2500.0, UnrealizedPnL: 5000.0, RealizedPnL: 1200.0, NetLiquidation: 105000.0}}},
			})

		case strings.Contains(path, "/orders") && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))

		case strings.Contains(path, "/order/") && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))

		case strings.Contains(path, "/orders") && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode([]ibkrOrderResponse{
				{OrderID: "ibkr-order-1", OrderStatus: "Submitted", LocalOrderID: "local-1"},
			})

		case strings.Contains(path, "/iserver/auth/status"):
			json.NewEncoder(w).Encode(map[string]interface{}{"authenticated": true, "connected": true})

		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"unknown endpoint"}`))
		}
	})

	server := httptest.NewServer(handler)
	return server, server.URL
}

func adapterWithServer(t *testing.T, server *httptest.Server, baseURL string, accountID string) *IBKRAdapter {
	t.Helper()
	cfg := IBKRConfig{Host: "127.0.0.1", Port: serverPort(server.URL), AccountID: accountID}
	adapter, err := NewAdapterWithConfig(cfg)
	if err != nil {
		t.Fatalf("NewAdapterWithConfig: %v", err)
	}
	adapter.client.baseURL = baseURL
	return adapter
}

func TestIBKRAdapter_HealthCheck(t *testing.T) {
	server, baseURL := newMockIBServer(t)
	defer server.Close()
	adapter := adapterWithServer(t, server, baseURL, "DU000000")

	if err := adapter.HealthCheck(context.Background()); err != nil {
		t.Errorf("HealthCheck should pass, got: %v", err)
	}
}

func TestIBKRAdapter_HealthCheck_NoClient(t *testing.T) {
	adapter := &IBKRAdapter{}
	if err := adapter.HealthCheck(context.Background()); err == nil {
		t.Error("HealthCheck should fail when client is nil")
	}
}

func TestIBKRAdapter_PlaceLimitOrder(t *testing.T) {
	server, baseURL := newMockIBServer(t)
	defer server.Close()
	adapter := adapterWithServer(t, server, baseURL, "DU000000")

	ctx := context.Background()
	resp, err := adapter.PlaceOrder(ctx, &broker.OrderRequest{
		Symbol: "SPY", Side: broker.Buy, Type: broker.Limit,
		Quantity: 100, LimitPrice: types.FromFloat64(580.0), TimeInForce: broker.Day,
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if resp.BrokerOrderID == "" {
		t.Error("expected non-empty BrokerOrderID")
	}
}

func TestIBKRAdapter_CancelOrder(t *testing.T) {
	server, baseURL := newMockIBServer(t)
	defer server.Close()
	adapter := adapterWithServer(t, server, baseURL, "DU000000")

	if err := adapter.CancelOrder(context.Background(), "ibkr-order-1"); err != nil {
		t.Errorf("CancelOrder: %v", err)
	}
}

func TestIBKRAdapter_CancelAllOrders(t *testing.T) {
	server, baseURL := newMockIBServer(t)
	defer server.Close()
	adapter := adapterWithServer(t, server, baseURL, "DU000000")

	if err := adapter.CancelAllOrders(context.Background()); err != nil {
		t.Errorf("CancelAllOrders: %v", err)
	}
}

func TestIBKRAdapter_GetPositions(t *testing.T) {
	server, baseURL := newMockIBServer(t)
	defer server.Close()
	adapter := adapterWithServer(t, server, baseURL, "DU000000")

	positions, err := adapter.GetPositions(context.Background())
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if len(positions) != 2 {
		t.Errorf("expected 2 positions, got %d", len(positions))
	}
	if positions[0].Symbol != "SPY" {
		t.Errorf("position[0].Symbol = %q, want SPY", positions[0].Symbol)
	}
}

func TestIBKRAdapter_GetAccount(t *testing.T) {
	server, baseURL := newMockIBServer(t)
	defer server.Close()
	adapter := adapterWithServer(t, server, baseURL, "DU000000")

	acct, err := adapter.GetAccount(context.Background())
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if acct.ID != "DU000000" {
		t.Errorf("ID = %q, want DU000000", acct.ID)
	}
	if acct.Status != "ACTIVE" {
		t.Errorf("Status = %q, want ACTIVE", acct.Status)
	}
}

func TestIBKRAdapter_ValidateCredentials(t *testing.T) {
	server, baseURL := newMockIBServer(t)
	defer server.Close()
	adapter := adapterWithServer(t, server, baseURL, "DU000000")

	if err := adapter.ValidateCredentials(context.Background()); err != nil {
		t.Errorf("ValidateCredentials: %v", err)
	}
}

func TestIBKRAdapter_Manifest(t *testing.T) {
	adapter, _ := NewAdapter()
	m := adapter.Manifest()
	if m.ID != "ibkr" {
		t.Errorf("ID = %q, want ibkr", m.ID)
	}
	if m.Priority != 2 {
		t.Errorf("Priority = %d, want 2", m.Priority)
	}
}

func TestIBKRAdapter_MultiAccountIsolation(t *testing.T) {
	server1, baseURL1 := newMockIBServer(t)
	defer server1.Close()
	server2, baseURL2 := newMockIBServer(t)
	defer server2.Close()

	adapter1 := adapterWithServer(t, server1, baseURL1, "ACC-1")
	adapter2 := adapterWithServer(t, server2, baseURL2, "ACC-2")

	ctx := context.Background()
	acct1, _ := adapter1.GetAccount(ctx)
	acct2, _ := adapter2.GetAccount(ctx)

	if acct1.ID != "ACC-1" {
		t.Errorf("acct1 ID = %q, want ACC-1", acct1.ID)
	}
	if acct2.ID != "ACC-2" {
		t.Errorf("acct2 ID = %q, want ACC-2", acct2.ID)
	}
}

func TestIBKRAdapter_TransactionalAdapter(t *testing.T) {
	adapter, _ := NewAdapter()
	if !adapter.IsTransactional() {
		t.Error("IBKR adapter should be transactional")
	}
}

func serverPort(url string) int {
	parts := strings.Split(strings.TrimPrefix(url, "http://"), ":")
	if len(parts) >= 2 {
		port := 0
		for _, c := range parts[1] {
			if c >= '0' && c <= '9' {
				port = port*10 + int(c-'0')
			} else {
				break
			}
		}
		return port
	}
	return 5000
}
