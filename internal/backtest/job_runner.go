package backtest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ValidationJobConfig struct {
	JobID          string
	StrategyIDs    []string
	Symbols        []string
	StartDate      time.Time
	EndDate        time.Time
	InitialCapital float64
	IVSEnabled     bool
	Simulations    int
	Engine         *Engine
}

type PreflightResult struct {
	DBConnected    bool     `json:"db_connected"`
	EngineReady    bool     `json:"engine_ready"`
	DataCandles    int64    `json:"data_candles"`
	SyntheticCount int64    `json:"synthetic_count"`
	DataRegimes    int64    `json:"data_regimes"`
	Passed         bool     `json:"passed"`
	Errors         []string `json:"errors,omitempty"`
}

type PreflightChecker interface {
	Check(ctx context.Context) PreflightResult
}

type jobPreflightChecker struct {
	engine *Engine
}

func (c *jobPreflightChecker) Check(ctx context.Context) PreflightResult {
	r := PreflightResult{}
	if c.engine != nil && c.engine.db != nil {
		if count, err := c.engine.db.CountCandles(ctx); err == nil {
			r.DataCandles = count
			r.DBConnected = true
		}
		if count, err := c.engine.db.CountSyntheticCandles(ctx); err == nil {
			r.SyntheticCount = count
		}
		if count, err := c.engine.db.CountRegimeLogs(ctx); err == nil {
			r.DataRegimes = count
		}
	}
	r.EngineReady = c.engine != nil
	if !r.DBConnected {
		r.Errors = append(r.Errors, "database not connected")
	}
	if !r.EngineReady {
		r.Errors = append(r.Errors, "backtest engine not initialized")
	}
	if r.DataCandles == 0 {
		r.Errors = append(r.Errors, "no candle data in database — run orca-fetch --source=stooq --universe")
	}
	if r.SyntheticCount > 0 {
		r.Errors = append(r.Errors, fmt.Sprintf("database contains %d synthetic candles — DELETE FROM candles WHERE source='synthetic'", r.SyntheticCount))
	}
	r.Passed = r.DBConnected && r.EngineReady && r.DataCandles > 0 && r.SyntheticCount == 0
	return r
}

type ValidationStage string

const (
	StagePreflight   ValidationStage = "preflight"
	StageBacktest    ValidationStage = "backtest"
	StageWalkForward ValidationStage = "walkforward"
	StageMonteCarlo  ValidationStage = "montecarlo"
	StageVerdict     ValidationStage = "verdict"
	StageComplete    ValidationStage = "complete"
)

type JobProgress struct {
	JobID    string          `json:"job_id"`
	Progress int             `json:"progress"`
	Stage    ValidationStage `json:"stage"`
	Status   string          `json:"status"`
	Error    string          `json:"error,omitempty"`
}

type JobRunner struct {
	mu   sync.RWMutex
	jobs map[string]*RunningJob
}

type RunningJob struct {
	Config   ValidationJobConfig
	Progress JobProgress
	Cancel   context.CancelFunc
	Result   *PipelineResult
}

type StrategyPipelineResult struct {
	StrategyID  string              `json:"strategy_id"`
	WalkForward *WalkForwardResult  `json:"walk_forward,omitempty"`
	MonteCarlo  *MonteCarloResult   `json:"monte_carlo,omitempty"`
	MultiMetric *MultiMetricVerdict `json:"multi_metric,omitempty"`
	BestParams  map[string]float64  `json:"best_params,omitempty"`
	Error       string              `json:"error,omitempty"`
}

type PipelineResult struct {
	JobID       string                   `json:"job_id"`
	Status      string                   `json:"status"`
	StartedAt   time.Time                `json:"started_at"`
	CompletedAt *time.Time               `json:"completed_at,omitempty"`
	Strategies  []StrategyPipelineResult `json:"strategies"`
	Summary     *PipelineSummary         `json:"summary,omitempty"`
	Error       string                   `json:"error,omitempty"`
}

type PipelineSummary struct {
	Passed int `json:"passed"`
	Failed int `json:"failed"`
	Total  int `json:"total"`
}

var defaultJobRunner = &JobRunner{
	jobs: make(map[string]*RunningJob),
}

func GetJobRunner() *JobRunner {
	return defaultJobRunner
}

func NewValidationJobConfig(strategyIDs []string, symbols []string, startDate, endDate time.Time, initialCapital float64) ValidationJobConfig {
	return ValidationJobConfig{
		JobID:          uuid.New().String()[:8],
		StrategyIDs:    strategyIDs,
		Symbols:        symbols,
		StartDate:      startDate,
		EndDate:        endDate,
		InitialCapital: initialCapital,
		IVSEnabled:     true,
		Simulations:    5000,
	}
}

type ProgressCallback func(JobProgress)

func (r *JobRunner) StartJob(ctx context.Context, config ValidationJobConfig, engine *Engine, callback ProgressCallback) *RunningJob {
	_, cancel := context.WithCancel(ctx)

	job := &RunningJob{
		Config: config,
		Progress: JobProgress{
			JobID:    config.JobID,
			Progress: 0,
			Stage:    StagePreflight,
			Status:   "running",
		},
		Cancel: cancel,
	}

	if engine != nil {
		config.Engine = engine
	}

	r.mu.Lock()
	r.jobs[config.JobID] = job
	r.mu.Unlock()

	go r.executeJob(job, engine, callback)

	return job
}

func (r *JobRunner) executeJob(job *RunningJob, engine *Engine, callback ProgressCallback) {
	defer func() {
		r.mu.Lock()
		r.mu.Unlock()
	}()

	job.Progress.Progress = 2
	job.Progress.Stage = StagePreflight
	callback(job.Progress)

	preflight := PreflightResult{DBConnected: false, EngineReady: engine != nil}
	if engine != nil && engine.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if count, err := engine.db.CountCandles(ctx); err == nil {
			preflight.DataCandles = count
			preflight.DBConnected = true
		} else {
			preflight.Errors = append(preflight.Errors, "database not connected")
		}
		if count, err := engine.db.CountSyntheticCandles(ctx); err == nil {
			preflight.SyntheticCount = count
		}
		if count, err := engine.db.CountRegimeLogs(ctx); err == nil {
			preflight.DataRegimes = count
		}
	}
	if !preflight.EngineReady {
		preflight.Errors = append(preflight.Errors, "backtest engine not initialized")
	}
	if preflight.DBConnected && preflight.DataCandles == 0 {
		preflight.Errors = append(preflight.Errors, "no candle data: run orca-fetch --source=stooq --universe")
	}
	if preflight.SyntheticCount > 0 {
		preflight.Errors = append(preflight.Errors, fmt.Sprintf("%d synthetic candles detected: DELETE FROM candles WHERE source='synthetic'", preflight.SyntheticCount))
	}
	preflight.Passed = preflight.DBConnected && preflight.EngineReady && preflight.DataCandles > 0 && preflight.SyntheticCount == 0

	if !preflight.Passed {
		job.Progress.Status = "failed"
		job.Progress.Stage = StagePreflight
		errMsg := "preflight failed"
		if len(preflight.Errors) > 0 {
			errMsg = strings.Join(preflight.Errors, "; ")
		}
		job.Progress.Error = errMsg
		job.Progress.Progress = 100
		callback(job.Progress)
		return
	}

	job.Progress.Progress = 10
	job.Progress.Stage = StagePreflight
	callback(job.Progress)

	totalStrategies := len(job.Config.StrategyIDs)
	if totalStrategies == 0 {
		job.Progress.Status = "completed"
		job.Progress.Stage = StageComplete
		job.Progress.Progress = 100
		callback(job.Progress)
		return
	}

	result := &PipelineResult{
		JobID:     job.Config.JobID,
		Status:    "running",
		StartedAt: time.Now(),
	}

	baseProgress := 10
	progressPerStrat := 80 / totalStrategies

	for i, strategyID := range job.Config.StrategyIDs {
		stratProgress := baseProgress + i*progressPerStrat
		job.Progress.Progress = stratProgress
		job.Progress.Stage = StageBacktest
		callback(job.Progress)

		stratResult := r.executeStrategyPipeline(job, engine, strategyID, func(prog JobProgress) {
			p := stratProgress + (prog.Progress * progressPerStrat / 400)
			if p > stratProgress+progressPerStrat {
				p = stratProgress + progressPerStrat - 1
			}
			job.Progress.Progress = p
			job.Progress.Stage = prog.Stage
			callback(job.Progress)
		})

		result.Strategies = append(result.Strategies, stratResult)
	}

	passed := 0
	failed := 0
	for _, s := range result.Strategies {
		if s.MultiMetric != nil && s.MultiMetric.Passed {
			passed++
		} else {
			failed++
		}
	}

	now := time.Now()
	result.CompletedAt = &now
	result.Status = "completed"
	result.Summary = &PipelineSummary{Passed: passed, Failed: failed, Total: totalStrategies}

	job.Result = result
	job.Progress.Progress = 100
	job.Progress.Stage = StageComplete
	job.Progress.Status = "completed"
	callback(job.Progress)
}

func (r *JobRunner) executeStrategyPipeline(job *RunningJob, engine *Engine, strategyID string, callback ProgressCallback) StrategyPipelineResult {
	stratResult := StrategyPipelineResult{StrategyID: strategyID}

	owfCfg := DefaultOptimizedWalkForwardConfig(
		strategyID,
		job.Config.Symbols,
		job.Config.StartDate,
		job.Config.EndDate,
		job.Config.InitialCapital,
	)
	owfCfg.IVSConfig = DefaultIVSConfig()
	owfCfg.IVSConfig.Enabled = job.Config.IVSEnabled

	callback(JobProgress{Progress: 25, Stage: StageBacktest})
	time.Sleep(100 * time.Millisecond)

	ctx := context.Background()
	owfResult, err := engine.RunOptimizedWalkForward(ctx, owfCfg)
	if err != nil {
		stratResult.Error = fmt.Sprintf("walk-forward: %v", err)
		return stratResult
	}
	stratResult.WalkForward = &owfResult.WalkForwardResult
	if len(owfResult.IVSRobustParamsPerWindow) > 0 {
		stratResult.BestParams = owfResult.IVSRobustParamsPerWindow[len(owfResult.IVSRobustParamsPerWindow)-1]
	} else if len(owfResult.BestParamsPerWindow) > 0 {
		stratResult.BestParams = owfResult.BestParamsPerWindow[len(owfResult.BestParamsPerWindow)-1]
	}

	callback(JobProgress{Progress: 50, Stage: StageMonteCarlo})

	var trades []Trade
	for _, w := range owfResult.Windows {
		trades = append(trades, Trade{PnL: w.OOSReturnPct})
	}
	mcResult, err := RunMonteCarloFromTrades(trades, job.Config.Simulations, job.Config.InitialCapital)
	if err != nil {
		stratResult.Error = fmt.Sprintf("monte carlo: %v", err)
		return stratResult
	}
	stratResult.MonteCarlo = mcResult

	callback(JobProgress{Progress: 75, Stage: StageVerdict})

	std := DefaultMultiMetricStandard()
	verdict := EvaluateOOSMultiMetric(owfResult, mcResult, std)
	stratResult.MultiMetric = &verdict

	callback(JobProgress{Progress: 100, Stage: StageComplete})
	return stratResult
}

func (r *JobRunner) GetJob(jobID string) *RunningJob {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.jobs[jobID]
}

func (r *JobRunner) GetJobStatus(jobID string) JobProgress {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if job, ok := r.jobs[jobID]; ok {
		return job.Progress
	}
	return JobProgress{JobID: jobID, Status: "not_found"}
}

func (r *JobRunner) GetJobResult(jobID string) *PipelineResult {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if job, ok := r.jobs[jobID]; ok {
		return job.Result
	}
	return nil
}

func (r *JobRunner) ListJobs() []JobProgress {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var jobs []JobProgress
	for _, j := range r.jobs {
		jobs = append(jobs, j.Progress)
	}
	return jobs
}

func (r *JobRunner) CancelJob(jobID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if job, ok := r.jobs[jobID]; ok {
		job.Cancel()
		job.Progress.Status = "cancelled"
		job.Progress.Stage = StageComplete
		return true
	}
	return false
}

func (v PipelineResult) JSON() string {
	data, _ := json.MarshalIndent(v, "", "  ")
	return string(data)
}
