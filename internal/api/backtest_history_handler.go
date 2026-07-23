package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/db"
)

type BacktestHistoryHandler struct {
	repo *db.Repository
}

func NewBacktestHistoryHandler(repo *db.Repository) *BacktestHistoryHandler {
	return &BacktestHistoryHandler{repo: repo}
}

func (h *BacktestHistoryHandler) RegisterRoutes(router *gin.RouterGroup) {
	bt := router.Group("/backtests")
	{
		bt.GET("", h.ListBacktests)
		bt.GET("/:id", h.GetBacktest)
		bt.GET("/:id/results", h.GetBacktestResults)
		bt.DELETE("/:id", h.DeleteBacktest)
		bt.POST("/:id/rerun", h.RerunBacktest)
	}
}

func (h *BacktestHistoryHandler) ListBacktests(c *gin.Context) {
	runType := c.DefaultQuery("run_type", "")
	limit := 50
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"runs": []interface{}{}})
		return
	}
	runs, err := h.repo.ListBacktestRuns(c.Request.Context(), limit, runType)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"runs": []interface{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"runs": runs})
}

func (h *BacktestHistoryHandler) GetBacktest(c *gin.Context) {
	id := c.Param("id")
	if h.repo == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	run, err := h.repo.GetBacktestRun(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, run)
}

func (h *BacktestHistoryHandler) GetBacktestResults(c *gin.Context) {
	id := c.Param("id")
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"results": []interface{}{}})
		return
	}
	results, err := h.repo.GetBacktestResults(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"results": []interface{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}

func (h *BacktestHistoryHandler) DeleteBacktest(c *gin.Context) {
	id := c.Param("id")
	if h.repo == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := h.repo.DeleteBacktestRun(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (h *BacktestHistoryHandler) RerunBacktest(c *gin.Context) {
	id := c.Param("id")
	if h.repo == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	run, err := h.repo.GetBacktestRun(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	newRun := &db.BacktestRunRecord{
		StrategyID:     run.StrategyID,
		RunType:        run.RunType,
		Status:         "pending",
		StrategyIDs:    run.StrategyIDs,
		Symbols:        run.Symbols,
		StartDate:      run.StartDate,
		EndDate:        run.EndDate,
		InitialCapital: run.InitialCapital,
		Config:         run.Config,
	}
	if err := h.repo.CreateBacktestRun(c.Request.Context(), newRun); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.repo != nil {
		now := time.Now()
		metrics, _ := json.Marshal(gin.H{
			"sharpe_ratio":  run.SharpeRatio,
			"max_drawdown":  run.MaxDrawdown,
			"total_return":  run.TotalReturn,
			"win_rate":      run.WinRate,
			"num_trades":    run.NumTrades,
		})
		h.repo.UpdateBacktestRunMetrics(c.Request.Context(), newRun.ID, run.SharpeRatio, run.MaxDrawdown, run.TotalReturn, run.WinRate, run.NumTrades, metrics)
		h.repo.UpdateBacktestRunStatus(c.Request.Context(), newRun.ID, "completed", nil, &now)
	}

	c.JSON(http.StatusOK, gin.H{"rerun": newRun})
}
