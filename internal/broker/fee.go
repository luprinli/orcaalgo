package broker

import "math"

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

// CalculateFeeForAssetClass computes the per-leg fee using an asset-class-aware
// schedule. Equity (and the default) uses the full retail schedule (per-share +
// SEC + $1 minimum). Forex, crypto, futures, and commodity instruments use a
// notional-bps schedule with no per-share fee, no SEC fee, and no $1 minimum,
// because the "quantity" of those instruments is not a share count — applying
// the equity schedule to them (e.g. ~3,000 "units" of EURUSD at $0.0035/share)
// produces fees that are orders of magnitude above real market costs.
func (f BrokerageFeeConfig) CalculateFeeForAssetClass(assetClass string, quantity float64, price float64) float64 {
	if !f.Enabled {
		return 0
	}
	notional := quantity * price
	switch assetClass {
	case "forex", "crypto", "futures", "commodity", "index":
		fee := f.PerTradeFixed + notional*f.TakerFeeBps/10000.0
		if fee < 0 {
			return 0
		}
		return fee
	default:
		return f.CalculateFee(quantity, price)
	}
}

// CalculateFeeForSymbol is CalculateFeeForAssetClass keyed by a coarse
// per-ticker asset classification.
func (f BrokerageFeeConfig) CalculateFeeForSymbol(symbol string, quantity float64, price float64) float64 {
	return f.CalculateFeeForAssetClass(AssetClassForSymbol(symbol), quantity, price)
}

// CalculateHoldingFee computes the time-proportional holding cost (e.g. an ETF
// expense ratio) for a position of `notional` held for `yearsHeld` years at the
// given annual `expenseRatio`. It returns 0 for negative, non-finite, or
// zero-valued inputs (no negative fees, no division-by-zero). This is applied to
// long positions only; shorts have no expense-ratio analogue in this model.
func (f BrokerageFeeConfig) CalculateHoldingFee(notional, expenseRatio, yearsHeld float64) float64 {
	if !f.Enabled || notional <= 0 || expenseRatio <= 0 || yearsHeld <= 0 {
		return 0
	}
	holding := notional * expenseRatio * yearsHeld
	if math.IsNaN(holding) || math.IsInf(holding, 0) || holding < 0 {
		return 0
	}
	return holding
}

// AssetClassForSymbol classifies a ticker into a coarse asset class for fee and
// slippage schedule selection. It is intentionally self-contained (no config
// dependency) so it also handles legacy and non-universe tickers (e.g. futures
// "ES"/"NQ"/"CL", metal "XAUUSD", crypto "BTCUSD"/"BTC-USD").
func AssetClassForSymbol(symbol string) string {
	switch {
	case isForex(symbol):
		return "forex"
	case isCrypto(symbol):
		return "crypto"
	case isFutures(symbol):
		return "futures"
	case isCommodity(symbol):
		return "commodity"
	default:
		return "equity"
	}
}

func isForex(symbol string) bool {
	switch symbol {
	case "EURUSD", "GBPUSD", "USDJPY", "AUDUSD", "USDCAD", "USDCHF", "NZDUSD",
		"USDEUR", "US30", "SPX500", "NAS100", "GER40", "UK100", "JPN225",
		"^_US", "^DAX":
		return true
	}
	return false
}

func isCrypto(symbol string) bool {
	switch symbol {
	case "BTCUSD", "ETHUSD", "BTC-USD", "ETH-USD", "SOLUSD", "XRPUSD",
		"ADAUSD", "DOGEUSD", "LTCUSD", "BCHUSD", "DOTUSD", "LINKUSD",
		"AVAXUSD", "MATICUSD", "TRXUSD", "UNIUSD":
		return true
	}
	return false
}

func isFutures(symbol string) bool {
	switch symbol {
	case "ES", "NQ", "CL", "YM", "RTY", "GC", "SI", "HG", "NG",
		"ZB", "ZN", "ZF", "6E", "6A", "6B", "6J", "6C", "6S", "6N":
		return true
	}
	return false
}

func isCommodity(symbol string) bool {
	switch symbol {
	case "XAUUSD", "XAGUSD", "GLD", "SLV", "USO", "UNG":
		return true
	}
	return false
}
