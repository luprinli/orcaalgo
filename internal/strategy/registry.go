package strategy

import (
	"sync"

	"github.com/lee-econ/orca-core/internal/types"
)

type Strategy interface {
	Name() string
	Type() string
	Version() (irVersion string, canonicalVersion string)
	SetVersion(irVersion, canonicalVersion string)
	SetInstanceHash(h string)
	InstanceHash() string
	Evaluate(candle Candle, regime int8) *Signal
	Reset()
	Params() map[string]float64
	SetParams(params map[string]float64)
	ParamDefs() []ParamDef
	OnFill(orderID string, symbol string, side string, entryPrice types.Price, fillPrice types.Price, quantity float64, filledQty float64)
	OnCancel(orderID string, reason string)
	OnOrderRejected(orderID string, reason string)
}

type Runner = Strategy

type Registry struct {
	entries   map[string]Strategy
	factories map[string]func() Strategy
}

var (
	globalReg     *Registry
	globalRegOnce sync.Once
)

func NewRegistry() *Registry {
	return &Registry{
		entries:   make(map[string]Strategy),
		factories: make(map[string]func() Strategy),
	}
}

func (r *Registry) Register(strategy Strategy) {
	r.entries[strategy.Name()] = strategy
}

// RegisterFactory registers a constructor that produces a fresh, independent
// runner instance for the given name. Used to hand each concurrent backtest
// engine its own runner so that mutable per-run state (e.g. GridRunner's
// openPositions map) is never shared across goroutines.
func (r *Registry) RegisterFactory(name string, fn func() Strategy) {
	r.factories[name] = fn
}

func (r *Registry) Get(name string) Strategy {
	return r.entries[name]
}

// Create returns a fresh runner instance for the given name via its registered
// factory. Falls back to the shared singleton (Get) when no factory exists.
// Callers that run backtests concurrently MUST use Create, not Get, because
// runners carry mutable per-run state that is not safe to share.
func (r *Registry) Create(name string) Strategy {
	if fn, ok := r.factories[name]; ok {
		return fn()
	}
	return r.entries[name]
}

func (r *Registry) GetVersion(strategyID string) (irVersion string, canonicalVersion string) {
	if s, ok := r.entries[strategyID]; ok {
		return s.Version()
	}
	return "", ""
}

func (r *Registry) All() []Strategy {
	out := make([]Strategy, 0, len(r.entries))
	for _, s := range r.entries {
		out = append(out, s)
	}
	return out
}

func (r *Registry) EvaluateAll(candle Candle, regime int8) []*Signal {
	var signals []*Signal
	for name, s := range r.entries {
		sig := s.Evaluate(candle, regime)
		if sig != nil {
			sig.StrategyID = name
			signals = append(signals, sig)
		}
	}
	return signals
}

func (r *Registry) SetActiveStrategyParams(strategyID string, params map[string]float64) {
	if runner, ok := r.entries[strategyID]; ok {
		runner.SetParams(params)
	}
}

func (r *Registry) SetAllStrategyParams(params map[string]float64) {
	for _, runner := range r.entries {
		runner.SetParams(params)
	}
}

// Factories returns all registered factory constructors. Used by LiveEngine to
// create per-account isolated strategy instances.
func (r *Registry) Factories() map[string]func() Strategy {
	return r.factories
}

func GlobalRegistry() *Registry {
	globalRegOnce.Do(func() {
		globalReg = NewRegistry()
		// Factory table: name -> constructor. Each entry produces an independent
		// runner instance so concurrent backtests never share mutable state.
		factories := map[string]func() Strategy{
			"opening_range_breakout": func() Strategy { return NewOrbRunner() },
			"orb":                    func() Strategy { return NewOrbRunner() },
			"breakout":               func() Strategy { return NewOrbRunner() },
			"orb_15m":                func() Strategy { r := NewOrbRunner(); r.RangeMinutes = 15; return r },
			"grid":                   func() Strategy { r := NewGridRunner(); r.Disabled = true; return r },
			"grid_trading":           func() Strategy { r := NewGridRunner(); r.Disabled = true; return r },
			"vol_grid":               func() Strategy { r := NewGridRunner(); r.AdjustByVolatility = true; return r },
			"trend_following":        func() Strategy { return NewTrendRunner() },
			"trend":                  func() Strategy { return NewTrendRunner() },
			"session_scalp":          func() Strategy { return NewSessionScalpRunner() },
			"scalp":                  func() Strategy { return NewSessionScalpRunner() },
			"mean_reversion":         func() Strategy { return NewMeanReversionRunner(20, 1.25, 0.5, 200) },
			"intraday_mr":            func() Strategy { return NewMeanReversionRunner(20, 1.25, 0.5, 200) },
			"ma_crossover":           func() Strategy { return NewMACrossoverRunner() },
			"macd_rsi":               func() Strategy { return NewMACrossoverRunner() },
			"rsi2_reversion":         func() Strategy { return NewRSI2MeanReversionRunner() },
			"rsi2":                   func() Strategy { return NewRSI2MeanReversionRunner() },
			"donchian_breakout":      func() Strategy { return NewDonchianBreakoutRunner() },
			"donchian":               func() Strategy { return NewDonchianBreakoutRunner() },
			"keltner_macd":           func() Strategy { return NewKeltnerMACDRunner() },
			"keltner":                func() Strategy { return NewKeltnerMACDRunner() },
			"ichimoku_cloud":         func() Strategy { return NewIchimokuRunner() },
			"ichimoku":               func() Strategy { return NewIchimokuRunner() },
			"pairs_trading":          func() Strategy { return NewPairsRunner("", "") },
			"stat_arb":               func() Strategy { return NewPairsRunner("", "") },
			"volatility_harvesting":  func() Strategy { return NewVolHarvestingRunner() },
			"vol_arb":                func() Strategy { return NewVolHarvestingRunner() },
			"dragon_trend":           func() Strategy { return NewDragonTrendRunner() },
			"vwap_mr":                func() Strategy { return &MeanReversionRunner{
				Lookback: 20, EntryZ: 1.5, ExitZ: 0.5, MaxHold: 40, TrendPeriod: 100, Mode: "vwap",
				closeHistory: make([]float64, 220), volumeHistory: make([]float64, 220),
			}},
			"volume_scalp":      func() Strategy { return NewVolumeScalpRunner() },
			"vix_futures_carry": func() Strategy { return NewVIXFuturesCarryRunner() },
		}
		for name, fn := range factories {
			globalReg.factories[name] = fn
			globalReg.entries[name] = fn() // shared singleton for live/param-def use
		}
	})
	return globalReg
}
