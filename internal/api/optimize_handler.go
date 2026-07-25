package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lee-econ/orca-core/internal/backtest"
	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/monitor"
)

func (s *Server) getOptimizationStatus(c *gin.Context) {
	runID := c.Param("id")
	bp, err := monitor.ReadBatchProgress("opt_" + runID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "optimization run not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"run_id":          bp.BatchID,
		"status":          bp.Status,
		"progress":        bp.ProgressPct,
		"elapsed_seconds": bp.ElapsedS,
	})
}

func (s *Server) getOptimizationResults(c *gin.Context) {
	runIDStr := c.Param("id")
	runUUID, err := uuid.Parse(runIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run ID"})
		return
	}

	if s.repo == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not available"})
		return
	}

	result, err := s.repo.GetOptimizationRunByID(c.Request.Context(), runUUID)
	if err != nil || result == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "optimization run not found"})
		return
	}

	bestMetric := 0.0
	if result.BestMetric != nil {
		bestMetric = *result.BestMetric
	}

	c.JSON(http.StatusOK, gin.H{
		"run_id":       result.ID.String(),
		"best_params":  result.BestParams,
		"best_metric":  bestMetric,
		"total_trials": result.TotalTrials,
		"trials":       []interface{}{},
	})
}

func (s *Server) listOptimizationRuns(c *gin.Context) {
	if s.repo == nil {
		c.JSON(http.StatusOK, gin.H{"runs": []interface{}{}})
		return
	}
	runs, err := s.repo.ListOptimizationRuns(c.Request.Context(), 50, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"runs": runs})
}

func (s *Server) submitBacktestWithOptimization(c *gin.Context) {
	var req struct {
		StrategyID      string                          `json:"strategy_id"`
		Symbols         []string                        `json:"symbols"`
		Objective       string                          `json:"objective"`
		MaxCombinations int                             `json:"max_combinations"`
		TrainYears      int                             `json:"train_years"`
		TestYears       int                             `json:"test_years"`
		StepMonths      int                             `json:"step_months"`
		Capital         float64                         `json:"capital"`
		Constraints     map[string]struct {
			Min  float64 `json:"min"`
			Max  float64 `json:"max"`
			Step float64 `json:"step"`
		} `json:"parameters"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if s.backtestEngine == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "backtest engine not available"})
		return
	}

	capital := req.Capital
	if capital <= 0 {
		capital = 100000
	}
	maxCombos := req.MaxCombinations
	if maxCombos <= 0 {
		maxCombos = backtest.LightOptBudget()
	}

	runID := uuid.New()
	now := time.Now()

	repSymbols := backtest.SelectRepresentativeSymbols(req.Symbols, backtest.LightOptSymbolCount())

	monitor.WriteBatchProgress("opt_"+runID.String(), monitor.BatchProgress{
		BatchID: runID.String(), Status: "running", ProgressPct: 0,
		StartedAt: now.Format(time.RFC3339),
	})

	go func() {
		ctx := context.Background()
		lightCfg := backtest.LightOptimizeConfig{
			StrategyID:         req.StrategyID,
			Symbols:            repSymbols,
			ValidationSymbols:  backtest.DiffStrings(req.Symbols, repSymbols),
			Timeframe:          "1d",
			StartDate:          now.AddDate(-req.TrainYears-req.TestYears, 0, 0),
			EndDate:            now.AddDate(-req.TestYears, 0, 0),
			InitialCapital:     capital,
			MaxCombos:          maxCombos,
			PropFirmEnabled:    false,
			EnableCache:        true,
			PerBacktestTimeout: backtest.LightOptTimeout(),
			PlateauPatience:    backtest.LightOptPlateauPatience(),
			TrainFraction:      backtest.LightOptTrainFraction(),
			ObjectiveWeights:   backtest.LightOptWeights(),
			CacheTTL:           backtest.LightOptCacheTTL(),
		}
		if lightCfg.StartDate.IsZero() {
			lightCfg.StartDate = now.AddDate(-4, 0, 0)
		}
		if lightCfg.EndDate.Before(lightCfg.StartDate) {
			lightCfg.EndDate = now
		}

		monitor.WriteBatchProgress("opt_"+runID.String(), monitor.BatchProgress{
			BatchID: runID.String(), Status: "running", ProgressPct: 50,
			StartedAt: now.Format(time.RFC3339),
		})

		params := backtest.RunLightOptimize(ctx, s.backtestEngine.GetDB(), lightCfg)

		bestMetric := 0.0
		bestParams := make(map[string]float64)

		if params == nil {
			slog.Warn("light optimize produced no params", "run_id", runID.String())
			monitor.WriteBatchProgress("opt_"+runID.String(), monitor.BatchProgress{
				BatchID: runID.String(), Status: "failed", ProgressPct: 100,
			})
			return
		}

		bestParams = params
		btCfg := backtest.BacktestConfig{
			StrategyID:     req.StrategyID,
			Symbols:        repSymbols,
			StartDate:      lightCfg.StartDate,
			EndDate:        lightCfg.EndDate,
			InitialCapital: capital,
			Timeframe:      "1d",
			StrategyParams: params,
		}
		if result, btErr := s.backtestEngine.Run(ctx, btCfg); btErr == nil && result != nil {
			bestMetric = result.SharpeRatio
		}

		bestMetricPtr := &bestMetric
		if s.repo != nil {
			s.repo.SaveOptimizationRun(ctx, &db.OptimizationRun{
				ID: runID, Method: "light_optimize",
				ObjectiveMetric: req.Objective, TotalTrials: maxCombos,
				BestParams: bestParams, BestMetric: bestMetricPtr,
				CreatedAt: now,
			})
		}

		monitor.WriteBatchProgress("opt_"+runID.String(), monitor.BatchProgress{
			BatchID: runID.String(), Status: "completed", ProgressPct: 100,
		})
	}()

	c.JSON(http.StatusAccepted, gin.H{"run_id": runID.String(), "status": "accepted"})
}
