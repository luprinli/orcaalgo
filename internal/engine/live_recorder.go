package engine

import (
	"log"
	"time"

	"github.com/lee-econ/orca-core/internal/model"
)

type LiveRecorder struct {
	states  []model.TradingState
	onOrder func(ts model.OrderTimestamp)
}

func NewLiveRecorder(onOrder func(model.OrderTimestamp)) *LiveRecorder {
	return &LiveRecorder{
		states:  make([]model.TradingState, 0, 10000),
		onOrder: onOrder,
	}
}

func (r *LiveRecorder) Record(state *model.TradingState, orders []*model.Order) {
	r.states = append(r.states, *state)
}

func (r *LiveRecorder) Flush() error {
	return nil
}

func (r *LiveRecorder) RecordOrderLatency(ts model.OrderTimestamp) {
	log.Printf("order latency: req=%v sent=%v ack=%v filled=%v",
		ts.RequestedAt.Format(time.RFC3339Nano),
		ts.SentAt.Format(time.RFC3339Nano),
		ts.AcknowledgedAt.Format(time.RFC3339Nano),
		ts.FilledAt.Format(time.RFC3339Nano),
	)
	if r.onOrder != nil {
		r.onOrder(ts)
	}
}

func (r *LiveRecorder) States() []model.TradingState {
	return r.states
}
