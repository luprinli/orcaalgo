package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/engine"
	"github.com/lee-econ/orca-core/internal/risk"
	"github.com/lee-econ/orca-core/internal/strategy"
)

var (
	tickCount atomic.Uint64
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("orca-engine starting on :8080...")

	eng := engine.NewLiveEngine()
	eng.RiskState = risk.NewGlobalRiskState()

	reg := strategy.GlobalRegistry()
	reg.Register(strategy.NewTrendRunner())
	reg.Register(strategy.NewOrbRunner())
	reg.Register(strategy.NewGridRunner())
	reg.Register(strategy.NewSessionScalpRunner())
	reg.Register(strategy.NewMeanReversionRunner(20, 2.0, 0.3, 200))
	reg.Register(strategy.NewMACrossoverRunner())
	reg.Register(strategy.NewRSI2MeanReversionRunner())
	reg.Register(strategy.NewDonchianBreakoutRunner())
	reg.Register(strategy.NewKeltnerMACDRunner())
	reg.Register(strategy.NewIchimokuRunner())

	brokerAddr := os.Getenv("ORCA_BROKER_ADDR")
	if brokerAddr == "" {
		brokerAddr = "http://localhost:9091"
	}
	log.Printf("engine: connected to broker at %s", brokerAddr)

	brokerClient := &http.Client{Timeout: 10 * time.Second}

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		simPrice := uint64(450_00000)
		simTs := time.Now().UnixNano()
		for range ticker.C {
			eng.ProcessTick(1, simPrice, 1000, simTs)
			simTs += 50_000_000
			tickCount.Add(1)
		}
	}()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "broker": brokerAddr, "ticks": tickCount.Load()})
	})

	api := router.Group("/api/v1")
	{
		api.POST("/orders", func(c *gin.Context) {
			var req struct {
				Symbol      string  `json:"symbol"`
				Side        string  `json:"side"`
				Type        string  `json:"type"`
				Quantity    float64 `json:"quantity"`
				LimitPrice  float64 `json:"limitPrice"`
				StopPrice   float64 `json:"stopPrice"`
				TimeInForce string  `json:"timeInForce"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			body, _ := json.Marshal(map[string]interface{}{
				"symbol":       req.Symbol,
				"side":         req.Side,
				"order_type":   req.Type,
				"quantity":     req.Quantity,
				"limit_price":  req.LimitPrice,
				"stop_price":   req.StopPrice,
				"time_in_force": req.TimeInForce,
			})
			resp, err := brokerClient.Post(brokerAddr+"/v1/place_order", "application/json", nil)
			if err != nil {
				c.JSON(502, gin.H{"error": "broker unreachable", "detail": err.Error()})
				return
			}
			defer resp.Body.Close()
			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			c.JSON(resp.StatusCode, result)
			_ = body
		})

		api.GET("/positions", func(c *gin.Context) {
			resp, err := brokerClient.Get(brokerAddr + "/v1/positions")
			if err != nil {
				c.JSON(502, gin.H{"positions": []interface{}{}})
				return
			}
			defer resp.Body.Close()
			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			c.JSON(resp.StatusCode, result)
		})

		api.GET("/accounts", func(c *gin.Context) {
			resp, err := brokerClient.Get(brokerAddr + "/v1/account")
			if err != nil {
				c.JSON(502, gin.H{"account": nil})
				return
			}
			defer resp.Body.Close()
			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			c.JSON(resp.StatusCode, result)
		})
	}

	ctx, cancel := signal.NotifyContext(nil, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srv := &http.Server{Addr: ":8080", Handler: router}
	go func() {
		<-ctx.Done()
		shutdownCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		srv.Shutdown(shutdownCtx)
	}()

	log.Println("orca-engine: HTTP API server listening on :8080")
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("engine server error: %v", err)
	}
	log.Printf("orca-engine shut down. Ticks processed: %d", tickCount.Load())
}
