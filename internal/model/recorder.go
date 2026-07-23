package model

import "time"

type TradingState struct {
	Timestamp     time.Time
	Balance       float64
	Position      float64
	MidPrice      uint64
	Fee           float64
	TradingVolume float64
	TradingValue  float64
	NumTrades     int64
}

type Recorder interface {
	Record(state *TradingState, orders []*Order)
	Flush() error
}

type Order struct {
	Symbol    string
	Side      string
	Price     uint64
	Quantity  float64
	FilledQty float64
	FillPrice uint64
	Fee       float64
	Maker     bool
	Status    string
	Timestamps OrderTimestamp
}

type OrderTimestamp struct {
	RequestedAt    time.Time
	SentAt         time.Time
	AcknowledgedAt time.Time
	FilledAt       time.Time
}
