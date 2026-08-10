package backtest

import (
	"math"
	"sort"

	"github.com/lee-econ/orca-core/internal/risk"
)

type RebalanceScheduler struct {
	T              int
	BarCount       int
	matrix         *risk.RegimeActivationMatrix
	trailingSharpe map[string][]float64
}

func NewRebalanceScheduler(barCadence int, matrix *risk.RegimeActivationMatrix) *RebalanceScheduler {
	return &RebalanceScheduler{
		T:              barCadence,
		BarCount:       0,
		matrix:         matrix,
		trailingSharpe: make(map[string][]float64),
	}
}

type EligibilityResult struct {
	StrategyID string
	Eligible   bool
	Sharpe     float64
	Kelly      float64
	Reason     string
}

func (s *RebalanceScheduler) EvaluateEligibility(strategyID string, regime int8, hasSignal bool, kellyOverride float64) EligibilityResult {
	result := EligibilityResult{
		StrategyID: strategyID,
		Kelly:      kellyOverride,
	}

	if !s.matrix.IsAllowed(strategyID, regime) {
		result.Reason = "regime_blocked"
		return result
	}

	kellyForRegime := s.matrix.KellyForRegime(strategyID, regime)
	if kellyForRegime > 0 {
		result.Kelly = kellyForRegime
	}

	if !hasSignal {
		result.Reason = "no_signal"
		return result
	}

	result.Eligible = true
	result.Sharpe = s.trailingSharpeFor(strategyID)
	result.Reason = "active"
	return result
}

func (s *RebalanceScheduler) ComputeWeights(active []EligibilityResult) map[string]float64 {
	if len(active) == 0 {
		return nil
	}

	numerator := make(map[string]float64)
	var denominator float64

	for _, e := range active {
		w := e.Kelly * floatMax(e.Sharpe, 0)
		numerator[e.StrategyID] = w
		denominator += w
	}

	weights := make(map[string]float64)
	if denominator <= 0 {
		equal := 1.0 / float64(len(active))
		for _, e := range active {
			weights[e.StrategyID] = equal
		}
		return weights
	}

	for _, e := range active {
		weights[e.StrategyID] = numerator[e.StrategyID] / denominator
	}
	return weights
}

func (s *RebalanceScheduler) IsFullRebalanceDue() bool {
	s.BarCount++
	if s.BarCount >= s.T {
		s.BarCount = 0
		return true
	}
	return false
}

func (s *RebalanceScheduler) RecordSharpe(strategyID string, trades []Trade) {
	sharpe := calculateTrailingSharpeFromTrades(trades, 20)
	list := s.trailingSharpe[strategyID]
	list = append(list, sharpe)
	if len(list) > 20 {
		list = list[len(list)-20:]
	}
	s.trailingSharpe[strategyID] = list
}

func (s *RebalanceScheduler) trailingSharpeFor(strategyID string) float64 {
	list, ok := s.trailingSharpe[strategyID]
	if !ok || len(list) == 0 {
		return 0
	}
	last := list[len(list)-1]
	if math.IsNaN(last) || math.IsInf(last, 0) {
		return 0
	}
	return last
}

func (s *RebalanceScheduler) ActiveWeight(weights map[string]float64, strategyID string) float64 {
	if w, ok := weights[strategyID]; ok {
		return w
	}
	return 0
}

func (s *RebalanceScheduler) CadenceForTimeframe(tf string) int {
	switch tf {
	case "1d":
		return 20
	case "4h":
		return 40
	case "1h":
		return 80
	case "30m":
		return 120
	case "15m":
		return 160
	default:
		return 40
	}
}

func calculateTrailingSharpeFromTrades(trades []Trade, window int) float64 {
	if len(trades) < 2 {
		return 0
	}
	subset := trades
	if len(trades) > window {
		subset = trades[len(trades)-window:]
	}
	returns := make([]float64, len(subset))
	for i, t := range subset {
		returns[i] = t.PnLPct
	}
	sort.Float64s(returns)
	return sharpeFromReturns(returns)
}

func sharpeFromReturns(returns []float64) float64 {
	n := len(returns)
	if n < 2 {
		return 0
	}
	var sum float64
	for _, r := range returns {
		sum += r
	}
	mean := sum / float64(n)
	var variance float64
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(n - 1)
	if variance < 1e-12 {
		return 0
	}
	return mean / math.Sqrt(variance)
}

func floatMax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
