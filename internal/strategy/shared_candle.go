package strategy

import "sync/atomic"

type SharedCandle struct {
	refs atomic.Int32
	Data Candle
}

func NewSharedCandle(c Candle) *SharedCandle {
	sc := &SharedCandle{Data: c}
	sc.refs.Store(1)
	return sc
}

func (s *SharedCandle) Acquire() *Candle {
	s.refs.Add(1)
	return &s.Data
}

func (s *SharedCandle) Release() {
	s.refs.Add(-1)
}
