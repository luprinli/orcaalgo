package strategy

import "time"

type Candle struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
	Symbol string
}

type Signal struct {
	Symbol   string
	Side     string
	Quantity float64
	PWin     float64 // ML meta-labeler win probability (0.0–1.0, 0 if unchecked)
}
