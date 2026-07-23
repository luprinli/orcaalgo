package monitor

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ticksProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "orca_ticks_processed_total",
		Help: "Total number of market ticks processed by the engine",
	})
	ordersTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "orca_orders_total",
		Help: "Total orders placed",
	}, []string{"strategy", "side"})
	ringBufferOverflows = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "orca_ring_buffer_overflow_total",
		Help: "Total ring buffer overflow events",
	})
	engineLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "orca_engine_latency_us",
		Help:    "Engine processing latency in microseconds",
		Buckets: []float64{0.5, 1, 2, 5, 10, 25, 50, 100, 250, 500, 1000},
	})
	regimeState = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orca_regime_state",
		Help: "Current HMM regime state (0=Calm, 1=Trending, 2=HighVol, 3=Crisis)",
	})
	killSwitchActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orca_kill_switch_active",
		Help: "Kill switch status (0=normal, 1=halted)",
	})
	dailyPnLPct = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orca_daily_pnl_pct",
		Help: "Current daily P&L as percentage",
	})
	wsConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orca_ws_connections",
		Help: "Active WebSocket connections",
	})
	wsBroadcastDropped = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "orca_ws_broadcast_dropped_total",
		Help: "WebSocket messages dropped due to broadcast buffer overflow",
	}, []string{"channel"})
	wsAuthFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "orca_ws_auth_failures_total",
		Help: "WebSocket connection attempts rejected due to invalid JWT token",
	})
)

func init() {
	prometheus.MustRegister(
		ticksProcessed,
		ordersTotal,
		ringBufferOverflows,
		engineLatency,
		regimeState,
		killSwitchActive,
		dailyPnLPct,
		wsConnections,
		wsBroadcastDropped,
		wsAuthFailures,
	)
}

type Metrics struct {
	engineTickCount     int64
	engineOverflowCount int64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

func (m *Metrics) RecordTick() {
	ticksProcessed.Inc()
}

func (m *Metrics) RecordOrder(strategy, side string) {
	ordersTotal.WithLabelValues(strategy, side).Inc()
}

func (m *Metrics) RecordOverflow() {
	ringBufferOverflows.Inc()
}

func (m *Metrics) RecordLatency(us float64) {
	engineLatency.Observe(us)
}

func (m *Metrics) SetRegime(regime int8) {
	regimeState.Set(float64(regime))
}

func (m *Metrics) SetKillSwitch(halted bool) {
	if halted {
		killSwitchActive.Set(1)
	} else {
		killSwitchActive.Set(0)
	}
}

func (m *Metrics) SetDailyPnL(pct float64) {
	dailyPnLPct.Set(pct)
}

func (m *Metrics) SetWSConnections(count int) {
	wsConnections.Set(float64(count))
}

type LatencyTracker struct {
	start    time.Time
	metrics  *Metrics
}

func StartLatencyTrack(metrics *Metrics) *LatencyTracker {
	return &LatencyTracker{
		start:   time.Now(),
		metrics: metrics,
	}
}

func (lt *LatencyTracker) Stop() {
	elapsed := time.Since(lt.start).Microseconds()
	lt.metrics.RecordLatency(float64(elapsed))
}

var metricsTickCounter int64

func AtomicIncTick() int64 {
	return atomic.AddInt64(&metricsTickCounter, 1)
}

func RecordWSBroadcastDropped(channel string) {
	wsBroadcastDropped.WithLabelValues(channel).Inc()
}

func RecordWSAuthFailure() {
	wsAuthFailures.Inc()
}

func AtomicGetTickCount() int64 {
	return atomic.LoadInt64(&metricsTickCounter)
}
