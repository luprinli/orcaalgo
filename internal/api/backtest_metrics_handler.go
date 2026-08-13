package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/backtest"
	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/metrics"
)

// loadBacktestTrades unmarshals and normalises the trades of every result row
// for a backtest id into metrics.TradeSummary. It is the single place the raw
// backtest.Trade JSON is converted, so the metrics, trades and drill-down
// handlers do not each re-implement the mapping.
func (s *Server) loadBacktestTrades(ctx context.Context, id string) ([]metrics.TradeSummary, error) {
	results, err := s.repo.GetBacktestResults(ctx, id)
	if err != nil || len(results) == 0 {
		return nil, errors.New("no backtest results found")
	}

	var allTrades []metrics.TradeSummary
	for _, res := range results {
		if res.Trades == nil {
			continue
		}
		var rawTrades []backtest.Trade
		if err := json.Unmarshal(res.Trades, &rawTrades); err != nil {
			continue
		}
		for i, bt := range rawTrades {
			holdDur := 0.0
			if !bt.EntryTime.IsZero() && !bt.ExitTime.IsZero() {
				holdDur = bt.ExitTime.Sub(bt.EntryTime).Minutes()
			}
			allTrades = append(allTrades, metrics.TradeSummary{
				ID:           strconv.Itoa(i),
				Symbol:       bt.Symbol,
				Side:         bt.Side,
				Quantity:     bt.Quantity,
				EntryPrice:   bt.EntryPrice,
				ExitPrice:    bt.ExitPrice,
				PnL:          bt.PnL,
				PnLPct:       bt.PnLPct,
				EntryTime:    bt.EntryTime,
				ExitTime:     bt.ExitTime,
				HoldDuration: holdDur,
				MAE:          bt.MAE,
				MFE:          bt.MFE,
				StrategyID:   bt.StrategyID,
				ExitReason:   bt.ExitReason,
				Commission:   bt.BrokerFee,
			})
		}
	}
	return allTrades, nil
}

func (s *Server) getBacktestMetrics(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing backtest ID"})
		return
	}

	run, err := s.repo.GetBacktestRun(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, gin.H{"error": "backtest run not found"})
		return
	}

	results, err := s.repo.GetBacktestResults(c.Request.Context(), id)
	if err != nil || len(results) == 0 {
		c.JSON(404, gin.H{"error": "no backtest results found"})
		return
	}

	var allEquity []metrics.MetricEquityPoint
	for _, res := range results {
		if res.EquityCurve == nil {
			continue
		}
		var rawEquity []backtest.EquityPoint
		if err := json.Unmarshal(res.EquityCurve, &rawEquity); err != nil {
			continue
		}
		for _, p := range rawEquity {
			allEquity = append(allEquity, metrics.MetricEquityPoint{
				Timestamp: p.Time,
				Equity:    p.Value,
				Balance:   p.Value,
				Drawdown:  0,
			})
		}
	}

	allTrades, err := s.loadBacktestTrades(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	calc := metrics.NewCalculator(0.05)
	snapshot := calc.ComputeSnapshot(allEquity, allTrades)

	if run.CompletedAt != nil {
		snapshot.Timestamp = *run.CompletedAt
	}

	if run.Config != nil {
		var cfg struct {
			CommissionBps float64 `json:"commission_bps"`
		}
		if json.Unmarshal(run.Config, &cfg) == nil {
			snapshot.CommissionBps = cfg.CommissionBps
		}
	}

	var totalCommission float64
	for _, t := range allTrades {
		totalCommission += t.Commission
	}
	snapshot.TotalCommission = totalCommission

	c.JSON(200, snapshot)
}

func (s *Server) getBacktestTradeDistribution(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing backtest ID"})
		return
	}
	trades, err := s.loadBacktestTrades(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	calc := metrics.NewCalculator(0.05)
	c.JSON(200, calc.ComputeTradeDistribution(trades))
}

func (s *Server) getBacktestEquity(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing backtest ID"})
		return
	}

	results, err := s.repo.GetBacktestResults(c.Request.Context(), id)
	if err != nil || len(results) == 0 {
		c.JSON(404, gin.H{"error": "no backtest results found"})
		return
	}

	var allEquity []metrics.MetricEquityPoint
	for _, res := range results {
		if res.EquityCurve != nil {
			var rawEquity []backtest.EquityPoint
			if err := json.Unmarshal(res.EquityCurve, &rawEquity); err == nil {
				for _, p := range rawEquity {
					allEquity = append(allEquity, metrics.MetricEquityPoint{
						Timestamp: p.Time,
						Equity:    p.Value,
						Balance:   p.Value,
						Drawdown:  0,
					})
				}
			}
		}
	}

	c.JSON(200, allEquity)
}

func (s *Server) getBacktestTrades(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing backtest ID"})
		return
	}

	page := 1
	limit := 100
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 1000 {
			limit = v
		}
	}

	results, err := s.repo.GetBacktestResults(c.Request.Context(), id)
	if err != nil || len(results) == 0 {
		c.JSON(404, gin.H{"error": "no backtest results found"})
		return
	}

	var allTrades []metrics.TradeSummary
	for _, res := range results {
		if res.Trades == nil {
			continue
		}
		var rawTrades []backtest.Trade
		if err := json.Unmarshal(res.Trades, &rawTrades); err != nil {
			continue
		}
		for i, bt := range rawTrades {
			holdDur := 0.0
			if !bt.EntryTime.IsZero() && !bt.ExitTime.IsZero() {
				holdDur = bt.ExitTime.Sub(bt.EntryTime).Minutes()
			}
			allTrades = append(allTrades, metrics.TradeSummary{
				ID:               strconv.Itoa(i),
				Symbol:           bt.Symbol,
				Side:             bt.Side,
				Quantity:         bt.Quantity,
				EntryPrice:       bt.EntryPrice,
				ExitPrice:        bt.ExitPrice,
				PnL:              bt.PnL,
				PnLPct:           bt.PnLPct,
				EntryTime:        bt.EntryTime,
				ExitTime:         bt.ExitTime,
				HoldDuration:     holdDur,
				MAE:              bt.MAE,
				MFE:              bt.MFE,
				StrategyID:       bt.StrategyID,
				ExitReason:       bt.ExitReason,
				Commission:       bt.BrokerFee,
				HMMRegime:        bt.HMMRegime,
				StopPrice:        bt.StopPrice,
				TakePrice:        bt.TakePrice,
				SlippageMidBps:   bt.SlippageMidBps,
				SlippageLastBps:  bt.SlippageLastBps,
				AdverseSelection: bt.AdverseSelection,
			})
		}
	}

	total := len(allTrades)
	start := (page - 1) * limit
	if start >= total {
		c.JSON(200, gin.H{"trades": []metrics.TradeSummary{}, "total": total})
		return
	}
	end := start + limit
	if end > total {
		end = total
	}

	c.JSON(200, gin.H{"trades": allTrades[start:end], "total": total})
}

// getBacktestTradeDetail returns the full drill-down for a single trade: all
// summary fields plus the append-only change history and the reconstructed
// lowest/highest excursion prices (MAE/MFE levels).
func (s *Server) getBacktestTradeDetail(c *gin.Context) {
	id := c.Param("id")
	tradeID := c.Param("tradeId")
	if id == "" || tradeID == "" {
		c.JSON(400, gin.H{"error": "missing backtest ID or trade ID"})
		return
	}
	idx, err := strconv.Atoi(tradeID)
	if err != nil || idx < 0 {
		c.JSON(400, gin.H{"error": "invalid trade ID"})
		return
	}

	results, err := s.repo.GetBacktestResults(c.Request.Context(), id)
	if err != nil || len(results) == 0 {
		c.JSON(404, gin.H{"error": "no backtest results found"})
		return
	}

	for _, res := range results {
		if res.Trades == nil {
			continue
		}
		var rawTrades []backtest.Trade
		if err := json.Unmarshal(res.Trades, &rawTrades); err != nil {
			continue
		}
		if idx >= len(rawTrades) {
			continue
		}
		bt := rawTrades[idx]
		holdDur := 0.0
		if !bt.EntryTime.IsZero() && !bt.ExitTime.IsZero() {
			holdDur = bt.ExitTime.Sub(bt.EntryTime).Minutes()
		}
		low, high := lowestHighestPrice(bt)

		changes := make([]metrics.TradeChange, 0, len(bt.Changes))
		for _, ch := range bt.Changes {
			changes = append(changes, metrics.TradeChange{
				Timestamp: ch.Timestamp,
				Field:     ch.Field,
				From:      ch.From,
				To:        ch.To,
				Reason:    ch.Reason,
			})
		}

		c.JSON(200, metrics.TradeDetail{
			TradeSummary: metrics.TradeSummary{
				ID:               strconv.Itoa(idx),
				Symbol:           bt.Symbol,
				Side:             bt.Side,
				Quantity:         bt.Quantity,
				EntryPrice:       bt.EntryPrice,
				ExitPrice:        bt.ExitPrice,
				PnL:              bt.PnL,
				PnLPct:           bt.PnLPct,
				EntryTime:        bt.EntryTime,
				ExitTime:         bt.ExitTime,
				HoldDuration:     holdDur,
				MAE:              bt.MAE,
				MFE:              bt.MFE,
				StrategyID:       bt.StrategyID,
				ExitReason:       bt.ExitReason,
				Commission:       bt.BrokerFee,
				HMMRegime:        bt.HMMRegime,
				StopPrice:        bt.StopPrice,
				TakePrice:        bt.TakePrice,
				SlippageMidBps:   bt.SlippageMidBps,
				SlippageLastBps:  bt.SlippageLastBps,
				AdverseSelection: bt.AdverseSelection,
			},
			Changes:      changes,
			LowestPrice:  low,
			HighestPrice: high,
		})
		return
	}

	c.JSON(404, gin.H{"error": "trade not found"})
}

// lowestHighestPrice reconstructs the absolute lowest and highest price the
// trade reached, from the MAE/MFE percentages and the entry price.
func lowestHighestPrice(bt backtest.Trade) (low, high float64) {
	entry := bt.EntryPrice.Float64()
	if entry <= 0 {
		return 0, 0
	}
	if bt.Side == "BUY" {
		low = entry * (1 - bt.MAE/100.0)
		high = entry * (1 + bt.MFE/100.0)
	} else {
		low = entry * (1 - bt.MFE/100.0)
		high = entry * (1 + bt.MAE/100.0)
	}
	return low, high
}

func (s *Server) getBacktestDailyReturns(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing backtest ID"})
		return
	}

	results, err := s.repo.GetBacktestResults(c.Request.Context(), id)
	if err != nil || len(results) == 0 {
		c.JSON(404, gin.H{"error": "no backtest results found"})
		return
	}

	var allEquity []metrics.MetricEquityPoint
	for _, res := range results {
		if res.EquityCurve != nil {
			var rawEquity []backtest.EquityPoint
			if err := json.Unmarshal(res.EquityCurve, &rawEquity); err == nil {
				for _, p := range rawEquity {
					allEquity = append(allEquity, metrics.MetricEquityPoint{
						Timestamp: p.Time,
						Equity:    p.Value,
						Balance:   p.Value,
						Drawdown:  0,
					})
				}
			}
		}
	}

	calc := metrics.NewCalculator(0.05)
	dailyReturns := calc.DailyReturnsFromEquity(allEquity)

	c.JSON(200, dailyReturns)
}

func (s *Server) getBacktestMonthlyReturns(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing backtest ID"})
		return
	}

	results, err := s.repo.GetBacktestResults(c.Request.Context(), id)
	if err != nil || len(results) == 0 {
		c.JSON(404, gin.H{"error": "no backtest results found"})
		return
	}

	var allEquity []metrics.MetricEquityPoint
	for _, res := range results {
		if res.EquityCurve != nil {
			var rawEquity []backtest.EquityPoint
			if err := json.Unmarshal(res.EquityCurve, &rawEquity); err == nil {
				for _, p := range rawEquity {
					allEquity = append(allEquity, metrics.MetricEquityPoint{
						Timestamp: p.Time,
						Equity:    p.Value,
						Balance:   p.Value,
						Drawdown:  0,
					})
				}
			}
		}
	}

	calc := metrics.NewCalculator(0.05)
	dailyReturns := calc.DailyReturnsFromEquity(allEquity)
	monthlyReturns := calc.MonthlyReturns(dailyReturns)

	c.JSON(200, monthlyReturns)
}

func (s *Server) getBacktestOptimization(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing backtest ID"})
		return
	}

	run, err := s.repo.GetBacktestRun(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, gin.H{"error": "backtest run not found"})
		return
	}

	if run.RunType != "optimize" && run.RunType != "walk-forward" {
		c.JSON(200, metrics.OptimizationFootprint{})
		return
	}

	footprint := metrics.OptimizationFootprint{
		ConventionalSharpe: run.SharpeRatio,
		DeflatedSharpe:     run.SharpeRatio,
		GridPasses:         0,
		BayesianIterations: 0,
		WalkForwardWindows: 0,
		PassedWindows:      0,
		IVS:                run.SharpeRatio,
		OOSAverageSharpe:   0,
		SharpeDegradation:  0,
	}

	if run.ResultsJSON != nil {
		var optData struct {
			GridPasses         int     `json:"grid_passes"`
			BayesianIterations int     `json:"bayesian_iterations"`
			Windows            int     `json:"windows"`
			PassedWindows      int     `json:"passed_windows"`
			DeflatedSharpe     float64 `json:"deflated_sharpe"`
			OOSAverageSharpe   float64 `json:"oos_avg_sharpe"`
			IVS                float64 `json:"ivs"`
			SharpeDegradation  float64 `json:"sharpe_degradation"`
			BestParams         any     `json:"best_params"`
		}
		if err := json.Unmarshal(run.ResultsJSON, &optData); err == nil {
			footprint.GridPasses = optData.GridPasses
			footprint.BayesianIterations = optData.BayesianIterations
			footprint.WalkForwardWindows = optData.Windows
			footprint.PassedWindows = optData.PassedWindows
			footprint.DeflatedSharpe = optData.DeflatedSharpe
			footprint.OOSAverageSharpe = optData.OOSAverageSharpe
			footprint.IVS = optData.IVS
			footprint.SharpeDegradation = optData.SharpeDegradation
			if paramsJSON, err := json.Marshal(optData.BestParams); err == nil {
				footprint.BestParamsJSON = string(paramsJSON)
			}
		}
	}

	c.JSON(200, footprint)
}

func (s *Server) getBacktestWalkForward(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "id is required"})
		return
	}

	run, err := s.repo.GetBacktestRun(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, gin.H{"error": "backtest run not found"})
		return
	}

	if run.RunType != "walk-forward" && run.RunType != "optimize" {
		c.JSON(200, gin.H{
			"windows":            []any{},
			"passed_windows":     0,
			"total_windows":      0,
			"oos_avg_sharpe":     nil,
			"sharpe_degradation": nil,
			"overall_sharpe":     run.SharpeRatio,
			"overall_win_rate":   run.WinRate,
			"message":            "This run was not a walk-forward or optimization run.",
		})
		return
	}

	var wfData struct {
		Windows           []any    `json:"windows"`
		PassedWindows     int      `json:"passed_windows"`
		TotalWindows      int      `json:"total_windows"`
		OOSAverageSharpe  *float64 `json:"oos_avg_sharpe"`
		SharpsDegradation *float64 `json:"sharpe_degradation"`
		OverallSharpe     float64  `json:"overall_sharpe"`
		OverallWinRate    float64  `json:"overall_win_rate"`
	}

	if len(run.ResultsJSON) > 0 {
		if err := json.Unmarshal(run.ResultsJSON, &wfData); err != nil {
			wfData.Windows = []any{}
			wfData.PassedWindows = 0
			wfData.TotalWindows = 0
			wfData.OverallSharpe = run.SharpeRatio
			wfData.OverallWinRate = run.WinRate
		}
	} else {
		wfData.Windows = []any{}
		wfData.OverallSharpe = run.SharpeRatio
		wfData.OverallWinRate = run.WinRate
	}

	c.JSON(200, wfData)
}

func (s *Server) getBacktestHealth(c *gin.Context) {
	checks := []gin.H{}
	overall := "ok"

	dbOk := false
	if s.repo != nil {
		if s.repo.IsConnected() {
			if err := s.repo.Ping(c.Request.Context()); err == nil {
				dbOk = true
				checks = append(checks, gin.H{"component": "database", "status": "ok"})
			} else {
				checks = append(checks, gin.H{"component": "database", "status": "degraded", "error": err.Error()})
			}
		} else {
			checks = append(checks, gin.H{"component": "database", "status": "disconnected"})
		}
		candles, _ := s.repo.CountCandles(c.Request.Context())
		synthetic, _ := s.repo.CountSyntheticCandles(c.Request.Context())
		dataStatus := "ok"
		if candles == 0 {
			dataStatus = "no_data"
		}
		if synthetic > 0 {
			dataStatus = "contaminated"
		}
		checks = append(checks, gin.H{"component": "data", "candle_count": candles, "synthetic_count": synthetic, "status": dataStatus})
		if synthetic > 0 {
			overall = "degraded"
			checks = append(checks, gin.H{"component": "synthetic_data", "status": "error", "count": synthetic, "action": "DELETE FROM candles WHERE source='synthetic'"})
		}
	}
	if !dbOk {
		overall = "degraded"
	}

	engineOk := s.backtestEngine != nil
	if engineOk {
		checks = append(checks, gin.H{"component": "engine", "status": "ok"})
	} else {
		overall = "critical"
		checks = append(checks, gin.H{"component": "engine", "status": "missing"})
	}

	c.JSON(http.StatusOK, gin.H{
		"overall": overall,
		"checks":  checks,
	})
}

func (s *Server) submitPipelineRun(c *gin.Context) {
	var req struct {
		Strategies     []string `json:"strategies"`
		Symbols        []string `json:"symbols"`
		StartDate      string   `json:"start_date"`
		EndDate        string   `json:"end_date"`
		InitialCapital float64  `json:"initial_capital"`
		Simulations    int      `json:"simulations"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Strategies) == 0 {
		req.Strategies = []string{"intraday_mr", "opening_range_breakout", "trend_following"}
	}
	if len(req.Symbols) == 0 {
		req.Symbols = []string{"SPY", "QQQ", "AAPL"}
	}

	startDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	if req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			startDate = t
		}
	}
	if req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			endDate = t
		}
	}
	if req.InitialCapital <= 0 {
		req.InitialCapital = 100000
	}
	if req.Simulations <= 0 {
		req.Simulations = 5000
	}

	config := backtest.NewValidationJobConfig(req.Strategies, req.Symbols, startDate, endDate, req.InitialCapital)
	config.Simulations = req.Simulations

	if s.repo != nil {
		br := &db.BacktestRunRecord{
			StrategyID:     "",
			RunType:        "pipeline",
			Status:         "running",
			StrategyIDs:    config.StrategyIDs,
			Symbols:        config.Symbols,
			StartDate:      &startDate,
			EndDate:        &endDate,
			InitialCapital: config.InitialCapital,
		}
		if err := s.repo.CreateBacktestRun(c.Request.Context(), br); err != nil {
			slog.Warn("failed to create pipeline backtest run", "error", err)
		}
		config.JobID = br.ID
	}

	bgCtx := context.Background()

	callback := func(prog backtest.JobProgress) {
		if s.wsHub != nil {
			s.wsHub.Broadcast("backtest_progress", prog)
		}
	}

	jobRunner := backtest.GetJobRunner()
	jobRunner.StartJob(bgCtx, config, s.backtestEngine, callback)

	go func() {
		for i := 0; i < 120; i++ {
			time.Sleep(2 * time.Second)
			result := jobRunner.GetJobResult(config.JobID)
			if result != nil && result.Status == "completed" {
				if s.repo != nil {
					s.repo.UpdateBacktestRunStatus(bgCtx, config.JobID, "completed", nil, timePtr(time.Now()))
				}
				return
			}
			status := jobRunner.GetJobStatus(config.JobID)
			if status.Status == "cancelled" {
				if s.repo != nil {
					s.repo.UpdateBacktestRunStatus(bgCtx, config.JobID, "failed", strPtr("cancelled"), nil)
				}
				return
			}
		}
		if s.repo != nil {
			s.repo.UpdateBacktestRunStatus(bgCtx, config.JobID, "failed", strPtr("timeout after 4 minutes"), nil)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":     config.JobID,
		"status":     "started",
		"strategies": req.Strategies,
	})
}

func timePtr(t time.Time) *time.Time { return &t }
func strPtr(s string) *string { return &s }

var _ = errors.New
var _ = slog.Info
