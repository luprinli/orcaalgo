package backtest

import (
	"context"
	"time"
)

// StartTimingConfig parameterizes an entry-date sensitivity sweep: the same
// strategy is run from many staggered start dates over a fixed forward horizon,
// so a trader can see whether performance depends on when the strategy was
// switched on (a robustness signal orthogonal to walk-forward windows).
type StartTimingConfig struct {
	StrategyID     string
	Symbols        []string
	StartDate      time.Time // earliest sample start
	EndDate        time.Time // latest sample start bound
	HorizonMonths  int       // forward window length per sample
	StepWeeks      int       // spacing between consecutive sample starts
	InitialCapital float64
	DataSource     string
	Timeframe      string
}

// StartTimingWindow is one (start, end) sample window.
type StartTimingWindow struct {
	StartDate time.Time
	EndDate   time.Time
}

// StartTimingSample is the result of running the strategy over one window.
type StartTimingSample struct {
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	TotalReturnPct float64   `json:"total_return_pct"`
	SharpeRatio    float64   `json:"sharpe_ratio"`
	MaxDrawdown    float64   `json:"max_drawdown"`
	WinRate        float64   `json:"win_rate"`
	NumTrades      int       `json:"num_trades"`
}

// GenerateStartTimingWindows produces the staggered sample windows between
// start and end. Each window opens `stepWeeks` after the previous one and runs
// forward `horizonMonths`. Windows whose end would exceed the global end date
// are truncated to the available data (they still yield a shorter-horizon
// result). The function is pure so the window layout is unit-testable without
// a database.
func GenerateStartTimingWindows(start, end time.Time, horizonMonths, stepWeeks int) []StartTimingWindow {
	if !end.After(start) || horizonMonths <= 0 {
		return nil
	}
	step := time.Duration(stepWeeks) * 7 * 24 * time.Hour
	if step <= 0 {
		step = 7 * 24 * time.Hour
	}
	horizon := time.Duration(horizonMonths) * 30 * 24 * time.Hour

	var windows []StartTimingWindow
	for s := start; s.Before(end); s = s.Add(step) {
		e := s.Add(horizon)
		if e.After(end) {
			e = end
		}
		if !e.After(s) {
			continue
		}
		windows = append(windows, StartTimingWindow{StartDate: s, EndDate: e})
	}
	return windows
}

// RunStartTiming runs the strategy over each generated window and returns the
// per-window results. Windows that produce no trades are still reported (with
// zero metrics) so the caller can see where the strategy went quiet.
func (e *Engine) RunStartTiming(ctx context.Context, cfg StartTimingConfig) ([]StartTimingSample, error) {
	if cfg.InitialCapital <= 0 {
		cfg.InitialCapital = 100000.0
	}
	if cfg.HorizonMonths <= 0 {
		cfg.HorizonMonths = 6
	}
	if cfg.StepWeeks <= 0 {
		cfg.StepWeeks = 4
	}
	windows := GenerateStartTimingWindows(cfg.StartDate, cfg.EndDate, cfg.HorizonMonths, cfg.StepWeeks)

	samples := make([]StartTimingSample, 0, len(windows))
	for _, w := range windows {
		btCfg := BacktestConfig{
			StrategyID:      cfg.StrategyID,
			Symbols:         cfg.Symbols,
			StartDate:       w.StartDate,
			EndDate:         w.EndDate,
			InitialCapital:  cfg.InitialCapital,
			DataSource:      cfg.DataSource,
			Timeframe:       cfg.Timeframe,
			PropFirmEnabled: true,
			StopLoss:        &StopLossConfig{Type: "atr", ATRPeriod: 14, ATRMultiplier: 2.0},
			TakeProfit:      &TakeProfitConfig{Type: "risk_reward", RRRatio: 2.0},
		}
		res, err := e.Run(ctx, btCfg)
		if err != nil {
			return nil, err
		}
		samples = append(samples, StartTimingSample{
			StartDate:      w.StartDate,
			EndDate:        w.EndDate,
			TotalReturnPct: res.TotalReturnPct,
			SharpeRatio:    res.SharpeRatio,
			MaxDrawdown:    res.MaxDrawdown,
			WinRate:        res.WinRate,
			NumTrades:      res.NumTrades,
		})
	}
	return samples, nil
}
