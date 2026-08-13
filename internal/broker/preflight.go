package broker

import "context"

// PreflightResult is the outcome of a dispatch preflight: whether an operation
// should be skipped before it reaches the broker, and why. It complements (not
// replaces) the RiskPipeline by guarding against state/reconciliation errors —
// duplicate opens, closes with no position, insufficient buying power — rather
// than risk-limit breaches.
type PreflightResult struct {
	Skip   bool   `json:"skip"`
	Reason string `json:"reason,omitempty"`
}

// Preflight checks a pending order against the broker's current state using only
// the Adapter interface (positions + account), so it works identically for every
// broker. When the broker cannot be queried it returns a non-skipping result
// (fail-open: the RiskPipeline still gates the order).
func Preflight(ctx context.Context, adapter Adapter, req *OrderRequest) PreflightResult {
	if adapter == nil || req == nil {
		return PreflightResult{}
	}

	positions, err := adapter.GetPositions(ctx)
	if err != nil {
		return PreflightResult{}
	}

	held := 0.0
	for _, p := range positions {
		if p.Symbol == req.Symbol {
			held = p.Quantity
			break
		}
	}

	switch req.Side {
	case Buy:
		if held > 0 {
			return PreflightResult{Skip: true, Reason: "position already open"}
		}
		// Buying-power check requires a reference price (limit order). Market
		// orders have no known fill price and are skipped.
		if req.LimitPrice.Float64() > 0 {
			acct, err := adapter.GetAccount(ctx)
			if err == nil && acct != nil && acct.BuyingPower.Float64() > 0 {
				if notional := req.Quantity * req.LimitPrice.Float64(); notional > acct.BuyingPower.Float64() {
					return PreflightResult{Skip: true, Reason: "insufficient buying power"}
				}
			}
		}
	case Sell:
		if held <= 0 {
			return PreflightResult{Skip: true, Reason: "no position to close"}
		}
	}

	return PreflightResult{}
}
