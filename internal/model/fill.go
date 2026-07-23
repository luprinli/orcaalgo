package model

import "math"

type FillModel interface {
	FillProbability(limitPrice, midPrice uint64, spread uint64, volume uint64, atr float64) float64
	FillPrice(limitPrice, midPrice uint64, side string, atr float64) uint64
}

type MidPriceFill struct{}

func (m MidPriceFill) FillProbability(limitPrice, midPrice uint64, spread uint64, volume uint64, atr float64) float64 {
	return 1.0
}

func (m MidPriceFill) FillPrice(limitPrice, midPrice uint64, side string, atr float64) uint64 {
	return midPrice
}

type ProbabilisticFill struct {
	BaseFillRate   float64
	SpreadSensitivity float64
	VolumeScale    float64
}

func NewProbabilisticFill(baseFillRate, spreadSensitivity, volumeScale float64) ProbabilisticFill {
	return ProbabilisticFill{
		BaseFillRate:       baseFillRate,
		SpreadSensitivity:  spreadSensitivity,
		VolumeScale:        volumeScale,
	}
}

func (p ProbabilisticFill) FillProbability(limitPrice, midPrice uint64, spread uint64, volume uint64, atr float64) float64 {
	if spread == 0 {
		return p.BaseFillRate
	}

	halfSpread := float64(spread) / 2.0
	distanceFromMid := math.Abs(float64(limitPrice) - float64(midPrice))

	volAdjust := 1.0
	if atr > 0 {
		volAdjust = float64(volume) / (atr * p.VolumeScale)
		if volAdjust > 2.0 {
			volAdjust = 2.0
		}
		if volAdjust < 0.1 {
			volAdjust = 0.1
		}
	}

	alpha := p.SpreadSensitivity * (halfSpread - distanceFromMid) / halfSpread
	prob := p.BaseFillRate * volAdjust * sigmoid(alpha)

	if prob > 1.0 {
		prob = 1.0
	}
	if prob < 0.0 {
		prob = 0.0
	}
	return prob
}

func (p ProbabilisticFill) FillPrice(limitPrice, midPrice uint64, side string, atr float64) uint64 {
	slip := 0.0
	if atr > 0 {
		slip = atr * 0.15
	}
	if side == "BUY" {
		fp := float64(limitPrice) + slip
		if fp > float64(midPrice)*1.01 {
			fp = float64(midPrice) * 1.01
		}
		if fp < float64(midPrice)*0.99 {
			fp = float64(midPrice) * 0.99
		}
		return uint64(fp)
	}
	fp := float64(limitPrice) - slip
	if fp < float64(midPrice)*0.99 {
		fp = float64(midPrice) * 0.99
	}
	if fp > float64(midPrice)*1.01 {
		fp = float64(midPrice) * 1.01
	}
	return uint64(fp)
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}
