package strategy

import (
	"math"
)

type GridRunner struct {
	GridLevels     float64
	GridSpacingPct float64
	PositionScale  float64
	MaxOpen        float64
	TakeProfitPct  float64
	StopLossPct    float64
	openCount      int
	openPositions  map[int]*gridPosition
	lastPrice      float64
	referencePrice float64
	priceInit      bool

	irVersion        string
	canonicalVersion string
	instanceHash     string
}

type gridPosition struct {
	Side       string
	Quantity   float64
	EntryPrice float64
	GridLevel  int
	TakePrice  float64
	StopPrice  float64
}

func NewGridRunner() *GridRunner {
	return &GridRunner{
		GridLevels:        5,
		GridSpacingPct:    1.0,
		PositionScale:     1.0,
		MaxOpen:           10,
		TakeProfitPct:     0.5,
		StopLossPct:       1.5,
		openPositions:     make(map[int]*gridPosition),
		irVersion:         "qst-ir/0.4",
		canonicalVersion:  "qst-canonical/0.4",
	}
}

func (r *GridRunner) Name() string {
	return "grid_trading"
}

func (r *GridRunner) Type() string {
	return "grid"
}

func (r *GridRunner) Version() (irVersion string, canonicalVersion string) {
	return r.irVersion, r.canonicalVersion
}
func (r *GridRunner) SetVersion(irVersion, canonicalVersion string) { r.irVersion = irVersion; r.canonicalVersion = canonicalVersion }
func (r *GridRunner) SetInstanceHash(h string)                      { r.instanceHash = h }
func (r *GridRunner) InstanceHash() string                          { return r.instanceHash }

func (r *GridRunner) Reset() {
	r.openCount = 0
	r.lastPrice = 0
	r.referencePrice = 0
	r.priceInit = false
	r.openPositions = make(map[int]*gridPosition)
}

func (r *GridRunner) Params() map[string]float64 {
	return map[string]float64{
		"grid_levels":      r.GridLevels,
		"grid_spacing_pct": r.GridSpacingPct,
		"position_scale":   r.PositionScale,
		"max_open":         r.MaxOpen,
		"take_profit_pct":  r.TakeProfitPct,
		"stop_loss_pct":    r.StopLossPct,
	}
}

func (r *GridRunner) SetParams(params map[string]float64) {
	if v, ok := params["grid_levels"]; ok {
		r.GridLevels = v
	}
	if v, ok := params["grid_spacing_pct"]; ok {
		r.GridSpacingPct = v
	}
	if v, ok := params["position_scale"]; ok {
		r.PositionScale = v
	}
	if v, ok := params["max_open"]; ok {
		r.MaxOpen = v
	}
	if v, ok := params["take_profit_pct"]; ok {
		r.TakeProfitPct = v
	}
	if v, ok := params["stop_loss_pct"]; ok {
		r.StopLossPct = v
	}
}

func (r *GridRunner) ParamDefs() []ParamDef {
	return []ParamDef{
		{Name: "grid_levels", Type: ParamInteger, Default: 5, Min: 2, Max: 8, Step: 1, Group: "Grid", Description: "Number of grid levels above and below reference price"},
		{Name: "grid_spacing_pct", Type: ParamContinuous, Default: 1.0, Min: 0.2, Max: 4.0, Step: 0.2, Group: "Grid", Description: "Percentage spacing between adjacent grid levels"},
		{Name: "take_profit_pct", Type: ParamContinuous, Default: 0.5, Min: 0.1, Max: 2.0, Step: 0.1, Group: "Exit", Description: "Take-profit percentage from entry level"},
		{Name: "stop_loss_pct", Type: ParamContinuous, Default: 1.5, Min: 0.5, Max: 5.0, Step: 0.1, Group: "Exit", Description: "Stop-loss percentage from entry level"},
		{Name: "max_open", Type: ParamInteger, Default: 10, Min: 1, Max: 10, Step: 1, Group: "Risk", Description: "Maximum number of simultaneously open grid positions"},
		{Name: "position_scale", Type: ParamContinuous, Default: 1.0, Min: 0.25, Max: 2.0, Step: 0.25, Group: "Sizing", Description: "Position size scale factor"},
	}
}

func (r *GridRunner) Evaluate(candle Candle, regime int8) *Signal {
	if regime == 3 {
		return nil
	}
	price := candle.Close
	if price <= 0 {
		return nil
	}

	if !r.priceInit {
		r.referencePrice = price
		r.lastPrice = price
		r.priceInit = true
		return nil
	}

	spacing := r.GridSpacingPct / 100.0
	if spacing <= 0 {
		spacing = 0.01
	}
	tpPct := r.TakeProfitPct / 100.0
	slPct := r.StopLossPct / 100.0

	levels := int(r.GridLevels)
	if levels > 20 {
		levels = 20
	}
	if levels < 2 {
		levels = 2
	}

	prices := make([]float64, levels*2+1)
	prices[levels] = r.referencePrice
	for i := 1; i <= levels; i++ {
		prices[levels+i] = r.referencePrice * (1.0 + spacing*float64(i))
		prices[levels-i] = r.referencePrice * (1.0 - spacing*float64(i))
	}

	closedAny := false
	exitSide := ""
	for level, pos := range r.openPositions {
		shouldClose := false
		if pos.Side == "BUY" && price >= pos.TakePrice {
			shouldClose = true
			exitSide = "SELL"
		} else if pos.Side == "SELL" && price <= pos.TakePrice {
			shouldClose = true
			exitSide = "BUY"
		} else if pos.Side == "BUY" && price <= pos.StopPrice {
			shouldClose = true
			exitSide = "SELL"
		} else if pos.Side == "SELL" && price >= pos.StopPrice {
			shouldClose = true
			exitSide = "BUY"
		}
		if shouldClose {
			delete(r.openPositions, level)
			r.openCount--
			closedAny = true
		}
	}

	if closedAny {
		return &Signal{Symbol: candle.Symbol, Side: exitSide, Quantity: 0}
	}

	if r.openCount >= int(r.MaxOpen) {
		r.lastPrice = price
		return nil
	}

	priceCrossedDown := r.lastPrice > price
	priceCrossedUp := r.lastPrice < price

	nearestGridAbove := -1
	nearestGridBelow := -1
	minDistAbove := math.MaxFloat64
	minDistBelow := math.MaxFloat64

	for i, gp := range prices {
		if gp > price && gp-price < minDistAbove {
			minDistAbove = gp - price
			nearestGridAbove = i - levels
		}
		if gp < price && price-gp < minDistBelow {
			minDistBelow = price - gp
			nearestGridBelow = i - levels
		}
	}

	var signal *Signal

	if priceCrossedDown && nearestGridBelow != -1 {
		if _, exists := r.openPositions[nearestGridBelow]; !exists {
			entryPrice := prices[nearestGridBelow+levels]
			takePrice := entryPrice * (1.0 + tpPct)
			stopPrice := entryPrice * (1.0 - slPct)
			r.openPositions[nearestGridBelow] = &gridPosition{
				Side:       "BUY",
				Quantity:   100 * r.PositionScale,
				EntryPrice: entryPrice,
				GridLevel:  nearestGridBelow,
				TakePrice:  takePrice,
				StopPrice:  stopPrice,
			}
			r.openCount++
			signal = &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 100 * r.PositionScale}
		}
	}

	if priceCrossedUp && nearestGridAbove != -1 && signal == nil {
		if _, exists := r.openPositions[nearestGridAbove]; !exists {
			entryPrice := prices[nearestGridAbove+levels]
			takePrice := entryPrice * (1.0 - tpPct)
			stopPrice := entryPrice * (1.0 + slPct)
			r.openPositions[nearestGridAbove] = &gridPosition{
				Side:       "SELL",
				Quantity:   100 * r.PositionScale,
				EntryPrice: entryPrice,
				GridLevel:  nearestGridAbove,
				TakePrice:  takePrice,
				StopPrice:  stopPrice,
			}
			r.openCount++
			signal = &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 100 * r.PositionScale}
		}
	}

	r.lastPrice = price
	return signal
}

func (r *GridRunner) OnFill(orderID string, symbol string, side string, entryPrice float64, fillPrice float64, quantity float64, filledQty float64) {}
func (r *GridRunner) OnCancel(orderID string, reason string) {}
func (r *GridRunner) OnOrderRejected(orderID string, reason string) {}
