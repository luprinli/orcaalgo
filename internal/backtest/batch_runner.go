package backtest

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lee-econ/orca-core/internal/config"
	"github.com/lee-econ/orca-core/internal/monitor"
	"github.com/lee-econ/orca-core/internal/risk"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type BatchTuple struct {
	Strategy  string
	Symbol    string
	Timeframe string
}

type batchTuple struct {
	Strategy  string
	Symbol    string
	Timeframe string
}

type BatchOptimizeConfig struct {
	StrategyID      string
	Symbols         []string
	StartDate       time.Time
	EndDate         time.Time
	InitialCapital  float64
	Params          []map[string]float64
	MonoChunkLen    int
	EnginePool      int
	GateProfile     string
	PropFirmEnabled bool
	FixedSeed       int64
	Timeframe       string
}

func cartesianProduct(strategies, symbols, timeframes []string) []batchTuple {
	if len(timeframes) == 0 {
		timeframes = []string{"1d"}
	}
	if len(strategies) == 0 {
		strategies = []string{"intraday_mr"}
	}
	combos := make([]batchTuple, 0, len(strategies)*len(symbols)*len(timeframes))
	for _, s := range strategies {
		for _, sym := range symbols {
			for _, tf := range timeframes {
				combos = append(combos, batchTuple{Strategy: s, Symbol: sym, Timeframe: tf})
			}
		}
	}
	return combos
}

func CartesianProduct(strategies, symbols, timeframes []string) []BatchTuple {
	combos := cartesianProduct(strategies, symbols, timeframes)
	result := make([]BatchTuple, len(combos))
	for i, c := range combos {
		result[i] = BatchTuple{Strategy: c.Strategy, Symbol: c.Symbol, Timeframe: c.Timeframe}
	}
	return result
}

func RunBatchOptimize(ctx context.Context, db Database, config BatchOptimizeConfig) ([]ComboResult, error) {
	if len(config.Params) == 0 {
		return nil, nil
	}
	chunkLen := config.MonoChunkLen
	if chunkLen <= 0 {
		chunkLen = 1
	}
	enginePool := config.EnginePool
	if enginePool <= 0 {
		enginePool = runtime.NumCPU()
		if enginePool > 8 {
			enginePool = 8
		}
	}

	results := make([]ComboResult, 0, len(config.Params))

	for chunkStart := 0; chunkStart < len(config.Params); chunkStart += chunkLen {
		chunkEnd := chunkStart + chunkLen
		if chunkEnd > len(config.Params) {
			chunkEnd = len(config.Params)
		}
		chunk := config.Params[chunkStart:chunkEnd]

		chunkResults := make([]ComboResult, len(chunk))
		var mu sync.Mutex
		sem := semaphore.NewWeighted(int64(enginePool))
		g, ctx := errgroup.WithContext(ctx)

		for i, params := range chunk {
			i, params := i, params
			for _, sym := range config.Symbols {
				sym := sym
				if err := sem.Acquire(ctx, 1); err != nil {
					break
				}
				g.Go(func() error {
					defer sem.Release(1)
					btConfig := BacktestConfig{
						StrategyID:     config.StrategyID,
						Symbols:        []string{sym},
						StartDate:      config.StartDate,
						EndDate:        config.EndDate,
						InitialCapital: config.InitialCapital,
						Timeframe:      config.Timeframe,
						StrategyParams: params,
						PropFirmEnabled: config.PropFirmEnabled,
						GateProfile:    config.GateProfile,
						FixedSeed:      config.FixedSeed,
					}
					engine := NewEngineWithFixedSeed(db, config.FixedSeed)
					result, err := engine.Run(ctx, btConfig)
					mu.Lock()
					defer mu.Unlock()
					if err != nil {
						chunkResults[i] = ComboResult{
							StrategyID: config.StrategyID,
							Symbol:     sym,
							Timeframe:  config.Timeframe,
							Error:      err.Error(),
						}
						return nil
					}
					chunkResults[i] = ComboResult{
						Symbol:             sym,
						StrategyID:         config.StrategyID,
						Timeframe:          config.Timeframe,
						SharpeRatio:        result.SharpeRatio,
						SortinoRatio:       result.SortinoRatio,
						MaxDrawdown:        result.MaxDrawdown,
						MaxDrawdownDur:     result.MaxDrawdownDuration,
						TotalReturn:        result.TotalReturnPct,
						WinRate:            result.WinRate,
						ProfitFactor:       result.ProfitFactor,
						AvgTrade:           result.AvgTrade,
						AvgWin:             result.AvgWin,
						AvgLoss:            result.AvgLoss,
						NumTrades:          result.NumTrades,
						NumWins:            result.NumWins,
						NumLosses:          result.NumLosses,
						AvgMAE:             result.AvgMAE,
						AvgMFE:             result.AvgMFE,
						Warnings:           result.Warnings,
						AdverseSelectRate:  result.AdverseSelectionRate,
						StrategyParams:     result.StrategyParams,
						EquityCurve:        result.EquityCurve,
						Trades:             result.Trades,
						MtmSharpeRatio:     result.MtmSharpeRatio,
						MtmMaxDrawdown:     result.MtmMaxDrawdown,
						MLFeatureEnabled:   result.MLFeatureEnabled,
						TotalFees:          result.TotalFees,
						AvgSlippageBps:     result.AvgSlippageBps,
						CalmarRatio:        result.CalmarRatio,
						CandleCount:        result.CandleCount,
						GrossReturnPct:     grossReturnPct(result.TotalReturnPct, result.TotalFees, config.InitialCapital),
						EngineVersion:      result.EngineVersion,
					}
		return nil
	})
}
}
_ = g.Wait()
results = append(results, chunkResults...)
	}

	return results, nil
}

type MatrixProgressFn func(index int, status string, errMsg string, result *ComboResult)

func RunMatrix(ctx context.Context, db Database, config MatrixBacktestConfig, onProgress MatrixProgressFn) (*MatrixResult, error) {
	return RunMatrixConcurrent(ctx, db, config, onProgress)
}

func RunMatrixConcurrent(ctx context.Context, db Database, config MatrixBacktestConfig, onProgress MatrixProgressFn) (*MatrixResult, error) {
	combos := cartesianProduct(config.StrategyIDs, config.Symbols, config.Timeframes)

	monitor.RecordMatrixBatchStart()

	if len(combos) == 0 {
		return &MatrixResult{RunID: uuid.New().String(), Combos: 0}, nil
	}

	// §1 per-strategy light optimization: each unique strategy gets a bounded
	// train/test-split parameter sweep on a representative symbol subset.
	// Optimized params are applied to all combos of that strategy in the matrix.
	// Skipped when config.SkipLightOptimize is true.
	optimizedParams := make(map[string]map[string]float64)
	if !config.SkipLightOptimize {
	seenStrats := make(map[string]bool)
	for _, c := range combos {
		if seenStrats[c.Strategy] {
			continue
		}
		seenStrats[c.Strategy] = true

		repSymbols := SelectRepresentativeSymbols(config.Symbols, LightOptSymbolCount())
		lightCfg := LightOptimizeConfig{
			StrategyID:         c.Strategy,
			Symbols:            repSymbols,
			ValidationSymbols:  DiffStrings(config.Symbols, repSymbols),
			Timeframe:          pickDominantTimeframe(config.Timeframes),
			StartDate:          config.StartDate,
			EndDate:            config.StartDate.AddDate(0, LightOptWindowMonths(), 0),
			InitialCapital:     config.InitialCapital,
			PropFirmEnabled:    config.PropFirmEnabled,
			GateProfile:        config.GateProfile,
			EnableCache:        true,
			MaxCombos:          LightOptBudget(),
			PerBacktestTimeout: LightOptTimeout(),
			PlateauPatience:    LightOptPlateauPatience(),
			TrainFraction:      LightOptTrainFraction(),
			ObjectiveWeights:   LightOptWeights(),
			CacheTTL:           LightOptCacheTTL(),
		}
		if lightCfg.EndDate.After(config.EndDate) {
			lightCfg.EndDate = config.EndDate
		}

		params := RunLightOptimize(ctx, db, lightCfg)
		if params == nil {
			continue
		}
		optimizedParams[c.Strategy] = params
		if onProgress != nil {
			onProgress(len(combos)-1, "optimized", "", &ComboResult{
				StrategyID: c.Strategy, Optimized: true,
			})
		}
	}
	} // end skipLightOptimize guard

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	maxWorkers := runtime.NumCPU()
	if maxWorkers > 8 {
		maxWorkers = 8
	}
	if maxWorkers < 2 {
		maxWorkers = 2
	}

	sem := semaphore.NewWeighted(int64(maxWorkers))
	g, ctx := errgroup.WithContext(ctx)

	results := make([]ComboResult, len(combos))
	var mu sync.Mutex

	applyGate := config.GateProfile != "" && config.GateProfile != "none"
	sizingPct := config.SizingPercent
	if sizingPct <= 0 {
		sizingPct = 0.02
	}
	kellyFrac := config.KellyFraction
	if kellyFrac <= 0 {
		kellyFrac = 0.25
	}

	for i, combo := range combos {
		i, combo := i, combo

		if config.DataSource == "synthetic" {
			switch combo.Strategy {
			case "breakout", "opening_range_breakout", "scalp", "session_scalp":
				mu.Lock()
				results[i] = ComboResult{
					StrategyID: combo.Strategy,
					Symbol:     combo.Symbol,
					Timeframe:  combo.Timeframe,
					Error:      "skipped: strategy requires real intraday data — not evaluated on synthetic data",
				}
				mu.Unlock()
				if onProgress != nil {
					onProgress(i, "skipped", "requires real intraday market data", nil)
				}
				continue
			}
		}

		if err := sem.Acquire(ctx, 1); err != nil {
			break
		}
		if !risk.DefaultHeapAdmission.Allow() {
			risk.DefaultHeapAdmission.ForceGC()
			time.Sleep(100 * time.Millisecond)
		}
		if onProgress != nil {
			onProgress(i, "running", "", nil)
		}
		g.Go(func() error {
			defer sem.Release(1)
			monitor.AdjustMatrixWorkers(1)
			defer monitor.AdjustMatrixWorkers(-1)
			start := time.Now()
			btConfig := BacktestConfig{
				StrategyID:      combo.Strategy,
				Symbols:         []string{combo.Symbol},
				StartDate:       config.StartDate,
				EndDate:         config.EndDate,
				InitialCapital:  config.InitialCapital,
				Timeframe:       combo.Timeframe,
				DataSource:      config.DataSource,
				PropFirmEnabled: config.PropFirmEnabled,
				SizingPercent:   sizingPct,
				KellyFraction:   kellyFrac,
				ApplyGate:       applyGate,
				GateProfile:     config.GateProfile,
				StopLoss: &StopLossConfig{
					Type:          "atr",
					ATRPeriod:     14,
					ATRMultiplier: 2.0,
				},
				TakeProfit: &TakeProfitConfig{
					Type:    "risk_reward",
					RRRatio: 2.0,
				},
			}
			if combo.Strategy == "pairs_trading" || combo.Strategy == "stat_arb" {
				if sec := defaultSecondarySymbol(combo.Symbol); sec != "" {
					btConfig.SecondarySymbols = map[string]string{combo.Symbol: sec}
					btConfig.Symbols = append(btConfig.Symbols, sec)
				}
			}
			if optParams, ok := optimizedParams[combo.Strategy]; ok {
				btConfig.StrategyParams = optParams
			}
			engine := NewEngine(db)
			if config.WirePipeline {
				engine.WirePipeline()
			}
			result, err := engine.Run(ctx, btConfig)
			monitor.RecordBacktestDuration(time.Since(start).Seconds())
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				monitor.RecordMatrixCombo("failed")
				results[i] = ComboResult{
					StrategyID: combo.Strategy,
					Symbol:     combo.Symbol,
					Timeframe:  combo.Timeframe,
					Error:      err.Error(),
				}
				if onProgress != nil {
					onProgress(i, "failed", err.Error(), nil)
				}
				return nil
			}

			// Optional walk-forward validation for optimized combos, mirroring the
			// CLI matrix-runner --walk-forward flag. Populates WfIS/WfOOS columns.
			var wfIS, wfOOS float64
			if config.WalkForward {
				if _, ok := optimizedParams[combo.Strategy]; ok {
					wfCfg := WalkForwardConfig{
						Config:             btConfig,
						TrainWindows:       3,
						TrainYears:         1,
						TestYears:          2,
						StepMonths:         6,
						PurgeTradingDays:   5,
						EmbargoTradingDays: 2,
					}
					if wfRes, wfErr := engine.RunWalkForward(ctx, wfCfg); wfErr == nil && wfRes != nil && wfRes.TotalWindows > 0 {
						wfIS = wfRes.OverallSharpe
						wfOOS = wfRes.AvgOOSSharpe
					}
				}
			}
			monitor.RecordMatrixCombo("completed")
			warnings := result.Warnings
			if (combo.Strategy == "grid_trading" || combo.Strategy == "grid") &&
				result.NumTrades == 0 {
				warnings = append(warnings, "disabled: strategy is disabled by default (HP agenda)")
			}
			results[i] = ComboResult{
				Symbol:             combo.Symbol,
				StrategyID:         combo.Strategy,
				Timeframe:          combo.Timeframe,
				SharpeRatio:        result.SharpeRatio,
				SortinoRatio:       result.SortinoRatio,
				MaxDrawdown:        result.MaxDrawdown,
				MaxDrawdownDur:     result.MaxDrawdownDuration,
				WinRate:            result.WinRate,
				AvgTrade:           result.AvgTrade,
				AvgWin:             result.AvgWin,
				AvgLoss:            result.AvgLoss,
				NumTrades:          result.NumTrades,
				NumWins:            result.NumWins,
				NumLosses:          result.NumLosses,
				AvgMAE:             result.AvgMAE,
				AvgMFE:             result.AvgMFE,
				Warnings:           result.Warnings,
				GatePassed:         gateBool(result.MetricGateStatus),
				AdverseSelectRate:  result.AdverseSelectionRate,
				StrategyParams:     result.StrategyParams,
				Optimized:          len(btConfig.StrategyParams) > 0,
				EquityCurve:        result.EquityCurve,
				Trades:             result.Trades,
				LongTrades:         result.LongShort.LongTrades,
				ShortTrades:        result.LongShort.ShortTrades,
				LongWinRate:        result.LongShort.LongWinRate,
				ShortWinRate:       result.LongShort.ShortWinRate,
				LongGrossPnL:       clampAbs(result.LongShort.LongGrossPnL, config.InitialCapital*100),
				ShortGrossPnL:      clampAbs(result.LongShort.ShortGrossPnL, config.InitialCapital*100),
				LongPF:             clampAbs(result.LongShort.LongPF, 1000),
				ShortPF:            clampAbs(result.LongShort.ShortPF, 1000),
				ProfitFactor:       clampAbs(result.ProfitFactor, 1000),
				TotalReturn:        clampAbs(result.TotalReturnPct, 10000),
				ZeroPnLTrades:      result.NumTrades - result.NumWins - result.NumLosses,
				ExpectedPF:         computeExpectedPF(result.WinRate, result.AvgWin, result.AvgLoss),
				RewardRiskRatio:    computeRewardRisk(result.AvgWin, result.AvgLoss),
				DailyVolatility:    computeDailyVolatility(result.DailyReturns),
				TrainPct:           result.TrainPct,
				MtmSharpeRatio:     result.MtmSharpeRatio,
				MtmMaxDrawdown:     result.MtmMaxDrawdown,
				MLFeatureEnabled:   result.MLFeatureEnabled,
				TotalFees:          result.TotalFees,
				AvgSlippageBps:     result.AvgSlippageBps,
				CalmarRatio:        result.CalmarRatio,
				CandleCount:        result.CandleCount,
				GrossReturnPct:     grossReturnPct(result.TotalReturnPct, result.TotalFees, config.InitialCapital),
				DataSource:         config.DataSource,
				EngineVersion:      result.EngineVersion,
				DataGenerationID:   result.DataGenerationID,
				WfISSharpe:         wfIS,
				WfOOSSharpe:        wfOOS,
			}
			if onProgress != nil {
				onProgress(i, "completed", "", &results[i])
			}
			return nil
		})
	}
	_ = g.Wait()

	runID := fmt.Sprintf("matrix-%s", time.Now().Format("20060102150405"))
	return &MatrixResult{
		RunID:        runID,
		Combos:       len(combos),
		Results:      results,
		Config:       config,
		Plausibility: FlagImplausibleCombos(results),
	}, nil
}

// grossReturnPct recomputes the return before fees/slippage so the cost drag is
// visible alongside the net return. Falls back to net return when capital is
// non-positive.
func grossReturnPct(netReturnPct, totalFees, initialCapital float64) float64 {
	if initialCapital <= 0 {
		return netReturnPct
	}
	return netReturnPct + totalFees/initialCapital*100.0
}

type CachedContext struct {
	RegimeLogs        []RegimeLog
	UniverseSnapshots map[time.Time][]string
	CandleCache       map[string][]Candle
}

func LoadCachedContext(ctx context.Context, db Database, config MatrixBacktestConfig) (*CachedContext, error) {
	cc := &CachedContext{
		CandleCache: make(map[string][]Candle),
	}

	if logs, err := db.LoadRegimeLogs(ctx, config.StartDate, config.EndDate); err == nil {
		cc.RegimeLogs = logs
	}

	if snaps, err := db.LoadUniverseSnapshots(ctx, config.StartDate, config.EndDate); err == nil && len(snaps) > 0 {
		cc.UniverseSnapshots = make(map[time.Time][]string)
		for _, snap := range snaps {
			cc.UniverseSnapshots[snap.Date.Truncate(24*time.Hour)] = snap.Symbols
		}
	}

	return cc, nil
}

func gateBool(v *MultiMetricVerdict) *bool {
	if v == nil {
		return nil
	}
	b := v.Passed
	return &b
}

// SelectRepresentativeSymbols picks up to n symbols deterministically from the
// provided list, preserving order. Used for the light optimizer's representative
// subset (§Det. Symbol Selection).
func SelectRepresentativeSymbols(symbols []string, n int) []string {
	if len(symbols) <= n {
		out := make([]string, len(symbols))
		copy(out, symbols)
		return out
	}
	return symbols[:n]
}

// DiffStrings returns elements in a that are not in b (set difference).
func DiffStrings(a, b []string) []string {
	bset := make(map[string]bool, len(b))
	for _, s := range b {
		bset[s] = true
	}
	var out []string
	for _, s := range a {
		if !bset[s] {
			out = append(out, s)
		}
	}
	return out
}

// pickDominantTimeframe returns the most appropriate timeframe for optimization,
// preferring daily bars then the longest-tick timeframe present.
func pickDominantTimeframe(timeframes []string) string {
	if len(timeframes) == 0 {
		return "1d"
	}
	for _, tf := range timeframes {
		if tf == "1d" || tf == "daily" {
			return "1d"
		}
	}
	return timeframes[0]
}

func RunWalkForwardOnTopCombos(results []ComboResult, topN int, db Database, baseConfig BacktestConfig) ([]WalkForwardResult, error) {
	if topN <= 0 {
		topN = 4
	}
	type combo struct {
		StrategyID string
		Symbol     string
		Timeframe  string
		Sharpe     float64
	}
	var ranked []combo
	for _, r := range results {
		if r.Error != "" || r.NumTrades < 20 {
			continue
		}
		ranked = append(ranked, combo{r.StrategyID, r.Symbol, r.Timeframe, r.SharpeRatio})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].Sharpe > ranked[j].Sharpe })
	if len(ranked) > topN {
		ranked = ranked[:topN]
	}

	var wfResults []WalkForwardResult
	for _, c := range ranked {
		wfCfg := WalkForwardConfig{
			Config: BacktestConfig{
				StrategyID:     c.StrategyID,
				Symbols:        []string{c.Symbol},
				StartDate:      baseConfig.StartDate,
				EndDate:        baseConfig.EndDate,
				InitialCapital: baseConfig.InitialCapital,
				Timeframe:      c.Timeframe,
				DataSource:     baseConfig.DataSource,
				SizingPercent:  baseConfig.SizingPercent,
				KellyFraction:  baseConfig.KellyFraction,
				GateProfile:    baseConfig.GateProfile,
			},
			TrainWindows: 3,
			PurgeTradingDays: 5,
			EmbargoTradingDays: 10,
		}
		engine := NewEngine(db)
		wfResult, err := engine.RunWalkForward(context.Background(), wfCfg)
		if err != nil {
			continue
		}
		wfResults = append(wfResults, *wfResult)
	}
	return wfResults, nil
}

func clampAbs(val, limit float64) float64 {
	if math.IsNaN(val) || math.IsInf(val, 0) {
		return 0
	}
	if val > limit {
		return limit
	}
	if val < -limit {
		return -limit
	}
	return val
}

func defaultSecondarySymbol(primary string) string {
	if sec := config.SecondaryTicker(primary); sec != "" {
		return sec
	}
	return ""
}

func computeExpectedPF(winRate, avgWin, avgLoss float64) float64 {
	if winRate <= 0 || winRate >= 100 || avgWin <= 0 || avgLoss <= 0 {
		return 0
	}
	wr := winRate / 100.0
	expected := (wr * avgWin) / ((1 - wr) * avgLoss)
	if math.IsNaN(expected) || math.IsInf(expected, 0) {
		return 0
	}
	return expected
}

func computeRewardRisk(avgWin, avgLoss float64) float64 {
	if avgLoss <= 0 {
		return 0
	}
	rr := avgWin / avgLoss
	if math.IsNaN(rr) || math.IsInf(rr, 0) {
		return 0
	}
	return rr
}

func computeDailyVolatility(returns []DailyReturn) float64 {
	if len(returns) < 2 {
		return 0
	}
	sum := 0.0
	for _, dr := range returns {
		sum += dr.Return
	}
	mean := sum / float64(len(returns))
	variance := 0.0
	for _, dr := range returns {
		diff := dr.Return - mean
		variance += diff * diff
	}
	variance /= float64(len(returns) - 1)
	return math.Sqrt(variance)
}
