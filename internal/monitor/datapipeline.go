package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/lee-econ/orca-core/internal/analytics"
	"github.com/lee-econ/orca-core/internal/ingest"
	"github.com/lee-econ/orca-core/internal/types"
)

type DataPipeline struct {
	hub         *WSHub
	ringBuf     *ingest.RingBuffer
	classifier  *analytics.TickClassifier
	accumulator *analytics.DeltaAccumulator
	cvdTracker  *analytics.CVDTracker
	divDetector *analytics.DivergenceDetector
	mu          sync.Mutex
	tickCount   int64
}

func NewDataPipeline(hub *WSHub, ringBuf *ingest.RingBuffer) *DataPipeline {
	return &DataPipeline{
		hub:         hub,
		ringBuf:     ringBuf,
		classifier:  analytics.NewTickClassifier(),
		accumulator: analytics.NewDeltaAccumulator(5 * time.Second),
		cvdTracker:  analytics.NewCVDTracker(),
		divDetector: analytics.NewDivergenceDetector(20),
	}
}

func (p *DataPipeline) Run(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.drainAndBroadcast()
		}
	}
}

func (p *DataPipeline) drainAndBroadcast() {
	p.mu.Lock()
	defer p.mu.Unlock()

	drained := 0
	for drained < 1000 {
		tick, ok := p.ringBuf.Pop()
		if !ok {
			break
		}
		drained++
		p.tickCount++

		price := types.PriceFromInt64(int64(tick.PriceRaw))
		volume := float64(tick.VolumeRaw)

		side := p.classifier.Classify(price, "")
		p.accumulator.AddTick(time.Now(), price, volume, analytics.TickSide(side))

		div := p.divDetector.Detect(price, 0)
		if div != nil && (div.Type == analytics.DivExhaustion || div.Type == analytics.DivAbsorption) {
			p.hub.Broadcast("divergence", map[string]interface{}{
				"type":       div.Type,
				"confidence": div.Confidence,
				"time":       time.Now().Format(time.RFC3339),
			})
		}
	}

	completed := p.accumulator.GetCompletedBars()
	for _, bar := range completed {
		p.cvdTracker.AddBar(bar)

		p.hub.Broadcast("cvd", map[string]interface{}{
			"bar": map[string]interface{}{
				"time":       bar.Time.Format(time.RFC3339),
				"open":       bar.Open,
				"high":       bar.High,
				"low":        bar.Low,
				"close":      bar.Close,
				"buy_volume": bar.BuyVolume,
				"sell_volume": bar.SellVolume,
				"delta":      bar.Delta,
				"num_trades": bar.NumTrades,
			},
		})
	}
}

func (p *DataPipeline) GetTickCount() int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.tickCount
}

func (p *DataPipeline) SetHub(hub *WSHub) {
	p.hub = hub
}
