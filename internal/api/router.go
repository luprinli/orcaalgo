package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/api/middleware"
	"github.com/lee-econ/orca-core/internal/backtest"
	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/email"
	"github.com/lee-econ/orca-core/internal/engine"
	"github.com/lee-econ/orca-core/internal/llm"
	"github.com/lee-econ/orca-core/internal/monitor"
	"github.com/lee-econ/orca-core/internal/notify"
	"github.com/lee-econ/orca-core/internal/propfirm"
	"github.com/lee-econ/orca-core/internal/reactive"
	"github.com/lee-econ/orca-core/internal/risk"
	"github.com/lee-econ/orca-core/internal/security"
	"github.com/lee-econ/orca-core/internal/types"
	"github.com/lee-econ/orca-core/internal/universe"
	"github.com/lee-econ/orca-core/internal/version"
)

type Server struct {
	router         *gin.Engine
	killSwitch     *risk.KillSwitch
	wsHub          *monitor.WSHub
	vault          risk.VaultProvider
	adapter        broker.Adapter
	brokerRegistry *broker.BrokerDriverRegistry
	accountManager *broker.AccountManager
	multiCapitalPool *risk.MultiAccountCapitalPool
	repo           *db.Repository
	backtestEngine *backtest.Engine
	liveEngine     *engine.LiveEngine
	eventBus       *reactive.EventBus
	emailService   email.EmailService
	notifyManager  *notify.Manager

	providerHandler     *ProviderHandler
	symbolHandler       *SymbolHandler
	credentialHandler   *CredentialHandler
	authHandler         *AuthHandler
	webhookHandler      *WebhookHandler
	adminHandler        *AdminHandler
	propFirmHandler     *PropFirmHandler
	propFirmManager     *propfirm.Manager
	dataSourceHandler   *DataSourceHandler
	settingsHandler     *SettingsHandler
	notificationHandler   *NotificationHandler
	auditHandler          *AuditHandler
	backtestHistoryHandler *BacktestHistoryHandler
	universeHandler        *UniverseHandler
	universeManager        *universe.UniverseManager
	reconciliationHandler  *ReconciliationHandler
	modelHandler           *ModelHandler
	indicatorHandler       *IndicatorHandler
	progressStore          *ProgressStore
	paramVersionHandler    *ParamVersionHandler
	orchestratorHandler    *OrchestratorHandler
	strategyStatusHandler  *StrategyStatusHandler
	monitoringHandler      *MonitoringHandler
	dataSource            string
}

func NewServer(vault risk.VaultProvider, adapter broker.Adapter, ks *risk.KillSwitch, hub *monitor.WSHub, repo *db.Repository, brokerReg *broker.BrokerDriverRegistry) *Server {
	s := &Server{
		router:         gin.Default(),
		killSwitch:     ks,
		wsHub:          hub,
		vault:          vault,
		adapter:        adapter,
		brokerRegistry: brokerReg,
		repo:           repo,
		eventBus:       reactive.NewEventBus(),
		progressStore:  NewProgressStoreWithHub(hub),
	}
	s.eventBus.Subscribe("signal_lifecycle", func(signal *reactive.Signal) {
		if s.wsHub != nil {
			s.wsHub.Broadcast("signal_lifecycle", signal)
		}
	})
	if repo != nil {
		s.providerHandler = NewProviderHandler(repo, vault, hub, brokerReg)
		s.symbolHandler = NewSymbolHandler(repo)
		s.credentialHandler = NewCredentialHandler(repo, vault)
		s.webhookHandler = NewWebhookHandlerWithAdapter(adapter)
		s.adminHandler = NewAdminHandler(repo)
		s.backtestEngine = backtest.NewEngine(&backtestRepoAdapter{repo})
		s.backtestEngine.WirePipeline()
		s.authHandler = NewAuthHandlerWithRepo(repo, nil, "")
		s.backtestHistoryHandler = NewBacktestHistoryHandler(repo)
		s.universeManager = universe.NewUniverseManager(repo, hub, slog.Default())
		s.universeHandler = NewUniverseHandler(repo, s.universeManager)
		s.reconciliationHandler = NewReconciliationHandler(repo)
		s.modelHandler = NewModelHandler(repo)
		s.notificationHandler = NewNotificationHandler(repo, nil)
		s.dataSource = os.Getenv("ORCA_DATA_MODE")
		if s.dataSource == "" {
			s.dataSource = "stooq"
		}
	}
	s.registerRoutes()
	return s
}

func NewServerWithServices(vault risk.VaultProvider, adapter broker.Adapter, ks *risk.KillSwitch, hub *monitor.WSHub, repo *db.Repository, brokerReg *broker.BrokerDriverRegistry, emailSvc email.EmailService, notifyMgr *notify.Manager, frontendURL string) *Server {
	s := NewServer(vault, adapter, ks, hub, repo, brokerReg)
	s.emailService = emailSvc
	s.notifyManager = notifyMgr

	if notifyMgr != nil && hub != nil {
		notifyMgr.SetPushHub(hub)
	}

	if repo != nil {
		s.authHandler = NewAuthHandlerWithRepo(repo, emailSvc, frontendURL)
		s.notificationHandler = NewNotificationHandler(repo, notifyMgr)
		s.registerRoutes()
	}

	return s
}

func (s *Server) registerRoutes() {
	s.router.Use(middleware.CORSMiddleware())
	v1 := s.router.Group("/api/v1")

	// Public endpoints — no auth required
	v1.GET("/backtests/health", s.getBacktestHealth)
	v1.GET("/system/health", s.getSystemHealth)

	// Protected endpoints — JWT required
	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/strategies", s.listStrategies)
		protected.GET("/strategies/:id", s.getStrategy)
		protected.POST("/strategies", s.createStrategy)
		protected.PUT("/strategies/:id", s.updateStrategy)
		protected.DELETE("/strategies/:id", s.deleteStrategy)
		protected.POST("/strategies/validate", s.validateStrategy)
		protected.POST("/strategies/:id/reload", s.reloadStrategy)
		protected.POST("/strategies/:id/clone", s.cloneStrategy)

		protected.GET("/candles", s.getCandles)
		protected.GET("/brokers", s.listBrokers)
		protected.GET("/accounts", s.getAccounts)
		protected.POST("/accounts", s.createAccount)
		protected.DELETE("/accounts/:id", s.deleteAccount)
		protected.POST("/accounts/:id/default", s.setDefaultAccount)
		protected.GET("/signals", s.getSignals)

		protected.POST("/backtests", s.submitBacktest)
		protected.POST("/backtests/pipeline", s.submitPipelineRun)
		protected.POST("/backtests/matrix", s.submitMatrix)
		protected.GET("/backtests/matrix/:id", s.getMatrixStatus)
		protected.GET("/backtests/matrix/:id/results", s.getMatrixResults)
		protected.POST("/backtests/matrix/:id/cancel", s.cancelMatrix)
		protected.GET("/backtests/:id/live-comparison", s.liveComparison)
		protected.GET("/backtests/:id/regime-stats", s.getBacktestRegimeStats)
		protected.GET("/backtests/:id/progress", s.getBacktestProgress)
		// §TODO: Consolidate optimization into the light optimizer within RunMatrixConcurrent.
		// The following standalone optimize endpoints are not yet implemented:
		// protected.POST("/optimize", s.submitOptimization)
		// protected.POST("/optimizations", s.submitBacktestWithOptimization)
		// protected.GET("/optimizations", s.listOptimizationRuns)
		// protected.GET("/optimizations/:id", s.getOptimizationStatus)
		// protected.GET("/optimizations/:id/results", s.getOptimizationResults)

		protected.GET("/backtests/:id/metrics", s.getBacktestMetrics)
		protected.GET("/backtests/:id/equity", s.getBacktestEquity)
		protected.GET("/backtests/:id/trades", s.getBacktestTrades)
		protected.GET("/backtests/:id/daily-returns", s.getBacktestDailyReturns)
		protected.GET("/backtests/:id/monthly-returns", s.getBacktestMonthlyReturns)
		protected.GET("/backtests/:id/optimization", s.getBacktestOptimization)
		protected.GET("/backtests/:id/walk-forward", s.getBacktestWalkForward)

		protected.GET("/live/metrics", s.getLiveMetrics)
		protected.GET("/live/equity", s.getLiveEquity)
		protected.GET("/live/trades", s.getLiveTrades)
		protected.GET("/live/daily-returns", s.getLiveDailyReturns)
		protected.GET("/live/rolling-sharpe", s.getLiveRollingSharpe)

		protected.GET("/positions", s.getPositions)
		protected.GET("/risk/status", s.getRiskStatus)

		protected.POST("/emergency/stop", middleware.TwoFAMiddleware(s.twoFAValidator()), s.triggerEmergencyStop)
		protected.POST("/emergency/resume", middleware.TwoFAMiddleware(s.twoFAValidator()), s.resumeTrading)

		protected.GET("/monitor/regime-history", s.getRegimeHistory)

		protected.POST("/orders", s.placeOrder)
		protected.DELETE("/orders/:id", s.cancelOrder)
		protected.DELETE("/orders", s.cancelAllOrders)
		protected.GET("/orders", s.listOrders)

		protected.POST("/llm/test", s.testLLM)
	}

	if s.providerHandler != nil {
		s.providerHandler.RegisterRoutes(v1)
	}
	if s.symbolHandler != nil {
		s.symbolHandler.RegisterRoutes(v1)
	}
	if s.credentialHandler != nil {
		s.credentialHandler.RegisterRoutes(v1)
	}
	if s.authHandler != nil {
		s.authHandler.RegisterRoutes(v1)
	}
	if s.webhookHandler != nil {
		s.webhookHandler.RegisterRoutes(v1)
	}
	if s.adminHandler != nil {
		s.adminHandler.RegisterRoutes(v1)
	}
	if s.backtestHistoryHandler != nil {
		s.backtestHistoryHandler.RegisterRoutes(v1)
	}
	if s.universeHandler != nil {
		s.universeHandler.RegisterRoutes(v1)
	}
	if s.notificationHandler != nil {
		s.notificationHandler.RegisterRoutes(v1)
	}
	if s.auditHandler != nil {
		s.auditHandler.RegisterRoutes(v1)
	}
	if s.propFirmHandler == nil && s.repo != nil {
		s.propFirmManager = propfirm.NewManager()
		s.propFirmHandler = NewPropFirmHandler(s.propFirmManager)
	}
	if s.propFirmHandler != nil {
		s.propFirmHandler.RegisterRoutes(v1)
	}
	if s.dataSourceHandler != nil {
		s.dataSourceHandler.RegisterRoutes(v1)
	}
	if s.settingsHandler == nil && s.repo != nil {
		s.settingsHandler = NewSettingsHandler(s.repo)
	}
	if s.settingsHandler != nil {
		s.settingsHandler.RegisterRoutes(v1)
	}
	if s.reconciliationHandler != nil {
		s.reconciliationHandler.RegisterRoutes(v1)
	}
	if s.modelHandler != nil {
		s.modelHandler.RegisterRoutes(v1)
	}
	if s.indicatorHandler == nil && s.wsHub != nil {
		s.indicatorHandler = NewIndicatorHandler(s.wsHub)
	}
	if s.indicatorHandler != nil {
		s.indicatorHandler.RegisterRoutes(v1)
	}
	if s.monitoringHandler == nil {
		s.monitoringHandler = NewMonitoringHandler("http://localhost:9090", "http://localhost:9093")
	}
	s.monitoringHandler.RegisterRoutes(v1)

	if s.repo != nil {
		s.paramVersionHandler = NewParamVersionHandler(s.repo)
		s.paramVersionHandler.RegisterRoutes(v1)

		s.orchestratorHandler = NewOrchestratorHandler(s.repo, &backtestRepoAdapter{repo: s.repo})
		s.orchestratorHandler.RegisterRoutes(v1)

		s.strategyStatusHandler = NewStrategyStatusHandler(s.repo)
		s.strategyStatusHandler.RegisterRoutes(v1)
	}

	s.router.GET("/ws", func(c *gin.Context) {
		token := c.Query("token")
		if token != "" {
			_, err := security.ValidateToken(token, middleware.GetJWTSecret())
			if err != nil {
				monitor.RecordWSAuthFailure()
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
				return
			}
		}
		s.wsHub.ServeWS(c.Writer, c.Request)
	})
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

func (s *Server) Engine() *gin.Engine {
	return s.router
}

func (s *Server) SetPropFirmManager(mgr *propfirm.Manager) {
	s.propFirmManager = mgr
	s.propFirmHandler = NewPropFirmHandler(mgr)
}

func (s *Server) SetAccountManager(am *broker.AccountManager) {
	s.accountManager = am
}

func (s *Server) SetMultiCapitalPool(mcp *risk.MultiAccountCapitalPool) {
	s.multiCapitalPool = mcp
}

// SetLiveEngine wires the live trading engine into the risk pipeline and
// KillSwitch infrastructure. When set, every ProcessTick runs through the
// shared pipeline, and KillSwitch triggers propagate to the capital pool.
func (s *Server) SetLiveEngine(le *engine.LiveEngine) {
	s.liveEngine = le

	if s.multiCapitalPool != nil {
		le.SetMultiAccountPool(s.multiCapitalPool)

		signalGate := risk.NewSignalGateImpl(
			risk.NewVolatilityHalt(risk.DefaultVolatilityHalt),
			risk.NewPositionSizer(nil),
			risk.NewExposureTracker(5.0, 0.25),
			risk.NewOrderRateLimiter(risk.DefaultOrderRateLimit),
		)
		signalGate.SetBacktestMode(false)

		pipeline := &risk.RiskPipeline{
			SignalGate:   signalGate,
			PropFirm:     s.propFirmManager,
			KellyMult:    0.25,
			RegimeMatrix: risk.NewRegimeActivationMatrix(),
		}
		le.SetRiskPipeline(pipeline)
	}

	if s.killSwitch != nil && s.multiCapitalPool != nil {
		s.killSwitch.OnTrigger(func(reason string, _ time.Time) {
			s.multiCapitalPool.MarkAllViolated(reason)
		})
	}
}

func (s *Server) SetEmailService(svc email.EmailService) {
	s.emailService = svc
}

func (s *Server) SetNotifyManager(mgr *notify.Manager) {
	s.notifyManager = mgr
	if mgr != nil && s.wsHub != nil {
		mgr.SetPushHub(s.wsHub)
	}
}

func (s *Server) SetAuditHandler(h *AuditHandler) {
	s.auditHandler = h
}

func (s *Server) SetDataSourceHandler(h *DataSourceHandler) {
	s.dataSourceHandler = h
}

func (s *Server) SetSettingsHandler(h *SettingsHandler) {
	s.settingsHandler = h
}

func (s *Server) listStrategies(c *gin.Context) {
	if s.repo == nil {
		c.JSON(http.StatusOK, gin.H{"strategies": []gin.H{}})
		return
	}
	strategies, err := s.repo.ListStrategies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"strategies": strategies})
}

func (s *Server) getStrategy(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	strategy, err := s.repo.GetStrategy(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if strategy == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "strategy not found"})
		return
	}
	c.JSON(http.StatusOK, strategy)
}

func (s *Server) createStrategy(c *gin.Context) {
	var req struct {
		Name       string                 `json:"name" binding:"required"`
		Type       string                 `json:"type" binding:"required"`
		Parameters map[string]interface{} `json:"parameters"`
		Enabled    bool                   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if s.repo == nil {
		c.JSON(http.StatusCreated, gin.H{"id": "strategy-uuid", "name": req.Name})
		return
	}
	st := &db.Strategy{
		ID:         "strat-" + time.Now().Format("20060102150405"),
		Name:       req.Name,
		Type:       req.Type,
		Parameters: req.Parameters,
		Enabled:    req.Enabled,
	}
	if err := s.repo.UpsertStrategy(c.Request.Context(), st); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": st.ID, "name": st.Name, "type": st.Type, "enabled": st.Enabled, "parameters": st.Parameters})
}

func (s *Server) updateStrategy(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name       string                 `json:"name"`
		Type       string                 `json:"type"`
		Parameters map[string]interface{} `json:"parameters"`
		Enabled    *bool                  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if s.repo == nil {
		c.JSON(http.StatusOK, gin.H{"updated": true})
		return
	}
	existing, err := s.repo.GetStrategy(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Strategy not found"})
		return
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Type != "" {
		existing.Type = req.Type
	}
	if req.Parameters != nil {
		existing.Parameters = req.Parameters
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if err := s.repo.UpsertStrategy(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": true, "strategy": existing})
}

func (s *Server) deleteStrategy(c *gin.Context) {
	id := c.Param("id")
	if s.repo == nil {
		c.JSON(http.StatusOK, gin.H{"deleted": true})
		return
	}
	if err := s.repo.DeleteStrategy(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (s *Server) validateStrategy(c *gin.Context) {
	var req struct {
		Name       string                 `json:"name"`
		Type       string                 `json:"type"`
		Parameters map[string]interface{} `json:"parameters"`
		Yaml       string                 `json:"yaml"`
		JSON       string                 `json:"json"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gkrJSON, err := json.Marshal(map[string]interface{}{
		"name":       req.Name,
		"type":       req.Type,
		"parameters": req.Parameters,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize strategy payload"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "orca", "validate", "--stdin")
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"valid": false, "errors": []string{fmt.Sprintf("subprocess pipe error: %v", err)}})
		return
	}

	go func() {
		defer stdinPipe.Close()
		stdinPipe.Write(gkrJSON)
	}()

	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := string(output)
		if errMsg == "" {
			errMsg = err.Error()
		}
		c.JSON(http.StatusOK, gin.H{
			"valid":  false,
			"errors": []string{errMsg},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":  true,
		"errors": []string{},
	})
}

func (s *Server) reloadStrategy(c *gin.Context) {
	id := c.Param("id")

	if s.repo != nil {
		_, err := s.repo.GetStrategy(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Strategy not found", "reloaded": false})
			return
		}
		log.Printf("strategy %s reload request received", id)
	}

	c.JSON(http.StatusOK, gin.H{"reloaded": true, "strategy_id": id})
}

func (s *Server) cloneStrategy(c *gin.Context) {
	id := c.Param("id")

	if s.repo == nil {
		c.JSON(http.StatusCreated, gin.H{"id": "clone-uuid", "name": "cloned-strategy"})
		return
	}

	existing, err := s.repo.GetStrategy(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Strategy not found"})
		return
	}

	cloned := &db.Strategy{
		ID:         id + "-copy-" + time.Now().Format("20060102150405"),
		Name:       existing.Name + "-copy",
		Type:       existing.Type,
		Parameters: existing.Parameters,
		Enabled:    false,
	}
	if err := s.repo.UpsertStrategy(c.Request.Context(), cloned); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id": cloned.ID, "name": cloned.Name, "type": cloned.Type,
		"enabled": cloned.Enabled, "parameters": cloned.Parameters,
	})
}

func (s *Server) getCandles(c *gin.Context) {
	symbol := c.Query("symbol")
	rangeStr := c.DefaultQuery("range", "1D")

	if symbol == "" {
		symbol = "SPY"
	}

	end := time.Now()
	var start time.Time
	switch rangeStr {
	case "1D":
		start = end.AddDate(0, 0, -1)
	case "1W":
		start = end.AddDate(0, 0, -7)
	case "1M":
		start = end.AddDate(0, -1, 0)
	case "3M":
		start = end.AddDate(0, -3, 0)
	case "1Y":
		start = end.AddDate(-1, 0, 0)
	case "ALL":
		start = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	default:
		start = end.AddDate(0, 0, -7)
	}

	if s.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not connected"})
		return
	}

	candles, err := s.repo.LoadCandles(c.Request.Context(), []string{symbol}, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load candles: " + err.Error()})
		return
	}

	if len(candles) == 0 || len(candles[0]) == 0 {
		c.JSON(http.StatusOK, gin.H{"symbol": symbol, "range": rangeStr, "candles": []gin.H{}, "warning": "no candle data available for this symbol and range"})
		return
	}

	var result []gin.H
	for _, cs := range candles {
		for _, candle := range cs {
			result = append(result, gin.H{
				"time":   candle.Time.Format(time.RFC3339),
				"open":   candle.Open,
				"high":   candle.High,
				"low":    candle.Low,
				"close":  candle.Close,
				"volume": candle.Volume,
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{"symbol": symbol, "range": rangeStr, "candles": result})
}

func (s *Server) getAccounts(c *gin.Context) {
	var accounts []gin.H

	if s.accountManager != nil {
		userID := c.GetString("user_id")
		for _, acct := range s.accountManager.ListAccountsByUser(c.Request.Context(), userID) {
			halted := s.killSwitch != nil && s.killSwitch.IsHalted()
			entry := gin.H{
				"id":                     acct.ID,
				"label":                  acct.Name,
				"firm":                   "Prop Firm",
				"broker_type":            acct.BrokerType,
				"type":                   "prop",
				"is_default":             acct.IsDefault,
				"halted":                 halted,
				"daily_loss_limit_pct":   5.0,
				"max_dd_pct":             10.0,
				"consistency_multiplier": 1.0,
				"balance":                acct.Balance,
				"equity":                 acct.Equity,
				"daily_pnl_pct":          acct.DailyPnL,
			}

			brokerAcct, brokerErr := acct.GetAccount(c.Request.Context())
			if brokerErr == nil && brokerAcct != nil {
				entry["buying_power"] = brokerAcct.BuyingPower
			}

			positions, posErr := acct.GetPositions(c.Request.Context())
			drawdownPct := 0.0
			if posErr == nil && brokerAcct != nil && brokerAcct.Balance.Float64() > 0 {
				for _, pos := range positions {
					if pos.UnrealizedPL < 0 {
						drawdownPct += (-pos.UnrealizedPL / brokerAcct.Balance.Float64()) * 100
					}
				}
			}
			entry["drawdown_pct"] = drawdownPct
			entry["profit_target_pct"] = nil

			accounts = append(accounts, entry)
		}
	}

	if len(accounts) == 0 && s.brokerRegistry != nil {
		adapterStatus := s.brokerRegistry.List()
		for id, st := range adapterStatus {
			halted := s.killSwitch != nil && s.killSwitch.IsHalted()
			entry := gin.H{
				"id":     id,
				"label":  id,
				"firm":   "Prop Firm",
				"type":   "prop",
				"status": st.BrokerType,
				"healthy": st.Healthy,
				"priority": st.Priority,
				"halted": halted,
			}

			adapter, ok := s.brokerRegistry.Get(id)
			if ok {
				acct, err := adapter.GetAccount(c.Request.Context())
				if err == nil && acct != nil {
					entry["balance"] = acct.Balance
					entry["equity"] = acct.Equity
					entry["daily_pnl_pct"] = acct.DailyPL
					entry["buying_power"] = acct.BuyingPower
				} else {
					entry["balance"] = 0.0
					entry["equity"] = 0.0
					entry["daily_pnl_pct"] = 0.0
				}

				positions, _ := adapter.GetPositions(c.Request.Context())
				drawdownPct := 0.0
				if acct != nil && acct.Balance.Float64() > 0 {
					for _, pos := range positions {
						if pos.UnrealizedPL < 0 {
							drawdownPct += (-pos.UnrealizedPL / acct.Balance.Float64()) * 100
						}
					}
				}
				entry["drawdown_pct"] = drawdownPct
			}

			entry["daily_loss_limit_pct"] = 5.0
			entry["max_dd_pct"] = 10.0
			entry["profit_target_pct"] = nil
			entry["consistency_multiplier"] = 1.0

			accounts = append(accounts, entry)
		}
	}

	if len(accounts) == 0 && s.adapter != nil {
		acct, err := s.adapter.GetAccount(c.Request.Context())
		halted := s.killSwitch != nil && s.killSwitch.IsHalted()
		entry := gin.H{
			"id":                     "primary",
			"label":                  "Primary Account",
			"firm":                   "Prop Firm",
			"type":                   "prop",
			"halted":                 halted,
			"daily_loss_limit_pct":   5.0,
			"max_dd_pct":             10.0,
			"consistency_multiplier": 1.0,
		}
		if err == nil && acct != nil {
			entry["balance"] = acct.Balance
			entry["equity"] = acct.Equity
			entry["daily_pnl_pct"] = acct.DailyPL
		} else {
			entry["balance"] = 100000.0
			entry["equity"] = 100000.0
			entry["daily_pnl_pct"] = 0.0
		}
		entry["drawdown_pct"] = 0.0
		accounts = append(accounts, entry)
	}

	c.JSON(http.StatusOK, gin.H{"accounts": accounts})
}

func (s *Server) createAccount(c *gin.Context) {
	var req struct {
		ID                string `json:"id"`
		Name              string `json:"name"`
		BrokerType        string `json:"broker_type"`
		PropFirmProfileID string `json:"prop_firm_profile_id"`
		IsDefault         bool   `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name == "" {
		req.Name = req.ID
	}

	var adapter broker.Adapter
	if s.brokerRegistry != nil {
		var ok bool
		adapter, ok = s.brokerRegistry.Get(req.BrokerType + ":" + req.ID)
		if !ok && s.adapter != nil {
			adapter = s.adapter
		}
	}
	if adapter == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no adapter available for broker type: " + req.BrokerType})
		return
	}

	if s.accountManager == nil {
		s.accountManager = broker.NewAccountManager(s.repo, s.brokerRegistry)
	}

	acct := broker.NewManagedAccount(req.ID, req.BrokerType, req.Name, adapter)
	acct.PropFirmProfileID = req.PropFirmProfileID
	acct.IsDefault = req.IsDefault

	if err := s.accountManager.RegisterAccount(acct); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// Register per-account isolated strategy instances with the live engine.
	if s.liveEngine != nil {
		s.liveEngine.RegisterAccountStrategies(req.ID, nil)
	}

	if s.repo != nil {
		if err := s.repo.InsertAccount(c.Request.Context(), acct.ToDBAccount()); err != nil {
			log.Printf("router: failed to persist account %s: %v", req.ID, err)
		}
	}

	c.JSON(http.StatusCreated, gin.H{"id": req.ID, "name": req.Name})
}

func (s *Server) deleteAccount(c *gin.Context) {
	id := c.Param("id")
	if s.accountManager == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account manager not available"})
		return
	}
	s.accountManager.UnregisterAccount(id)
	if s.repo != nil {
		if err := s.repo.DeleteAccount(c.Request.Context(), id); err != nil {
			log.Printf("router: failed to delete account %s from db: %v", id, err)
		}
	}
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

func (s *Server) setDefaultAccount(c *gin.Context) {
	id := c.Param("id")
	if s.accountManager == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account manager not available"})
		return
	}
	if err := s.accountManager.SetDefaultAccount(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"default": id})
}

func (s *Server) getSignals(c *gin.Context) {
	signals := s.eventBus.GetSignals(100)
	c.JSON(http.StatusOK, gin.H{"signals": signals})
}

func (s *Server) submitOptimization(c *gin.Context) {
	var req struct {
		StrategyID      string                       `json:"strategy_id"`
		Objective       string                       `json:"objective"`
		MaxCombinations int                          `json:"max_combinations"`
		TrainYears      int                          `json:"train_years"`
		TestYears       int                          `json:"test_years"`
		StepMonths      int                          `json:"step_months"`
		Symbols         []string                     `json:"symbols"`
		Capital         float64                      `json:"capital"`
		Constraints     map[string]struct {
			Min  float64 `json:"min"`
			Max  float64 `json:"max"`
			Step float64 `json:"step"`
		} `json:"constraints"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.StrategyID == "" {
		req.StrategyID = "trend_following"
	}
	if req.MaxCombinations <= 0 {
		req.MaxCombinations = 100
	}
	if req.TrainYears <= 0 {
		req.TrainYears = 1
	}
	if req.TestYears <= 0 {
		req.TestYears = 1
	}
	if req.StepMonths <= 0 {
		req.StepMonths = 3
	}
	if len(req.Symbols) == 0 {
		req.Symbols = []string{"SPY"}
	}
	if req.Capital <= 0 {
		req.Capital = 100000
	}

	searchSpace := make(backtest.SearchSpace)
	for name, c := range req.Constraints {
		pType := backtest.ParamContinuous
		if c.Step == 1.0 || c.Step == float64(int(c.Step)) {
			if c.Min == float64(int(c.Min)) && c.Max == float64(int(c.Max)) {
				pType = backtest.ParamInteger
			}
		}
		searchSpace[name] = backtest.ParamConstraint{
			Name: name, Type: pType, Min: c.Min, Max: c.Max, Step: c.Step,
		}
	}
	if len(searchSpace) == 0 {
		searchSpace = backtest.DefaultSearchSpace(req.StrategyID)
	}

	objType := backtest.ObjectiveSharpe
	switch req.Objective {
	case "sharpe":
		objType = backtest.ObjectiveSharpe
	case "sortino":
		objType = backtest.ObjectiveSortino
	case "profit_factor":
		objType = backtest.ObjectiveProfitFactor
	case "win_rate":
		objType = backtest.ObjectiveWinRate
	case "min_drawdown":
		objType = backtest.ObjectiveMinDD
	case "sharpe_over_dd":
		objType = backtest.ObjectiveDDRatio
	case "composite":
		objType = backtest.ObjectiveComposite
	}

	config := backtest.OptimizedWalkForwardConfig{
		WalkForwardConfig: backtest.WalkForwardConfig{
			Config: backtest.BacktestConfig{
				StrategyID:     req.StrategyID,
				Symbols:        req.Symbols,
			StartDate:      time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:        time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
				InitialCapital: req.Capital,
			},
			TrainWindows: 5,
			TrainYears:   req.TrainYears,
			TestYears:    req.TestYears,
			StepMonths:   req.StepMonths,
		},
		OptimizationConfig: backtest.OptimizationConfig{
			StrategyID:      req.StrategyID,
			SearchSpace:     searchSpace,
			ObjectiveType:   objType,
			MaxCombinations: req.MaxCombinations,
		},
	}

	result, err := s.backtestEngine.RunOptimizedWalkForward(c.Request.Context(), config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var mcResult *backtest.MonteCarloResult
	var verdict backtest.MultiMetricVerdict
	owfResult := &backtest.OptimizedWalkForwardResult{
		WalkForwardResult: backtest.WalkForwardResult{
			Windows:         result.Windows,
			OverallSharpe:   result.OverallSharpe,
			AvgOOSSharpe:    result.AvgOOSSharpe,
			SharpeDegradation: result.SharpeDegradation,
			PassedWindows:   result.PassedWindows,
			TotalWindows:    result.TotalWindows,
		},
	}

	if result.AvgOOSSharpe != 0 {
		var oosTrades []backtest.Trade
		for _, win := range result.Windows {
			oosTrades = append(oosTrades, backtest.Trade{PnLPct: win.OOSReturnPct})
		}
		if len(oosTrades) > 1 {
			if r, err := backtest.RunMonteCarloFromTrades(oosTrades, 1000, req.Capital); err == nil {
				mcResult = r
			}
		}
		std := backtest.DefaultMultiMetricStandard()
		verdict = backtest.EvaluateOOSMultiMetric(owfResult, mcResult, std)
	}

	c.JSON(http.StatusOK, gin.H{
		"best_params_per_window": result.BestParamsPerWindow,
		"overall_sharpe":         result.OverallSharpe,
		"avg_oos_sharpe":         result.AvgOOSSharpe,
		"sharpe_degradation":     result.SharpeDegradation,
		"passed_windows":         result.PassedWindows,
		"total_windows":          result.TotalWindows,
		"windows":                result.Windows,
		"monte_carlo":            mcResult,
		"verdict":                verdict,
	})
}

func (s *Server) submitBacktest(c *gin.Context) {
	var req struct {
		Mode          string   `json:"mode"`
		StrategyID    string   `json:"strategy_id"`
		StrategyIDs   []string `json:"strategy_ids"`
		Symbols       []string `json:"symbols" binding:"required"`
		Timeframes    []string `json:"timeframes"`
		StartDate     string   `json:"start_date" binding:"required"`
		EndDate       string   `json:"end_date" binding:"required"`
		Capital       float64  `json:"capital"`
		GateProfile   string   `json:"gate_profile"`
		SizingPercent float64  `json:"sizing_percent"`
		KellyFraction float64  `json:"kelly_fraction"`
		LightOptimize *bool    `json:"light_optimize,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if s.backtestEngine == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "backtest engine not available"})
		return
	}
	if req.Capital == 0 {
		req.Capital = 100000.0
	}
	if len(req.Timeframes) == 0 {
		req.Timeframes = []string{"1d"}
	}
	ds := s.dataSource
	if ds == "" {
		ds = "stooq"
	}
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format, use YYYY-MM-DD"})
		return
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format, use YYYY-MM-DD"})
		return
	}

	if req.Mode == "matrix" {
		if len(req.StrategyIDs) == 0 {
			req.StrategyIDs = []string{"intraday_mr", "opening_range_breakout", "trend_following",
				"grid_trading", "session_scalp", "pairs_trading", "volatility_harvesting"}
		}
		mc := backtest.MatrixBacktestConfig{
			StrategyIDs:       req.StrategyIDs,
			Symbols:           req.Symbols,
			Timeframes:        req.Timeframes,
			StartDate:         startDate,
			EndDate:           endDate,
			InitialCapital:    req.Capital,
			DataSource:        ds,
			GateProfile:       req.GateProfile,
			PropFirmEnabled:   true,
			SizingPercent:     req.SizingPercent,
			KellyFraction:     req.KellyFraction,
			SkipLightOptimize: req.LightOptimize != nil && !*req.LightOptimize,
		}
		combos := backtest.CartesianProduct(mc.StrategyIDs, mc.Symbols, mc.Timeframes)
		batchID := fmt.Sprintf("matrix-%s", time.Now().Format("20060102150405"))

		cp := make([]ComboProgress, len(combos))
		for i, co := range combos {
			cp[i] = ComboProgress{
				Symbol:     co.Symbol,
				StrategyID: co.Strategy,
				Timeframe:  co.Timeframe,
				Status:     "pending",
			}
		}
		s.progressStore.Create(batchID, len(combos), cp)

		go func(bid string) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("MATRIX BACKTEST PANIC", "batch_id", bid, "panic", r)
					ps := s.progressStore
					mp := ps.Get(bid)
					if mp != nil {
						mp.Status = "failed"
						for i := range mp.Combos {
							if mp.Combos[i].Status == "pending" || mp.Combos[i].Status == "running" {
								ps.UpdateCombo(bid, i, "failed", fmt.Sprintf("server panic: %v", r), nil)
							}
						}
					}
				}
			}()

			dbAdapter := &backtestRepoAdapter{repo: s.repo}
			_, _ = backtest.RunMatrixConcurrent(context.Background(), dbAdapter, mc,
				func(index int, status string, errMsg string, result *backtest.ComboResult) {
					switch status {
					case "running":
						s.progressStore.UpdateCombo(bid, index, "running", "", nil)
					case "failed":
						s.progressStore.UpdateCombo(bid, index, "failed", errMsg, nil)
					case "completed":
						if result == nil {
							return
						}
						crQuick := *result
						crQuick.EquityCurve = nil
						crQuick.Trades = nil
						crQuick.RunID = fmt.Sprintf("%s-%s-%s-%d", result.StrategyID, result.Symbol, result.Timeframe, index)
						s.progressStore.UpdateCombo(bid, index, "completed", "", &crQuick)
						if s.repo != nil {
							sd := startDate
							ed := endDate
							rec := &db.BacktestRunRecord{
								StrategyID:       result.StrategyID,
								RunType:          "matrix",
								Status:           "completed",
								StrategyIDs:      []string{result.StrategyID},
								Symbols:          []string{result.Symbol},
								StartDate:        &sd,
								EndDate:          &ed,
								InitialCapital:   mc.InitialCapital,
								SharpeRatio:      result.SharpeRatio,
								SortinoRatio:     result.SortinoRatio,
								MaxDrawdown:      result.MaxDrawdown,
								MaxDrawdownDur:   result.MaxDrawdownDur,
								TotalReturn:      result.TotalReturn,
								WinRate:          result.WinRate,
								ProfitFactor:     result.ProfitFactor,
								AvgTrade:         result.AvgTrade,
								AvgWin:           result.AvgWin,
								AvgLoss:          result.AvgLoss,
								NumTrades:        result.NumTrades,
								NumWins:          result.NumWins,
								NumLosses:        result.NumLosses,
								GatePassed:       result.GatePassed,
							}
							fullMetricsJSON, merr := json.Marshal(result)
							if merr != nil {
								slog.Error("matrix: failed to marshal metrics", "err", merr)
								return
							}
							rec.ResultsJSON = fullMetricsJSON
							if cerr := s.repo.CreateBacktestRun(context.Background(), rec); cerr != nil {
								slog.Error("matrix: failed to persist combo", "symbol", result.Symbol, "strategy", result.StrategyID, "tf", result.Timeframe, "err", cerr)
							} else {
								cr := *result
								cr.RunID = rec.ID
								cr.EquityCurve = nil
								cr.Trades = nil
								s.progressStore.UpdateCombo(bid, index, "completed", "", &cr)
								eqJSON, _ := json.Marshal(result.EquityCurve)
								tradesJSON, _ := json.Marshal(result.Trades)
								_ = s.repo.InsertBacktestResult(context.Background(), &db.BacktestResultRecord{
									RunID:         rec.ID,
									StrategyID:    result.StrategyID,
									ResultType:    "matrix",
									TrialIndex:    0,
									SchemaVersion: 1,
									Metrics:       rec.ResultsJSON,
									EquityCurve:   eqJSON,
									Trades:        tradesJSON,
								})
							}
						}
					}
				},
			)
		}(batchID)

		c.JSON(http.StatusAccepted, gin.H{
			"batch_run_id": batchID,
			"status":       "running",
			"total_combos": len(combos),
		})
		return
	}

	strategyID := req.StrategyID
	if len(req.StrategyIDs) > 0 {
		strategyID = req.StrategyIDs[0]
	}
	if strategyID == "" {
		strategyID = "intraday_mr"
	}
	tf := req.Timeframes[0]

	baseConfig := backtest.BacktestConfig{
		StrategyID:     strategyID,
		Symbols:        req.Symbols,
		StartDate:      startDate,
		EndDate:        endDate,
		InitialCapital: req.Capital,
		DataSource:     ds,
		Timeframe:      tf,
		PropFirmEnabled: true,
		StopLoss: &backtest.StopLossConfig{
			Type:          "atr",
			ATRPeriod:     14,
			ATRMultiplier: 2.0,
		},
		TakeProfit: &backtest.TakeProfitConfig{
			Type:    "risk_reward",
			RRRatio: 2.0,
		},
		ApplyGate:   req.GateProfile != "" && req.GateProfile != "none",
		GateProfile: req.GateProfile,
	}

	if req.LightOptimize != nil && *req.LightOptimize {
		batchID := "single-" + time.Now().Format("20060102150405") + "-" + strategyID
		totalCombos := len(req.Symbols) * 4
		combos := make([]ComboProgress, 0, totalCombos)
		sizings := []float64{0.02, 0.04, 0.06, 0.08}
		for _, sym := range req.Symbols {
			for _, _ = range sizings {
				combos = append(combos, ComboProgress{
					StrategyID: strategyID, Symbol: sym, Timeframe: tf,
					Status: "pending",
				})
			}
		}
		s.progressStore.Create(batchID, totalCombos, combos)

		go func(bid string) {
			defer func() { recover() }()
			idx := 0
			sizes := []float64{0.02, 0.04, 0.06, 0.08}
			for _, sym := range req.Symbols {
				for _, sp := range sizes {
					cfg := baseConfig
					cfg.Symbols = []string{sym}
					cfg.SizingPercent = sp
					engine := backtest.NewEngine(s.backtestEngine.GetDB())
					engine.WirePipeline()
					result, err := engine.Run(context.Background(), cfg)
					if err != nil {
						s.progressStore.UpdateCombo(bid, idx, "failed", err.Error(), nil)
						idx++
						continue
					}
					passed := result.SharpeRatio >= 1.0 && result.MaxDrawdown <= 8.0 && result.ProfitFactor >= 1.5
					cr := backtest.ComboResult{
						StrategyID:     strategyID,
						Symbol:         sym,
						Timeframe:      tf,
						SharpeRatio:    result.SharpeRatio,
						SortinoRatio:   result.SortinoRatio,
						MaxDrawdown:    result.MaxDrawdown,
						TotalReturn:    result.TotalReturnPct,
						WinRate:        result.WinRate,
						ProfitFactor:   result.ProfitFactor,
						NumTrades:      result.NumTrades,
						AvgTrade:       result.AvgTrade,
						AvgWin:         result.AvgWin,
						AvgLoss:        result.AvgLoss,
						AvgMAE:         result.AvgMAE,
						AvgMFE:         result.AvgMFE,
						StrategyParams: map[string]float64{"sizing_percent": sp, "kelly_fraction": cfg.KellyFraction},
						Optimized:      true,
						GatePassed:     &passed,
						EquityCurve:    result.EquityCurve,
						Trades:         result.Trades,
						LongTrades:     result.LongShort.LongTrades,
						ShortTrades:    result.LongShort.ShortTrades,
						LongWinRate:    result.LongShort.LongWinRate,
						ShortWinRate:   result.LongShort.ShortWinRate,
						LongGrossPnL:   result.LongShort.LongGrossPnL,
						ShortGrossPnL:  result.LongShort.ShortGrossPnL,
						LongPF:         result.LongShort.LongPF,
						ShortPF:        result.LongShort.ShortPF,
					}
					s.progressStore.UpdateCombo(bid, idx, "completed", "", &cr)
					idx++
				}
			}
		}(batchID)

		c.JSON(http.StatusAccepted, gin.H{
			"batch_run_id": batchID,
			"status":       "running",
			"total_combos": totalCombos,
		})
		return
	}

	result, err := s.backtestEngine.Run(c.Request.Context(), baseConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	runID := "bt-" + time.Now().Format("20060102150405") + "-" + strategyID
	if s.repo != nil {
		metricsJSON, _ := json.Marshal(gin.H{
			"sharpe_ratio":  result.SharpeRatio,
			"max_drawdown":  result.MaxDrawdown,
			"total_return":  result.TotalReturnPct,
			"win_rate":      result.WinRate,
			"num_trades":    result.NumTrades,
			"profit_factor": result.ProfitFactor,
			"data_source":   ds,
			"timeframe":     tf,
		})
		runRecord := &db.BacktestRunRecord{
			StrategyID:     strategyID,
			RunType:        "single",
			Status:         "completed",
			StrategyIDs:    []string{strategyID},
			Symbols:        req.Symbols,
			StartDate:      &startDate,
			EndDate:        &endDate,
			InitialCapital: req.Capital,
			SharpeRatio:    result.SharpeRatio,
			MaxDrawdown:    result.MaxDrawdown,
			TotalReturn:    result.TotalReturnPct,
			WinRate:        result.WinRate,
			NumTrades:      result.NumTrades,
			ResultsJSON:    metricsJSON,
		}
		if err := s.repo.CreateBacktestRun(c.Request.Context(), runRecord); err != nil {
			log.Printf("backtest history: failed to persist run: %v", err)
		} else {
			runID = runRecord.ID
			eqJSON, _ := json.Marshal(result.EquityCurve)
			tradesJSON, _ := json.Marshal(result.Trades)
			fullMetrics, _ := json.Marshal(result)
			if insErr := s.repo.InsertBacktestResult(c.Request.Context(), &db.BacktestResultRecord{
				RunID:         runID,
				StrategyID:    strategyID,
				ResultType:    "single",
				TrialIndex:    0,
				SchemaVersion: 1,
				Metrics:       fullMetrics,
				EquityCurve:   eqJSON,
				Trades:        tradesJSON,
			}); insErr != nil {
				log.Printf("backtest: failed to persist result detail: %v", insErr)
			}
		}
	}

	c.JSON(http.StatusAccepted, gin.H{
		"id":            runID,
		"status":        "completed",
		"sharpe_ratio":  result.SharpeRatio,
		"max_drawdown":  result.MaxDrawdown,
		"total_return":  result.TotalReturnPct,
		"win_rate":      result.WinRate,
		"num_trades":    result.NumTrades,
		"profit_factor": result.ProfitFactor,
		"data_source":   ds,
		"timeframe":     tf,
		"warnings":      result.Warnings,
		"equity_curve":  result.EquityCurve,
		"regime_stats":  result.RegimeStats,
	})
}

func (s *Server) getBacktestRegimeStats(c *gin.Context) {
	id := c.Param("id")
	if s.repo == nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	results, err := s.repo.GetBacktestResults(c.Request.Context(), id)
	if err != nil || len(results) == 0 {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	var allRegimeStats []gin.H
	for _, res := range results {
		if res.Metrics != nil {
			var engineResult backtest.BacktestResult
			if err := json.Unmarshal(res.Metrics, &engineResult); err == nil {
				for _, rs := range engineResult.RegimeStats {
					allRegimeStats = append(allRegimeStats, gin.H{
						"regime":        rs.Regime,
						"label":         rs.Label,
						"num_trades":    rs.NumTrades,
						"win_rate":      rs.WinRate,
						"total_return":  rs.TotalReturn,
						"max_drawdown":  rs.MaxDrawdown,
						"profit_factor": rs.ProfitFactor,
					})
				}
			}
		}
	}
	c.JSON(http.StatusOK, allRegimeStats)
}

func (s *Server) getBacktestProgress(c *gin.Context) {
	id := c.Param("id")
	mp := s.progressStore.Get(id)
	if mp == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "batch run not found"})
		return
	}
	c.JSON(http.StatusOK, mp)
}

func (s *Server) getPositions(c *gin.Context) {
	accountID := c.Query("account_id")

	var positions []broker.Position
	var err error

	if s.accountManager != nil && accountID != "" {
		positions, err = s.accountManager.GetPositions(c.Request.Context(), accountID)
	} else if s.accountManager != nil {
		defaultID := s.accountManager.GetDefaultAccountID()
		if defaultID != "" {
			positions, err = s.accountManager.GetPositions(c.Request.Context(), defaultID)
		} else {
			positions, err = s.adapter.GetPositions(c.Request.Context())
		}
	} else {
		positions, err = s.adapter.GetPositions(c.Request.Context())
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"positions": positions})
}

func (s *Server) getRiskStatus(c *gin.Context) {
	halted, reason, lastTrigger := s.killSwitch.Status()
	balance := 100000.0
	equity := 100000.0
	dailyPL := 0.0

	if s.accountManager != nil {
		defaultAcct, err := s.accountManager.GetDefaultAccount()
		if err == nil && defaultAcct != nil {
			balance = defaultAcct.Balance.Float64()
			equity = defaultAcct.Equity.Float64()
			dailyPL = defaultAcct.DailyPnL
		}
	} else if s.adapter != nil {
		acct, err := s.adapter.GetAccount(c.Request.Context())
		if err == nil && acct != nil {
			balance = acct.Balance.Float64()
			equity = acct.Equity.Float64()
			dailyPL = acct.DailyPL
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"halted":                 halted,
		"reason":                 reason,
		"last_trigger":           lastTrigger.Format(time.RFC3339),
		"balance":                balance,
		"equity":                 equity,
		"daily_pnl_pct":          dailyPL,
		"daily_loss_used":        0.0,
		"drawdown_used":          0.0,
		"daily_limit_pct":        5.0,
		"max_dd_pct":             10.0,
		"consistency_multiplier": 1.0,
	})
}

func (s *Server) getRegimeHistory(c *gin.Context) {
	if s.repo == nil {
		c.JSON(http.StatusOK, gin.H{"history": []gin.H{}})
		return
	}
	now := time.Now()
	logs, err := s.repo.LoadRegimeLogs(c.Request.Context(), now.AddDate(0, 0, -30), now)
	if err != nil || len(logs) == 0 {
		c.JSON(http.StatusOK, gin.H{"history": []gin.H{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"history": logs})
}

func (s *Server) listOrders(c *gin.Context) {
	if s.adapter == nil {
		c.JSON(http.StatusOK, gin.H{"orders": []interface{}{}})
		return
	}
	pos, err := s.adapter.GetPositions(c.Request.Context())
	if err != nil || pos == nil {
		c.JSON(http.StatusOK, gin.H{"orders": []interface{}{}})
		return
	}
	type orderItem struct {
		Symbol    string  `json:"symbol"`
		Quantity  float64 `json:"quantity"`
		Entry     float64 `json:"entry_price"`
		Mark      float64 `json:"mark_price"`
	}
	var items []orderItem
	for _, p := range pos {
		items = append(items, orderItem{
			Symbol:   p.Symbol,
			Quantity: p.Quantity,
			Entry:    p.AvgEntryPrice.Float64(),
			Mark:     p.MarketValue.Float64(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"orders": items})
}

func (s *Server) triggerEmergencyStop(c *gin.Context) {
	err := s.killSwitch.Trigger("manual")
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"halted": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"halted": true})
}

func (s *Server) resumeTrading(c *gin.Context) {
	s.killSwitch.Resume()
	c.JSON(http.StatusOK, gin.H{"halted": false})
}

func (s *Server) placeOrder(c *gin.Context) {
	var req struct {
		AccountID   string       `json:"account_id"`
		Symbol      string       `json:"symbol" binding:"required"`
		Side        string       `json:"side" binding:"required"`
		Type        string       `json:"type" binding:"required"`
		Quantity    float64      `json:"quantity" binding:"required"`
		LimitPrice  types.Price  `json:"limitPrice"`
		StopPrice   types.Price  `json:"stopPrice"`
		TimeInForce string       `json:"timeInForce"`
		StrategyID  string       `json:"strategy_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderReq := &broker.OrderRequest{
		AccountID:   req.AccountID,
		Symbol:      req.Symbol,
		Side:        broker.OrderSide(req.Side),
		Type:        broker.OrderType(req.Type),
		Quantity:    req.Quantity,
		LimitPrice:  req.LimitPrice,
		StopPrice:   req.StopPrice,
		TimeInForce: broker.TimeInForce(req.TimeInForce),
		StrategyID:  req.StrategyID,
	}
	if orderReq.TimeInForce == "" {
		orderReq.TimeInForce = broker.Day
	}

	var resp *broker.OrderResponse
	var err error

	if s.accountManager != nil && req.AccountID != "" {
		resp, err = s.accountManager.PlaceOrder(c.Request.Context(), req.AccountID, orderReq)
	} else if s.accountManager != nil {
		defaultID := s.accountManager.GetDefaultAccountID()
		if defaultID != "" {
			resp, err = s.accountManager.PlaceOrder(c.Request.Context(), defaultID, orderReq)
		} else {
			resp, err = s.adapter.PlaceOrder(c.Request.Context(), orderReq)
		}
	} else {
		resp, err = s.adapter.PlaceOrder(c.Request.Context(), orderReq)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	accountIDUsed := req.AccountID
	if accountIDUsed == "" && s.accountManager != nil {
		accountIDUsed = s.accountManager.GetDefaultAccountID()
	}

	if resp.Status == broker.Filled || resp.Status == broker.PartiallyFilled {
		if s.wsHub != nil {
			s.wsHub.Broadcast("fill", gin.H{
				"account_id":     accountIDUsed,
				"broker_order_id": resp.BrokerOrderID,
				"symbol":          req.Symbol,
				"side":            req.Side,
				"filled_qty":      resp.FilledQty,
				"avg_fill_price":  resp.AvgFillPrice,
				"status":          string(resp.Status),
				"strategy_id":     req.StrategyID,
			})
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":              resp.BrokerOrderID,
		"status":          resp.Status,
		"filled_qty":      resp.FilledQty,
		"avg_fill_price":  resp.AvgFillPrice,
		"account_id":      accountIDUsed,
	})
}

func (s *Server) cancelOrder(c *gin.Context) {
	orderID := c.Param("id")
	accountID := c.Query("account_id")

	var err error
	if s.accountManager != nil && accountID != "" {
		err = s.accountManager.CancelOrder(c.Request.Context(), accountID, orderID)
	} else if s.accountManager != nil {
		defaultID := s.accountManager.GetDefaultAccountID()
		if defaultID != "" {
			err = s.accountManager.CancelOrder(c.Request.Context(), defaultID, orderID)
		} else {
			err = s.adapter.CancelOrder(c.Request.Context(), orderID)
		}
	} else {
		err = s.adapter.CancelOrder(c.Request.Context(), orderID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cancelled": true, "id": orderID})
}

func (s *Server) cancelAllOrders(c *gin.Context) {
	accountID := c.Query("account_id")

	var err error
	if s.accountManager != nil && accountID != "" {
		err = s.accountManager.CancelAllOrders(c.Request.Context(), accountID)
	} else if s.accountManager != nil {
		defaultID := s.accountManager.GetDefaultAccountID()
		if defaultID != "" {
			err = s.accountManager.CancelAllOrders(c.Request.Context(), defaultID)
		} else {
			err = s.adapter.CancelAllOrders(c.Request.Context())
		}
	} else {
		err = s.adapter.CancelAllOrders(c.Request.Context())
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cancelled": true})
}

func (s *Server) testLLM(c *gin.Context) {
	var req struct {
		Provider string `json:"provider"`
		APIKey   string `json:"api_key"`
		BaseURL  string `json:"base_url"`
		Model    string `json:"model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client := llm.NewClient(llm.Provider(req.Provider))
	resp, err := client.Chat(&llm.ChatRequest{
		Model:    req.Model,
		Messages: []llm.Message{{Role: "user", Content: "ping"}},
		MaxTokens: 5,
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"reachable": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reachable": true, "model": req.Model, "response": resp.Choices[0].Message.Content})
}


func (s *Server) twoFAValidator() func(username, code string) bool {
	return func(username, code string) bool {
		if s.authHandler == nil {
			return true
		}
		return s.authHandler.ValidateTOTP(username, code)
	}
}

type backtestRepoAdapter struct {
	repo *db.Repository
}

func (a *backtestRepoAdapter) LoadCandles(ctx context.Context, symbols []string, start, end time.Time) ([][]backtest.Candle, error) {
	results, err := a.repo.LoadCandles(ctx, symbols, start, end)
	if err != nil {
		return nil, err
	}
	out := make([][]backtest.Candle, len(results))
	for i, row := range results {
		out[i] = make([]backtest.Candle, len(row))
		for j, c := range row {
			out[i][j] = backtest.Candle{
				Time:   c.Time,
				Open:   c.Open,
				High:   c.High,
				Low:    c.Low,
				Close:  c.Close,
				Volume: c.Volume,
				Symbol: c.Symbol,
			}
		}
	}
	return out, nil
}

func (a *backtestRepoAdapter) LoadCandlesFiltered(ctx context.Context, symbols []string, start, end time.Time, source string) ([][]backtest.Candle, error) {
	if source == "synthetic" {
		return generateSyntheticCandles(symbols, start, end, "1d"), nil
	}
	candles, err := a.LoadCandles(ctx, symbols, start, end)
	if err != nil || isEmptyCandleResult(candles) {
		return generateSyntheticCandles(symbols, start, end, "1d"), nil
	}
	return candles, nil
}

func (a *backtestRepoAdapter) LoadCandlesTFFiltered(ctx context.Context, symbols []string, start, end time.Time, timeframe string, source string) ([][]backtest.Candle, error) {
	if source == "synthetic" {
		return generateSyntheticCandles(symbols, start, end, timeframe), nil
	}
	candles, err := a.LoadCandles(ctx, symbols, start, end)
	if err != nil || isEmptyCandleResult(candles) {
		return generateSyntheticCandles(symbols, start, end, timeframe), nil
	}
	return candles, nil
}

func (a *backtestRepoAdapter) LoadRegimeLogs(ctx context.Context, start, end time.Time) ([]backtest.RegimeLog, error) {
	logs, err := a.repo.LoadRegimeLogs(ctx, start, end)
	if err != nil {
		return nil, nil
	}
	out := make([]backtest.RegimeLog, len(logs))
	for i, l := range logs {
		out[i] = backtest.RegimeLog{
			Time:       l.Time,
			HMMState:   l.HMMState,
			Confidence: l.Confidence,
			Symbol:     l.Symbol,
		}
	}
	return out, nil
}

func isEmptyCandleResult(candles [][]backtest.Candle) bool {
	if len(candles) == 0 {
		return true
	}
	for _, row := range candles {
		if len(row) > 0 {
			return false
		}
	}
	return true
}

type assetClass int

const (
	acEquity    assetClass = iota
	acForex
	acCrypto
	acCommodity
	acIndex
	acUnknown
)

type syntheticSymbolConfig struct {
	Class      assetClass
	BasePrice  float64
	AnnDrift   float64
	AnnVol     float64
	Momentum   float64
	MRStrength float64
	JumpProb   float64
	MinPrice   float64
	LotSize    float64
}

func symbolConfig(sym string) syntheticSymbolConfig {
	switch sym {
	case "AAPL":
		return syntheticSymbolConfig{acEquity, 185.0, 0.10, 0.28, 0.05, 0.0, 0.04, 1.0, 0.01}
	case "MSFT":
		return syntheticSymbolConfig{acEquity, 420.0, 0.12, 0.26, 0.04, 0.0, 0.03, 1.0, 0.01}
	case "GOOGL":
		return syntheticSymbolConfig{acEquity, 175.0, 0.09, 0.30, 0.06, 0.0, 0.04, 1.0, 0.01}
	case "AMZN":
		return syntheticSymbolConfig{acEquity, 185.0, 0.11, 0.32, 0.05, 0.0, 0.05, 1.0, 0.01}
	case "NVDA":
		return syntheticSymbolConfig{acEquity, 900.0, 0.18, 0.45, 0.08, 0.0, 0.06, 1.0, 0.01}
	case "META":
		return syntheticSymbolConfig{acEquity, 510.0, 0.13, 0.35, 0.06, 0.0, 0.05, 1.0, 0.01}
	case "TSLA":
		return syntheticSymbolConfig{acEquity, 250.0, 0.08, 0.50, 0.07, 0.0, 0.08, 1.0, 0.01}
	case "SPY", "VOO":
		return syntheticSymbolConfig{acEquity, 520.0, 0.07, 0.18, 0.02, 0.0, 0.01, 1.0, 0.01}
	case "QQQ":
		return syntheticSymbolConfig{acEquity, 450.0, 0.09, 0.22, 0.03, 0.0, 0.02, 1.0, 0.01}
	case "IWM":
		return syntheticSymbolConfig{acEquity, 210.0, 0.05, 0.24, 0.04, 0.0, 0.03, 1.0, 0.01}
	case "DIA":
		return syntheticSymbolConfig{acEquity, 400.0, 0.06, 0.17, 0.02, 0.0, 0.01, 1.0, 0.01}
	case "EURUSD":
		return syntheticSymbolConfig{acForex, 1.08, 0.0, 0.075, 0.0, 0.03, 0.01, 0.50, 0.00001}
	case "GBPUSD":
		return syntheticSymbolConfig{acForex, 1.27, 0.0, 0.085, 0.0, 0.03, 0.01, 0.50, 0.00001}
	case "USDJPY":
		return syntheticSymbolConfig{acForex, 150.0, 0.0, 0.09, 0.0, 0.04, 0.02, 50.0, 0.001}
	case "USDCAD":
		return syntheticSymbolConfig{acForex, 1.36, 0.0, 0.07, 0.0, 0.04, 0.01, 0.50, 0.00001}
	case "AUDUSD":
		return syntheticSymbolConfig{acForex, 0.66, 0.0, 0.10, 0.0, 0.05, 0.02, 0.30, 0.00001}
	case "NZDUSD":
		return syntheticSymbolConfig{acForex, 0.61, 0.0, 0.10, 0.0, 0.05, 0.02, 0.30, 0.00001}
	case "USDCHF":
		return syntheticSymbolConfig{acForex, 0.88, 0.0, 0.075, 0.0, 0.04, 0.01, 0.50, 0.00001}
	case "BTCUSD":
		return syntheticSymbolConfig{acCrypto, 65000.0, 0.15, 0.70, 0.05, 0.0, 0.12, 1.0, 0.01}
	case "ETHUSD":
		return syntheticSymbolConfig{acCrypto, 3500.0, 0.12, 0.85, 0.06, 0.0, 0.15, 1.0, 0.01}
	case "XAUUSD":
		return syntheticSymbolConfig{acCommodity, 2350.0, 0.04, 0.16, 0.02, 0.01, 0.02, 500.0, 0.01}
	case "XAGUSD":
		return syntheticSymbolConfig{acCommodity, 27.0, 0.03, 0.25, 0.02, 0.01, 0.04, 5.0, 0.001}
	case "GLD":
		return syntheticSymbolConfig{acCommodity, 180.0, 0.04, 0.15, 0.02, 0.01, 0.02, 50.0, 0.01}
	case "USO":
		return syntheticSymbolConfig{acCommodity, 78.0, 0.02, 0.28, 0.03, 0.02, 0.06, 10.0, 0.01}
	case "TLT":
		return syntheticSymbolConfig{acCommodity, 92.0, 0.01, 0.16, 0.01, 0.03, 0.02, 50.0, 0.01}
	case "CL":
		return syntheticSymbolConfig{acCommodity, 78.0, 0.03, 0.32, 0.03, 0.02, 0.08, 10.0, 0.01}
	case "SPX500", "ES":
		return syntheticSymbolConfig{acIndex, 5500.0, 0.07, 0.17, 0.02, 0.0, 0.01, 100.0, 0.01}
	case "NAS100", "NQ":
		return syntheticSymbolConfig{acIndex, 20000.0, 0.09, 0.22, 0.03, 0.0, 0.02, 100.0, 0.01}
	case "US30":
		return syntheticSymbolConfig{acIndex, 40000.0, 0.06, 0.16, 0.02, 0.0, 0.01, 100.0, 0.01}
	case "GER40":
		return syntheticSymbolConfig{acIndex, 18500.0, 0.05, 0.19, 0.02, 0.0, 0.02, 100.0, 0.01}
	case "JPN225":
		return syntheticSymbolConfig{acIndex, 39000.0, 0.04, 0.20, 0.02, 0.0, 0.02, 100.0, 0.01}
	case "UK100":
		return syntheticSymbolConfig{acIndex, 8200.0, 0.04, 0.16, 0.02, 0.0, 0.01, 10.0, 0.01}
	default:
		return syntheticSymbolConfig{acUnknown, 100.0, 0.05, 0.20, 0.02, 0.0, 0.02, 1.0, 0.01}
	}
}

func barsPerTradingDay(tf string) int {
	switch tf {
	case "1d", "daily", "":
		return 1
	case "4h":
		return 2
	case "1h", "hourly", "60m":
		return 7
	case "30m":
		return 13
	case "15m":
		return 26
	case "5m", "5min":
		return 78
	case "1m":
		return 390
	default:
		return 1
	}
}

func generateSyntheticCandles(symbols []string, start, end time.Time, timeframe string) [][]backtest.Candle {
	barsPerDay := barsPerTradingDay(timeframe)
	out := make([][]backtest.Candle, len(symbols))
	for si, sym := range symbols {
		cfg := symbolConfig(sym)
		rng := rand.New(rand.NewPCG(uint64(start.Unix())+uint64(si)*1000000, uint64(end.Unix())+uint64(si)*2000000))
		out[si] = generateAssetPath(cfg, sym, start, end, barsPerDay, rng)
	}
	return out
}

func generateAssetPath(cfg syntheticSymbolConfig, sym string, start, end time.Time, barsPerDay int, rng *rand.Rand) []backtest.Candle {
	dailyVol := cfg.AnnVol / math.Sqrt(252)
	price := cfg.BasePrice
	lastReturn := 0.0
	currentVol := dailyVol

	regimeTransition := [4][4]float64{
		{0.85, 0.10, 0.04, 0.01},
		{0.08, 0.80, 0.10, 0.02},
		{0.03, 0.10, 0.80, 0.07},
		{0.01, 0.02, 0.10, 0.87},
	}
	driftMult := [4]float64{1.5, 2.0, 0.5, -1.0}
	volMult := [4]float64{1.0, 1.5, 2.5, 3.5}
	mrMult := [4]float64{1.0, 0.5, 0.5, 0.1}
	jumpMult := [4]float64{0.1, 0.3, 1.0, 1.5}
	currentRegime := 0

	var candles []backtest.Candle
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}

		p := rng.Float64()
		cum := 0.0
		nextRegime := 0
		for j := 0; j < 4; j++ {
			cum += regimeTransition[currentRegime][j]
			if p < cum {
				nextRegime = j
				break
			}
		}
		currentRegime = nextRegime

		regimeVolMult := volMult[currentRegime]
		regimeDriftMult := driftMult[currentRegime]
		regimeMRMult := mrMult[currentRegime]
		regimeJumpMult := jumpMult[currentRegime]

		shockMag := math.Abs(lastReturn)
		currentVol = 0.85*currentVol + 0.15*dailyVol + 0.08*shockMag
		jumpReturn := 0.0
		effectiveJumpProb := cfg.JumpProb * regimeJumpMult / 252.0
		if rng.Float64() < effectiveJumpProb {
			jumpReturn = (rng.Float64() - 0.5) * 4.0 * currentVol * regimeVolMult
			if cfg.Class == acCrypto {
				jumpReturn = (rng.Float64() - 0.4) * 6.0 * currentVol * regimeVolMult
			}
		}
		dailyDrift := cfg.AnnDrift / 252.0 * regimeDriftMult
		momentumTerm := cfg.Momentum * lastReturn
		mrTerm := cfg.MRStrength * regimeMRMult * (cfg.BasePrice - price) / cfg.BasePrice
		noise := randNorm(rng) * currentVol * regimeVolMult
		ret := dailyDrift + momentumTerm + mrTerm + noise + jumpReturn
		if cfg.Class == acCrypto && math.Abs(ret/(currentVol*regimeVolMult)) > 2.5 {
			ret *= 1.8
		}
		if cfg.Class == acEquity && math.Abs(noise/(currentVol*regimeVolMult)) > 2.5 {
			ret += noise * (0.1 + rng.Float64()*0.4)
		}
		open := price + randNorm(rng)*currentVol*0.3
		close := price * (1.0 + ret)
		if close < cfg.MinPrice {
			close = cfg.MinPrice
			open = close * (1.0 + math.Abs(randNorm(rng)*currentVol*0.5))
		}
		dayRange := math.Abs(close-open) + math.Abs(randNorm(rng)*currentVol*0.8)
		mid := (open + close) / 2.0
		high := mid + dayRange/2.0
		low := mid - dayRange/2.0
		if low < cfg.MinPrice {
			low = cfg.MinPrice
		}
		dailyBars := expandIntraday(open, high, low, close, bgVol(currentVol, barsPerDay), d, barsPerDay, sym, rng)
		candles = append(candles, dailyBars...)
		price = close
		lastReturn = ret
	}
	return candles
}

func expandIntraday(open, high, low, close, barVol float64, day time.Time, barsPerDay int, sym string, rng *rand.Rand) []backtest.Candle {
	if barsPerDay <= 1 {
		return []backtest.Candle{{
			Time:   day,
			Open:   types.PriceFromFloat(open),
			High:   types.PriceFromFloat(high),
			Low:    types.PriceFromFloat(low),
			Close:  types.PriceFromFloat(close),
			Volume: math.Max(1000, math.Abs(randNorm(rng)*50000+50000)),
			Symbol: sym,
		}}
	}
	bars := make([]backtest.Candle, barsPerDay)
	highIdx := rng.IntN(barsPerDay)
	lowIdx := rng.IntN(barsPerDay)
	if highIdx == lowIdx {
		lowIdx = (highIdx + barsPerDay/2) % barsPerDay
	}
	prices := make([]float64, barsPerDay+1)
	prices[0] = open
	baseVol := barVol * 0.5
	for i := 1; i <= barsPerDay; i++ {
		t := float64(i) / float64(barsPerDay)
		drift := (close - open) * t
		bridgeNoise := randNorm(rng) * math.Sqrt(t*(1.0-t)) * barVol
		prices[i] = open + drift + bridgeNoise
		if i-1 == highIdx && prices[i] < high {
			prices[i] = high - math.Abs(randNorm(rng)*baseVol*0.1)
		}
		if i-1 == lowIdx && prices[i] > low {
			prices[i] = low + math.Abs(randNorm(rng)*baseVol*0.1)
		}
	}
	baseTime := time.Date(day.Year(), day.Month(), day.Day(), 9, 30, 0, 0, day.Location())
	barDuration := time.Duration(390/barsPerDay) * time.Minute
	for i := 0; i < barsPerDay; i++ {
		o := prices[i]
		c := prices[i+1]
		h := math.Max(o, c)
		l := math.Min(o, c)
		bars[i] = backtest.Candle{
			Time:   baseTime.Add(time.Duration(i) * barDuration),
			Open:   types.PriceFromFloat(o),
			High:   types.PriceFromFloat(h),
			Low:    types.PriceFromFloat(l),
			Close:  types.PriceFromFloat(c),
			Volume: math.Max(100, math.Abs(randNorm(rng)*float64(1000000/int64(barsPerDay))+float64(500000/int64(barsPerDay)))),
			Symbol: sym,
		}
	}
	return bars
}

func bgVol(dailyVol float64, barsPerDay int) float64 {
	if barsPerDay <= 1 {
		return dailyVol
	}
	return dailyVol / math.Sqrt(float64(barsPerDay))
}

func randNorm(rng *rand.Rand) float64 {
	u1 := rng.Float64()
	u2 := rng.Float64()
	if u1 < 1e-12 {
		u1 = 1e-12
	}
	return math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
}

func (a *backtestRepoAdapter) LoadVIXLogs(ctx context.Context, start, end time.Time) ([]backtest.VIXLog, error) {
	logs, err := a.repo.LoadVIXLogs(ctx, start, end)
	if err != nil {
		return nil, nil
	}
	out := make([]backtest.VIXLog, len(logs))
	for i, l := range logs {
		out[i] = backtest.VIXLog{
			Time:      l.Time,
			VIXValue:  l.VIXValue,
			VIXChange: l.VIXChange,
		}
	}
	return out, nil
}

func (a *backtestRepoAdapter) LoadSentimentLogs(ctx context.Context, start, end time.Time) ([]backtest.SentimentLog, error) {
	return nil, nil
}

func (a *backtestRepoAdapter) SaveBacktestResult(ctx context.Context, result *backtest.BacktestResult) error {
	br := &db.BacktestResult{
		ID:             result.Config.StrategyID,
		StrategyID:     result.Config.StrategyID,
		Config:         result.Config,
		SharpeRatio:    result.SharpeRatio,
	}
	return a.repo.SaveBacktestResult(ctx, br)
}

func (a *backtestRepoAdapter) CountCandles(ctx context.Context) (int64, error) {
	return a.repo.CountCandles(ctx)
}

func (a *backtestRepoAdapter) CountSyntheticCandles(ctx context.Context) (int64, error) {
	return a.repo.CountSyntheticCandles(ctx)
}

func (a *backtestRepoAdapter) CountRegimeLogs(ctx context.Context) (int64, error) {
	return a.repo.CountTable(ctx, "regime_logs")
}

func (a *backtestRepoAdapter) LoadUniverseSnapshots(ctx context.Context, start, end time.Time) ([]backtest.UniverseSnapshot, error) {
	snaps, err := a.repo.ListUniverseSnapshots(ctx, "00000000-0000-0000-0000-000000000001", start, end)
	if err != nil {
		return nil, err
	}
	out := make([]backtest.UniverseSnapshot, len(snaps))
	for i, snap := range snaps {
		tickers, tickerErr := a.repo.ResolveSnapshotSymbols(ctx, snap.SymbolIDs)
		if tickerErr != nil {
			tickers = []string{}
		}
		out[i] = backtest.UniverseSnapshot{
			Date:    snap.SnapshotDate,
			Symbols: tickers,
		}
	}
	return out, nil
}

func (a *backtestRepoAdapter) LoadCandlesTF(ctx context.Context, symbols []string, start, end time.Time, timeframe string) ([][]backtest.Candle, error) {
	candles, err := a.repo.LoadCandlesByTimeframe(ctx, symbols, start, end, timeframe)
	if err != nil {
		return nil, err
	}
	out := make([][]backtest.Candle, len(candles))
	for i, symCandles := range candles {
		out[i] = make([]backtest.Candle, len(symCandles))
		for j, c := range symCandles {
			out[i][j] = backtest.Candle{Time: c.Time, Open: c.Open, High: c.High, Low: c.Low, Close: c.Close, Volume: c.Volume, Symbol: c.Symbol}
		}
	}
	return out, nil
}

func (a *backtestRepoAdapter) LoadAllCandles(ctx context.Context, symbols []string, start, end time.Time, timeframe string) (map[string][]backtest.Candle, error) {
	raw, err := a.repo.LoadAllCandles(ctx, symbols, start, end, timeframe)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]backtest.Candle, len(raw))
	for ticker, dbCandles := range raw {
		candles := make([]backtest.Candle, len(dbCandles))
		for i, c := range dbCandles {
			candles[i] = backtest.Candle{Time: c.Time, Open: c.Open, High: c.High, Low: c.Low, Close: c.Close, Volume: c.Volume, Symbol: c.Symbol}
		}
		out[ticker] = candles
	}
	return out, nil
}

func (s *Server) submitMatrix(c *gin.Context) {
	var req struct {
		StrategyIDs      []string `json:"strategy_ids"`
		Symbols          []string `json:"symbols"`
		Timeframes       []string `json:"timeframes"`
		StartDate        string   `json:"start_date"`
		EndDate          string   `json:"end_date"`
		InitialCapital   float64  `json:"initial_capital"`
		DataSource       string   `json:"data_source"`
		GateProfile      string   `json:"gate_profile"`
		SizingPercent    float64  `json:"sizing_percent"`
		KellyFraction    float64  `json:"kelly_fraction"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.StrategyIDs) == 0 {
		req.StrategyIDs = []string{"ma_crossover"}
	}
	if len(req.Symbols) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbols required"})
		return
	}
	if len(req.Timeframes) == 0 {
		req.Timeframes = []string{"1d"}
	}
	if req.InitialCapital <= 0 {
		req.InitialCapital = 100000
	}
	if req.DataSource == "" {
		req.DataSource = "synthetic"
	}

	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	endDate, _ := time.Parse("2006-01-02", req.EndDate)
	if startDate.IsZero() {
		startDate = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if endDate.IsZero() {
		endDate = time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	}

	config := backtest.MatrixBacktestConfig{
		StrategyIDs:       req.StrategyIDs,
		Symbols:           req.Symbols,
		Timeframes:        req.Timeframes,
		StartDate:         startDate,
		EndDate:           endDate,
		InitialCapital:    req.InitialCapital,
		DataSource:        req.DataSource,
		GateProfile:       req.GateProfile,
		PropFirmEnabled:   true,
		SizingPercent:     req.SizingPercent,
		KellyFraction:     req.KellyFraction,
		SkipLightOptimize: false,
	}

	dbAdapter := &backtestRepoAdapter{repo: s.repo}
	combos := len(req.StrategyIDs) * len(req.Symbols) * len(req.Timeframes)
	batchID := fmt.Sprintf("matrix-%s", time.Now().Format("20060102150405"))

	progresses := make([]ComboProgress, combos)
	idx := 0
	for _, st := range req.StrategyIDs {
		for _, sym := range req.Symbols {
			for _, tf := range req.Timeframes {
				progresses[idx] = ComboProgress{StrategyID: st, Symbol: sym, Timeframe: tf, Status: "pending"}
				idx++
			}
		}
	}
	s.progressStore.Create(batchID, combos, progresses)

	go func() {
		_, err := backtest.RunMatrixConcurrent(context.Background(), dbAdapter, config,
			func(index int, status string, errMsg string, result *backtest.ComboResult) {
				switch status {
				case "running":
					s.progressStore.UpdateCombo(batchID, index, "running", "", nil)
				case "failed":
					s.progressStore.UpdateCombo(batchID, index, "failed", errMsg, nil)
				case "completed":
					if result == nil {
						return
					}
					crQuick := *result
					crQuick.EquityCurve = nil
					crQuick.Trades = nil
					s.progressStore.UpdateCombo(batchID, index, "completed", "", &crQuick)
					if s.repo != nil {
						sd := startDate
						ed := endDate
						rec := &db.BacktestRunRecord{
							StrategyID:       result.StrategyID,
							RunType:          "matrix",
							Status:           "completed",
							StrategyIDs:      []string{result.StrategyID},
							Symbols:          []string{result.Symbol},
							StartDate:        &sd,
							EndDate:          &ed,
							InitialCapital:   config.InitialCapital,
							SharpeRatio:      result.SharpeRatio,
							SortinoRatio:     result.SortinoRatio,
							MaxDrawdown:      result.MaxDrawdown,
							MaxDrawdownDur:   result.MaxDrawdownDur,
							TotalReturn:      result.TotalReturn,
							WinRate:          result.WinRate,
							ProfitFactor:     result.ProfitFactor,
							AvgTrade:         result.AvgTrade,
							AvgWin:           result.AvgWin,
							AvgLoss:          result.AvgLoss,
							NumTrades:        result.NumTrades,
							NumWins:          result.NumWins,
							NumLosses:        result.NumLosses,
							GatePassed:       result.GatePassed,
						}
						fullMetricsJSON, merr := json.Marshal(result)
						if merr == nil {
							rec.ResultsJSON = fullMetricsJSON
							if cerr := s.repo.CreateBacktestRun(context.Background(), rec); cerr != nil {
								slog.Error("matrix: failed to persist combo", "symbol", result.Symbol, "strategy", result.StrategyID, "tf", result.Timeframe, "err", cerr)
							} else {
								cr := *result
								cr.RunID = rec.ID
								cr.EquityCurve = nil
								cr.Trades = nil
								s.progressStore.UpdateCombo(batchID, index, "completed", "", &cr)
								eqJSON, _ := json.Marshal(result.EquityCurve)
								tradesJSON, _ := json.Marshal(result.Trades)
								_ = s.repo.InsertBacktestResult(context.Background(), &db.BacktestResultRecord{
									RunID:         rec.ID,
									StrategyID:    result.StrategyID,
									ResultType:    "matrix",
									TrialIndex:    0,
									SchemaVersion: 1,
									Metrics:       rec.ResultsJSON,
									EquityCurve:   eqJSON,
									Trades:        tradesJSON,
								})
							}
						}
					}
				}
			},
		)
		if err != nil {
			slog.Error("matrix failed", "batch_id", batchID, "error", err)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"batch_id":      batchID,
		"total_combos":  combos,
		"status":        "running",
	})
}

func (s *Server) getMatrixStatus(c *gin.Context) {
	id := c.Param("id")
	mp := s.progressStore.Get(id)
	if mp == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "batch not found"})
		return
	}
	c.JSON(http.StatusOK, mp)
}

func (s *Server) getMatrixResults(c *gin.Context) {
	id := c.Param("id")
	sinceSeq := 0
	if s := c.Query("since"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			sinceSeq = v
		}
	}
	results, nextSeq, _ := s.progressStore.GetSince(id, sinceSeq)
	summary := s.progressStore.GetSummary(id)
	c.JSON(http.StatusOK, gin.H{
		"summary": summary,
		"results": results,
		"seq":     nextSeq,
	})
}

func (s *Server) cancelMatrix(c *gin.Context) {
	id := c.Param("id")
	mp := s.progressStore.Get(id)
	if mp == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "batch not found"})
		return
	}
	// Mark remaining combos as cancelled
	s.progressStore.Cancel(id)
	c.JSON(http.StatusOK, gin.H{"batch_id": id, "status": "cancelled"})
}

func (s *Server) getSystemHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"engine_version": version.Engine(),
		"status":         "ok",
	})
}

func (s *Server) listBrokers(c *gin.Context) {
	if s.providerHandler != nil {
		s.providerHandler.ListProviders(c)
		return
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "provider service not available"})
}
