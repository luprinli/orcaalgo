package strategy

import (
	"math"

	"github.com/lee-econ/orca-core/internal/types"
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
	lastPrice      types.Price
	referencePrice types.Price
	priceInit      bool
	barCount       int
	Disabled       bool

	// Vol-adjusted grid: when AdjustByVolatility is true, grid spacing is
	// dynamically scaled based on current ATR / VIX. Wider spacing in high
	// volatility, tighter in low volatility.
	AdjustByVolatility bool
	CurrentVIX         float64
	CurrentATR         float64
	VolMaxSpacingMult  float64

	irVersion        string
	canonicalVersion string
	instanceHash     string
}

type gridPosition struct {
	Side       string
	Quantity   float64
	EntryPrice types.Price
	GridLevel  int
	TakePrice  types.Price
	StopPrice  types.Price
}

func NewGridRunner() *GridRunner {
	return &GridRunner{
		GridLevels:        5,
		GridSpacingPct:    1.0,
		PositionScale:     1.0,
		MaxOpen:           10,
		TakeProfitPct:     0.5,
		StopLossPct:       1.5,
		openPositions:      make(map[int]*gridPosition),
		Disabled:           false,
		AdjustByVolatility: false,
		VolMaxSpacingMult:  2.0,
		irVersion:          "qst-ir/0.4",
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

func (r *GridRunner) SetVIX(vix float64) { r.CurrentVIX = vix }
func (r *GridRunner) SetATR(atr float64) { r.CurrentATR = atr }

// computeVolatilityMultiplier returns a spacing multiplier based on current
// VIX and ATR relative to baseline levels. Higher vol -> wider grid spacing.
func (r *GridRunner) computeVolatilityMultiplier() float64 {
	// ATR-based: if current ATR is available, use it as the primary signal.
	if r.CurrentATR > 0 {
		baselineATR := 5.0
		mult := 1.0 + (r.CurrentATR-baselineATR)/baselineATR*0.5
		if mult < 0.7 {
			mult = 0.7
		}
		if mult > r.VolMaxSpacingMult {
			mult = r.VolMaxSpacingMult
		}
		return mult
	}

	// VIX-based: fall back to VIX if ATR is not available.
	if r.CurrentVIX > 0 {
		mult := 1.0 + (r.CurrentVIX-15.0)/15.0*0.5
		if mult < 0.7 {
			mult = 0.7
		}
		if mult > r.VolMaxSpacingMult {
			mult = r.VolMaxSpacingMult
		}
		return mult
	}

	return 1.0
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
		{Name: "adjust_by_volatility", Type: ParamInteger, Default: 0, Min: 0, Max: 1, Step: 1, Group: "Grid", Description: "Dynamically scale grid spacing based on ATR/VIX (0=off, 1=on)"},
	}
}

func (r *GridRunner) Evaluate(candle Candle, regime int8) *Signal {
	if r.Disabled {
		return nil
	}
	if regime == 3 {
		return nil
	}
	price := candle.Close
	if price.IsZero() {
		return nil
	}

	effectiveMaxOpen := r.MaxOpen
	effectiveSpacing := r.GridSpacingPct
	switch regime {
	case 0:
		effectiveSpacing *= 0.8
	case 2:
		effectiveMaxOpen = math.Max(1, r.MaxOpen*0.5)
		effectiveSpacing *= 1.5
	}

	// Vol-adjusted grid: dynamically scale spacing based on VIX or ATR.
	if r.AdjustByVolatility {
		volMult := r.computeVolatilityMultiplier()
		effectiveSpacing *= volMult
	}

	if !r.priceInit {
		r.referencePrice = price
		r.lastPrice = price
		r.priceInit = true
		return nil
	}

	r.barCount++
	if r.barCount > 0 && r.barCount%100 == 0 && r.openCount == 0 {
		r.referencePrice = price
	}

	spacing := effectiveSpacing / 100.0
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

	prices := make([]types.Price, levels*2+1)
	prices[levels] = r.referencePrice
	for i := 1; i <= levels; i++ {
		prices[levels+i] = r.referencePrice.MulFloat(1.0 + spacing*float64(i))
		prices[levels-i] = r.referencePrice.MulFloat(1.0 - spacing*float64(i))
	}

	closedAny := false
	exitSide := ""
	for level, pos := range r.openPositions {
		shouldClose := false
		if pos.Side == "BUY" && price.Compare(pos.TakePrice) >= 0 {
			shouldClose = true
			exitSide = "SELL"
		} else if pos.Side == "SELL" && price.Compare(pos.TakePrice) <= 0 {
			shouldClose = true
			exitSide = "BUY"
		} else if pos.Side == "BUY" && price.Compare(pos.StopPrice) <= 0 {
			shouldClose = true
			exitSide = "SELL"
		} else if pos.Side == "SELL" && price.Compare(pos.StopPrice) >= 0 {
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

	if r.openCount >= int(effectiveMaxOpen) {
		r.lastPrice = price
		return nil
	}

	priceCrossedDown := r.lastPrice.Compare(price) > 0
	priceCrossedUp := r.lastPrice.Compare(price) < 0

	nearestGridAbove := -1
	nearestGridBelow := -1
	minDistAbove := math.MaxFloat64
	minDistBelow := math.MaxFloat64

	for i, gp := range prices {
		gpF := gp.Float64()
		priceF := price.Float64()
		if gpF > priceF && gpF-priceF < minDistAbove {
			minDistAbove = gpF - priceF
			nearestGridAbove = i - levels
		}
		if gpF < priceF && priceF-gpF < minDistBelow {
			minDistBelow = priceF - gpF
			nearestGridBelow = i - levels
		}
	}

	var signal *Signal

	if priceCrossedDown && nearestGridBelow != -1 {
		if _, exists := r.openPositions[nearestGridBelow]; !exists {
			entryPrice := prices[nearestGridBelow+levels]
			takePrice := entryPrice.MulFloat(1.0 + tpPct)
			stopPrice := entryPrice.MulFloat(1.0 - slPct)
			r.openPositions[nearestGridBelow] = &gridPosition{
				Side:       "BUY",
				Quantity:   100 * r.PositionScale,
				EntryPrice: entryPrice,
				GridLevel:  nearestGridBelow,
				TakePrice:  takePrice,
				StopPrice:  stopPrice,
			}
			r.openCount++
			signal = &Signal{Symbol: candle.Symbol, Side: "BUY", Quantity: 1.0}
		}
	}

	if priceCrossedUp && nearestGridAbove != -1 && signal == nil {
		if _, exists := r.openPositions[nearestGridAbove]; !exists {
			entryPrice := prices[nearestGridAbove+levels]
			takePrice := entryPrice.MulFloat(1.0 - tpPct)
			stopPrice := entryPrice.MulFloat(1.0 + slPct)
			r.openPositions[nearestGridAbove] = &gridPosition{
				Side:       "SELL",
				Quantity:   100 * r.PositionScale,
				EntryPrice: entryPrice,
				GridLevel:  nearestGridAbove,
				TakePrice:  takePrice,
				StopPrice:  stopPrice,
			}
			r.openCount++
			signal = &Signal{Symbol: candle.Symbol, Side: "SELL", Quantity: 1.0}
		}
	}

	r.lastPrice = price
	return signal
}

func (r *GridRunner) OnFill(orderID string, symbol string, side string, entryPrice types.Price, fillPrice types.Price, quantity float64, filledQty float64) {}
func (r *GridRunner) OnCancel(orderID string, reason string) {}
func (r *GridRunner) OnOrderRejected(orderID string, reason string) {}
