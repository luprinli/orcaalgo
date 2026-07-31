# Prometheus / Grafana / React — Combined Evaluation & Integration Report

**Date:** 2026-07-27
**Scope:** Full architecture evaluation of the three visualization/monitoring surfaces in OrcaAlgo
**Current stack:** Prometheus (port 9090) → Grafana (port 3000) + standalone React SPA (port 5173)

---

## 1. Architecture Overview

```
┌──────────────────────────────────────────────────────────────────────┐
│  Go Server (:8080 API + :9090 /metrics)                             │
│                                                                      │
│  ┌─────────────────────┐        ┌──────────────────────┐            │
│  │  REST /api/v1/*     │        │  /metrics endpoint    │            │
│  │  (Gin handlers)     │        │  (promhttp.Handler)   │            │
│  │  • backtests.run    │        │                        │            │
│  │  • orders.place     │        │  10 instrumented       │            │
│  │  • strategies.*     │        │  prometheus gauges/    │            │
│  │  • risk.status      │        │  counters/histograms   │            │
│  │  • live.metrics     │        │                        │            │
│  └──────┬──────────────┘        └──────────┬─────────────┘            │
│         │                                  │                          │
│  ┌──────▼──────────────┐        ┌──────────▼─────────────┐            │
│  │  WebSocket /ws      │        │  Go runtime metrics    │            │
│  │  (gorilla)          │        │  (heap, goroutines,    │            │
│  │  • ticks (50ms)     │        │   GC, DB pool)         │            │
│  │  • risk (5s)        │        └────────────────────────┘            │
│  │  • positions        │                                              │
│  └─────────┬───────────┘                                              │
└────────────┼──────────────────────────────────────────────────────────┘
             │                            │
     ┌───────▼────────┐         ┌─────────▼──────────┐
     │  React SPA     │         │  Prometheus :9090   │
     │  (:5173)       │         │  (15s scrape)       │
     │                │         │  + 7 alert rules    │
     │  Interactive   │         └─────────┬──────────┘
     │  trading app   │                   │
     └────────────────┘          ┌─────────▼──────────┐
                                 │  Grafana :3000     │
                                 │  8 dashboards      │
                                 │  (Prometheus DS)   │
                                 └────────────────────┘
```

---

## 2. Prometheus — Metric Store + Alert Engine

### 2.1 What It Does

| Function | Implementation |
|----------|---------------|
| **Time-series storage** | TSDB on disk, 15s scrape interval from `app:9091/metrics` |
| **Metric collection** | `promhttp.Handler()` exposes Go runtime + custom business metrics |
| **Alerting** | 7 alert rules in `configs/alerts.yml` evaluated every 30s |
| **PromQL query engine** | Ad-hoc metric exploration, rate calculations, histograms |
| **Retention** | Default 15d (configurable) |

### 2.2 Instrumented Metrics (`internal/monitor/metrics.go`)

| Metric | Type | Purpose |
|--------|------|---------|
| `orca_ticks_processed_total` | Counter | Total market ticks ingested |
| `orca_orders_total` | CounterVec | Orders placed, labeled by strategy + side |
| `orca_ring_buffer_overflow_total` | Counter | Ring buffer overflow events (data loss indicator) |
| `orca_engine_latency_us` | Histogram | Per-tick processing latency (0.5µs–1ms buckets) |
| `orca_regime_state` | Gauge | Current HMM regime (0=Calm, 1=Trending, 2=HighVol, 3=Crisis) |
| `orca_kill_switch_active` | Gauge | Kill switch state (0=normal, 1=halted) |
| `orca_daily_pnl_pct` | Gauge | Current daily P&L as percentage |
| `orca_ws_connections` | Gauge | Active WebSocket connections |
| `orca_ws_broadcast_dropped_total` | CounterVec | Dropped WS messages per channel |
| `orca_ws_auth_failures_total` | Counter | JWT-rejected WS connection attempts |

Plus auto-collected Go runtime metrics: `go_goroutines`, `go_memstats_heap_inuse_bytes`, `go_gc_duration_seconds`, `go_threads`.

### 2.3 Alert Rules (`configs/alerts.yml`)

| Alert | Severity | Trigger |
|-------|----------|---------|
| KillSwitchActive | **critical** | `orca_kill_switch_active == 1` for 10s |
| HighRejectRate | **critical** | `rate(orca_reject_count_total[5m]) > 3` |
| RingBufferOverflow | warning | `orca_ring_buffer_overflow_total > 0` for 5m |
| LatencySpike | warning | P99 tick latency > 100µs for 5m |
| BrokerDisconnected | **critical** | `orca_broker_connected == 0` for 30s |
| HighDrawdown | warning | `orca_daily_loss_pct > 4.0` for 1m |
| RegimeChange | info | `orca_hmm_regime > 2` for 5m |

### 2.4 Critical Gap: Uninstrumented Alert Metrics

4 of 7 alert rules reference metrics that are **not instrumented** in `metrics.go`:

| Alert Rule | Referenced Metric | Instrumented? | Gap |
|------------|------------------|---------------|-----|
| HighRejectRate | `orca_reject_count_total` | **No** | Signal rejection counter never recorded |
| LatencySpike | `orca_tick_processing_latency_seconds_bucket` | **No** | Uses seconds bucket; instrumented metric is `orca_engine_latency_us` (microseconds) — **naming mismatch** |
| BrokerDisconnected | `orca_broker_connected` | **No** | Broker connectivity gauge never set |
| RegimeChange | `orca_hmm_regime` | **No** | Metric exists as `orca_regime_state` — **naming mismatch** |

**Impact:** Alerts will never fire for HighRejectRate and BrokerDisconnected. LatencySpike and RegimeChange reference wrong metric names and will silently fail.

---

## 3. Grafana — Operational Dashboards

### 3.1 What It Does

| Function | Implementation |
|----------|---------------|
| **Visual dashboards** | 8 pre-built dashboards from `docs/grafana/dashboards/` |
| **Data source** | Prometheus (`configs/grafana/provisioning/datasources/prometheus.yml`) |
| **Ad-hoc exploration** | Explore view with PromQL query builder |
| **CI/CD visibility** | GitHub + Codecov datasource plugins |
| **Dashboards as code** | JSON files in `docs/grafana/`, provisioned via `configs/grafana/provisioning/` |

### 3.2 Dashboard Inventory

| Dashboard | Panels | Audience |
|-----------|--------|----------|
| **Backtest Execution** | Combo throughput, active workers, heap/DB pressure, duration p50/p95/p99, batch totals, failed ratio | DevOps |
| **Risk Status** | Kill switch state, max drawdown %, WS auth failures, broadcast drops | DevOps / Trading |
| **Equity Curve** | Paper balance, PnL %, orders/sec, ticks/sec | Trading |
| **Regime Gauge** | Current HMM regime + confidence | Trading |
| **Broker Health** | DB pool usage, heap memory, WS connections, ring buffer overflows | DevOps |
| **Trade Log** | Order rate, matrix backtest workers, batch submissions | Trading |
| **CI/CD Health** | Pipeline success rate (7d), mean CI duration, Go/Python test coverage | DevOps |
| **Backtest Matrix** | Matrix completion status, failed combo ratio | DevOps |

### 3.3 Interaction Model

- **Read-only** — watches metrics, cannot trigger actions
- **Time range filtering** — built-in picker for any window
- **Auto-refresh** — configurable, typically 10–30s
- **No authentication per user** — single admin account (`GRAFANA_USER`/`GRAFANA_PASSWORD`)
- **Embeddable** — supports `?kiosk` mode for iframe embedding without header/sidebar

---

## 4. React Frontend — Interactive Trading Application

### 4.1 What It Does

| Function | Implementation |
|----------|---------------|
| **Write workflows** | Place/cancel orders, submit backtests, deploy strategies, emergency stop, manage accounts/credentials |
| **Real-time display** | WebSocket on `/ws` — ticks (50ms), risk (5s), positions, orders |
| **Interactive charts** | 9 lightweight-charts components with crosshair tooltips, keyboard zoom, drawing tools, indicator overlay |
| **Backtest analysis** | 17-metric detail view, equity/daily-returns/Monte-Carlo charts, trade list with MAE/MFE, calendar heatmap |
| **Auth + roles** | JWT login, 2FA, admin/user roles |
| **Multi-tenant** | Per-user config isolation |

### 4.2 Monitoring Capabilities (subset of full app)

| Page/Tab | What It Monitors | Real-time |
|----------|-----------------|-----------|
| MonitorPage → Overview | 9 KPIs (Balance, Equity, Daily P&L, Sharpe, MaxDD, Win Rate, PF, Regime, Total Trades), equity curve chart, risk limits with progress bars, system status | WebSocket + 10s polling |
| MonitorPage → Risk | Emergency stop/resume (2FA-gated), kill history, regime history timeline | WebSocket + polling |
| MonitorPage → Positions | Active positions table, order status | WebSocket (5s) |
| MonitorPage → Signals | Signal stream with PWin, size, reason, regime | REST polling |
| AdminPage → Health | System health, DB pool, memory, reconciliation status | REST polling |

---

## 5. Functional Overlap Matrix

| Capability | Prometheus | Grafana | React | Overlap? |
|-----------|:---:|:---:|:---:|:---:|
| Kill switch state | Gauge | Stat panel | Badge (OverviewTab) | **3-way** |
| Daily PnL % | Gauge | Gauge/stat | MetricCard | **3-way** |
| Regime state | Gauge | Stat panel | MetricCard label | **3-way** |
| WebSocket connections | Gauge | Timeseries | System status badge | **2-way** |
| Ticks processed | Counter | Timeseries | Not shown | — |
| Orders placed | CounterVec | Timeseries | Orders table | Grafana+React |
| Engine latency | Histogram | Not panelized | Not shown | — |
| Trade history | Not stored | Not stored | Filterable table | **Only React** |
| Equity curve | Not stored | Not stored | Interactive chart | **Only React** |
| Backtest metrics | Not stored | Not stored | 17-metric detail | **Only React** |
| Strategy management | N/A | N/A | Full CRUD | **Only React** |
| Order placement | N/A | N/A | Market/Limit/Stop | **Only React** |
| Emergency stop | N/A | N/A | 2FA-gated button | **Only React** |
| Alerting | 7 alert rules | Alertmanager UI | None | **Only Prometheus** |
| PromQL exploration | Scrape engine | Explore UI | None | **Prometheus+Grafana** |
| CI/CD visibility | N/A | GitHub+Codecov DS | None | **Only Grafana** |

### Overlap Assessment

The only **3-way overlap** is on 3 read-only status indicators (kill switch, PnL, regime). The overlap is **superficial** — each surface presents the same underlying data for different audiences and purposes:

- **Prometheus:** Raw time-series storage + alert evaluation
- **Grafana:** Operational dashboard for infrastructure engineers
- **React:** Trader-facing interactive application with workflow execution

---

## 6. What Each Can Do (That the Others Cannot)

### Only Prometheus
- 15-second scrape interval → sub-minute anomaly detection
- 7 automated alert rules → email/Slack/PagerDuty notifications without human monitoring
- Long-term metric retention (15d+) → trend analysis, capacity planning
- PromQL ad-hoc querying → investigate correlations between any two metrics
- Zero-dependency alert evaluation (no Grafana needed for alerts to fire)

### Only Grafana
- Multi-source dashboards (Prometheus + GitHub + Codecov in single view)
- Drag-and-drop dashboard builder for non-developers
- CI/CD pipeline health visualization (GitHub Actions success rate by branch)
- Ad-hoc metric exploration with auto-complete (Explore view)
- Alertmanager integration for alert lifecycle (silence, acknowledge, escalate)
- `?kiosk` embeddable mode for integration into other tools

### Only React
- **Write workflows:** Backtest submission, order placement, strategy deployment, account management
- **Interactive charting:** Crosshair with trade markers at individual bar level, keyboard zoom, drawing tools
- **Auth + RBAC:** Per-user login, JWT tokens, 2FA, role-based access
- **Workflow gating:** Promote-to-Live 3-step wizard (quality gates → pre-flight → deploy)
- **Rich backtest analysis:** 17-metric detail, calendar heatmap, yearly summary, Monte Carlo from trades
- **Real-time WebSocket streaming:** Ticks at 50ms intervals for live candle painting

---

## 7. Gaps & Broken Wiring

### 7.1 CRITICAL: Uninstrumented Alert Metrics
4 alert rules reference metrics that don't exist in `metrics.go`. Alerts will never fire for broker disconnection and high reject rate. Two metrics have naming mismatches with the instrumented code.

**Fix:** Line 15 in `src`
- Add `orca_reject_count_total` Counter to `metrics.go`
- Add `orca_broker_connected` Gauge to `metrics.go`
- Rename alerts to match instrumented names: `orca_hmm_regime` → `orca_regime_state`, `orca_tick_processing_latency_seconds_bucket` → `orca_engine_latency_us`

### 7.2 MEDIUM: Grafana Dashboard Metric Gaps
Several Grafana dashboards reference metrics that are never populated:
- `orca_matrix_active_workers` (Trade Log dashboard) — not instrumented
- `orca_matrix_combos_completed` (Trade Log dashboard) — not instrumented
- `orca_matrix_batches_total` (Backtest Matrix) — not instrumented
- `orca_backtest_duration_seconds` (Backtest Execution) — not instrumented
- `orca_db_pool_in_use` (Broker Health) — not instrumented
- `orca_heap_inuse_bytes` (Broker Health) — not instrumented
- `orca_hmm_confidence` (Regime Gauge) — not instrumented

**Fix:** Add Prometheus instrumentation in the backtest batch runner and the DB connection pool.

### 7.3 MEDIUM: React Missing System Health Depth
AdminPage → Health tab shows basic `systemHealth` JSON as MetricCards. The Grafana Broker Health and Backtest Execution dashboards show deeper infrastructure metrics (heap growth over time, DB pool saturation, combo throughput history) that no React component visualizes.

### 7.4 LOW: No React Metric Cards for Grafana-Only Metrics
The MonitorPage OverviewTab shows 9 KPIs but misses operational metrics that Grafana surfaces: ring buffer overflow count, engine latency P99, WebSocket connection count, signal reject rate.

---

## 8. Integration Recommendations

### 8.1 Low Effort, Immediate Impact (P0)

#### A. Fix Alert Metric Mismatches (`internal/monitor/metrics.go`)
Add the 4 missing metrics so all 7 alert rules function:
```go
rejectCountTotal = prometheus.NewCounter(prometheus.CounterOpts{
    Name: "orca_reject_count_total", Help: "Total signal rejections",
})
brokerConnected = prometheus.NewGauge(prometheus.GaugeOpts{
    Name: "orca_broker_connected", Help: "Broker connection status (1=connected, 0=disconnected)",
})
```
Rename alert references to match instrumented names:
- `orca_hmm_regime` → `orca_regime_state`
- `orca_tick_processing_latency_seconds_bucket` → `orca_engine_latency_us`
**Effort:** 0.5d | **Risk:** None | **Impact:** Critical — 2 missing alert rules start working

#### B. Add Signal Reject Reason Counter
The RiskPipeline already returns a `reason` string on rejection. Instrument it:
```go
signalRejects = prometheus.NewCounterVec(prometheus.CounterOpts{
    Name: "orca_signal_rejects_total", Help: "Signals rejected by pipeline stage",
}, []string{"stage", "strategy_id"})
```
Call in `pipeline.go:ProcessSignal` on rejection:
```go
signalRejects.WithLabelValues("exposure", req.StrategyID).Inc()
```
**Effort:** 0.5d | **Risk:** None | **Impact:** High — closes gap #7.2, enables Grafana dashboard for signal quality

### 8.2 Medium Effort, Strategic Value (P1)

#### C. Instrument DB Pool and Runtime Health
The Go server already tracks pool stats internally. Expose them:
```go
dbPoolInUse = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
    Name: "orca_db_pool_in_use", Help: "Database connections in use",
}, func() float64 { return float64(repo.Pool().Stat().AcquiredConns()) })

heapInUse = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
    Name: "orca_heap_inuse_bytes", Help: "Go heap in use",
}, func() float64 { 
    var m runtime.MemStats; runtime.ReadMemStats(&m); return float64(m.HeapInuse) 
})
```
**Effort:** 0.5d | **Risk:** None | **Impact:** Medium — makes Broker Health and Backtest Execution dashboards functional

#### D. Instrument Backtest Execution Metrics
Add metrics for the matrix backtest runner (`internal/backtest/batch_runner.go`):
```go
matrixActiveWorkers = prometheus.NewGauge(...)  // "orca_matrix_active_workers"
matrixCombosCompleted = prometheus.NewCounterVec(...) // "orca_matrix_combos_completed_total"
matrixBatchesTotal = prometheus.NewCounter(...) // "orca_matrix_batches_total"
backtestDuration = prometheus.NewHistogram(...) // "orca_backtest_duration_seconds"
```
Wire into `RunMatrixConcurrent` — increment/decrement workers, record combo completion + duration.
**Effort:** 2d | **Risk:** Low | **Impact:** Medium — makes 3 Grafana dashboards functional

#### E. Instrument Business Metrics
Add strategy-level performance tracking so Prometheus can graph P&L trends, backtest pass rates, and prop-firm breaches over time:

| Metric | Type | Labels | Purpose |
|--------|------|--------|---------|
| `orca_daily_pnl_time_series` | Gauge | `strategy_id` | Track P&L over time |
| `orca_backtest_result` | Counter | `strategy_id`, `gate_profile`, `passed` | Pass/fail rate by strategy |
| `orca_propfirm_breach` | Counter | `breach_type` (daily_loss, drawdown) | Breach event tracking |
| `orca_strategy_sharpe` | Gauge | `strategy_id`, `window` (30d, 90d) | Rolling Sharpe by strategy |

These enable a "Strategy Performance" Grafana dashboard that tracks per-strategy health over time — a visibility gap neither React nor Grafana currently fills.
**Effort:** 2d | **Risk:** Low | **Impact:** High — strategic business intelligence

### 8.3 React ↔ Monitoring Integration (P2) — "React Monitoring Hub"

These four tasks form a single cohesive feature: giving traders infrastructure and alert visibility inside the React app without leaving the trading workflow.

#### F. Add Prometheus-Backed System Health Tab
Add a new `SystemHealthTab` to MonitorPage that queries Prometheus via a Go proxy endpoint and displays operational metrics the current OverviewTab misses:
- Engine latency p50/p95/p99 (last 5 minutes)
- Signal reject rate by stage (last 5 minutes)
- Ring buffer overflow count
- Broker connectivity status

```tsx
// web/src/pages/monitor/SystemHealthTab.tsx (new)
const { data } = useQuery({
  queryKey: ['prometheus', 'latency'],
  queryFn: () => fetch('/api/v1/metrics/query?query=orca_engine_latency_us{quantile="0.95"}'),
  refetchInterval: 10_000,
})
```

Add a Go proxy endpoint (`GET /api/v1/metrics/query?query=...`) to avoid exposing Prometheus directly, since the React app already has authenticated REST access.
**Effort:** 1.5d | **Risk:** Low | **Impact:** Medium

#### G. Embed Grafana Dashboards in AdminPage → Infrastructure Tab
Add a new `InfrastructureTab` to AdminPage that embeds key Grafana dashboards in kiosk mode for a single-pane-of-glass operational view:

```tsx
// web/src/pages/admin/InfrastructureTab.tsx (new)
<iframe
  src="http://grafana:3000/d/orca-broker?kiosk&theme=dark&refresh=10s"
  className="w-full h-[500px] border-0 rounded-lg"
  title="Broker Health Dashboard"
/>
<iframe
  src="http://grafana:3000/d/orca-backtest?kiosk&theme=dark&refresh=10s"
  className="w-full h-[500px] border-0 rounded-lg mt-4"
  title="Backtest Execution Dashboard"
/>
```

Also provide a "View in Grafana" link to open the full interactive dashboard in a new tab.
**Effort:** 0.5d | **Risk:** None (iframe isolation preserves Grafaana auth boundary) | **Impact:** Medium

#### H. Push Alerts to React via WebSocket
Currently Prometheus fires alerts via Alertmanager (email/Slack/PagerDuty), but traders in the React app see nothing. Push alert state through the existing WebSocket infrastructure:

```go
// In internal/monitor/ws_hub.go
func (h *WSHub) PushAlert(alert Alert) {
    h.Broadcast("alerts", alert)
}
```

| Option | Mechanism | Latency | Effort |
|--------|-----------|---------|--------|
| A | React polls Alertmanager API | 30–60s | 1d |
| B | Go evaluates alert state, pushes via WS | <1s | 1.5d |
| C | Go reads Alertmanager state, pushes via WS | 5–10s | 1d |

**Recommendation:** Option B — Go evaluates alert rules internally (same rules as `alerts.yml`) and pushes state changes via `/ws`. This reuses the existing WebSocket channel and gives real-time alert toasts in the React header.
**Effort:** 1.5d | **Risk:** Medium (alert rule duplication between Go and Prometheus) | **Impact:** High

#### I. Add Alertmanager UI to AdminPage
Add an `AlertsTab` to AdminPage that polls Alertmanager's API (`/api/v2/alerts`) to display active alerts, provide silence/acknowledge buttons, and show alert history:

```tsx
// web/src/pages/admin/AlertsTab.tsx (new)
const { data: alerts } = useQuery({
  queryKey: ['alertmanager', 'alerts'],
  queryFn: () => fetch('/api/v1/monitoring/alerts').then(r => r.json()),
  refetchInterval: 15_000,
})
```

A Go proxy endpoint (`GET /api/v1/monitoring/alerts`) forwards to Alertmanager to keep auth in one place.
**Effort:** 1d | **Risk:** Low | **Impact:** Medium

### 8.4 Governance & Long-Term Strategy (P3)

#### J. Grafana JWT OAuth Integration
Currently Grafana uses a single admin account (`GRAFANA_USER`/`GRAFANA_PASSWORD`). Integrate Orca's JWT for per-user access:

1. Enable Grafana's `auth.jwt` configuration
2. Configure it to validate Orca's JWT tokens using the same secret
3. Map Orca roles to Grafana roles (`admin` → Admin, `trader` → Viewer)

```yaml
# configs/grafana/grafana.ini
[auth.jwt]
enabled = true
header_name = Authorization
header_prefix = Bearer
email_claim = sub
role_attribute_path = role
auto_sign_up = true
```

**Effort:** 2d | **Risk:** Medium (auth boundary change) | **Impact:** Medium — per-user dashboard access, audit trail

#### K. Dashboard Versioning & CI Validation
There is currently no CI check that Grafana dashboard JSON is valid or that data sources are correctly wired. Add a validation script:

```python
# scripts/validate_grafana_dashboard.py
import json, sys
with open(sys.argv[1]) as f:
    data = json.load(f)
    assert "title" in data, "Missing title"
    assert "panels" in data, "Missing panels"
    assert "schemaVersion" in data, "Missing schemaVersion"
    for panel in data["panels"]:
        if "targets" in panel:
            for target in panel["targets"]:
                assert "expr" in target or "query" in target, \
                    f"Panel '{panel.get('title', '?')}' has no query"
    print(f"OK: {data['title']}")
```

Run in CI:
```yaml
# .github/workflows/ci.yml
- name: Validate Grafana dashboards
  run: for f in configs/grafana/dashboards/*.json; do python scripts/validate_grafana_dashboard.py "$f"; done
```
**Effort:** 1d | **Risk:** None | **Impact:** Low — prevents broken dashboards from reaching production

#### L. Long-Term Metric Retention (15d → permanent)
Prometheus retains metrics for only 15 days by default. For year-over-year comparison and regulatory compliance, export daily aggregated metrics to TimescaleDB:

| Option | Pros | Cons |
|--------|------|------|
| Increase Prometheus retention | Simple | Storage cost, slower queries |
| Thanos + S3 | Drop-in Prometheus compatibility | Complex setup |
| **Export to TimescaleDB** | Already have TimescaleDB, fits existing data pipeline | Manual export job |

**Recommendation:** Export daily aggregates (min, max, mean, p95) to a `metrics_daily` hypertable in TimescaleDB via a cron job. This enables long-term P&L analysis, backtest performance trends, and regime analysis without running Prometheus at large retention.
**Effort:** 2d | **Risk:** Low | **Impact:** Medium — enables regulatory audits and year-over-year analysis

---

## 9. Recommendation Summary

### Keep All Three
Prometheus, Grafana, and React serve fundamentally different roles. None can replace the others:

| If you remove... | You lose... |
|-----------------|-------------|
| **Prometheus** | Automated alerting, 15s metric scrape, PromQL, 15d+ metric retention, infrastructure monitoring |
| **Grafana** | Multi-source dashboards, CI/CD visibility, ad-hoc metric explorer, non-developer dashboard editing |
| **React** | All write workflows (orders, backtests, deploy), interactive charts, auth, WebSocket real-time feed |

### Merged Prioritized Action Plan

| Wave | Priority | Tasks | Effort | Cumulative |
|------|----------|-------|--------|------------|
| **Wave 1: Fix Broken Wiring** | P0 | A: Fix 4 alert metric mismatches + name corrections | 0.5d | 0.5d |
| | P0 | B: Add signal reject reason counter | 0.5d | 1d |
| **Wave 2: Infrastructure Metrics** | P1 | C: DB pool gauge + heap in-use (GaugeFunc) | 0.5d | 1.5d |
| | P1 | D: Backtest matrix worker/throughput histogram | 2d | 3.5d |
| | P1 | E: Business metrics (Sharpe, P&L trend, prop-firm breaches) | 2d | 5.5d |
| **Wave 3: React Monitoring Hub** | P2 | F: SystemHealthTab — Prometheus-backed operational KPIs | 1.5d | 7d |
| | P2 | G: InfrastructureTab — embedded Grafana kiosk iframes | 0.5d | 7.5d |
| | P2 | H: WebSocket-pushed alerts to React toast notifications | 1.5d | 9d |
| | P2 | I: AlertsTab — Alertmanager poll with acknowledge/silence | 1d | 10d |
| **Wave 4: Governance + Longevity** | P3 | J: Grafana JWT OAuth integration | 2d | 12d |
| | P3 | K: Dashboard JSON CI validation script | 1d | 13d |
| | P3 | L: TimescaleDB metric retention (daily aggregates) | 2d | 15d |

### Dependency Graph

```
Wave 1 (P0) ── Fix Broken Wiring ── No dependencies, standalone
    │
    ▼
Wave 2 (P1) ── Infrastructure + Business Metrics ── Depends on Wave 1
    │                                              (metrics.go must be clean first)
    ▼
Wave 3 (P2) ── React Monitoring Hub ── Depends on Wave 2
    │          (SystemHealthTab)         (needs instrumented metrics to display)
    │          (Embedded Grafana)        (needs dashboards to be functional)
    │          (Alerts push)             (needs alert rules to fire)
    │
    ▼
Wave 4 (P3) ── Governance + Longevity ── Independent, parallelizable
               (Grafana OAuth)
               (CI validation)
               (Metric retention)
```

### Immediate Next Step

Execute Wave 1 (1 day) — fix all broken alert wiring. This has zero side effects (only adds new metric registrations and corrects names in existing config files) and is the single highest-impact change: it makes 4 silent alert rules functional and enables signal quality monitoring.

**Verification gate:** After Wave 1, `curl localhost:9091/metrics | grep orca_` should show all 14 instrumented metrics. The Prometheus "Alerts" page should show 0 "unhealthy" rules.

---

## 10. Frontend/Backend Drift Audit

Each recommendation is checked for counterpart changes on the opposite side to prevent implementation drift.

### Wave 1: Fix Broken Wiring

#### A. Fix 4 Alert Metric Mismatches
| Layer | Change | Status |
|-------|--------|--------|
| Backend | Add `orca_reject_count_total`, `orca_broker_connected` to `internal/monitor/metrics.go` | **Specified** |
| Backend | Rename `orca_hmm_regime` → `orca_regime_state` in `configs/alerts.yml` (lines 59-61) | **Specified** |
| Backend | Rename `orca_tick_processing_latency_seconds_bucket` → `orca_engine_latency_us` in `configs/alerts.yml` (line 33) | **Specified** |
| Frontend | None needed — alerts.yml and metrics.go are backend-only consumers | **Correctly absent** |

**Verdict:** Balanced. No frontend counterpart needed.

#### B. Add Signal Reject Reason Counter
| Layer | Change | Status |
|-------|--------|--------|
| Backend | Add `orca_signal_rejects_total` CounterVec to `internal/monitor/metrics.go` | **Specified** |
| Backend | Call `signalRejects.WithLabelValues(...).Inc()` in `internal/risk/pipeline.go:ProcessSignal` on rejection | **Specified** |
| Frontend | None needed — this is a Prometheus-only metric consumed by Grafana dashboards | **Correctly absent** |

**Verdict:** Balanced. No frontend counterpart needed.

---

### Wave 2: Infrastructure Metrics

#### C. DB Pool Gauge + Heap In-Use
| Layer | Change | Status |
|-------|--------|--------|
| Backend | Add `orca_db_pool_in_use`, `orca_heap_inuse_bytes` GaugeFuncs to `internal/monitor/metrics.go` | **Specified** |
| Frontend | None needed — consumed by Grafana Broker Health and Backtest Execution dashboards | **Correctly absent** |

**Verdict:** Balanced.

#### D. Backtest Matrix Metrics
| Layer | Change | Status |
|-------|--------|--------|
| Backend | Add `orca_matrix_active_workers`, `orca_matrix_combos_completed_total`, `orca_matrix_batches_total`, `orca_backtest_duration_seconds` to `internal/monitor/metrics.go` | **Specified** |
| Backend | Wire into `internal/backtest/batch_runner.go:RunMatrixConcurrent` — increment/decrement workers, record durations | **Specified** |
| Frontend | None needed — consumed by Grafana Trade Log and Backtest Execution dashboards | **Correctly absent** |

**Verdict:** Balanced.

#### E. Business Metrics
| Layer | Change | Status |
|-------|--------|--------|
| Backend | Add `orca_daily_pnl_time_series`, `orca_backtest_result`, `orca_propfirm_breach`, `orca_strategy_sharpe` to `internal/monitor/metrics.go` | **Specified** |
| Backend | Wire `orca_propfirm_breach` into `internal/backtest/propfirm_enforcer.go:OnFill` | **Not specified** — implicit |
| Backend | Wire `orca_backtest_result` into `internal/backtest/engine.go:Run` after metrics computation | **Not specified** — implicit |
| Backend | Wire `orca_strategy_sharpe` into `internal/backtest/engine.go:calculateSharpe` or post-run | **Not specified** — implicit |
| Frontend | None needed — intended for a future Grafana "Strategy Performance" dashboard | **Correctly absent** |

**Verdict:** Backend wiring points are underspecified. The report states the metric names and labels but does not identify which Go functions should call `.Inc()`, `.Set()`, or `.Observe()`. This is a **drift risk** — the metrics will be registered but never populated.

**Fix:** Add explicit wiring instructions:
- `orca_propfirm_breach`: Call in `PropfirmEnforcer.CheckDailyLoss` (line 74) and `CheckDrawdown` (line 97)
- `orca_backtest_result`: Call in `Engine.Run` after `calculateWinRate`/`calculateSharpe` block (~line 880)
- `orca_strategy_sharpe`: Update in `Engine.Run` after Sharpe computation
- `orca_daily_pnl_time_series`: Update in `CapitalPoolManager.RecordFill` (line 172) and `CapitalPoolSim.RecordFill` (line 132)

---

### Wave 3: React Monitoring Hub

#### F. SystemHealthTab — Prometheus-Backed Operational KPIs
| Layer | Change | Status |
|-------|--------|--------|
| Backend | Add `GET /api/v1/metrics/query?query={promql}` proxy endpoint | **Specified** (line 346) but not listed as a task in the action plan |
| Backend | The endpoint must proxy to `http://prometheus:9090/api/v1/query`, adding JWT auth | **Not specified** — implicit |
| Backend | Rate limiting on this endpoint (PromQL queries can be expensive) | **Not mentioned** — gap |
| Frontend | New `web/src/pages/monitor/SystemHealthTab.tsx` | **Specified** |
| Frontend | Add to `MonitorPage` tabs (`"systemHealth"` tab trigger) | **Not specified** — implicit |
| Frontend | Add type definitions for the proxy response (`PrometheusQueryResponse`) | **Not specified** — gap |
| Frontend | Error state when Prometheus is unreachable | **Not mentioned** — gap |

**Verdict:** Backend task (proxy endpoint) is underspecified. Frontend types and error handling are missing.

**Fix:**
1. Add explicit backend task: `GET /api/v1/monitoring/prometheus/query?query=...` (nested under `/monitoring/` for discoverability). Include rate limiting (max 1 req/s per user, max 5 concurrent).
2. Frontend type:
```typescript
// web/src/types/api.ts
export interface PrometheusQueryResponse {
  status: 'success' | 'error'
  data?: { resultType: string; result: Array<{ metric: Record<string,string>; value: [number, string] }> }
  error?: string
}
```
3. Error state: If Prometheus returns error, show `ErrorCard` with "Metrics service unavailable" and a retry button.

#### G. InfrastructureTab — Embedded Grafana Kiosk Iframes
| Layer | Change | Status |
|-------|--------|--------|
| Backend | Grafana URL configuration — currently hardcoded as `http://grafana:3000` in the iframe `src` | **Not specified** — gap |
| Backend | Mixed content: if React is served over HTTPS and Grafana is HTTP, browsers block the iframe | **Not mentioned** — gap |
| Frontend | New `web/src/pages/admin/InfrastructureTab.tsx` | **Specified** |
| Frontend | Add to `AdminPage` tabs | **Not specified** — implicit |
| Frontend | "View in Grafana" link to open full dashboard in new tab | **Specified** (line 366) |
| Frontend | Loading/error state when Grafana is unreachable | **Not mentioned** — gap |

**Verdict:** The Grafana URL is hardcoded. No mechanism to discover or configure it from the Go backend. Mixed content issue not addressed.

**Fix:**
1. Add backend endpoint `GET /api/v1/settings` must include `grafana_url` field (or add to the existing settings handler).
2. The `InfrastructureTab` fetches `settings.grafana_url` and constructs iframe `src` dynamically.
3. If Grafana is on the same origin (via reverse proxy), use relative URLs to avoid mixed content. If separate origin, document the CORS/HTTPS requirement.
4. Frontend loading state: show `Skeleton` placeholder while iframe loads. On error: show placeholder with "Grafana dashboard unavailable" text.

#### H. WebSocket-Pushed Alerts to React Toast Notifications
| Layer | Change | Status |
|-------|--------|--------|
| Backend | Go evaluates alert rules internally (same rules as `alerts.yml`) | **Specified** (Option B, line 385) |
| Backend | `WSHub.PushAlert(alert Alert)` broadcasts on `"alerts"` channel | **Specified** |
| Backend | `Alert` struct definition (severity, summary, description, active bool, timestamp) | **Not specified** — gap |
| Backend | Alert rule evaluation loop (goroutine in `main.go` or `scheduler.go`) | **Not specified** — implicit |
| Frontend | Subscribe to `"alerts"` channel in `useWebSocket` hook | **Specified implicitly** |
| Frontend | Toast notification component using `react-hot-toast` (already a dependency) | **Not specified** — gap |
| Frontend | `WSAlertData` TypeScript type matching Go `Alert` struct | **Not specified** — gap |
| Frontend | Deduplication: don't toast the same active alert repeatedly | **Not mentioned** — gap |

**Verdict:** The Go `Alert` struct and frontend `WSAlertData` type are not defined. Without matching types, the Go→React contract is undefined, creating drift.

**Fix:**
1. Go type:
```go
// internal/monitor/alert.go (new)
type Alert struct {
    Name        string `json:"name"`
    Severity    string `json:"severity"`    // "critical" | "warning" | "info"
    Summary     string `json:"summary"`
    Description string `json:"description"`
    Active      bool   `json:"active"`
    FiredAt     string `json:"fired_at"`
    ResolvedAt  string `json:"resolved_at,omitempty"`
}
```
2. Frontend type in `web/src/types/ws.ts`:
```typescript
export interface WSAlertData {
  name: string
  severity: 'critical' | 'warning' | 'info'
  summary: string
  description: string
  active: boolean
  fired_at: string
  resolved_at?: string
}
```
3. Frontend hook: `useAlertToast()` that subscribes to `"alerts"` WS channel and calls `toast.error()` / `toast.warning()` from `react-hot-toast` with deduplication on `alert.name`.

#### I. AlertsTab — Alertmanager Poll with Acknowledge/Silence
| Layer | Change | Status |
|-------|--------|--------|
| Backend | `GET /api/v1/monitoring/alerts` proxy to Alertmanager `/api/v2/alerts` | **Specified** (line 400) |
| Backend | `POST /api/v1/monitoring/alerts/{id}/silence` with `{duration, comment}` body | **Not specified** — critical gap |
| Backend | `POST /api/v1/monitoring/alerts/{id}/acknowledge` with `{comment}` body | **Not specified** — critical gap |
| Frontend | New `web/src/pages/admin/AlertsTab.tsx` with polling + silence/acknowledge buttons | **Specified** |
| Frontend | Add to `AdminPage` tabs | **Not specified** — implicit |
| Frontend | `AlertmanagerAlert` TypeScript type | **Not specified** — gap |
| Frontend | Form for silence duration (1h, 4h, 1d, 1w) + comment | **Not specified** — implicit |
| Frontend | Error state when Alertmanager is unreachable | **Not mentioned** — gap |

**Verdict:** Backend spec is incomplete — only the GET endpoint is described. The POST endpoints for silence and acknowledge are essential to the frontend's interactive functionality. Without them, the `AlertsTab` can display alerts but cannot act on them.

**Fix:**
1. Add backend tasks:
   - `POST /api/v1/monitoring/alerts/{id}/silence` — proxy to `POST http://alertmanager:9093/api/v2/silences` with body `{matchers: [{name: "alertname", value: id, isRegex: false}], startsAt, endsAt, comment, createdBy}`
   - `POST /api/v1/monitoring/alerts/{id}/acknowledge` — records in audit log (Alertmanager has no native acknowledge API)
2. Frontend type:
```typescript
export interface AlertmanagerAlert {
  fingerprint: string
  labels: Record<string, string>
  annotations: { summary: string; description: string }
  startsAt: string
  endsAt: string
  status: { state: 'active' | 'suppressed' | 'unprocessed' }
}
```

---

### Wave 4: Governance + Longevity

#### J. Grafana JWT OAuth Integration
| Layer | Change | Status |
|-------|--------|--------|
| Backend | `configs/grafana/grafana.ini` with `[auth.jwt]` configuration | **Specified** |
| Backend | Ensure JWT secret is shared between Go server and Grafana | **Specified implicitly** |
| Backend | Role mapping: `admin` → Grafana Admin, `trader` → Grafana Viewer | **Specified** |
| Frontend | None needed — transparent to users (iframe already handles auth) | **Correctly absent** |

**Verdict:** Balanced. No frontend counterpart needed.

#### K. Dashboard JSON CI Validation Script
| Layer | Change | Status |
|-------|--------|--------|
| Backend | New `scripts/validate_grafana_dashboard.py` | **Specified** |
| Backend | CI workflow step in `.github/workflows/ci.yml` | **Specified** |
| Frontend | None needed — CI-only change | **Correctly absent** |

**Verdict:** Balanced.

#### L. TimescaleDB Metric Retention
| Layer | Change | Status |
|-------|--------|--------|
| Backend | New `metrics_daily` hypertable in TimescaleDB | **Specified** |
| Backend | Cron job to aggregate Prometheus metrics → TimescaleDB | **Specified** |
| Backend | Prometheus remote_write or batch export script | **Not specified** — implementation detail |
| Frontend | None needed — unless a "Historical Metrics" tab is desired later | **Correctly absent** |

**Verdict:** Balanced. Implementation detail (export mechanism) is TBD, which is appropriate for a P3 item.

---

## 11. Drift Audit Summary

### Items with Both Sides Correctly Specified (8/12)

| Task | Backend | Frontend | Rating |
|------|---------|----------|--------|
| A: Alert metric fixes | ✓ metrics.go + alerts.yml | N/A (no frontend consumer) | Balanced |
| B: Signal reject counter | ✓ metrics.go + pipeline.go | N/A | Balanced |
| C: DB/heap gauges | ✓ metrics.go | N/A | Balanced |
| D: Backtest matrix metrics | ✓ metrics.go + batch_runner.go | N/A | Balanced |
| G: InfrastructureTab | ⚠️ URL hardcoded | ✓ iframe component | **Gap: Grafana URL discovery** |
| H: WebSocket alerts | ⚠️ Alert struct undefined | ⚠️ WS type + toast missing | **Gap: Contract undefined** |
| J: Grafana OAuth | ✓ grafana.ini | N/A | Balanced |
| K: CI validation | ✓ script + workflow | N/A | Balanced |
| L: Metric retention | ✓ hypertable + cron | N/A | Balanced |

### Items with Gaps (4/12)

| Task | Gap | Severity |
|------|-----|----------|
| **E: Business metrics** | Wiring points not specified — metrics registered but never populated | **Medium** — added to section 10.E fix |
| **F: SystemHealthTab** | Backend proxy endpoint not listed as a task. Frontend types missing. No error state. | **Medium** — added to section 10.F fix |
| **G: InfrastructureTab** | Grafana URL hardcoded. Mixed content risk. No loading/error state. | **Medium** — added to section 10.G fix |
| **H: WebSocket alerts** | Go `Alert` struct and frontend `WSAlertData` type undefined. Toast dedup not specified. | **High** — contract undefined, cannot implement |
| **I: AlertsTab** | POST silence/acknowledge endpoints missing. Frontend type missing. | **High** — frontend interactive features unusable without backend |

### Updated Action Plan with Drift Corrections

| Wave | Task | Original Effort | Drift Correction | Revised Effort |
|------|------|----------------|------------------|----------------|
| 1 | A: Alert metric fixes | 0.5d | — | 0.5d |
| 1 | B: Signal reject counter | 0.5d | — | 0.5d |
| 2 | C: DB/heap gauges | 0.5d | — | 0.5d |
| 2 | D: Backtest matrix metrics | 2d | — | 2d |
| 2 | **E: Business metrics** | 2d | +0.25d to specify wiring points | 2.25d |
| 3 | **F: SystemHealthTab** | 1.5d | +0.5d for proxy endpoint task + frontend types + error state | 2d |
| 3 | **G: InfrastructureTab** | 0.5d | +0.25d for settings.grafana_url + loading state | 0.75d |
| 3 | **H: WebSocket alerts** | 1.5d | +0.5d for Alert struct, WS type, toast hook, dedup | 2d |
| 3 | **I: AlertsTab** | 1d | +0.5d for POST endpoints + frontend type | 1.5d |
| 4 | J: Grafana OAuth | 2d | — | 2d |
| 4 | K: CI validation | 1d | — | 1d |
| 4 | L: Metric retention | 2d | — | 2d |

**Original total:** 15d | **Revised total:** 17d (+2d for drift corrections)

### Contract-First Implementation Rule

All **Wave 3 tasks** (F–I) share a pattern: a Go endpoint or WebSocket channel produces data that a React component consumes. Before implementing either side, the **data contract** (Go type + TypeScript type + endpoint signature) must be defined. This prevents the most common form of frontend/backend drift:

```
Define contract → Implement backend → Verify with curl → Implement frontend → E2E test
```

Concrete contract artifacts needed before Wave 3 begins:

| Task | Contract Artifact |
|------|-------------------|
| F | `GET /api/v1/monitoring/prometheus/query` response type (Go + TypeScript) |
| G | `GET /api/v1/settings` must include `grafana_url` field (already in settings type, verify) |
| H | `WSAlertData` type in `web/src/types/ws.ts`, `monitor.Alert` struct in Go |
| I | `GET /api/v1/monitoring/alerts` + `POST /api/v1/monitoring/alerts/{id}/silence` signatures |
