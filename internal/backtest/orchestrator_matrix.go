package backtest

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/lee-econ/orca-core/internal/db"
)

type OrchMatrixConfig struct {
	Sets          [][]OrchestratorStrategy `json:"sets"`
	StartDate     time.Time                `json:"start_date"`
	EndDate       time.Time                `json:"end_date"`
	InitialCapital float64                 `json:"initial_capital"`
	RebalanceBars  int                     `json:"rebalance_bars"`
	KellyFraction  float64                 `json:"kelly_fraction"`
	MaxPositionPct float64                 `json:"max_position_pct"`
	FrictionModel  string                  `json:"friction_model"`
}

type OrchMatrixResult struct {
	SetIndex   int                       `json:"set_index"`
	Strategies []OrchestratorStrategy    `json:"strategies"`
	PoolSharpe float64                   `json:"pool_sharpe"`
	PoolMaxDD  float64                   `json:"pool_maxdd"`
	PoolReturn float64                   `json:"pool_return_pct"`
	NumTrades  int                       `json:"num_trades"`
	StrategyPnL map[string]float64       `json:"strategy_pnl"`
	Status     string                    `json:"status"`
	Error      string                    `json:"error,omitempty"`
	RunID      string                    `json:"run_id,omitempty"`
	Completed  int                       `json:"completed"`
	Total      int                       `json:"total"`
}

type OrchMatrixStreamResult struct {
	Results    []OrchMatrixResult `json:"results"`
	Telemetry  OrchMatrixTelemetry `json:"telemetry"`
	Seq        int                `json:"seq"`
}

type OrchMatrixTelemetry struct {
	Total       int     `json:"total"`
	Completed   int     `json:"completed"`
	BestSharpe  float64 `json:"best_sharpe"`
	BestSet     int     `json:"best_set"`
	Status      string  `json:"status"`
}

type orchMatrixJob struct {
	index int
	set   []OrchestratorStrategy
}

func RunOrchestratorMatrix(dbAdapter Database, repo *db.Repository, cfg OrchMatrixConfig, batchID string) {
	results := make([]OrchMatrixResult, len(cfg.Sets))
	resultsBySeq := make([]OrchMatrixResult, 0)
	var mu sync.Mutex
	var seq int

	jobs := make(chan orchMatrixJob, len(cfg.Sets))
	sem := make(chan struct{}, 2)

	for i, set := range cfg.Sets {
		if len(set) == 0 {
			results[i] = OrchMatrixResult{SetIndex: i, Status: "skipped", Completed: i + 1, Total: len(cfg.Sets)}
			mu.Lock()
			resultsBySeq = append(resultsBySeq, results[i])
			seq++
			mu.Unlock()
			continue
		}
		jobs <- orchMatrixJob{index: i, set: set}
	}
	close(jobs)

	var wg sync.WaitGroup
	for job := range jobs {
		wg.Add(1)
		go func(j orchMatrixJob) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			orchCfg := OrchestratorConfig{
				Strategies:     j.set,
				StartDate:      cfg.StartDate,
				EndDate:        cfg.EndDate,
				InitialCapital: cfg.InitialCapital,
				RebalanceBars:  cfg.RebalanceBars,
				KellyFraction:  cfg.KellyFraction,
				MaxPositionPct: cfg.MaxPositionPct,
				FrictionModel:  cfg.FrictionModel,
			}

			o, err := NewOrchestrator(dbAdapter, orchCfg)
			if err != nil {
				mu.Lock()
				results[j.index] = OrchMatrixResult{
					SetIndex: j.index, Strategies: j.set, Status: "failed",
					Error: err.Error(), Completed: j.index + 1, Total: len(cfg.Sets),
				}
				resultsBySeq = append(resultsBySeq, results[j.index])
				seq++
				mu.Unlock()
				return
			}

			for _, s := range j.set {
				if err := o.AddStrategy(s.Symbol, s.Timeframe, s.StrategyID); err != nil {
					mu.Lock()
					results[j.index] = OrchMatrixResult{
						SetIndex: j.index, Strategies: j.set, Status: "failed",
						Error: err.Error(), Completed: j.index + 1, Total: len(cfg.Sets),
					}
					resultsBySeq = append(resultsBySeq, results[j.index])
					seq++
					mu.Unlock()
					return
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()

			result, runErr := o.Run(ctx)

			mu.Lock()
			r := OrchMatrixResult{
				SetIndex: j.index, Strategies: j.set,
				Status: "completed", Completed: j.index + 1, Total: len(cfg.Sets),
			}
			if runErr != nil {
				r.Status = "failed"
				r.Error = runErr.Error()
			} else {
				r.PoolSharpe = result.PoolSharpe
				r.PoolMaxDD = result.PoolMaxDD
				r.PoolReturn = result.PoolReturnPct
				r.NumTrades = len(result.Trades)
				r.StrategyPnL = result.StrategyPnL

				enriched := EnrichResultJSON(result)
				enrichedJSON, _ := json.Marshal(enriched)
				run := &db.OrchestrationRun{
					StartDate:     cfg.StartDate,
					EndDate:       cfg.EndDate,
					InitialCapital: cfg.InitialCapital,
					Status:        "completed",
				}
				for _, s := range j.set {
					run.StrategyIDs = append(run.StrategyIDs, s.StrategyID)
					run.SymbolTFPairs = append(run.SymbolTFPairs, s.Symbol+":"+s.Timeframe)
				}
				run.PoolSharpe = &result.PoolSharpe
				run.PoolMaxDD = &result.PoolMaxDD
				run.PoolReturnPct = &result.PoolReturnPct
				if repo != nil {
					_ = repo.SaveOrchestrationRun(context.Background(), run)
					repo.UpdateOrchestrationRunWithJSON(context.Background(), run.ID, "completed",
						&db.OrchestrationResult{
							PoolSharpe: result.PoolSharpe, PoolSortino: result.PoolSortino,
							PoolMaxDD: result.PoolMaxDD, PoolReturnPct: result.PoolReturnPct,
						}, enrichedJSON)
					r.RunID = run.ID
				}
			}
			results[j.index] = r
			resultsBySeq = append(resultsBySeq, r)
			seq++
			mu.Unlock()
		}(job)
	}
	wg.Wait()
}
