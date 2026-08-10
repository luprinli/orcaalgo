package backtest

import (
	"time"
)

type StrategyState string

const (
	StrategyActive    StrategyState = "active"
	StrategyInactive  StrategyState = "inactive"
	StrategyStandby   StrategyState = "standby"
	StrategyViolated  StrategyState = "violated"
	StrategyValidated StrategyState = "validated"
)

type ReevaluationConfig struct {
	SharpeDegradationPct float64
	DegradationDays      int
	RecoveryDays         int
	MaxDDThreshold       map[string]float64
	RegimeExitBars       int
	CorrelationBrakeDays int
	FillDegradationBps   float64
	FillDegradationCount int
	OOSValidationDays    int
	OOSMinSharpe         float64
	OOSMaxSharpeDropPct  float64
}

func DefaultReevaluationConfig() ReevaluationConfig {
	return ReevaluationConfig{
		SharpeDegradationPct: 0.30,
		DegradationDays:      30,
		RecoveryDays:         10,
		MaxDDThreshold: map[string]float64{
			"grid_trading":         15.0,
			"rsi2_reversion":       10.0,
			"trend_following":      25.0,
			"volatility_harvesting": 15.0,
			"donchian_breakout":    20.0,
		},
		RegimeExitBars:       5,
		CorrelationBrakeDays: 10,
		FillDegradationBps:   2.0,
		FillDegradationCount: 20,
		OOSValidationDays:    60,
		OOSMinSharpe:         0.0,
		OOSMaxSharpeDropPct:  0.50,
	}
}

type StrategyReevaluator struct {
	config           ReevaluationConfig
	benchmarkSharpe  map[string]float64
	sharpeHistory    map[string][]sharpePoint
	fillSlippage     map[string][]float64
	maxDD            map[string]float64
	peakEquity       map[string]float64
}

type sharpePoint struct {
	time   time.Time
	sharpe float64
}

func NewStrategyReevaluator(config ReevaluationConfig, benchmarkSharpe map[string]float64) *StrategyReevaluator {
	if config.MaxDDThreshold == nil {
		config.MaxDDThreshold = make(map[string]float64)
	}
	return &StrategyReevaluator{
		config:          config,
		benchmarkSharpe: benchmarkSharpe,
		sharpeHistory:   make(map[string][]sharpePoint),
		fillSlippage:    make(map[string][]float64),
		maxDD:           make(map[string]float64),
		peakEquity:      make(map[string]float64),
	}
}

type ReevaluationResult struct {
	StrategyID string
	Action     string
	Reason     string
	NewState   StrategyState
	NewWeight  float64
}

func (sr *StrategyReevaluator) Evaluate(currentSharpe map[string]float64, currentDD map[string]float64,
	currentState map[string]StrategyState, currentWeight map[string]float64,
	now time.Time) []ReevaluationResult {

	var results []ReevaluationResult

	for sid := range currentState {
		state := currentState[sid]
		sharpe := currentSharpe[sid]
		dd := currentDD[sid]
		weight := currentWeight[sid]
		benchmark, hasBench := sr.benchmarkSharpe[sid]

		sr.recordSharpe(sid, sharpe, now)

		result := sr.evaluateDemotion(sid, state, sharpe, dd, weight, benchmark, hasBench, now)
		if result != nil {
			results = append(results, *result)
			continue
		}

		result = sr.evaluatePromotion(sid, state, sharpe, dd, weight, benchmark, hasBench, now)
		if result != nil {
			results = append(results, *result)
		}
	}

	return results
}

func (sr *StrategyReevaluator) evaluateDemotion(sid string, state StrategyState, sharpe, dd, weight, benchmark float64,
	hasBench bool, now time.Time) *ReevaluationResult {

	if state == StrategyViolated {
		return nil
	}

	maxDDThreshold, ok := sr.config.MaxDDThreshold[sid]
	if !ok {
		maxDDThreshold = 15.0
	}
	if dd > maxDDThreshold {
		return &ReevaluationResult{
			StrategyID: sid, Action: "hard_halt", Reason: "maxdd_breach",
			NewState: StrategyViolated, NewWeight: 0,
		}
	}

	if hasBench {
		daysBelow := sr.countDaysBelow(sid, benchmark*sr.config.SharpeDegradationPct, sr.config.DegradationDays)
		if daysBelow >= sr.config.DegradationDays {
			newWeight := weight * 0.25
			reason := "sharpe_degradation"
			if state == StrategyActive && daysBelow >= 60 {
				return &ReevaluationResult{
					StrategyID: sid, Action: "deactivate", Reason: reason,
					NewState: StrategyInactive, NewWeight: 0,
				}
			}
			return &ReevaluationResult{
				StrategyID: sid, Action: "reduce_allocation", Reason: reason,
				NewState: StrategyActive, NewWeight: newWeight,
			}
		}
	}

	return nil
}

func (sr *StrategyReevaluator) evaluatePromotion(sid string, state StrategyState, sharpe, dd, weight, benchmark float64,
	hasBench bool, now time.Time) *ReevaluationResult {

	if state == StrategyActive || state == StrategyViolated {
		return nil
	}

	if hasBench && state == StrategyInactive {
		daysAbove := sr.countDaysAbove(sid, benchmark*0.50, sr.config.RecoveryDays)
		if daysAbove >= sr.config.RecoveryDays {
			return &ReevaluationResult{
				StrategyID: sid, Action: "restore_50pct", Reason: "sharpe_recovery",
				NewState: StrategyActive, NewWeight: weight * 0.5 * 2,
			}
		}
	}

	if state == StrategyStandby {
		return &ReevaluationResult{
			StrategyID: sid, Action: "activate", Reason: "regime_reentry",
			NewState: StrategyActive, NewWeight: 0.10,
		}
	}

	return nil
}

func (sr *StrategyReevaluator) recordSharpe(sid string, sharpe float64, t time.Time) {
	history := sr.sharpeHistory[sid]
	history = append(history, sharpePoint{time: t, sharpe: sharpe})
	if len(history) > 60 {
		history = history[len(history)-60:]
	}
	sr.sharpeHistory[sid] = history
}

func (sr *StrategyReevaluator) countDaysBelow(sid string, threshold float64, window int) int {
	history := sr.sharpeHistory[sid]
	if len(history) == 0 {
		return 0
	}
	count := 0
	start := len(history) - window
	if start < 0 {
		start = 0
	}
	for i := start; i < len(history); i++ {
		if history[i].sharpe < threshold {
			count++
		}
	}
	return count
}

func (sr *StrategyReevaluator) countDaysAbove(sid string, threshold float64, window int) int {
	history := sr.sharpeHistory[sid]
	if len(history) == 0 {
		return 0
	}
	count := 0
	start := len(history) - window
	if start < 0 {
		start = 0
	}
	for i := start; i < len(history); i++ {
		if history[i].sharpe > threshold {
			count++
		}
	}
	return count
}

func (sr *StrategyReevaluator) RecordFillSlippage(sid string, slippageBps float64) {
	list := sr.fillSlippage[sid]
	list = append(list, slippageBps)
	if len(list) > 40 {
		list = list[len(list)-40:]
	}
	sr.fillSlippage[sid] = list
}

func (sr *StrategyReevaluator) AverageFillSlippage(sid string) float64 {
	list, ok := sr.fillSlippage[sid]
	if !ok || len(list) == 0 {
		return 0
	}
	var sum float64
	for _, s := range list {
		sum += s
	}
	return sum / float64(len(list))
}

func (sr *StrategyReevaluator) GetState(current map[string]StrategyState, sid string) StrategyState {
	if s, ok := current[sid]; ok {
		return s
	}
	return StrategyInactive
}
