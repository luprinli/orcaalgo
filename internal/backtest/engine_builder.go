package backtest

import (
	"github.com/lee-econ/orca-core/internal/ml"
	"github.com/lee-econ/orca-core/internal/risk"
	strategy "github.com/lee-econ/orca-core/internal/strategy"
)

// EngineBuilder uses the Builder pattern to construct Engine instances
// with configurable slippage model, strategy, and random seed. Call
// Build() to produce the final Engine.
type EngineBuilder struct {
	db       Database
	slippage SlippageModel
	strategy strategy.Strategy
	seed     int64
}

// NewEngineBuilder creates an EngineBuilder with the given database and
// a default equity slippage model.
func NewEngineBuilder(db Database) *EngineBuilder {
	return &EngineBuilder{
		db:       db,
		slippage: DefaultEquitySlippage(),
	}
}

// WithSlippage sets the slippage model used for fill simulation.
func (b *EngineBuilder) WithSlippage(m SlippageModel) *EngineBuilder {
	b.slippage = m
	return b
}

// WithStrategy registers a strategy under the "default" symbol key.
func (b *EngineBuilder) WithStrategy(s strategy.Strategy) *EngineBuilder {
	b.strategy = s
	return b
}

// WithSeed sets the random seed for fill simulation. A zero seed (default)
// uses a non-deterministic source.
func (b *EngineBuilder) WithSeed(seed int64) *EngineBuilder {
	b.seed = seed
	return b
}

// Build assembles the Engine with the configured options, initializing
// fill simulator, order rate limiter, volatility halt, exposure tracker,
// Kelly multiplier, position sizer, and meta-labeler config.
func (b *EngineBuilder) Build() *Engine {
	var fillSim *FillSimulator
	if b.seed != 0 {
		fillSim = NewFillSimulatorWithSeed(b.slippage, b.seed)
	} else {
		fillSim = NewFillSimulator(b.slippage)
	}

	e := &Engine{
		db:            b.db,
		fillSim:       fillSim,
		stratBySymbol: make(map[string]strategy.Strategy),
		orderLimiter:  risk.NewOrderRateLimiter(10),
		volHalt:       risk.NewVolatilityHalt(3.0),
		exposure:      risk.NewExposureTracker(5.0, 0.25),
		kellyMult:     0.25,
		positionSizer: risk.NewPositionSizer(nil),
		regimeMatrix:  risk.NewRegimeActivationMatrix(),
		metaCfg:       ml.DefaultMetaLabelerConfig(),
	}

	if b.strategy != nil {
		e.stratBySymbol["default"] = b.strategy
	}

	return e
}
