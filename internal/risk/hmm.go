package risk

import "math"

type HMMState int8

const (
	HMMCalm     HMMState = 0
	HMMTrending HMMState = 1
	HMMHighVol  HMMState = 2
	HMMCrisis   HMMState = 3
)

type HMMModel struct {
	Transition     [4][4]float64
	InitialProbs   [4]float64
	EmissionMeans  [4]float64
	EmissionSDs    [4]float64
	Loaded         bool
}

type HMMTracker struct {
	Model        HMMModel
	CurrentState HMMState
	Confidence   float64
	Alpha        [4]float64
	LastReturns  [256]float64
	ReturnCount  int
	ReturnIdx    int
}

func DefaultHMM() HMMModel {
	return HMMModel{
		Transition: [4][4]float64{
			{0.85, 0.10, 0.04, 0.01},
			{0.08, 0.80, 0.10, 0.02},
			{0.03, 0.10, 0.80, 0.07},
			{0.01, 0.02, 0.10, 0.87},
		},
		InitialProbs:  [4]float64{0.70, 0.20, 0.08, 0.02},
		EmissionMeans: [4]float64{0.0002, 0.0005, -0.0003, -0.0015},
		EmissionSDs:   [4]float64{0.005, 0.012, 0.025, 0.060},
		Loaded:        true,
	}
}

func NewHMMTracker(model HMMModel) *HMMTracker {
	t := &HMMTracker{Model: model}
	t.Alpha = model.InitialProbs
	return t
}

func (t *HMMTracker) Update(priceRaw, prevPrice int64) {
	if prevPrice <= 0 || !t.Model.Loaded {
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

	vol := computeVolatility(t.LastReturns[:], t.ReturnCount)
	var newAlpha [4]float64

	for j := 0; j < 4; j++ {
		var sum float64
		for i := 0; i < 4; i++ {
			sum += t.Alpha[i] * t.Model.Transition[i][j]
		}
		newAlpha[j] = sum * emissionProb(j, ret, vol, t.Model.EmissionMeans, t.Model.EmissionSDs)
	}

	var total float64
	for j := 0; j < 4; j++ {
		total += newAlpha[j]
	}
	if total > 0 {
		for j := 0; j < 4; j++ {
			newAlpha[j] /= total
		}
		t.Alpha = newAlpha
	}

	bestState := int8(0)
	bestConf := 0.0
	for j := 0; j < 4; j++ {
		if t.Alpha[j] > bestConf {
			bestConf = t.Alpha[j]
			bestState = int8(j)
		}
	}
	t.CurrentState = HMMState(bestState)
	t.Confidence = bestConf
}

func (t *HMMTracker) GetRegime() (HMMState, float64) {
	return t.CurrentState, t.Confidence
}

func (t *HMMTracker) LoadCalibratedParams(transition [4][4]float64, initialProbs, means, sds [4]float64) {
	t.Model.Transition = transition
	t.Model.InitialProbs = initialProbs
	t.Model.EmissionMeans = means
	t.Model.EmissionSDs = sds
	t.Model.Loaded = true
	t.Alpha = initialProbs
}

func VIXModulateSDs(sds [4]float64, vixValue float64) [4]float64 {
	result := sds
	if vixValue > 30 {
		for i := 0; i < 4; i++ {
			result[i] *= 1.5
		}
	} else if vixValue > 25 {
		result[3] *= 1.3
	} else if vixValue < 12 {
		for i := 0; i < 4; i++ {
			result[i] *= 0.75
		}
	}
	return result
}

func computeVolatility(returns []float64, count int) float64 {
	if count < 2 {
		return 0.01
	}
	var mean float64
	for i := 0; i < count; i++ {
		mean += returns[i]
	}
	mean /= float64(count)
	var variance float64
	for i := 0; i < count; i++ {
		diff := returns[i] - mean
		variance += diff * diff
	}
	variance /= float64(count - 1)
	return math.Sqrt(variance)
}

func emissionProb(state int, ret, vol float64, means, sds [4]float64) float64 {
	mu := means[state]
	sd := sds[state]
	if sd < vol {
		sd = vol
	}
	if sd == 0 {
		sd = 0.001
	}
	z := (ret - mu) / sd
	return math.Exp(-0.5*z*z) / (sd * 2.50662827463)
}
