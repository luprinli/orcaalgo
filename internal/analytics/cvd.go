package analytics

import (
	"sync"
	"time"
)

type TickSide uint8

const (
	TickBuy  TickSide = 1
	TickSell TickSide = 2
)

type TickClassifier struct {
	mu          sync.Mutex
	prevPrice   float64
	prevSymbol  string
	prevSide    TickSide
}

func NewTickClassifier() *TickClassifier {
	return &TickClassifier{}
}

func (c *TickClassifier) Classify(price float64, symbol string) TickSide {
	c.mu.Lock()
	defer c.mu.Unlock()

	var side TickSide
	switch {
	case price > c.prevPrice:
		side = TickBuy
	case price < c.prevPrice:
		side = TickSell
	default:
		side = c.prevSide
	}

	if symbol == c.prevSymbol {
		c.prevPrice = price
	}
	c.prevSymbol = symbol
	c.prevSide = side
	return side
}

type DeltaBar struct {
	Time       time.Time
	Open       float64
	High       float64
	Low        float64
	Close      float64
	BuyVolume  float64
	SellVolume float64
	Delta      float64
	NumTrades  int
}

type DeltaAccumulator struct {
	mu       sync.Mutex
	barDuration time.Duration
	currentBar  *DeltaBar
	completed   []DeltaBar
}

func NewDeltaAccumulator(barDuration time.Duration) *DeltaAccumulator {
	now := time.Now().Truncate(barDuration)
	return &DeltaAccumulator{
		barDuration: barDuration,
		currentBar: &DeltaBar{Time: now},
	}
}

func (a *DeltaAccumulator) AddTick(timestamp time.Time, price float64, volume float64, side TickSide) {
	a.mu.Lock()
	defer a.mu.Unlock()

	barTime := timestamp.Truncate(a.barDuration)
	if !barTime.Equal(a.currentBar.Time) {
		a.completed = append(a.completed, *a.currentBar)
		if len(a.completed) > 10000 {
			a.completed = a.completed[len(a.completed)-5000:]
		}
		a.currentBar = &DeltaBar{Time: barTime}
	}

	if a.currentBar.Open == 0 {
		a.currentBar.Open = price
	}
	a.currentBar.Close = price
	if price > a.currentBar.High || a.currentBar.High == 0 {
		a.currentBar.High = price
	}
	if price < a.currentBar.Low || a.currentBar.Low == 0 {
		a.currentBar.Low = price
	}

	if side == TickBuy {
		a.currentBar.BuyVolume += volume
	} else {
		a.currentBar.SellVolume += volume
	}
	a.currentBar.Delta = a.currentBar.BuyVolume - a.currentBar.SellVolume
	a.currentBar.NumTrades++
}

func (a *DeltaAccumulator) GetCompletedBars() []DeltaBar {
	a.mu.Lock()
	defer a.mu.Unlock()
	result := make([]DeltaBar, len(a.completed))
	copy(result, a.completed)
	return result
}

func (a *DeltaAccumulator) GetCurrentBar() DeltaBar {
	a.mu.Lock()
	defer a.mu.Unlock()
	return *a.currentBar
}

type CVDTracker struct {
	mu     sync.Mutex
	bars   []CVDPoint
}

type CVDPoint struct {
	Time  time.Time
	Value float64
	Delta float64
}

func NewCVDTracker() *CVDTracker {
	return &CVDTracker{}
}

func (t *CVDTracker) AddBar(bar DeltaBar) {
	t.mu.Lock()
	defer t.mu.Unlock()

	point := CVDPoint{
		Time:  bar.Time,
		Delta: bar.Delta,
		Value: bar.Delta,
	}
	if len(t.bars) > 0 {
		point.Value += t.bars[len(t.bars)-1].Value
	}
	t.bars = append(t.bars, point)
	if len(t.bars) > 50000 {
		t.bars = t.bars[len(t.bars)-25000:]
	}
}

func (t *CVDTracker) GetHistory(limit int) []CVDPoint {
	t.mu.Lock()
	defer t.mu.Unlock()
	start := len(t.bars) - limit
	if start < 0 {
		start = 0
	}
	result := make([]CVDPoint, len(t.bars)-start)
	copy(result, t.bars[start:])
	return result
}

type DivergenceType uint8

const (
	DivNone        DivergenceType = 0
	DivExhaustion  DivergenceType = 1
	DivAbsorption  DivergenceType = 2
)

type DivergenceSignal struct {
	Type       DivergenceType
	BarTime    time.Time
	Price      float64
	CVDValue   float64
	Confidence float64
}

type DivergenceDetector struct {
	mu          sync.Mutex
	prices      []float64
	cvdValues   []float64
	windowSize  int
}

func NewDivergenceDetector(windowSize int) *DivergenceDetector {
	return &DivergenceDetector{windowSize: windowSize}
}

func (d *DivergenceDetector) Detect(price, cvdValue float64) *DivergenceSignal {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.prices = append(d.prices, price)
	d.cvdValues = append(d.cvdValues, cvdValue)
	if len(d.prices) > d.windowSize {
		d.prices = d.prices[len(d.prices)-d.windowSize:]
		d.cvdValues = d.cvdValues[len(d.cvdValues)-d.windowSize:]
	}

	if len(d.prices) < d.windowSize {
		return nil
	}

	mid := d.windowSize / 2
	prevPrice := d.prices[0]
	currPrice := d.prices[len(d.prices)-1]
	prevCVD := d.cvdValues[0]
	currCVD := d.cvdValues[len(d.cvdValues)-1]

	_ = mid

	if currPrice > prevPrice && currCVD < prevCVD {
		return &DivergenceSignal{
			Type:       DivExhaustion,
			Confidence: (currPrice - prevPrice) / prevPrice * 100,
		}
	}

	if currPrice < prevPrice && currCVD > prevCVD {
		return &DivergenceSignal{
			Type:       DivAbsorption,
			Confidence: (prevPrice - currPrice) / currPrice * 100,
		}
	}

	return nil
}
