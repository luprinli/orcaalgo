package risk

import (
	"encoding/json"
	"math"
	"os"
)

type EnhancedHMMParams struct {
	Transition       [4][4]float64  `json:"transition"`
	InitialProbs     [4]float64     `json:"initial_probs"`
	EmissionMeans    [4][5]float64  `json:"emission_means"`
	EmissionCovars   [4][5][5]float64 `json:"emission_covars"`
	EmissionDiagSDs  [4][5]float64  `json:"emission_diag_sds"`
	Loaded           bool
}

func LoadEnhancedHMMParams(path string) (EnhancedHMMParams, error) {
	var p EnhancedHMMParams
	data, err := os.ReadFile(path)
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return p, err
	}
	p.Loaded = true
	return p, nil
}

func emissionProbMultiDim(state int, obs [5]float64, params EnhancedHMMParams) float64 {
	if state < 0 || state >= 4 {
		return 0
	}
	means := params.EmissionMeans[state]
	sds := params.EmissionDiagSDs[state]

	var logProb float64
	const pi2 = 2.0 * 3.141592653589793

	for d := 0; d < 5; d++ {
		mu := means[d]
		sd := sds[d]
		if sd < 1e-4 {
			sd = 1e-4
		}
		z := (obs[d] - mu) / sd
		logProb += -0.5*z*z - math.Log(sd) - 0.5*math.Log(pi2)
	}

	return math.Exp(logProb)
}

func (t *HMMTracker) UpdateEnhanced(priceRaw, prevPrice int64, obs [5]float64, params EnhancedHMMParams) {
	if prevPrice <= 0 || !params.Loaded {
		return
	}

	ret := float64(priceRaw-prevPrice) / float64(prevPrice)
	t.LastReturns[t.ReturnIdx] = ret
	t.ReturnIdx = (t.ReturnIdx + 1) % 256
	if t.ReturnCount < 256 {
		t.ReturnCount++
	}
	if t.ReturnCount < 10 {
		return
	}

	var newAlpha [4]float64
	var bestConf float64
	bestState := 0

	for j := 0; j < 4; j++ {
		var sum float64
		for i := 0; i < 4; i++ {
			sum += t.Alpha[i] * params.Transition[i][j]
		}
		ep := emissionProbMultiDim(j, obs, params)
		newAlpha[j] = sum * ep
	}

	var total float64
	for j := 0; j < 4; j++ {
		total += newAlpha[j]
	}
	if total > 0 {
		for j := 0; j < 4; j++ {
			newAlpha[j] /= total
			if newAlpha[j] > bestConf {
				bestConf = newAlpha[j]
				bestState = j
			}
		}
	}

	t.Alpha = newAlpha
	t.CurrentState = HMMState(bestState)
	t.Confidence = bestConf
}
