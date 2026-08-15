package backtest

import (
	"math"
	"sync"
)

type CorrelationTracker struct {
	mu                sync.RWMutex
	equityHistories   map[string][]float64
	window            int
	threshold         float64
	brakeDuration     int
	brakes            map[string]int
	velocityThreshold float64
	cooldown          int
	velocityCooldown  map[string]int
	rhoHistory        map[string]float64
}

func NewCorrelationTracker(window int, threshold float64) *CorrelationTracker {
	return &CorrelationTracker{
		equityHistories:   make(map[string][]float64),
		window:            window,
		threshold:         threshold,
		brakeDuration:     10,
		brakes:            make(map[string]int),
		velocityThreshold: 0.3,
		cooldown:          3,
		velocityCooldown:  make(map[string]int),
		rhoHistory:        make(map[string]float64),
	}
}

func (ct *CorrelationTracker) SetVelocityBrakeParams(dailyThreshold float64, cooldownBars int) {
	ct.velocityThreshold = dailyThreshold
	ct.cooldown = cooldownBars
}

func (ct *CorrelationTracker) RecordEquity(strategyID string, equity float64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	history := ct.equityHistories[strategyID]
	history = append(history, equity)
	if len(history) > ct.window {
		history = history[len(history)-ct.window:]
	}
	ct.equityHistories[strategyID] = history
}

type PairCorrelation struct {
	StrategyA   string
	StrategyB   string
	Correlation float64
	BrakeActive bool
}

type BreachEvent struct {
	StrategyA   string
	StrategyB   string
	Correlation float64
	Action      string
}

func (ct *CorrelationTracker) CheckCorrelations() ([]PairCorrelation, []BreachEvent) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ids := ct.activeIDs()
	var correlations []PairCorrelation
	var breaches []BreachEvent

	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			a, b := ids[i], ids[j]
			rho := ct.computePearson(a, b)
			pairKey := pairKey(a, b)
			prevRho := ct.previousRho(pairKey)

			pc := PairCorrelation{
				StrategyA:   a,
				StrategyB:   b,
				Correlation: rho,
			}

			if rho > ct.threshold {
				ct.brakes[pairKey] = ct.brakeDuration
				pc.BrakeActive = true
				breaches = append(breaches, BreachEvent{
					StrategyA:   a,
					StrategyB:   b,
					Correlation: rho,
					Action:      "brake_applied",
				})
			} else if ct.brakes[pairKey] > 0 {
				ct.brakes[pairKey]--
				pc.BrakeActive = true
				if ct.brakes[pairKey] == 0 {
					breaches = append(breaches, BreachEvent{
						StrategyA:   a,
						StrategyB:   b,
						Correlation: rho,
						Action:      "brake_released",
					})
				}
			}

			if ct.velocityCooldown[pairKey] == 0 && prevRho != 0 {
				rhoDelta := math.Abs(rho - prevRho)
				if rhoDelta > ct.velocityThreshold {
					ct.brakes[pairKey] = ct.brakeDuration
					pc.BrakeActive = true
					ct.velocityCooldown[pairKey] = ct.cooldown
					breaches = append(breaches, BreachEvent{
						StrategyA:   a,
						StrategyB:   b,
						Correlation: rho,
						Action:      "brake_applied_velocity",
					})
				}
			}
			if ct.velocityCooldown[pairKey] > 0 {
				ct.velocityCooldown[pairKey]--
			}
			ct.storeRho(pairKey, rho)

			correlations = append(correlations, pc)
		}
	}

	return correlations, breaches
}

func (ct *CorrelationTracker) IsBrakeActive(strategyA, strategyB string) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.brakes[pairKey(strategyA, strategyB)] > 0
}

func (ct *CorrelationTracker) BrakeDiscount(strategyA, strategyB string) float64 {
	if ct.IsBrakeActive(strategyA, strategyB) {
		return 0.70
	}
	return 1.0
}

func (ct *CorrelationTracker) PairMatrix() map[string]map[string]float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	ids := ct.activeIDs()
	matrix := make(map[string]map[string]float64)

	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			a, b := ids[i], ids[j]
			rho := ct.computePearson(a, b)
			if matrix[a] == nil {
				matrix[a] = make(map[string]float64)
			}
			if matrix[b] == nil {
				matrix[b] = make(map[string]float64)
			}
			matrix[a][b] = rho
			matrix[b][a] = rho
		}
	}
	return matrix
}

func (ct *CorrelationTracker) activeIDs() []string {
	var ids []string
	for id, history := range ct.equityHistories {
		if len(history) >= 2 {
			ids = append(ids, id)
		}
	}
	return ids
}

func (ct *CorrelationTracker) computePearson(a, b string) float64 {
	eqA := ct.equityHistories[a]
	eqB := ct.equityHistories[b]
	if len(eqA) < 2 || len(eqB) < 2 {
		return 0
	}

	returnsA := returnsFromEquity(eqA)
	returnsB := returnsFromEquity(eqB)

	n := intMin(len(returnsA), len(returnsB))
	if n < 2 {
		return 0
	}

	returnsA = returnsA[len(returnsA)-n:]
	returnsB = returnsB[len(returnsB)-n:]

	var sumA, sumB, sumAB, sumA2, sumB2 float64
	for i := 0; i < n; i++ {
		sumA += returnsA[i]
		sumB += returnsB[i]
		sumAB += returnsA[i] * returnsB[i]
		sumA2 += returnsA[i] * returnsA[i]
		sumB2 += returnsB[i] * returnsB[i]
	}

	num := float64(n)*sumAB - sumA*sumB
	den := math.Sqrt((float64(n)*sumA2 - sumA*sumA) * (float64(n)*sumB2 - sumB*sumB))

	if den < 1e-12 {
		return 0
	}
	rho := num / den
	if math.IsNaN(rho) || math.IsInf(rho, 0) {
		return 0
	}
	return rho
}

func returnsFromEquity(equity []float64) []float64 {
	if len(equity) < 2 {
		return nil
	}
	returns := make([]float64, len(equity)-1)
	for i := 1; i < len(equity); i++ {
		if equity[i-1] > 0 {
			returns[i-1] = (equity[i] - equity[i-1]) / equity[i-1]
		}
	}
	return returns
}

func pairKey(a, b string) string {
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}

func (ct *CorrelationTracker) previousRho(key string) float64 {
	return ct.rhoHistory[key]
}

func (ct *CorrelationTracker) storeRho(key string, rho float64) {
	ct.rhoHistory[key] = rho
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
