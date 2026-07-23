package api

import (
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
	bp, err := monitor.ReadBatchProgress("optimize_" + runID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "optimization run not found"})
		return
	}
	c.JSON(http.StatusOK, bp)
}

func (s *Server) getOptimizationResults(c *gin.Context) {
	runIDStr := c.Param("id")
	runUUID, err := uuid.Parse(runIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run ID"})
		return
	}

	var result *db.OptimizationRun
	if s.repo != nil {
		result, err = s.repo.GetOptimizationRunByID(c.Request.Context(), runUUID)
		if err != nil || result == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "optimization run not found"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"run":      result,
		"endpoint": "optimization_results",
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
		StartDate       string                          `json:"start_date"`
		EndDate         string                          `json:"end_date"`
		Timeframe       string                          `json:"timeframe"`
		Capital         float64                         `json:"capital"`
		Objective       string                          `json:"objective"`
		MaxCombinations int                             `json:"max_combinations"`
		TrainYears      int                             `json:"train_years"`
		TestYears       int                             `json:"test_years"`
		StepMonths      int                             `json:"step_months"`
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

	runID := uuid.New().String()

	go func() {
		searchSpace := make(backtest.SearchSpace)
		for name, con := range req.Constraints {
			searchSpace[name] = backtest.ParamConstraint{
				Name: name, Type: backtest.ParamContinuous,
				Min: con.Min, Max: con.Max, Step: con.Step,
			}
		}
		if len(searchSpace) == 0 {
			searchSpace = backtest.DefaultSearchSpace(req.StrategyID)
		}

		if s.repo != nil {
			s.repo.SaveOptimizationRun(c.Request.Context(), &db.OptimizationRun{
				ID: uuid.MustParse(runID), Method: "walkforward",
				ObjectiveMetric: req.Objective, TotalTrials: req.MaxCombinations,
				CreatedAt: time.Now(),
			})
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{"run_id": runID, "status": "accepted"})
}
