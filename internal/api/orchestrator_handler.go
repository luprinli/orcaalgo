package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/backtest"
	"github.com/lee-econ/orca-core/internal/db"

	"golang.org/x/sync/semaphore"
)

var orchSem = semaphore.NewWeighted(2)

type OrchestratorHandler struct {
	repo *db.Repository
	db   backtest.Database
}

func NewOrchestratorHandler(repo *db.Repository, btDB backtest.Database) *OrchestratorHandler {
	return &OrchestratorHandler{repo: repo, db: btDB}
}

type submitRunRequest struct {
	Strategies []struct {
		StrategyID string `json:"strategy_id" binding:"required"`
		Symbol     string `json:"symbol" binding:"required"`
		Timeframe  string `json:"timeframe" binding:"required"`
	} `json:"strategies" binding:"required"`
	StartDate              string  `json:"start_date" binding:"required"`
	EndDate                string  `json:"end_date" binding:"required"`
	InitialCapital         float64 `json:"initial_capital"`
	RebalanceBars          int     `json:"rebalance_bars"`
	KellyFraction          float64 `json:"kelly_fraction"`
	MaxPositionPct         float64 `json:"max_position_pct"`
	EnableCorrelationBrake bool    `json:"enable_correlation_brake"`
	CorrelationThreshold   float64 `json:"correlation_threshold"`
	FrictionModel          string  `json:"friction_model"`
}

func (h *OrchestratorHandler) SubmitMatrix(c *gin.Context) {
	var req struct {
		Sets []struct {
			Strategies []struct {
				StrategyID string `json:"strategy_id" binding:"required"`
				Symbol     string `json:"symbol" binding:"required"`
				Timeframe  string `json:"timeframe" binding:"required"`
			} `json:"strategies" binding:"required"`
		} `json:"sets" binding:"required"`
		StartDate       string  `json:"start_date" binding:"required"`
		EndDate         string  `json:"end_date" binding:"required"`
		InitialCapital  float64 `json:"initial_capital"`
		RebalanceBars   int     `json:"rebalance_bars"`
		KellyFraction   float64 `json:"kelly_fraction"`
		MaxPositionPct  float64 `json:"max_position_pct"`
		FrictionModel   string  `json:"friction_model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	parsedStart, err := parseDate(req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date"})
		return
	}
	parsedEnd, err := parseDate(req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date"})
		return
	}

	cfg := backtest.OrchMatrixConfig{
		StartDate:      parsedStart,
		EndDate:        parsedEnd,
		InitialCapital: req.InitialCapital,
		RebalanceBars:  req.RebalanceBars,
		KellyFraction:  req.KellyFraction,
		MaxPositionPct: req.MaxPositionPct,
		FrictionModel:  req.FrictionModel,
	}

	if cfg.InitialCapital <= 0 {
		cfg.InitialCapital = 100000
	}
	if cfg.RebalanceBars <= 0 {
		cfg.RebalanceBars = 20
	}
	if cfg.KellyFraction <= 0 {
		cfg.KellyFraction = 0.25
	}

	for _, set := range req.Sets {
		var strategies []backtest.OrchestratorStrategy
		for _, s := range set.Strategies {
			strategies = append(strategies, backtest.OrchestratorStrategy{
				StrategyID: s.StrategyID,
				Symbol:     s.Symbol,
				Timeframe:  s.Timeframe,
			})
		}
		cfg.Sets = append(cfg.Sets, strategies)
	}

	batchID := time.Now().Format("orchmat_20060102_150405") + "_" + fmt.Sprintf("%d", len(cfg.Sets))
	go backtest.RunOrchestratorMatrix(h.db, h.repo, cfg, batchID)

	c.JSON(http.StatusAccepted, gin.H{
		"batch_id":    batchID,
		"total_sets":  len(cfg.Sets),
		"message":     "matrix run submitted; poll GET /api/v1/orchestrator/runs for results",
	})
}

func (h *OrchestratorHandler) SubmitRun(c *gin.Context) {
	var req submitRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.InitialCapital <= 0 {
		req.InitialCapital = 100000
	}
	if req.RebalanceBars <= 0 {
		req.RebalanceBars = 20
	}
	if req.KellyFraction <= 0 {
		req.KellyFraction = 0.25
	}
	if req.CorrelationThreshold <= 0 {
		req.CorrelationThreshold = 0.6
	}
	if req.FrictionModel == "" {
		req.FrictionModel = "realistic"
	}

	cfg := backtest.OrchestratorConfig{
		InitialCapital:         req.InitialCapital,
		RebalanceBars:          req.RebalanceBars,
		KellyFraction:          req.KellyFraction,
		MaxPositionPct:         req.MaxPositionPct,
		EnableCorrelationBrake: req.EnableCorrelationBrake,
		CorrelationThreshold:   req.CorrelationThreshold,
		FrictionModel:          req.FrictionModel,
	}

	for _, s := range req.Strategies {
		cfg.Strategies = append(cfg.Strategies, backtest.OrchestratorStrategy{
			StrategyID: s.StrategyID,
			Symbol:     s.Symbol,
			Timeframe:  s.Timeframe,
		})
	}

	parsedStart, err := parseDate(req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date: " + err.Error()})
		return
	}
	parsedEnd, err := parseDate(req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date: " + err.Error()})
		return
	}
	cfg.StartDate = parsedStart
	cfg.EndDate = parsedEnd

	run := &db.OrchestrationRun{
		StartDate:      parsedStart,
		EndDate:        parsedEnd,
		InitialCapital: req.InitialCapital,
		Status:         "running",
	}

	for _, s := range req.Strategies {
		run.StrategyIDs = append(run.StrategyIDs, s.StrategyID)
		run.SymbolTFPairs = append(run.SymbolTFPairs, s.Symbol+":"+s.Timeframe)
	}

	if err := h.repo.SaveOrchestrationRun(c.Request.Context(), run); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save run: " + err.Error()})
		return
	}

	runID := run.ID
	dbAdapter := h.db

	go func() {
		defer func() {
			if r := recover(); r != nil {
				h.repo.UpdateOrchestrationRun(context.Background(), runID, "failed", nil)
			}
		}()

		if !orchSem.TryAcquire(1) {
			h.repo.UpdateOrchestrationRun(context.Background(), runID, "queued", nil)
			_ = orchSem.Acquire(context.Background(), 1)
			h.repo.UpdateOrchestrationRun(context.Background(), runID, "running", nil)
		}
		defer orchSem.Release(1)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		orch, err := backtest.NewOrchestrator(dbAdapter, cfg)
		if err != nil {
			h.repo.UpdateOrchestrationRun(context.Background(), runID, "failed", nil)
			return
		}

		for _, s := range req.Strategies {
			if err := orch.AddStrategy(s.Symbol, s.Timeframe, s.StrategyID); err != nil {
				h.repo.UpdateOrchestrationRun(context.Background(), runID, "failed", nil)
				return
			}
		}

		result, runErr := orch.Run(ctx)
		if runErr != nil {
			h.repo.UpdateOrchestrationRun(context.Background(), runID, "failed", nil)
			return
		}

		dbResult := &db.OrchestrationResult{
			PoolSharpe:     result.PoolSharpe,
			PoolSortino:    result.PoolSortino,
			PoolMaxDD:      result.PoolMaxDD,
			PoolReturnPct:  result.PoolReturnPct,
			RebalanceCosts: result.RebalanceCosts,
		}

		enrichedJSON := backtest.EnrichResultJSON(result)
		resultJSON, _ := json.Marshal(enrichedJSON)
		run.ResultJSON = resultJSON
		run.PoolSharpe = &result.PoolSharpe
		run.PoolSortino = &result.PoolSortino
		run.PoolMaxDD = &result.PoolMaxDD
		run.PoolReturnPct = &result.PoolReturnPct
		run.RebalanceCosts = &result.RebalanceCosts

		h.repo.UpdateOrchestrationRunWithJSON(context.Background(), runID, "completed", dbResult, resultJSON)

		allocEntries := make([]db.AllocationEntry, len(result.AllocationHistory))
		for i, p := range result.AllocationHistory {
			posSize := new(float64)
			allocEntries[i] = db.AllocationEntry{
				RunID:            runID,
				BarTime:          p.BarTime,
				StrategyID:       p.StrategyID,
				Weight:           p.Weight,
				AllocatedCapital: p.Capital,
				PositionSize:     posSize,
				IsActive:         p.IsActive,
			}
		}
		if len(allocEntries) > 0 {
			h.repo.SaveAllocationHistory(context.Background(), runID, allocEntries)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"run_id":  runID,
		"status":  "running",
		"message": "run submitted; use GET /api/v1/orchestrator/runs/:id to poll for completion",
	})
}

func (h *OrchestratorHandler) ListRuns(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	runs, total, err := h.repo.ListOrchestrationRuns(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"runs":  runs,
		"total": total,
	})
}

func (h *OrchestratorHandler) GetRun(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	run, err := h.repo.LoadOrchestrationRun(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}

	c.JSON(http.StatusOK, run)
}

func (h *OrchestratorHandler) GetEquity(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	run, err := h.repo.LoadOrchestrationRun(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}

	enriched := extractEnrichedResult(run.ResultJSON)
	if enriched == nil || len(enriched.PoolEquity) == 0 {
		c.JSON(http.StatusOK, []backtest.EquityPoint{})
		return
	}
	c.JSON(http.StatusOK, enriched.PoolEquity)
}

func (h *OrchestratorHandler) GetTrades(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	run, err := h.repo.LoadOrchestrationRun(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}

	enriched := extractEnrichedResult(run.ResultJSON)
	if enriched == nil || len(enriched.Trades) == 0 {
		c.JSON(http.StatusOK, []backtest.Trade{})
		return
	}
	c.JSON(http.StatusOK, enriched.Trades)
}

func (h *OrchestratorHandler) GetDailyReturns(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	run, err := h.repo.LoadOrchestrationRun(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}

	enriched := extractEnrichedResult(run.ResultJSON)
	if enriched == nil || len(enriched.DailyReturns) == 0 {
		c.JSON(http.StatusOK, []backtest.DailyReturn{})
		return
	}
	c.JSON(http.StatusOK, enriched.DailyReturns)
}

func (h *OrchestratorHandler) GetMetrics(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	run, err := h.repo.LoadOrchestrationRun(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}

	metrics := gin.H{
		"pool_sharpe":     run.PoolSharpe,
		"pool_sortino":    run.PoolSortino,
		"pool_maxdd":      run.PoolMaxDD,
		"pool_return_pct": run.PoolReturnPct,
		"rebalance_costs": run.RebalanceCosts,
		"num_trades":      0,
		"eq_final":        run.InitialCapital,
	}

	enriched := extractEnrichedResult(run.ResultJSON)
	if enriched != nil {
		metrics["num_trades"] = len(enriched.Trades)
		metrics["strategy_pnl"] = enriched.StrategyPnL
		metrics["active_count"] = len(enriched.ActiveCount)
		if len(enriched.PoolEquity) > 0 {
			metrics["eq_final"] = enriched.PoolEquity[len(enriched.PoolEquity)-1].Value
		}
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *OrchestratorHandler) GetAllocation(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	entries, err := h.repo.LoadAllocationHistory(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if entries == nil {
		entries = []db.AllocationEntry{}
	}
	c.JSON(http.StatusOK, entries)
}

func (h *OrchestratorHandler) GetCorrelation(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	run, err := h.repo.LoadOrchestrationRun(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"run_id":        id,
		"strategy_ids":  run.StrategyIDs,
		"result_json":   run.ResultJSON,
	})
}

func (h *OrchestratorHandler) CancelRun(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	if err := h.repo.UpdateOrchestrationRun(c.Request.Context(), id, "cancelled", nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"run_id": id, "status": "cancelled"})
}

func (h *OrchestratorHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/orchestrator/run", h.SubmitRun)
	r.POST("/orchestrator/matrix", h.SubmitMatrix)
	r.GET("/orchestrator/runs", h.ListRuns)
	r.GET("/orchestrator/runs/:id", h.GetRun)
	r.GET("/orchestrator/runs/:id/equity", h.GetEquity)
	r.GET("/orchestrator/runs/:id/trades", h.GetTrades)
	r.GET("/orchestrator/runs/:id/daily-returns", h.GetDailyReturns)
	r.GET("/orchestrator/runs/:id/metrics", h.GetMetrics)
	r.GET("/orchestrator/runs/:id/allocation", h.GetAllocation)
	r.GET("/orchestrator/runs/:id/correlation", h.GetCorrelation)
	r.DELETE("/orchestrator/runs/:id", h.CancelRun)
}

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

type enrichedResult struct {
	PoolEquity          []backtest.EquityPoint       `json:"pool_equity"`
	Trades             []backtest.Trade              `json:"trades"`
	DailyReturns        []backtest.DailyReturn        `json:"daily_returns"`
	StrategyPnL         map[string]float64            `json:"strategy_pnl"`
	AllocationHistory   []backtest.OrchAllocationPoint `json:"allocation_history"`
	CorrelationBreaches []backtest.BreachEvent        `json:"correlation_breaches"`
	ActiveCount         []int                          `json:"active_count"`
}

func extractEnrichedResult(raw json.RawMessage) *enrichedResult {
	if len(raw) == 0 {
		return nil
	}
	var e enrichedResult
	if err := json.Unmarshal(raw, &e); err != nil {
		return nil
	}
	return &e
}
