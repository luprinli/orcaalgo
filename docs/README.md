# `docs/` — Project Documentation

Architecture specifications, technical references, and operations guides.

[↑ Back to Root README](../README.md)

## Reference Documents

| File | Content |
|------|---------|
| `Synthetic Data Generation Best Practices 2026-08-11.md` | Data generation methodology, 5-minute base resampling, validation checklist |
| `Backtest Readiness Audit matrix_results (7) 2026-08-12.md` | Backtest matrix audit — prior-issue resolution, anomaly catalog, prioritized enhancements (E1–E15), and post-audit implementation status |
| `Data Quality Fix Plan 2026-08-12.md` | Dev-seed synthetic generator fix plan (2-decimal rounding → full fixed-point unconstrained GBM, aligned with `stooq_synthetic.py`) |
| `grafana/` | Grafana monitoring dashboard setup, JWT authentication, and CI validation documents |
| `runbooks/` | Operational runbooks: startup/shutdown, kill-switch, database migrations, incident response |
| `archive/` | Archived audit reports and historical assessments |

## Operational Runbooks

| File | Content |
|------|---------|
| `runbooks/startup-shutdown.md` | Service startup and graceful shutdown procedures |
| `runbooks/kill-switch.md` | Kill-switch activation and recovery procedures |
| `runbooks/database-migrations.md` | Database migration guide |
| `runbooks/incident-response.md` | Incident classification and response protocols |
