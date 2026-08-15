package backtest

import (
	"context"
	"math"
	"math/rand/v2"
	"sort"
	"time"

	"golang.org/x/sync/errgroup"
)

type MonteCarloResult = MCResult

type MCConfig struct {
	Iterations int   `json:"iterations"`
	BarsPerSim int   `json:"bars_per_sim"`
	Seed       int64 `json:"seed"`
}

type MCIterationResult struct {
	PnlPct   float64 `json:"pnl_pct"`
	MaxDDPct float64 `json:"max_dd_pct"`
}

type MCSummary struct {
	NumSimulations  int     `json:"num_simulations"`
	NumDays         int     `json:"num_days"`
	AvgPnlPct       float64 `json:"avg_pnl_pct"`
	MedianPnlPct    float64 `json:"median_pnl_pct"`
	P5PnlPct        float64 `json:"p5_pnl_pct"`
	P10PnlPct       float64 `json:"p10_pnl_pct"`
	AvgMaxDDPct     float64 `json:"avg_max_dd_pct"`
	MedianMaxDDPct  float64 `json:"median_max_dd_pct"`
	P95MaxDDPct     float64 `json:"p95_max_dd_pct"`
	BustProbability float64 `json:"bust_probability"`
}

type MCResult struct {
	Config         MCConfig             `json:"config"`
	Iterations     []MCIterationResult  `json:"iterations"`
	Summary        MCSummary            `json:"summary"`
	PassProbability float64             `json:"pass_probability"`
	CreatedAt      time.Time            `json:"created_at"`
}

func newMCResult(cfg MCConfig, iters []MCIterationResult) *MCResult {
	summary := computeMCSummary(iters, cfg)
	passProb := 100.0
	if summary.BustProbability > 0 {
		passProb = (1.0 - summary.BustProbability) * 100.0
	}
	return &MCResult{
		Config:         cfg,
		Iterations:     iters,
		Summary:        summary,
		PassProbability: passProb,
		CreatedAt:      time.Now(),
	}
}

func computeMCSummary(iterations []MCIterationResult, cfg MCConfig) MCSummary {
	s := MCSummary{
		NumSimulations: cfg.Iterations,
		NumDays:        cfg.BarsPerSim,
	}

	if len(iterations) == 0 {
		return s
	}

	sortedDD := make([]float64, 0, len(iterations))
	for _, it := range iterations {
		if it.MaxDDPct < 100 {
			sortedDD = append(sortedDD, it.MaxDDPct)
		}
	}

	sortedIterations := make([]MCIterationResult, len(iterations))
	copy(sortedIterations, iterations)
	sort.Slice(sortedIterations, func(i, j int) bool {
		return sortedIterations[i].PnlPct < sortedIterations[j].PnlPct
	})

	var sumPnl float64
	bustCount := 0
	for _, it := range sortedIterations {
		sumPnl += it.PnlPct
		if it.PnlPct < 0 || it.MaxDDPct > 20.0 {
			bustCount++
		}
	}

	n := len(sortedIterations)
	s.AvgPnlPct = sumPnl / float64(n)
	s.MedianPnlPct = sortedIterations[n/2].PnlPct
	s.P5PnlPct = sortedIterations[max(0, n*5/100)].PnlPct
	s.P10PnlPct = sortedIterations[max(0, n*10/100)].PnlPct
	s.BustProbability = float64(bustCount) / float64(n)

	if len(sortedDD) > 0 {
		sort.Float64s(sortedDD)
		var sumDD float64
		for _, d := range sortedDD {
			sumDD += d
		}
		nd := len(sortedDD)
		s.AvgMaxDDPct = sumDD / float64(nd)
		s.MedianMaxDDPct = sortedDD[nd/2]
		s.P95MaxDDPct = sortedDD[nd*95/100]
	}

	return s
}

func RunMonteCarlo(
	baseResult *BacktestResult,
	cfg MCConfig,
) (*MCResult, error) {
	if cfg.Iterations <= 0 {
		cfg.Iterations = 500
	}
	if cfg.BarsPerSim <= 0 {
		cfg.BarsPerSim = 252
	}

	dailyReturns := extractReturns(baseResult.DailyReturns)
	if len(dailyReturns) < 2 {
		return nil, ErrInsufficientReturns
	}

	var (
		iters = make([]MCIterationResult, cfg.Iterations)
		g     errgroup.Group
	)

	numWorkers := 4
	chunkSize := (cfg.Iterations + numWorkers - 1) / numWorkers

	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		end := min(start+chunkSize, cfg.Iterations)
		if start >= end {
			continue
		}

		g.Go(func() error {
			localRng := rand.New(rand.NewPCG(uint64(start), uint64(cfg.Seed)))
			for i := start; i < end; i++ {
				path := bootstrapPath(dailyReturns, cfg.BarsPerSim, localRng)
				pnlPct, maxDD := computePathMetrics(path)
				iters[i] = MCIterationResult{
					PnlPct:   pnlPct,
					MaxDDPct: maxDD,
				}
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return newMCResult(cfg, iters), nil
}

func extractReturns(daily []DailyReturn) []float64 {
	out := make([]float64, 0, len(daily))
	for _, d := range daily {
		if !math.IsNaN(d.Return) && !math.IsInf(d.Return, 0) {
			out = append(out, d.Return)
		}
	}
	return out
}

func bootstrapPath(returns []float64, bars int, rng *rand.Rand) []float64 {
	return bootstrapBlockPath(returns, bars, rng, 7)
}

func bootstrapBlockPath(returns []float64, bars int, rng *rand.Rand, blockLen int) []float64 {
	if blockLen <= 1 || blockLen > len(returns)/4 {
		blockLen = 1
	}
	path := make([]float64, bars)
	equity := 1.0
	busted := false

	n := len(returns)
	for i := 0; i < bars; {
		start := rng.IntN(n)
		for j := 0; j < blockLen && i < bars; j++ {
			idx := (start + j) % n
			equity *= (1.0 + returns[idx])
			if equity <= 0 {
				equity = 0.0001
				if !busted {
					busted = true
				}
			}
			if busted {
				equity = 0.0001
			}
			path[i] = equity
			i++
		}
	}
	return path
}

func computePathMetrics(path []float64) (pnlPct float64, maxDDPct float64) {
	if len(path) == 0 {
		return 0, 0
	}

	busted := false
	for _, e := range path {
		if e == 0.0001 {
			busted = true
			break
		}
	}
	if busted {
		return -100.0, 100.0
	}

	startEquity := 1.0
	finalEquity := path[len(path)-1]
	pnlPct = ((finalEquity - startEquity) / startEquity) * 100.0

	peak := 1.0
	for _, e := range path {
		if e > peak {
			peak = e
		}
		dd := 0.0
		if peak > 0 {
			dd = ((peak - e) / peak) * 100.0
		}
		if dd > maxDDPct {
			maxDDPct = dd
		}
	}

	return pnlPct, maxDDPct
}

var ErrInsufficientReturns = errInsufficientReturns{}

type errInsufficientReturns struct{}

func (e errInsufficientReturns) Error() string {
	return "insufficient daily returns for Monte Carlo simulation (need at least 2)"
}

func MonteCarloFromTrades(trades []Trade, simulations int, initialCapital float64) MCConfig {
	cfg := MCConfig{
		Iterations: simulations,
		BarsPerSim: 252,
		Seed:       time.Now().UnixNano(),
	}
	if cfg.Iterations <= 0 {
		cfg.Iterations = 500
	}
	if len(trades) > 0 {
		cfg.BarsPerSim = len(trades)
	}
	return cfg
}

func RunMonteCarloFromTrades(trades []Trade, simulations int, initialCapital float64) (*MonteCarloResult, error) {
	if len(trades) < 2 {
		return nil, ErrInsufficientReturns
	}

	returns := make([]float64, 0, len(trades))
	for _, t := range trades {
		pnlPct := t.PnLPct
		if pnlPct == 0 && t.PnL != 0 {
			pnlPct = t.PnL / initialCapital * 100.0
		}
		if !math.IsNaN(pnlPct) && !math.IsInf(pnlPct, 0) {
			returns = append(returns, pnlPct/100.0)
		}
	}

	if len(returns) < 2 {
		return nil, ErrInsufficientReturns
	}

	cfg := MCConfig{
		Iterations: simulations,
		BarsPerSim: len(returns),
		Seed:       time.Now().UnixNano(),
	}
	if cfg.Iterations <= 0 {
		cfg.Iterations = 500
	}

	var (
		iters = make([]MCIterationResult, cfg.Iterations)
		g     errgroup.Group
	)

	numWorkers := 4
	chunkSize := (cfg.Iterations + numWorkers - 1) / numWorkers

	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		end := min(start+chunkSize, cfg.Iterations)
		if start >= end {
			continue
		}

		g.Go(func() error {
			localRng := rand.New(rand.NewPCG(uint64(start), uint64(cfg.Seed)))
			for i := start; i < end; i++ {
				path := bootstrapPath(returns, cfg.BarsPerSim, localRng)
				pnlPct, maxDD := computePathMetrics(path)
				iters[i] = MCIterationResult{
					PnlPct:   pnlPct,
					MaxDDPct: maxDD,
				}
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return newMCResult(cfg, iters), nil
}

func RunMonteCarloWithContext(ctx context.Context, cfg MCConfig) (*MonteCarloResult, error) {
	if cfg.Iterations <= 0 {
		cfg.Iterations = 500
	}
	if cfg.BarsPerSim <= 0 {
		cfg.BarsPerSim = 252
	}

	var (
		iters = make([]MCIterationResult, cfg.Iterations)
		g     errgroup.Group
	)

	numWorkers := 4
	chunkSize := (cfg.Iterations + numWorkers - 1) / numWorkers

	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		end := min(start+chunkSize, cfg.Iterations)
		if start >= end {
			continue
		}

		g.Go(func() error {
			localRng := rand.New(rand.NewPCG(uint64(start), uint64(cfg.Seed)))
			for i := start; i < end; i++ {
				path := randomWalkPath(cfg.BarsPerSim, localRng)
				pnlPct, maxDD := computePathMetrics(path)
				iters[i] = MCIterationResult{
					PnlPct:   pnlPct,
					MaxDDPct: maxDD,
				}
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return newMCResult(cfg, iters), nil
}

func randomWalkPath(bars int, rng *rand.Rand) []float64 {
	path := make([]float64, bars)
	equity := 1.0
	for i := 0; i < bars; i++ {
		equity *= (1.0 + (rng.Float64()-0.5)*0.02)
		if equity <= 0 {
			equity = 0.0001
		}
		path[i] = equity
	}
	return path
}
