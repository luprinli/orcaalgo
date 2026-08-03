package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/lee-econ/orca-core/internal/backtest"
	"github.com/lee-econ/orca-core/internal/db"
)

func defaultSymbols(strategyID string) []string {
	switch strategyID {
	case "trend_following":
		return []string{"SPY", "QQQ", "ES", "NQ", "GLD", "TLT"}
	case "mean_reversion":
		return []string{"SPY", "QQQ", "IWM", "AAPL", "MSFT"}
	case "session_scalp":
		return []string{"SPY", "QQQ", "ES"}
	case "opening_range_breakout":
		return []string{"SPY", "QQQ", "ES"}
	case "pairs_trading":
		return []string{"SPY"} // single symbol — pair adds secondary in runner
	case "volatility_harvesting":
		return []string{"SPY", "QQQ"}
	default:
		return []string{"SPY"}
	}
}

// ReoptimizationConfig configures the automatic parameter re-optimization job.
type ReoptimizationConfig struct {
	// Interval is how often to check for degradation and trigger re-optimization.
	// Default: 24h (daily, runs at 16:00 EST market close).
	Interval time.Duration

	// DegradationThreshold is the OOS Sharpe drop (%) that triggers re-optimization.
	// Default: 20.0 (re-optimize if OOS Sharpe degrades > 20% from active params).
	DegradationThreshold float64

	// MaxAgeDays is the maximum age of active params before forced re-optimization.
	// Default: 90 (3 months).
	MaxAgeDays int

	// Engine is the backtest engine used to run optimization.
	Engine *backtest.Engine

	// Repo is the database repository for parameter version persistence.
	Repo *db.Repository

	// Strategies is the list of strategy IDs to include in scheduled optimization.
	Strategies []string

	// Symbols is the list of symbols to use for in-sample optimization.
	Symbols []string

	// ValidationSymbols is the list of symbols for out-of-sample validation.
	ValidationSymbols []string

	// EnableAutoActivate controls whether the scheduler automatically promotes
	// re-optimized params to active. When false, params are saved but not
	// activated — manual review required.
	EnableAutoActivate bool

	mu       sync.Mutex
	running  bool
	stopCh   chan struct{}
}

// DefaultReoptimizationConfig returns sensible defaults for daily re-optimization.
func DefaultReoptimizationConfig() ReoptimizationConfig {
	return ReoptimizationConfig{
		Interval:             24 * time.Hour,
		DegradationThreshold: 20.0,
		MaxAgeDays:           90,
		EnableAutoActivate:   false, // safe default: manual review required
	}
}

// Start begins the re-optimization loop in a background goroutine.
func (c *ReoptimizationConfig) Start(ctx context.Context) {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.stopCh = make(chan struct{})
	c.mu.Unlock()

	go c.run(ctx)
	log.Printf("[reopt] started: interval=%s, degradation_threshold=%.0f%%, max_age=%dd, auto_activate=%v",
		c.Interval, c.DegradationThreshold, c.MaxAgeDays, c.EnableAutoActivate)
}

// Stop terminates the re-optimization loop.
func (c *ReoptimizationConfig) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running {
		return
	}
	close(c.stopCh)
	c.running = false
	log.Println("[reopt] stopped")
}

func (c *ReoptimizationConfig) run(ctx context.Context) {
	ticker := time.NewTicker(c.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.CheckAndOptimize(ctx)
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (c *ReoptimizationConfig) CheckAndOptimize(ctx context.Context) {
	if c.Repo == nil || c.Engine == nil {
		log.Println("[reopt] skipped: engine or repo not configured")
		return
	}
	strategies := c.Strategies
	if len(strategies) == 0 {
		// Default: optimize all registered strategies in the regime matrix.
		for _, s := range []string{
			"trend_following", "session_scalp", "mean_reversion",
			"opening_range_breakout", "pairs_trading", "volatility_harvesting",
		} {
			strategies = append(strategies, s)
		}
	}

	for _, strategyID := range strategies {
		shouldOptimize, reason := c.shouldReoptimize(ctx, strategyID)
		if !shouldOptimize {
			log.Printf("[reopt] %s: skipping — %s", strategyID, reason)
			continue
		}

		log.Printf("[reopt] %s: triggering re-optimization — %s", strategyID, reason)
		if err := c.reoptimize(ctx, strategyID); err != nil {
			log.Printf("[reopt] %s: re-optimization failed: %v", strategyID, err)
		}
	}
}

func (c *ReoptimizationConfig) shouldReoptimize(ctx context.Context, strategyID string) (bool, string) {
	if c.Repo == nil {
		return false, "no_repo_configured"
	}
	active, err := c.Repo.GetActiveParams(ctx, strategyID)
	if err != nil {
		return true, fmt.Sprintf("db error: %v", err)
	}
	if active == nil {
		return true, "no active params (will run initial optimization)"
	}

	// Age-based trigger.
	if c.MaxAgeDays > 0 {
		age := time.Since(active.CreatedAt)
		if age.Hours() > float64(c.MaxAgeDays*24) {
			return true, fmt.Sprintf("params %s (%.0f days old, max=%d)",
				active.VersionTag, age.Hours()/24, c.MaxAgeDays)
		}
	}

	// Degradation trigger: run a quick backtest with active params on recent data.
	if c.DegradationThreshold > 0 && active.OOSSharpe != nil && *active.OOSSharpe > 0 {
		currentOOS := c.runQuickBacktest(ctx, strategyID, active.Params)
		if currentOOS > 0 {
			drop := (*active.OOSSharpe - currentOOS) / *active.OOSSharpe * 100.0
			if drop > c.DegradationThreshold {
				return true, fmt.Sprintf("OOS Sharpe degraded %.1f%% (%.3f → %.3f, threshold=%.0f%%)",
					drop, *active.OOSSharpe, currentOOS, c.DegradationThreshold)
			}
		}
	}

	return false, "params are within tolerance"
}

func (c *ReoptimizationConfig) runQuickBacktest(ctx context.Context, strategyID string, params map[string]float64) float64 {
	endDate := time.Now()
	startDate := endDate.AddDate(0, -3, 0) // 3-month lookback

	cfg := backtest.BacktestConfig{
		StrategyID:     strategyID,
		Symbols:        c.ValidationSymbols,
		StartDate:      startDate,
		EndDate:        endDate,
		InitialCapital: 100000,
		DataSource:     "stooq",
		Timeframe:      "1d",
		StrategyParams: params,
		SizingPercent:  0.02,
	}

	result, err := c.Engine.Run(ctx, cfg)
	if err != nil || result == nil {
		return 0
	}
	return result.SharpeRatio
}

func (c *ReoptimizationConfig) reoptimize(ctx context.Context, strategyID string) error {
	if c.Repo == nil {
		return fmt.Errorf("repo not configured")
	}
	symbols := c.Symbols
	if len(symbols) == 0 {
		symbols = defaultSymbols(strategyID)
	}
	valSymbols := c.ValidationSymbols
	if len(valSymbols) == 0 {
		// Use last symbol as validation if none specified.
		if len(symbols) > 1 {
			valSymbols = symbols[len(symbols)-1:]
			symbols = symbols[:len(symbols)-1]
		}
	}

	endDate := time.Now()
	startDate := endDate.AddDate(-1, 0, 0) // 1-year IS window

	lightCfg := backtest.LightOptimizeConfig{
		StrategyID:         strategyID,
		Symbols:            symbols,
		ValidationSymbols:  valSymbols,
		StartDate:          startDate,
		EndDate:            endDate,
		InitialCapital:     100000,
		DataSource:         "stooq",
		MaxCombos:          48,
		EnableCache:        false,
		PropFirmEnabled:    false,
	}

	params := backtest.RunLightOptimize(ctx, c.Engine.GetDB(), lightCfg)
	if params == nil {
		return fmt.Errorf("light optimization returned no viable params")
	}

	// Validate OOS.
	oosSharpe := c.runQuickBacktest(ctx, strategyID, params)

	// Get previous active for comparison.
	active, _ := c.Repo.GetActiveParams(ctx, strategyID)
	if active != nil && active.OOSSharpe != nil && oosSharpe > 0 && *active.OOSSharpe > 0 {
		drop := (*active.OOSSharpe - oosSharpe) / *active.OOSSharpe * 100.0
		if drop > c.DegradationThreshold {
			log.Printf("[reopt] %s: new params degrade OOS by %.1f%% — keeping current active params",
				strategyID, drop)
			// Still save for audit, but don't activate.
			c.saveVersion(ctx, strategyID, params, startDate, endDate, oosSharpe, false)
			return nil
		}
	}

	versionTag := fmt.Sprintf("reopt-%s", time.Now().Format("20060102-150405"))
	shouldActivate := c.EnableAutoActivate
	if active == nil {
		shouldActivate = true // Always activate first optimization.
	}

	return c.saveVersion(ctx, strategyID, params, startDate, endDate, oosSharpe, shouldActivate, versionTag)
}

func (c *ReoptimizationConfig) saveVersion(ctx context.Context, strategyID string, params map[string]float64,
	isStart, isEnd time.Time, oosSharpe float64, activate bool, versionTag ...string) error {

	if c.Repo == nil {
		return fmt.Errorf("repo not configured")
	}

	tag := fmt.Sprintf("reopt-%s", time.Now().Format("20060102-150405"))
	if len(versionTag) > 0 && versionTag[0] != "" {
		tag = versionTag[0]
	}

	pv := &db.ParamVersion{
		StrategyID:    strategyID,
		VersionTag:    tag,
		Params:        params,
		InSampleStart: &isStart,
		InSampleEnd:   &isEnd,
		OOSSharpe:     &oosSharpe,
		IsActive:      activate,
	}

	if err := c.Repo.SaveParamVersion(ctx, pv); err != nil {
		return fmt.Errorf("save param version: %w", err)
	}

	if activate {
		if err := c.Repo.ActivateParams(ctx, strategyID, tag); err != nil {
			return fmt.Errorf("activate params: %w", err)
		}
		log.Printf("[reopt] %s: activated version %s (OOS Sharpe=%.3f)", strategyID, tag, oosSharpe)
	} else {
		log.Printf("[reopt] %s: saved version %s (OOS Sharpe=%.3f, manual review required)", strategyID, tag, oosSharpe)
	}

	return nil
}
