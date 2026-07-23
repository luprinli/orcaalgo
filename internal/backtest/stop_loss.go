package backtest

import "math"

type StopLossType string

const (
	StopLossNone    StopLossType = "none"
	StopLossFixed   StopLossType = "fixed"
	StopLossATR     StopLossType = "atr"
	StopLossTrail   StopLossType = "trailing"
)

type TakeProfitType string

const (
	TakeProfitNone     TakeProfitType = "none"
	TakeProfitFixed    TakeProfitType = "fixed"
	TakeProfitATR      TakeProfitType = "atr"
	TakeProfitRR       TakeProfitType = "risk_reward"
)

type StopLossConfig struct {
	Type          StopLossType
	StopPoints    float64
	StopPercent   float64
	ATRPeriod     int
	ATRMultiplier float64
	TrailActivation float64
	TrailDistance   float64
}

type TakeProfitConfig struct {
	Type           TakeProfitType
	TakePoints     float64
	TakePercent    float64
	ATRMultiplier  float64
	RRRatio        float64
}

type ActiveStop struct {
	TradeID        int
	EntryPrice     float64
	Side           string
	StopPrice      float64
	TakePrice      float64
	PeakPrice      float64
	ATRValue       float64
	StopType       StopLossType
	TakeType       TakeProfitType
}

func DefaultStopLossConfig() *StopLossConfig {
	return &StopLossConfig{
		Type:          StopLossNone,
		StopPoints:    0,
		StopPercent:   1.0,
		ATRPeriod:     14,
		ATRMultiplier: 2.0,
	}
}

func DefaultTakeProfitConfig() *TakeProfitConfig {
	return &TakeProfitConfig{
		Type:      TakeProfitNone,
		TakePoints: 0,
		TakePercent: 2.0,
		RRRatio:   2.0,
	}
}

func CalculateStopPrice(entryPrice float64, side string, cfg *StopLossConfig, atrValue float64, currentHigh float64) float64 {
	if cfg == nil || cfg.Type == StopLossNone {
		return 0
	}

	var stopOffset float64

	switch cfg.Type {
	case StopLossFixed:
		if cfg.StopPercent > 0 {
			stopOffset = entryPrice * cfg.StopPercent / 100.0
		} else if cfg.StopPoints > 0 {
			stopOffset = cfg.StopPoints
		} else {
			return 0
		}
	case StopLossATR:
		if cfg.ATRMultiplier > 0 && atrValue > 0 {
			stopOffset = cfg.ATRMultiplier * atrValue
		} else {
			return 0
		}
	case StopLossTrail:
		if cfg.TrailDistance > 0 {
			stopOffset = cfg.TrailDistance
		} else if cfg.ATRMultiplier > 0 && atrValue > 0 {
			stopOffset = cfg.ATRMultiplier * atrValue
		} else {
			return 0
		}
	default:
		return 0
	}

	if side == "BUY" {
		return entryPrice - stopOffset
	}
	return entryPrice + stopOffset
}

func CalculateTakeProfitPrice(entryPrice float64, side string, cfg *TakeProfitConfig, stopPrice float64, atrValue float64) float64 {
	if cfg == nil || cfg.Type == TakeProfitNone {
		return 0
	}

	var takeOffset float64

	switch cfg.Type {
	case TakeProfitFixed:
		if cfg.TakePercent > 0 {
			takeOffset = entryPrice * cfg.TakePercent / 100.0
		} else if cfg.TakePoints > 0 {
			takeOffset = cfg.TakePoints
		} else {
			return 0
		}
	case TakeProfitATR:
		if cfg.ATRMultiplier > 0 && atrValue > 0 {
			takeOffset = cfg.ATRMultiplier * atrValue
		} else {
			return 0
		}
	case TakeProfitRR:
		if stopPrice > 0 && entryPrice > 0 {
			risk := math.Abs(entryPrice - stopPrice)
			takeOffset = risk * cfg.RRRatio
		} else if cfg.TakePercent > 0 {
			takeOffset = entryPrice * cfg.TakePercent / 100.0
		} else {
			return 0
		}
	default:
		return 0
	}

	if side == "BUY" {
		return entryPrice + takeOffset
	}
	return entryPrice - takeOffset
}

func CheckStopHit(candle Candle, stop *ActiveStop) (bool, float64) {
	if stop == nil || stop.StopPrice <= 0 {
		return false, 0
	}

	if stop.Side == "BUY" {
		if candle.Low > 0 && candle.Low <= stop.StopPrice {
			exitPrice := stop.StopPrice
			if candle.Open < stop.StopPrice {
				exitPrice = candle.Open
			}
			return true, exitPrice
		}
		if candle.Open > 0 && candle.Open <= stop.StopPrice {
			return true, candle.Open
		}
	} else {
		if candle.High > 0 && candle.High >= stop.StopPrice {
			exitPrice := stop.StopPrice
			if candle.Open > stop.StopPrice {
				exitPrice = candle.Open
			}
			return true, exitPrice
		}
		if candle.Open > 0 && candle.Open >= stop.StopPrice {
			return true, candle.Open
		}
	}

	return false, 0
}

func CheckTakeProfitHit(candle Candle, stop *ActiveStop) (bool, float64) {
	if stop == nil || stop.TakePrice <= 0 {
		return false, 0
	}

	if stop.Side == "BUY" {
		if candle.High > 0 && candle.High >= stop.TakePrice {
			exitPrice := stop.TakePrice
			if candle.Open > stop.TakePrice {
				exitPrice = candle.Open
			}
			return true, exitPrice
		}
		if candle.Open > 0 && candle.Open >= stop.TakePrice {
			return true, candle.Open
		}
	} else {
		if candle.Low > 0 && candle.Low <= stop.TakePrice {
			exitPrice := stop.TakePrice
			if candle.Open < stop.TakePrice {
				exitPrice = candle.Open
			}
			return true, exitPrice
		}
		if candle.Open > 0 && candle.Open <= stop.TakePrice {
			return true, candle.Open
		}
	}

	return false, 0
}

func UpdateTrailingStop(stop *ActiveStop, candle Candle) {
	if stop == nil || stop.StopType != StopLossTrail || stop.StopPrice <= 0 {
		return
	}

	if stop.Side == "BUY" {
		if candle.High > stop.PeakPrice {
			stop.PeakPrice = candle.High
			delta := stop.PeakPrice - stop.EntryPrice
			if delta > 0 {
				stop.StopPrice = stop.PeakPrice - (stop.StopPrice - stop.EntryPrice)
				if stop.StopPrice < stop.EntryPrice {
					stop.StopPrice = stop.EntryPrice
				}
			}
		}
	} else {
		if candle.Low > 0 && candle.Low < stop.PeakPrice {
			stop.PeakPrice = candle.Low
			delta := stop.EntryPrice - stop.PeakPrice
			if delta > 0 {
				stop.StopPrice = stop.PeakPrice + (stop.EntryPrice - stop.StopPrice)
				if stop.StopPrice > stop.EntryPrice {
					stop.StopPrice = stop.EntryPrice
				}
			}
		}
	}
}

func ComputeATR(candles []Candle, period int) float64 {
	if len(candles) < period+1 {
		return 0
	}

	trSum := 0.0
	end := len(candles)
	start := end - period
	if start < 0 {
		start = 0
	}
	count := 0

	for i := start; i < end; i++ {
		c := candles[i]
		high := c.High
		low := c.Low
		prevClose := 0.0
		if i > 0 {
			prevClose = candles[i-1].Close
		} else {
			prevClose = c.Close
		}

		tr := high - low
		if prevClose > 0 {
			hc := math.Abs(high - prevClose)
			lc := math.Abs(low - prevClose)
			if hc > tr {
				tr = hc
			}
			if lc > tr {
				tr = lc
			}
		}
		trSum += tr
		count++
	}

	if count == 0 {
		return 0
	}
	return trSum / float64(count)
}
