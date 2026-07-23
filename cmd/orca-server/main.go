package main

import (
	"bufio"
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/lee-econ/orca-core/internal/api"
	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/broker/alpaca"
	"github.com/lee-econ/orca-core/internal/broker/paper"
	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/ingest"
	"github.com/lee-econ/orca-core/internal/monitor"
	"github.com/lee-econ/orca-core/internal/risk"
	"github.com/lee-econ/orca-core/internal/scheduler"
	"github.com/lee-econ/orca-core/internal/security"
)

var (
	lastVIX       float64
	lastSentiment int
	viMux         sync.Mutex
	sentMux       sync.Mutex
)

func main() {
	loadDotEnv()
	runtime.GOMAXPROCS(runtime.NumCPU())
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	vault := &risk.EnvVault{}
	_ = vault.ValidateScope("alpaca")

	repo, err := db.NewRepository(db.DefaultConfig())
	if err != nil {
		log.Fatalf("FATAL: database unavailable: %v", err)
	}
	defer repo.Close()

	slog.Info("database connected", "host", db.DefaultConfig().Host)

	if err := repo.RunMigrations(context.Background()); err != nil {
		slog.Warn("migration check", "error", err)
	} else {
		slog.Info("migrations OK")
	}

	if err := repo.SeedSymbols(context.Background()); err != nil {
		slog.Warn("symbol seeding", "error", err)
	}

	seeder := db.NewSeeder(repo)
	report, vErr := seeder.VerifyIntegrity(context.Background())
	if vErr != nil {
		slog.Warn("database integrity check failed", "error", vErr)
	} else if !report.Passed {
		slog.Warn("integrity: some tables empty. Run POST /api/v1/admin/seed")
	} else {
		slog.Info("database integrity OK", "tables", len(report.Checks))
		for _, chk := range report.Checks {
			slog.Info("  table", "name", chk.Table, "count", chk.Count)
		}
	}

	dataMode := os.Getenv("ORCA_DATA_MODE")
	if dataMode == "" { dataMode = "stooq" }
	slog.Info("data mode", "mode", dataMode)
	if dataMode == "stooq" || dataMode == "mock" {
		if err := seeder.Run(context.Background(), false); err != nil {
			slog.Warn("auto-seed failed", "error", err)
		}
	}

	ringBuf := ingest.NewRingBuffer(ingest.MaxTickBuffer)
	metrics := monitor.NewMetrics()
	pipeline := monitor.NewDataPipeline(nil, ringBuf)

	brokerReg := broker.NewBrokerDriverRegistry()

	usePaper := os.Getenv("PAPER_TRADING") == "true"
	if usePaper {
		paperAdapter := paper.NewAdapter(100000.0)
		brokerReg.Register(paperAdapter)
		slog.Info("paper trading mode enabled", "balance", 100000.0)
	} else {
		alpacaAdapter, aErr := alpaca.NewAdapter()
		if aErr != nil {
			slog.Warn("alpaca not configured, falling back to paper", "error", aErr)
			paperAdapter := paper.NewAdapter(100000.0)
			brokerReg.Register(paperAdapter)
		} else {
			brokerReg.Register(alpacaAdapter)
			paperAdapter := paper.NewAdapter(100000.0)
			brokerReg.Register(paperAdapter)
			slog.Info("alpaca (primary) + paper (fallback) registered")
		}
	}

	// Run initial health checks to mark adapters ready
	brokerReg.RunHealthChecks(context.Background())

	// Resolve the primary adapter for the kill-switch (falls back through priority chain)
	adapter, resolveErr := brokerReg.Resolve(context.Background(), broker.CapPlaceOrder)
	if resolveErr != nil {
		slog.Error("no broker adapter available", "error", resolveErr)
		log.Fatalf("no broker adapter: %v", resolveErr)
	}

	ks := risk.NewKillSwitch(adapter)
	wsHub := monitor.NewWSHub()

	// Wire WS auth — validate JWT tokens on WebSocket upgrade
	jwtSecret := os.Getenv("ORCA_JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-jwt-secret-do-not-use-in-production-32chars"
	}
	wsHub.SetAuthValidator(func(token string) bool {
		if token == "" {
			return false
		}
		_, err := security.ValidateToken(token, []byte(jwtSecret))
		return err == nil
	})

	telegram := monitor.NewTelegramBot()
	pipeline.SetHub(wsHub)
	brokerReg.RunHealthChecks(context.Background())

	ks.OnTrigger(func(reason string, t time.Time) {
		telegram.Send("Critical", "KillSwitchTriggered", reason)
		metrics.SetKillSwitch(true)
		brokerReg.CancelAllOrders(context.Background())
		brokerReg.CloseAllPositions(context.Background())
		wsHub.Broadcast("risk", map[string]interface{}{"halted": true, "reason": reason, "time": t.Format(time.RFC3339), "daily_loss_used": 0.0, "drawdown_used": 0.0})
	})

	sched := scheduler.NewScheduler(vault, telegram)
	sched.RegisterKeyRotationJob()
	sched.RegisterDailyHealthJob()
	sched.RegisterDailyResetJob()
	sched.Start()

	server := api.NewServer(vault, adapter, ks, wsHub, repo, brokerReg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) { monitor.MetricsHandler().ServeHTTP(w, r) })
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); w.Write([]byte(`{"status":"ok"}`)) })

	go func() {
		slog.Info("metrics server starting", "addr", ":9090")
		if err := http.ListenAndServe(":9090", mux); err != nil { slog.Error("metrics error", "error", err) }
	}()
	go func() {
		if err := wsClientConnect(ctx, ringBuf, metrics); err != nil { slog.Error("ws connect error", "error", err) }
	}()
	go pipeline.Run(ctx)
	simFeed := monitor.NewSimulatedFeed(wsHub)
	go simFeed.Run(ctx)
	go func() {
		slog.Info("API server starting", "addr", ":8080")
		if err := server.Run(":8080"); err != nil { log.Fatalf("server error: %v", err) }
	}()
	go func() {
		vixClient := ingest.NewVIXClient()
		t := time.NewTicker(60 * time.Second); defer t.Stop()
		for {
			select {
			case <-ctx.Done(): return
			case <-t.C:
				vix, _, err := vixClient.FetchLatest(ctx, os.Getenv("POLYGON_API_KEY"))
				if err != nil {
					slog.Warn("vix fetch failed", "error", err)
					continue
				}
				viMux.Lock()
				lastVIX = vix
				viMux.Unlock()
			}
		}
	}()
	go func() {
		sentClient := ingest.NewSentimentClient()
		t := time.NewTicker(3600 * time.Second); defer t.Stop()
		for {
			select {
			case <-ctx.Done(): return
			case <-t.C:
				score, _, err := sentClient.Fetch(ctx)
				if err != nil {
					slog.Warn("sentiment fetch failed", "error", err)
					continue
				}
				sentMux.Lock()
				lastSentiment = score
				sentMux.Unlock()
			}
		}
	}()
	go func() {
		t := time.NewTicker(5 * time.Second); defer t.Stop()
		for {
			select {
			case <-ctx.Done(): return
			case <-t.C:
				regime := int8(0)
				confidence := 0.85
				viMux.Lock()
				vix := lastVIX
				viMux.Unlock()
				sentMux.Lock()
				sent := lastSentiment
				sentMux.Unlock()
			wsHub.Broadcast("risk", map[string]interface{}{
				"halted": ks.IsHalted(), "connections": len(wsHub.Clients()),
				"regime": regime, "confidence": confidence,
				"vix": vix, "sentiment": sent,
				"daily_loss_used": 0.0, "drawdown_used": 0.0,
				"daily_limit_pct": 5.0, "max_dd_pct": 10.0,
				"consistency_multiplier": 1.0,
				"tick_count": pipeline.GetTickCount(),
				"balance": 100000.0, "equity": 100000.0, "daily_pnl_pct": 0.0,
			})
			wsHub.Broadcast("ticks", map[string]interface{}{"tick_count": pipeline.GetTickCount(), "time": time.Now().Format(time.RFC3339)})
			pos, _ := adapter.GetPositions(ctx)
			if pos != nil {
				wsHub.Broadcast("positions", map[string]interface{}{"positions": pos})
			}
			orders, _ := adapter.GetPositions(ctx)
			if orders != nil {
				wsHub.Broadcast("orders", map[string]interface{}{"orders": orders})
			} else {
				wsHub.Broadcast("orders", map[string]interface{}{"orders": []interface{}{}})
			}
			wsHub.Broadcast("pnl_history", map[string]interface{}{
				"daily_pnl_pct": 0.0,
				"cumulative_pnl": 0.0,
				"equity": 100000.0,
				"time": time.Now().Format(time.RFC3339),
			})
			wsHub.Broadcast("strategy_metrics", map[string]interface{}{
				"strategies": []interface{}{},
				"updated_at": time.Now().Format(time.RFC3339),
			})
			}
		}
	}()

	slog.Info("Orca Core v0.5.0 starting", "api", "http://localhost:8080", "metrics", "http://localhost:9090/metrics", "health", "http://localhost:9090/healthz", "ws", "ws://localhost:8080/ws", "paper", usePaper, "db", true, "data_mode", dataMode)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	slog.Info("shutting down...")
	cancel()
	sched.Stop()
}

func wsClientConnect(ctx context.Context, ringBuf *ingest.RingBuffer, metrics *monitor.Metrics) error {
	wsURL := os.Getenv("ALPACA_DATA_STREAM_URL")
	if wsURL == "" {
		wsURL = "wss://stream.data.alpaca.markets/v2/sip"
	}
	wsClient := ingest.NewWSClient(wsURL, ringBuf)

	apiKey := os.Getenv("ALPACA_API_KEY")
	apiSecret := os.Getenv("ALPACA_API_SECRET")
	if apiKey != "" && apiSecret != "" {
		wsClient.SetAuth(apiKey, apiSecret)
	}

	wsClient.RegisterSymbol("AAPL", 1); wsClient.RegisterSymbol("MSFT", 2); wsClient.RegisterSymbol("SPY", 3)
	wsClient.Subscribe("AAPL", "MSFT", "SPY")
	if err := wsClient.Connect(ctx); err != nil { return err }
	wsClient.ReadLoop(ctx)
	return nil
}

func loadDotEnv() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			if os.Getenv(key) == "" {
				os.Setenv(key, val)
			}
		}
	}
}
