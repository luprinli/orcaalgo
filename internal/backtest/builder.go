package backtest

import (
	"fmt"
	"time"

	"github.com/lee-econ/orca-core/internal/broker"
)

type BacktestBuilder struct {
	strategyID     string
	symbols        []string
	startDate      time.Time
	endDate        time.Time
	initialCapital float64
	commissionBps  float64
	brokerFee      broker.BrokerageFeeConfig
	gkrPath        string
	timeframe      string
	dataSource     string
	fixedSeed      int64
	enableFTMO     bool
	enablePrefetch bool
	built          bool
}

func NewBacktestBuilder() *BacktestBuilder {
	return &BacktestBuilder{
		initialCapital: 100_000,
		commissionBps:  5.0,
		timeframe:      "1d",
		dataSource:     "db",
		enablePrefetch: false,
	}
}

func (b *BacktestBuilder) WithStrategy(id string) *BacktestBuilder {
	b.strategyID = id
	return b
}

func (b *BacktestBuilder) WithSymbols(symbols ...string) *BacktestBuilder {
	b.symbols = symbols
	return b
}

func (b *BacktestBuilder) WithDateRange(start, end time.Time) *BacktestBuilder {
	b.startDate = start
	b.endDate = end
	return b
}

func (b *BacktestBuilder) WithInitialCapital(capital float64) *BacktestBuilder {
	b.initialCapital = capital
	return b
}

func (b *BacktestBuilder) WithCommission(bps float64) *BacktestBuilder {
	b.commissionBps = bps
	return b
}

func (b *BacktestBuilder) WithBrokerFee(fee broker.BrokerageFeeConfig) *BacktestBuilder {
	b.brokerFee = fee
	return b
}

func (b *BacktestBuilder) WithGKRPath(path string) *BacktestBuilder {
	b.gkrPath = path
	return b
}

func (b *BacktestBuilder) WithTimeframe(tf string) *BacktestBuilder {
	b.timeframe = tf
	return b
}

func (b *BacktestBuilder) WithDataSource(source string) *BacktestBuilder {
	b.dataSource = source
	return b
}

func (b *BacktestBuilder) WithSeed(seed int64) *BacktestBuilder {
	b.fixedSeed = seed
	return b
}

func (b *BacktestBuilder) WithPrefetch() *BacktestBuilder {
	b.enablePrefetch = true
	return b
}

func (b *BacktestBuilder) Build() (*BacktestConfig, error) {
	if b.built {
		return nil, fmt.Errorf("builder already used")
	}
	b.built = true

	if b.strategyID == "" {
		return nil, fmt.Errorf("strategy ID is required")
	}
	if len(b.symbols) == 0 {
		return nil, fmt.Errorf("at least one symbol is required")
	}

	cfg := &BacktestConfig{
		StrategyID:     b.strategyID,
		Symbols:        b.symbols,
		StartDate:      b.startDate,
		EndDate:        b.endDate,
		InitialCapital: b.initialCapital,
		CommissionBps:  b.commissionBps,
		BrokerFee:      b.brokerFee,
		GKRPath:        b.gkrPath,
		Timeframe:      b.timeframe,
		DataSource:     b.dataSource,
		FixedSeed:      b.fixedSeed,
		EnablePrefetch: b.enablePrefetch,
	}

	if b.enableFTMO {
		cfg.PropFirmEnabled = true
	}

	return cfg, nil
}


