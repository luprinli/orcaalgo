package backtest

import (
	"math"
	"math/rand"
	"time"

	"github.com/lee-econ/orca-core/internal/types"
)

type SlippageModel struct {
	Type       string
	SpreadBps  float64
	MaxSlippage float64
	LatencyMs  float64
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
	return s.SimulateFillWithTCA(orderID, symbol, limitPrice, quantity, side, tickPrice, tickTime, tickPrice, tickPrice)
}

func (s *FillSimulator) SimulateFillWithTCA(orderID uint32, symbol string, limitPrice float64, quantity float64, side string, tickPrice float64, tickTime time.Time, midPrice float64, lastPrice float64) *SimulatedFill {
	delay := time.Duration(s.model.LatencyMs) * time.Millisecond

	slippageBps := s.model.SpreadBps
	if s.model.MaxSlippage > 0 {
		randomFactor := (s.rng.Float64()*2 - 1) * s.model.MaxSlippage
		slippageBps += randomFactor
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
		Type:        "relative",
		SpreadBps:   0.5,
		MaxSlippage: 2.0,
		LatencyMs:   5.0,
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
