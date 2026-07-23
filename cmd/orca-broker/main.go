package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/broker/paper"
	"github.com/lee-econ/orca-core/internal/risk"
	"github.com/lee-econ/orca-core/internal/types"
)

var adapter broker.Adapter

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("orca-broker starting on :9091...")

	switch mode := os.Getenv("ORCA_BROKER_MODE"); mode {
	case "alpaca":
		log.Fatal("alpaca mode requires ALPACA_API_KEY + ALPACA_API_SECRET env vars")
	case "ibkr":
		log.Fatal("ibkr mode not yet wired")
	default:
		adapter = paper.NewAdapter(100000.0)
		log.Println("broker mode: paper (100k initial)")
	}

	ks := risk.NewKillSwitch(nil)
	ks.OnTrigger(func(reason string, t time.Time) {
		log.Printf("KILL SWITCH TRIGGERED: %s at %s", reason, t.Format(time.RFC3339))
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/v1/place_order", handlePlaceOrder)
	mux.HandleFunc("/v1/cancel_order", handleCancelOrder)
	mux.HandleFunc("/v1/positions", handleGetPositions)
	mux.HandleFunc("/v1/account", handleGetAccount)

	srv := &http.Server{Addr: ":9091", Handler: mux}

	ctx, cancel := signal.NotifyContext(nil, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func() {
		<-ctx.Done()
		shutdownCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		srv.Shutdown(shutdownCtx)
	}()

	log.Println("orca-broker: HTTP server listening on :9091")
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("broker server error: %v", err)
	}
	log.Println("orca-broker shut down")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok", "mode": "paper"})
}

func handlePlaceOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "method not allowed")
		return
	}
	var req struct {
		Symbol      string  `json:"symbol"`
		Side        string  `json:"side"`
		OrderType   string  `json:"order_type"`
		Quantity    float64 `json:"quantity"`
		LimitPrice  float64 `json:"limit_price"`
		StopPrice   float64 `json:"stop_price"`
		TimeInForce string  `json:"time_in_force"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid body")
		return
	}

	resp, err := adapter.PlaceOrder(r.Context(), &broker.OrderRequest{
		Symbol:      req.Symbol,
		Side:        broker.OrderSide(req.Side),
		Type:        broker.OrderType(req.OrderType),
		Quantity:    req.Quantity,
		LimitPrice:  types.FromFloat64(req.LimitPrice),
		StopPrice:   types.FromFloat64(req.StopPrice),
		TimeInForce: broker.TimeInForce(req.TimeInForce),
	})
	if err != nil {
		writeJSON(w, map[string]interface{}{"status": "rejected", "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{
		"broker_order_id": resp.BrokerOrderID,
		"status":          resp.Status,
		"avg_fill_price":  resp.AvgFillPrice,
		"filled_qty":      resp.FilledQty,
	})
}

func handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderID string `json:"order_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid body")
		return
	}
	if err := adapter.CancelOrder(r.Context(), req.OrderID); err != nil {
		writeJSON(w, map[string]bool{"success": false})
		return
	}
	writeJSON(w, map[string]bool{"success": true})
}

func handleGetPositions(w http.ResponseWriter, r *http.Request) {
	pos, err := adapter.GetPositions(r.Context())
	if err != nil {
		writeJSON(w, map[string]interface{}{"positions": []interface{}{}})
		return
	}
	if pos == nil {
		pos = []broker.Position{}
	}
	writeJSON(w, map[string]interface{}{"positions": pos})
}

func handleGetAccount(w http.ResponseWriter, r *http.Request) {
	acct, err := adapter.GetAccount(r.Context())
	if err != nil {
		writeJSON(w, map[string]interface{}{"account": nil})
		return
	}
	writeJSON(w, map[string]interface{}{"account": acct})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	writeJSON(w, map[string]string{"error": msg})
}
