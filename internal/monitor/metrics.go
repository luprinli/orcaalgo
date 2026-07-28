package monitor

import (
	"net/http"
	"runtime"
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
	rejectCountTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "orca_reject_count_total",
		Help: "Total signal rejections by the risk pipeline",
	})
	signalRejects = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "orca_signal_rejects_total",
		Help: "Signals rejected by pipeline stage",
	}, []string{"stage", "strategy_id"})
	brokerConnected = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orca_broker_connected",
		Help: "Broker connection status (1=connected, 0=disconnected)",
	})
	dbPoolInUse = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "orca_db_pool_in_use",
		Help: "Database connections currently in use",
	}, func() float64 { return float64(atomic.LoadInt64(&dbPoolInUseVal)) })
	heapInUseBytes = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "orca_heap_inuse_bytes",
		Help: "Go heap memory currently in use",
	}, func() float64 {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return float64(m.HeapInuse)
	})
	matrixActiveWorkers = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "orca_matrix_active_workers",
		Help: "Number of backtest workers currently executing",
	})
	matrixCombosCompleted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "orca_matrix_combos_completed_total",
		Help: "Total matrix backtest combos completed",
	}, []string{"status"})
	matrixBatchesTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "orca_matrix_batches_total",
		Help: "Total matrix backtest batches submitted",
	})
	backtestDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "orca_backtest_duration_seconds",
		Help:    "Per-combo backtest duration in seconds",
		Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300},
	})
	propfirmBreach = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "orca_propfirm_breach_total",
		Help: "Prop-firm rule breach events",
	}, []string{"breach_type"})
	strategySharpe = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "orca_strategy_sharpe",
		Help: "Rolling Sharpe ratio per strategy",
	}, []string{"strategy_id", "window"})
)

var dbPoolInUseVal int64

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
		rejectCountTotal,
		signalRejects,
		brokerConnected,
		matrixActiveWorkers,
		matrixCombosCompleted,
		matrixBatchesTotal,
		backtestDuration,
		propfirmBreach,
		strategySharpe,
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

func RecordReject() {
	rejectCountTotal.Inc()
}

func RecordSignalReject(stage, strategyID string) {
	signalRejects.WithLabelValues(stage, strategyID).Inc()
}

func SetBrokerConnected(connected bool) {
	if connected {
		brokerConnected.Set(1)
	} else {
		brokerConnected.Set(0)
	}
}

func SetDBPoolInUse(count int) {
	atomic.StoreInt64(&dbPoolInUseVal, int64(count))
}

func RecordMatrixBatchStart() {
	matrixBatchesTotal.Inc()
}

func RecordMatrixCombo(status string) {
	matrixCombosCompleted.WithLabelValues(status).Inc()
}

func AdjustMatrixWorkers(delta int) {
	matrixActiveWorkers.Add(float64(delta))
}

func RecordBacktestDuration(seconds float64) {
	backtestDuration.Observe(seconds)
}

func RecordPropfirmBreach(breachType string) {
	propfirmBreach.WithLabelValues(breachType).Inc()
}

func SetStrategySharpe(strategyID, window string, sharpe float64) {
	strategySharpe.WithLabelValues(strategyID, window).Set(sharpe)
}
