# OrcaAlgo Grafana Dashboards

## Backtest & Matrix Execution

`orca_backtest_execution_dashboard.json` — live matrix-execution health for the
execution framework (see `../backtest_execution_framework_plan.md` §5.5).

### Metrics it uses (exported on `:9090/metrics`)

| Metric | Type | Meaning |
|--------|------|---------|
| `orca_backtest_duration_seconds` | histogram | per-combo backtest time |
| `orca_matrix_combos_completed_total{status}` | counter | combos completed (`completed`/`failed`) |
| `orca_matrix_batches_total` | counter | matrix batches submitted |
| `orca_matrix_active_workers` | gauge | backtests executing across all batches |
| `orca_heap_inuse_bytes` | gauge | Go heap in-use (resource pressure) |
| `orca_db_pool_in_use` | gauge | DB connections acquired |

### Panels

- **Combo throughput (/min)**, **Active workers**, **Heap in-use (MB)**, **DB pool in-use** (stat row)
- **Combo completion rate by status** (stacked)
- **Backtest duration p50/p95/p99** (from the histogram)
- **Heap in-use vs budget** (2048 MB line = `ORCA_MATRIX_MEM_BUDGET_MB`)
- **Concurrency & DB pool**
- **Batches / combos totals** and **failed-combo ratio**

### Setup

1. **Prometheus** — scrape the server's metrics endpoint:
   ```yaml
   scrape_configs:
     - job_name: orca
       static_configs:
         - targets: ['localhost:9090']   # OrcaAlgo /metrics
   ```
   (Compose already runs Prometheus; add the job if not present.)
2. **Grafana** → Dashboards → Import → upload `orca_backtest_execution_dashboard.json`
   → select your Prometheus datasource for the `DS_PROMETHEUS` input.
3. The heap thresholds (yellow 1433 MB ≈ 70%, red 1843 MB ≈ 90%) assume the default
   2048 MB budget; adjust if you change `ORCA_MATRIX_MEM_BUDGET_MB`.
