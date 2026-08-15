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
	"github.com/lee-econ/orca-core/internal/benchmark"
	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/metrics"
)

// benchmarkConfigJSON serializes the benchmark spec into the run's `config`
// JSONB so the benchmark choice is persisted (and hashed) with the run and can
// be used as the default by the benchmark-eval / benchmark-overlay endpoints.
func benchmarkConfigJSON(kind, symbol string) json.RawMessage {
	if kind == "" && symbol == "" {
		return nil
	}
	if kind == "" {
		kind = "equity_index"
	}
	if symbol == "" {
		symbol = "SPY"
	}
	b, _ := json.Marshal(map[string]interface{}{
		"benchmark": map[string]string{"kind": kind, "symbol": symbol},
	})
	return b
}

// benchmarkFromConfig extracts the benchmark spec from a run's config JSONB,
// returning (kind, symbol) with the equity_index/SPY defaults when absent.
func benchmarkFromConfig(config json.RawMessage) (string, string) {
	kind, symbol := "equity_index", "SPY"
	if len(config) == 0 {
		return kind, symbol
	}
	var cfg struct {
		Benchmark struct {
			Kind   string `json:"kind"`
			Symbol string `json:"symbol"`
		} `json:"benchmark"`
	}
	if json.Unmarshal(config, &cfg) != nil {
		return kind, symbol
	}
	if cfg.Benchmark.Kind != "" {
		kind = cfg.Benchmark.Kind
	}
	if cfg.Benchmark.Symbol != "" {
		symbol = cfg.Benchmark.Symbol
	}
	return kind, symbol
}

// getBacktestBenchmarkEval computes the market-based benchmark filter verdict
// for a backtest's daily returns, persists it to benchmark_evals (Phase 1), and
// returns it. The math runs in the `orca benchmark-filter` subprocess (HP #1).
func (s *Server) getBacktestBenchmarkEval(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing backtest ID"})
		return
	}

	run, err := s.repo.GetBacktestRun(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backtest run not found"})
		return
	}

	defaultKind, defaultSymbol := benchmarkFromConfig(run.Config)
	benchmarkSymbol := c.DefaultQuery("benchmark_symbol", defaultSymbol)
	kind := c.DefaultQuery("kind", defaultKind)
	if kind == "risk_free" && benchmarkSymbol == "SPY" {
		benchmarkSymbol = "risk_free_3m"
	}
	hurdle := 0.0
	if h := c.Query("hurdle"); h != "" {
		if parsed, err := strconv.ParseFloat(h, 64); err == nil {
			hurdle = parsed
		}
	}
	nTrials := 1
	if t := c.Query("n_trials"); t != "" {
		if parsed, err := strconv.Atoi(t); err == nil && parsed >= 1 {
			nTrials = parsed
		}
	}

	daily, err := s.collectDailyReturnsWithDates(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if len(daily) < 3 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "insufficient daily returns to benchmark"})
		return
	}

	start, end := daily[0].Date, daily[len(daily)-1].Date
	var bench map[string]float64
	if kind == "risk_free" {
		bench, err = s.riskFreeDailyReturns(c.Request.Context(), benchmarkSymbol, start, end)
	} else {
		bench, err = s.benchmarkDailyReturns(c.Request.Context(), benchmarkSymbol, start, end)
	}
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	strat, benchAligned := alignDailyReturns(daily, bench)
	if len(strat) < 3 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "insufficient aligned strategy/benchmark observations"})
		return
	}

	verdict, err := benchmark.Evaluate(c.Request.Context(), strat, benchAligned, kind, benchmarkSymbol, hurdle, nTrials)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "benchmark filter unavailable (orca toolchain)"})
		return
	}

	strategyID := run.StrategyID
	if strategyID == "" && len(run.StrategyIDs) > 0 {
		strategyID = run.StrategyIDs[0]
	}
	if strategyID == "" {
		strategyID = id
	}
	s.persistBenchmarkEval(c.Request.Context(), strategyID, kind, benchmarkSymbol, start, end, nTrials, verdict)

	c.JSON(http.StatusOK, verdict)
}

// collectDailyReturnsWithDates returns the strategy's daily returns with their
// calendar dates, derived from the equity curve.
func (s *Server) collectDailyReturnsWithDates(ctx context.Context, id string) ([]metrics.DailyReturn, error) {
	results, err := s.repo.GetBacktestResults(ctx, id)
	if err != nil || len(results) == 0 {
		return nil, fmt.Errorf("no backtest results found")
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
	calc := metrics.NewCalculator(0.05)
	return calc.DailyReturnsFromEquity(allEquity), nil
}

// benchmarkDailyReturns loads 1d stooq candles for a symbol and returns
// close-to-close decimal returns keyed by "2006-01-02" date.
func (s *Server) benchmarkDailyReturns(ctx context.Context, symbol string, start, end time.Time) (map[string]float64, error) {
	candles, err := s.repo.LoadCandlesByTimeframeFiltered(ctx, []string{symbol}, start, end, "1d", "stooq")
	if err != nil {
		return nil, err
	}
	if len(candles) == 0 || len(candles[0]) < 2 {
		return nil, fmt.Errorf("no benchmark candles for %s", symbol)
	}
	bars := candles[0]
	out := make(map[string]float64, len(bars)-1)
	for i := 1; i < len(bars); i++ {
		prev := bars[i-1].Close.Float64()
		cur := bars[i].Close.Float64()
		if prev > 0 {
			out[bars[i].Time.Format("2006-01-02")] = cur/prev - 1.0
		}
	}
	return out, nil
}

// riskFreeDailyReturns loads a named benchmark_series (fractional annualized
// yield) and converts it to daily risk-free returns (yield/252), keyed by date.
func (s *Server) riskFreeDailyReturns(ctx context.Context, name string, start, end time.Time) (map[string]float64, error) {
	points, err := s.repo.LoadBenchmarkSeries(ctx, name, start, end)
	if err != nil {
		return nil, err
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("no benchmark series for %s (run `orca ingest-risk-free`)", name)
	}
	out := make(map[string]float64, len(points))
	for _, p := range points {
		out[p.Time.Format("2006-01-02")] = p.Value / 252.0
	}
	return out, nil
}

// alignDailyReturns intersects strategy and benchmark returns by calendar date,
// returning aligned per-period decimal returns (strategy, benchmark).
func alignDailyReturns(daily []metrics.DailyReturn, bench map[string]float64) ([]float64, []float64) {
	var strat, benchOut []float64
	for _, d := range daily {
		if r, ok := bench[d.Date.Format("2006-01-02")]; ok {
			strat = append(strat, d.ReturnPct/100.0)
			benchOut = append(benchOut, r)
		}
	}
	return strat, benchOut
}

// persistBenchmarkEval stores the verdict as an append-only benchmark_evals row.
func (s *Server) persistBenchmarkEval(ctx context.Context, strategyID, kind, symbol string, start, end time.Time, nTrials int, v *benchmark.Verdict) {
	nt := nTrials
	_ = s.repo.InsertBenchmarkEval(ctx, db.BenchmarkEval{
		StrategyID:           strategyID,
		BenchmarkSpecHash:    benchmark.SpecHash(kind, symbol),
		BenchmarkKind:        kind,
		BenchmarkSymbols:     symbol,
		WindowStart:          start,
		WindowEnd:            end,
		InformationRatio:     v.Metrics.InformationRatio,
		AlphaAnnualized:      v.Metrics.AlphaAnnualized,
		Beta:                 v.Metrics.Beta,
		DeflatedActiveSharpe: &v.DeflatedActiveSharpe,
		NTrials:              &nt,
		Passed:               v.Passed,
	})
}
