package indicator

import (
	"log"
	"sync"
	"time"

	"github.com/lee-econ/orca-core/internal/monitor"
)

type LiveStreamRequest struct {
	Symbol    string                 `json:"symbol"`
	Timeframe string                 `json:"timeframe"`
	Indicator string                 `json:"indicator"`
	Params    map[string]interface{} `json:"parameters"`
}

type LiveStreamService struct {
	hub             *monitor.WSHub
	mu              sync.RWMutex
	activeStreams   map[string]*liveStream
	candleProvider  func(symbol, timeframe string, limit int) ([]Candle, error)
}

type liveStream struct {
	req       LiveStreamRequest
	stopChan  chan struct{}
	candles   []Candle
}

func NewLiveStreamService(hub *monitor.WSHub, candleProvider func(symbol, timeframe string, limit int) ([]Candle, error)) *LiveStreamService {
	return &LiveStreamService{
		hub:            hub,
		activeStreams:  make(map[string]*liveStream),
		candleProvider: candleProvider,
	}
}

func streamKey(req LiveStreamRequest) string {
	return req.Symbol + ":" + req.Timeframe + ":" + req.Indicator
}

func (s *LiveStreamService) Start(req LiveStreamRequest) {
	key := streamKey(req)

	s.mu.Lock()
	if _, exists := s.activeStreams[key]; exists {
		s.mu.Unlock()
		return
	}

	ls := &liveStream{
		req:      req,
		stopChan: make(chan struct{}),
	}
	s.activeStreams[key] = ls
	s.mu.Unlock()

	go s.runStream(ls)
}

func (s *LiveStreamService) Stop(req LiveStreamRequest) {
	key := streamKey(req)

	s.mu.Lock()
	ls, exists := s.activeStreams[key]
	if exists {
		delete(s.activeStreams, key)
		close(ls.stopChan)
	}
	s.mu.Unlock()
}

func (s *LiveStreamService) runStream(ls *liveStream) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ls.stopChan:
			return
		case <-ticker.C:
			s.computeAndPublish(ls)
		}
	}
}

func (s *LiveStreamService) computeAndPublish(ls *liveStream) {
	if s.candleProvider == nil {
		return
	}

	candles, err := s.candleProvider(ls.req.Symbol, ls.req.Timeframe, 500)
	if err != nil {
		log.Printf("indicator stream: failed to fetch candles for %s: %v", ls.req.Symbol, err)
		return
	}

	if len(candles) == 0 {
		return
	}

	result, err := Compute(ls.req.Indicator, candles, ls.req.Params)
	if err != nil {
		log.Printf("indicator stream: compute error for %s/%s: %v", ls.req.Indicator, ls.req.Symbol, err)
		return
	}

	if len(result.Data) == 0 {
		return
	}

	last := result.Data[len(result.Data)-1]

	payload := map[string]interface{}{
		"indicator":    ls.req.Indicator,
		"symbol":       ls.req.Symbol,
		"timeframe":    ls.req.Timeframe,
		"timestamp_ms": last.Time,
		"values":       last.Values,
	}

	channelID := "indicator_update:" + ls.req.Symbol + ":" + ls.req.Timeframe
	s.hub.Broadcast(channelID, payload)
}

func (s *LiveStreamService) ActiveCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.activeStreams)
}

func (s *LiveStreamService) SetCandleProvider(provider func(symbol, timeframe string, limit int) ([]Candle, error)) {
	s.candleProvider = provider
}
