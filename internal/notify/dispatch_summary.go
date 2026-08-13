package notify

import (
	"math"

	"github.com/lee-econ/orca-core/internal/types"
)

// DispatchSummaryOperation is a single pending order considered by the dispatch
// summary. It mirrors the fields an operator needs to review before a broker
// order is sent, plus the order-type metadata used for fill estimation.
type DispatchSummaryOperation struct {
	Ticker        string      `json:"ticker"`
	OperationType string      `json:"operation_type"` // open_position | close_position | update_stop_loss
	Quantity      float64     `json:"quantity"`
	Price         types.Price `json:"price"`
	OrderType     string      `json:"order_type"` // market | limit
}

// OrderSizeStats is the min/avg/max absolute notional across a set of orders.
type OrderSizeStats struct {
	Min float64 `json:"min"`
	Avg float64 `json:"avg"`
	Max float64 `json:"max"`
}

// CashImpactSummary estimates the net cash effect of a batch of pending orders.
type CashImpactSummary struct {
	TotalImpact     float64 `json:"total_impact"`
	EstimatedImpact float64 `json:"estimated_impact"` // limit orders weighted by fill probability
	Considered      int     `json:"considered"`
	MissingPricing  int     `json:"missing_pricing"`
	Eligible        int     `json:"eligible"`
	LimitOrders     int     `json:"limit_orders"`
	LimitAdjusted   int     `json:"limit_adjusted"`
	LimitMissing    int     `json:"limit_missing"`
}

// DefaultSigmaFallback is used when a ticker has too little price history to
// estimate its own volatility (2% daily).
const DefaultSigmaFallback = 0.02

// DefaultSigmaFloor prevents a zero-volatility estimate from making a limit
// order look like a certain fill (0.2%).
const DefaultSigmaFloor = 0.002

// normalCDF returns the standard normal CDF using the Go stdlib complementary
// error function (Phi(z) = 0.5 * erfc(-z / sqrt(2))). No hand-rolled erf.
func normalCDF(z float64) float64 {
	return 0.5 * math.Erfc(-z/math.Sqrt2)
}

// LimitFillProbability estimates the probability that a limit order at `distancePct`
// away from the reference price fills, given a daily log-return sigma.
//
// For a buy limit `n` below reference: z = log(1/(1-n)) / sigma.
// For a sell limit `n` above reference: z = log(1+n) / sigma.
// p = 2 * (1 - Phi(z)), clamped to [0, 1].
//
// This is the "expected fill" model used to size cash-impact estimates in
// dispatch summary emails. `side` is "BUY" or "SELL".
func LimitFillProbability(side string, distancePct float64, sigma float64) float64 {
	if distancePct <= 0 {
		return 1.0 // market or at-market limit always fills
	}
	if sigma < DefaultSigmaFloor {
		sigma = DefaultSigmaFloor
	}
	var z float64
	switch side {
	case "BUY":
		z = math.Log(1.0/(1.0-distancePct)) / sigma
	default: // SELL
		z = math.Log(1.0+distancePct) / sigma
	}
	p := 2.0 * (1.0 - normalCDF(z))
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

// EstimateSigmaRolling computes the rolling standard deviation of daily log
// returns (sample variance, n-1) over the trailing `lookback` closes.
func EstimateSigmaRolling(closes []float64, lookback int) float64 {
	if lookback <= 1 {
		lookback = 20
	}
	if len(closes) < 2 {
		return DefaultSigmaFallback
	}
	start := 0
	if len(closes) > lookback {
		start = len(closes) - lookback
	}
	window := closes[start:]
	var prev float64
	prevSet := false
	ret := make([]float64, 0, len(window)-1)
	for _, c := range window {
		if c <= 0 {
			continue
		}
		if prevSet && prev > 0 {
			ret = append(ret, math.Log(c/prev))
		}
		prev = c
		prevSet = true
	}
	if len(ret) < 2 {
		return DefaultSigmaFallback
	}
	var mean, sumSq float64
	for _, r := range ret {
		mean += r
	}
	mean /= float64(len(ret))
	for _, r := range ret {
		d := r - mean
		sumSq += d * d
	}
	sigma := math.Sqrt(sumSq / float64(len(ret)-1))
	if sigma <= 0 || math.IsNaN(sigma) || math.IsInf(sigma, 0) {
		return DefaultSigmaFallback
	}
	return sigma
}

// EstimateLimitFillWeight returns a 0..1 fill probability for a limit order at
// the given distance, using the ticker's trailing closes to estimate sigma.
func EstimateLimitFillWeight(side string, distancePct float64, closes []float64, lookback int) float64 {
	return LimitFillProbability(side, distancePct, EstimateSigmaRolling(closes, lookback))
}

// operationCashDirection returns +1 (cash in / sell) for close_position and
// update_stop_loss, -1 (cash out / buy) for open_position, and 0 otherwise.
func operationCashDirection(op DispatchSummaryOperation) float64 {
	switch op.OperationType {
	case "close_position", "update_stop_loss":
		return 1
	case "open_position":
		return -1
	default:
		return 0
	}
}

// CalculateCashImpact estimates the net cash effect of a batch of operations.
// Market orders are counted at full notional; limit orders are weighted by their
// expected fill probability from the ticker's trailing volatility. `closes` maps
// a ticker to its trailing close series; `limitDistances` maps a ticker to the
// limit order's distance-from-reference fraction.
func CalculateCashImpact(
	ops []DispatchSummaryOperation,
	closes map[string][]float64,
	limitDistances map[string]float64,
	lookback int,
) CashImpactSummary {
	out := CashImpactSummary{Considered: len(ops)}
	for _, op := range ops {
		dir := operationCashDirection(op)
		if dir == 0 {
			continue
		}
		if op.Price.Float64() <= 0 || op.Quantity <= 0 {
			out.MissingPricing++
			continue
		}
		out.Eligible++
		notional := op.Quantity * op.Price.Float64()
		out.TotalImpact += dir * notional

		if op.OrderType == "limit" {
			out.LimitOrders++
			dist, ok := limitDistances[op.Ticker]
			if !ok {
				out.LimitMissing++
				out.EstimatedImpact += dir * notional // unknown distance: assume full fill
				continue
			}
			series, hasSeries := closes[op.Ticker]
			if !hasSeries || len(series) < 2 {
				out.LimitMissing++
				out.EstimatedImpact += dir * notional
				continue
			}
			side := "SELL"
			if op.OperationType == "open_position" {
				side = "BUY"
			}
			weight := EstimateLimitFillWeight(side, dist, series, lookback)
			out.EstimatedImpact += dir * notional * weight
			out.LimitAdjusted++
		} else {
			out.EstimatedImpact += dir * notional
		}
	}
	return out
}

// CalculateOrderSizeStats returns min/avg/max absolute notional across orders.
func CalculateOrderSizeStats(ops []DispatchSummaryOperation) OrderSizeStats {
	if len(ops) == 0 {
		return OrderSizeStats{}
	}
	stats := OrderSizeStats{Min: math.Inf(1)}
	var sum float64
	n := 0
	for _, op := range ops {
		if op.Price.Float64() <= 0 || op.Quantity <= 0 {
			continue
		}
		notional := math.Abs(op.Quantity * op.Price.Float64())
		sum += notional
		n++
		if notional < stats.Min {
			stats.Min = notional
		}
		if notional > stats.Max {
			stats.Max = notional
		}
	}
	if n == 0 {
		return OrderSizeStats{}
	}
	if math.IsInf(stats.Min, 1) {
		stats.Min = 0
	}
	stats.Avg = sum / float64(n)
	return stats
}

// DispatchSummary aggregates a batch of dispatched operations into a single
// email-ready review summary: order counts, expected cash impact (limit orders
// weighted by fill probability) and order-size statistics.
type DispatchSummary struct {
	AccountName  string                    `json:"account_name"`
	Provider     string                    `json:"provider"`
	Environment  string                    `json:"environment"`
	SentCount    int                       `json:"sent_count"`
	FailedCount  int                       `json:"failed_count"`
	SkippedCount int                       `json:"skipped_count"`
	CashImpact   CashImpactSummary         `json:"cash_impact"`
	OrderSize    OrderSizeStats            `json:"order_size"`
	Operations   []DispatchSummaryOperation `json:"operations"`
}

// BuildDispatchSummary computes a review summary for a batch of operations. It
// is pure over its inputs (no DB/broker access) so it can be unit-tested and
// reused by the API, scheduler, and paper/live dispatch paths identically.
func BuildDispatchSummary(
	accountName, provider, environment string,
	sent, failed, skipped int,
	ops []DispatchSummaryOperation,
	closes map[string][]float64,
	limitDistances map[string]float64,
	lookback int,
) DispatchSummary {
	return DispatchSummary{
		AccountName:  accountName,
		Provider:     provider,
		Environment:  environment,
		SentCount:    sent,
		FailedCount:  failed,
		SkippedCount: skipped,
		CashImpact:   CalculateCashImpact(ops, closes, limitDistances, lookback),
		OrderSize:    CalculateOrderSizeStats(ops),
		Operations:   ops,
	}
}
