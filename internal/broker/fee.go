package broker

// BrokerageFeeConfig is the single source of truth for commission/fee calculations
// shared by the backtest engine and paper broker (R3 — docs/backtest_live_parity_audit.md).
type BrokerageFeeConfig struct {
	PerTradeFixed float64
	PerShare      float64
	MinFee        float64
	SECTxnFee     float64
	MakerFeeBps   float64
	TakerFeeBps   float64
	Enabled       bool
}

func DefaultBrokerageFee() BrokerageFeeConfig {
	return BrokerageFeeConfig{
		PerTradeFixed: 0.35,
		PerShare:      0.0035,
		MinFee:        1.00,
		SECTxnFee:     0.0000229,
		MakerFeeBps:   0.0,
		TakerFeeBps:   0.5,
		Enabled:       true,
	}
}

func (f BrokerageFeeConfig) CalculateFee(quantity float64, price float64) float64 {
	if !f.Enabled {
		return 0
	}
	fee := f.PerTradeFixed + (quantity * f.PerShare) + (quantity * price * f.SECTxnFee)
	fee += quantity * price * f.TakerFeeBps / 10000.0
	if fee < f.MinFee {
		return f.MinFee
	}
	return fee
}

func (f BrokerageFeeConfig) CalculateMakerFee(quantity float64, price float64) float64 {
	if !f.Enabled {
		return 0
	}
	fee := f.PerTradeFixed + (quantity * f.PerShare) + (quantity * price * f.SECTxnFee)
	fee += quantity * price * f.MakerFeeBps / 10000.0
	if fee < f.MinFee {
		return f.MinFee
	}
	return fee
}
