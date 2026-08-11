package strategy

import (
	"math"

	"github.com/lee-econ/orca-core/internal/types"
)

type MeanReversionRunner struct {
	Lookback    int
	EntryZ      float64
	ExitZ       float64
	MaxHold     int
	TrendPeriod int
	VolPeriod   int
	VolMaxMult  float64
	Mode        string // "sma" (default) or "vwap"

	closeHistory  []float64
	volumeHistory []float64
	histIndex     int
	histCount     int
	emaMean       types.Price
	trendEMA      types.Price
	histVariance  float64

	openPosition *position
	barsHeld     int

	irVersion        string
	canonicalVersion string
	instanceHash     string
}

type position struct {
	Symbol     string
	Side       string
	EntryPrice types.Price
	EntryBar   int
}

func NewMeanReversionRunner(lookback int, entryZ, exitZ float64, maxHold int) *MeanReversionRunner {
	return &MeanReversionRunner{
		Lookback:          lookback,
		EntryZ:            entryZ,
		ExitZ:             exitZ,
		MaxHold:           maxHold,
		TrendPeriod:       200,
		VolPeriod:         20,
		VolMaxMult:        2.5,
		closeHistory:      make([]float64, lookback+200),
		irVersion:         "qst-ir/0.4",
		canonicalVersion:  "qst-canonical/0.4",
	}
}

func (sr *MeanReversionRunner) Name() string { return "mean_reversion" }
func (sr *MeanReversionRunner) Type() string { return "mr" }
func (sr *MeanReversionRunner) Version() (irVersion string, canonicalVersion string) {
	return sr.irVersion, sr.canonicalVersion
}
func (sr *MeanReversionRunner) SetVersion(irVersion, canonicalVersion string) { sr.irVersion = irVersion; sr.canonicalVersion = canonicalVersion }
func (sr *MeanReversionRunner) SetInstanceHash(h string) { sr.instanceHash = h }

func (sr *MeanReversionRunner) computeVWAP(start, end int) float64 {
	if len(sr.volumeHistory) == 0 {
		return 0
	}
	var totalPV, totalV float64
	for i := start; i < end; i++ {
		idx := i % (sr.Lookback + 200)
		p := sr.closeHistory[idx]
		v := 0.0
		if idx < len(sr.volumeHistory) {
			v = sr.volumeHistory[idx]
		}
		if p > 0 && v > 0 {
			totalPV += p * v
			totalV += v
		}
	}
	if totalV <= 0 {
		return 0
	}
	return totalPV / totalV
}
func (sr *MeanReversionRunner) InstanceHash() string                          { return sr.instanceHash }

func (sr *MeanReversionRunner) Reset() {
	sr.histIndex = 0
	sr.histCount = 0
	sr.emaMean = 0
	sr.trendEMA = 0
	sr.histVariance = 0
	sr.openPosition = nil
	sr.barsHeld = 0
	for i := range sr.closeHistory {
		sr.closeHistory[i] = 0
	}
}

func (sr *MeanReversionRunner) Evaluate(candle Candle, regime int8) *Signal {
	_ = regime
	idx := sr.histIndex % (sr.Lookback + 200)
	sr.closeHistory[idx] = candle.Close.Float64()
	if idx >= len(sr.volumeHistory) {
		newVol := make([]float64, sr.Lookback+200)
		copy(newVol, sr.volumeHistory)
		sr.volumeHistory = newVol
	}
	if idx < len(sr.volumeHistory) {
		sr.volumeHistory[idx] = candle.Volume
	}
	sr.histIndex++
	if sr.histCount < sr.Lookback+200 {
		sr.histCount++
	}

	if sr.histCount < sr.Lookback {
		return nil
	}

	mean, std, atrVol := sr.computeStats()
	if std <= 0 {
		return nil
	}

	isTrendingUp := candle.Close.Compare(sr.trendEMA) > 0
	isTrendingDown := candle.Close.Compare(sr.trendEMA) < 0

	isHighVol := false
	normStd := atrVol / math.Max(candle.Close.Float64(), 0.0001)
	if normStd > sr.VolMaxMult*math.Sqrt(sr.histVariance)/math.Max(candle.Close.Float64(), 0.0001)*0.1 {
		isHighVol = true
	}

	z := (candle.Close.Float64() - mean) / std

	if sr.openPosition != nil {
		sr.barsHeld++
		meanReversionComplete := false
		if sr.openPosition.Side == "BUY" && candle.Close.Float64() >= mean {
			meanReversionComplete = true
		}
		if sr.openPosition.Side == "SELL" && candle.Close.Float64() <= mean {
			meanReversionComplete = true
		}
		if sr.barsHeld >= sr.MaxHold || meanReversionComplete || math.Abs(z) < sr.ExitZ {
			exitSide := "SELL"
			if sr.openPosition.Side == "SELL" {
				exitSide = "BUY"
			}
			sr.openPosition = nil
			sr.barsHeld = 0
			return &Signal{Symbol: candle.Symbol, Side: exitSide, Quantity: 0}
		}
		return nil
	}

	if isHighVol {
		return nil
	}

	if z < -sr.EntryZ {
		if !isTrendingUp {
			return nil
		}
		sr.openPosition = &position{
			Symbol:     candle.Symbol,
			Side:       "BUY",
			EntryPrice: candle.Close,
			EntryBar:   1,
		}
		sr.barsHeld = 1
		return &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 1.0}
	}

	if z > sr.EntryZ {
		if !isTrendingDown {
			return nil
		}
		sr.openPosition = &position{
			Symbol:     candle.Symbol,
			Side:       "SELL",
			EntryPrice: candle.Close,
			EntryBar:   1,
		}
		sr.barsHeld = 1
		return &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 1.0}
	}

	return nil
}

func (sr *MeanReversionRunner) computeStats() (float64, float64, float64) {
	n := sr.histCount
	if n > sr.Lookback+200 {
		n = sr.Lookback + 200
	}

	var lookbackPrices []float64
	start := sr.histCount - sr.Lookback
	if start < 0 {
		start = 0
	}
	for i := start; i < sr.histCount; i++ {
		idx := i % (sr.Lookback + 200)
		lookbackPrices = append(lookbackPrices, sr.closeHistory[idx])
	}
	nLB := len(lookbackPrices)

	sum := 0.0
	for _, p := range lookbackPrices {
		sum += p
	}
	simpleMean := sum / float64(nLB)

	// VWAP mode: use volume-weighted average instead of EMA/SMA.
	if sr.Mode == "vwap" && len(sr.volumeHistory) > 0 {
		vwapMean := sr.computeVWAP(start, sr.histCount)
		if vwapMean > 0 {
			simpleMean = vwapMean
		}
	}

	alpha := 2.0 / float64(sr.Lookback+1)
	if sr.emaMean.IsZero() {
		sr.emaMean = types.PriceFromFloat(simpleMean)
	}
	sr.emaMean = types.PriceFromFloat(alpha*lookbackPrices[nLB-1] + (1.0-alpha)*sr.emaMean.Float64())

	trendAlpha := 2.0 / float64(sr.TrendPeriod+1)
	if sr.trendEMA.IsZero() {
		sr.trendEMA = types.PriceFromFloat(simpleMean)
	}
	sr.trendEMA = types.PriceFromFloat(trendAlpha*lookbackPrices[nLB-1] + (1.0-trendAlpha)*sr.trendEMA.Float64())

	variance := 0.0
	for _, p := range lookbackPrices {
		diff := p - simpleMean
		variance += diff * diff
	}
	variance /= float64(nLB - 1)
	sr.histVariance = variance

	volN := sr.VolPeriod
	if volN > n {
		volN = n
	}
	atrSum := 0.0
	atrCount := 0
	for i := start + 1; i < sr.histCount && atrCount < volN; i++ {
		idxCurr := i % (sr.Lookback + 200)
		idxPrev := (i - 1) % (sr.Lookback + 200)
		diff := math.Abs(sr.closeHistory[idxCurr] - sr.closeHistory[idxPrev])
		atrSum += diff
		atrCount++
	}
	atrVol := 0.0
	if atrCount > 0 {
		atrVol = atrSum / float64(atrCount)
	}

	return sr.emaMean.Float64(), math.Sqrt(variance), atrVol
}

func (sr *MeanReversionRunner) Params() map[string]float64 {
	return map[string]float64{
		"lookback":     float64(sr.Lookback),
		"entry_z":      sr.EntryZ,
		"exit_z":       sr.ExitZ,
		"max_hold":     float64(sr.MaxHold),
		"trend_period": float64(sr.TrendPeriod),
	}
}

func (sr *MeanReversionRunner) SetParams(params map[string]float64) {
	if v, ok := params["lookback"]; ok && v > 0 {
		sr.Lookback = int(v)
	}
	if v, ok := params["entry_z"]; ok {
		sr.EntryZ = v
	}
	if v, ok := params["exit_z"]; ok {
		sr.ExitZ = v
	}
	if v, ok := params["max_hold"]; ok && v > 0 {
		sr.MaxHold = int(v)
	}
	if v, ok := params["trend_period"]; ok && v > 0 {
		sr.TrendPeriod = int(v)
	}
}

func (sr *MeanReversionRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "lookback", Type: ParamInteger, Default: float64(sr.Lookback), Min: 5, Max: 60, Step: 5, Group: "Signal", Description: "Lookback period for z-score computation"},
		{Name: "entry_z", Type: ParamContinuous, Default: sr.EntryZ, Min: 0.5, Max: 4.0, Step: 0.25, Group: "Signal", Description: "Z-score threshold for entry (lower = more signals)"},
		{Name: "exit_z", Type: ParamContinuous, Default: sr.ExitZ, Min: 0.1, Max: 2.0, Step: 0.1, Group: "Signal", Description: "Z-score threshold for exit (must be < entry_z)"},
		{Name: "max_hold", Type: ParamInteger, Default: float64(sr.MaxHold), Min: 10, Max: 200, Step: 10, Group: "Exit", Description: "Maximum bars to hold position before forced exit"},
		{Name: "trend_period", Type: ParamInteger, Default: float64(sr.TrendPeriod), Min: 50, Max: 400, Step: 50, Group: "Filter", Description: "Long-term EMA period for trend direction filter"},
	}
}

func (sr *MeanReversionRunner) OnFill(orderID string, symbol string, side string, entryPrice types.Price, fillPrice types.Price, quantity float64, filledQty float64) {}
func (sr *MeanReversionRunner) OnCancel(orderID string, reason string) {}
func (sr *MeanReversionRunner) OnOrderRejected(orderID string, reason string) {}
