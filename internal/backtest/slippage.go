package backtest

import (
	"math"
	"math/rand"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

type SlippageModel struct {
	Type               string
	SpreadBps          float64
	MaxSlippage        float64
	LatencyMs          float64
	VolumeImpactFactor float64
	AdverseSelectBps   float64
}

// LimitFillProbability estimates the probability that a limit order at a given
// distance from mid is filled. Available for future limit-order fill simulation
// (not yet wired into SimulateFillWithTCA).
func (m SlippageModel) LimitFillProbability(distanceFromMidBps float64) float64 {
	if distanceFromMidBps <= 0 {
		return 0.98
	}
	decay := math.Exp(-distanceFromMidBps / (m.SpreadBps + 0.01))
	return math.Max(0.05, decay)
}

func CalibrateSlippageModel(model SlippageModel, observedSlippageBps float64, sampleCount int) SlippageModel {
	if sampleCount < 10 || observedSlippageBps < 0 {
		return model
	}
	alpha := 0.2
	lowerBound := 0.7
	upperBound := 1.5
	adjustment := observedSlippageBps / (model.SpreadBps + model.AdverseSelectBps + model.MaxSlippage*0.4 + 0.001)
	adjustment = math.Max(lowerBound, math.Min(upperBound, adjustment))
	model.SpreadBps *= 1.0 + alpha*(adjustment-1.0)
	model.MaxSlippage *= 1.0 + alpha*(adjustment-1.0)
	model.AdverseSelectBps *= 1.0 + alpha*(adjustment-1.0)
	if model.SpreadBps < 0 {
		model.SpreadBps = 0
	}
	if model.MaxSlippage < 0 {
		model.MaxSlippage = 0
	}
	return model
}

type FillSimulator struct {
	model    SlippageModel
	rng      *rand.Rand
}

func NewFillSimulator(model SlippageModel) *FillSimulator {
	return &FillSimulator{
		model: model,
		rng:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func NewFillSimulatorWithSeed(model SlippageModel, seed int64) *FillSimulator {
	return &FillSimulator{
		model: model,
		rng:   rand.New(rand.NewSource(seed)),
	}
}

type SimulatedFill struct {
	OrderID       uint32
	Symbol        string
	FillPrice     types.Price
	FillQuantity  float64
	SlippageBps   float64
	LatencyMs     float64
	Delay         time.Duration
	SlippageMidBps  float64
	SlippageLastBps float64
}

func (s *FillSimulator) SimulateFill(orderID uint32, symbol string, limitPrice float64, quantity float64, side string, tickPrice float64, tickTime time.Time) *SimulatedFill {
	return s.SimulateFillWithTCA(orderID, symbol, limitPrice, quantity, side, tickPrice, tickTime, tickPrice, tickPrice, 0)
}

func (s *FillSimulator) SimulateFillWithTCA(orderID uint32, symbol string, limitPrice float64, quantity float64, side string, tickPrice float64, tickTime time.Time, midPrice float64, lastPrice float64, barVolume float64) *SimulatedFill {
	delay := time.Duration(s.model.LatencyMs) * time.Millisecond

	sm := s.model
	if s.model.Type == "relative" && s.model.SpreadBps == 0.5 && s.model.MaxSlippage == 2.0 &&
		s.model.VolumeImpactFactor == 0.5 && s.model.AdverseSelectBps == 0.5 {
		base := SlippageForSymbol(symbol)
		sm.SpreadBps = base.SpreadBps
		sm.MaxSlippage = base.MaxSlippage
		sm.VolumeImpactFactor = base.VolumeImpactFactor
		sm.AdverseSelectBps = base.AdverseSelectBps
	}

	slippageBps := sm.SpreadBps + sm.AdverseSelectBps
	if sm.MaxSlippage > 0 {
		randomFactor := math.Abs(s.rng.NormFloat64()) * sm.MaxSlippage * 0.5
		if s.rng.Float64() < 0.05 {
			randomFactor = -randomFactor * 0.5
		}
		slippageBps += randomFactor
	}
	if barVolume > 0 && sm.VolumeImpactFactor > 0 && quantity > 0 {
		slippageBps += sm.VolumeImpactFactor * math.Sqrt(quantity / barVolume)
	}
	if slippageBps < 0 {
		slippageBps = 0
	}

	var fillPrice float64
	if side == "BUY" {
		fillPrice = tickPrice * (1 + slippageBps/10000.0)
	} else {
		fillPrice = tickPrice * (1 - slippageBps/10000.0)
	}

	fillQuantity := quantity
	if barVolume > 0 {
		maxQtyByVolume := barVolume * 0.01
		if fillQuantity > maxQtyByVolume {
			fillQuantity = maxQtyByVolume
		}
	}
	if fillQuantity > 0 && s.rng.Float64() < 0.05 {
		fillQuantity *= 0.5 + s.rng.Float64()*0.5
	}

	var slippageMidBps, slippageLastBps float64
	if midPrice > 0 {
		if side == "BUY" {
			slippageMidBps = (fillPrice - midPrice) / midPrice * 10000.0
		} else {
			slippageMidBps = (midPrice - fillPrice) / midPrice * 10000.0
		}
	}
	if lastPrice > 0 {
		if side == "BUY" {
			slippageLastBps = (fillPrice - lastPrice) / lastPrice * 10000.0
		} else {
			slippageLastBps = (lastPrice - fillPrice) / lastPrice * 10000.0
		}
	}

	return &SimulatedFill{
		OrderID:         orderID,
		Symbol:          symbol,
		FillPrice:       types.FromFloat64(math.Round(fillPrice*10000) / 10000),
		FillQuantity:    math.Round(fillQuantity*10000) / 10000,
		SlippageBps:     slippageBps,
		LatencyMs:       s.model.LatencyMs,
		Delay:           delay,
		SlippageMidBps:  slippageMidBps,
		SlippageLastBps: slippageLastBps,
	}
}

func DefaultEquitySlippage() SlippageModel {
	return SlippageModel{
		Type:               "relative",
		SpreadBps:          0.5,
		MaxSlippage:        2.0,
		LatencyMs:          5.0,
		VolumeImpactFactor: 0.5,
		AdverseSelectBps:   0.5,
	}
}

func LowLatencySlippage() SlippageModel {
	return SlippageModel{
		Type:        "relative",
		SpreadBps:   0.1,
		MaxSlippage: 0.5,
		LatencyMs:   0.1,
	}
}

func RealisticEquitySlippage() SlippageModel {
	return SlippageModel{
		Type:               "relative",
		SpreadBps:          2.0,
		MaxSlippage:        5.0,
		LatencyMs:          10.0,
		VolumeImpactFactor: 1.0,
		AdverseSelectBps:   1.0,
	}
}

func SmallCapSlippage() SlippageModel {
	return SlippageModel{
		Type:               "relative",
		SpreadBps:          8.0,
		MaxSlippage:        15.0,
		LatencyMs:          15.0,
		VolumeImpactFactor: 2.0,
		AdverseSelectBps:   2.0,
	}
}

func ForexSlippage() SlippageModel {
	return SlippageModel{
		Type:               "relative",
		SpreadBps:          0.3,
		MaxSlippage:        1.0,
		LatencyMs:          5.0,
		VolumeImpactFactor: 0.1,
		AdverseSelectBps:   0.2,
	}
}

func CryptoSlippage() SlippageModel {
	return SlippageModel{
		Type:               "relative",
		SpreadBps:          12.0,
		MaxSlippage:        25.0,
		LatencyMs:          20.0,
		VolumeImpactFactor: 3.0,
		AdverseSelectBps:   3.0,
	}
}

func CommoditySlippage() SlippageModel {
	return SlippageModel{
		Type:               "relative",
		SpreadBps:          4.0,
		MaxSlippage:        8.0,
		LatencyMs:          10.0,
		VolumeImpactFactor: 1.5,
		AdverseSelectBps:   1.0,
	}
}

func SlippageForSymbol(symbol string) SlippageModel {
	switch {
	case symbol == "BTCUSD" || symbol == "ETHUSD":
		return CryptoSlippage()
	case symbol == "XAUUSD" || symbol == "XAGUSD" || symbol == "CL" || symbol == "USO":
		return CommoditySlippage()
	case symbol == "EURUSD" || symbol == "GBPUSD" || symbol == "USDJPY" || symbol == "USDCHF" ||
		symbol == "AUDUSD" || symbol == "USDCAD" || symbol == "NZDUSD":
		return ForexSlippage()
	case symbol == "IWM" || symbol == "TSLA" || symbol == "NVDA":
		return SmallCapSlippage()
	case symbol == "GLD" || symbol == "TLT":
		return CommoditySlippage()
	default:
		return RealisticEquitySlippage()
	}
}
