# Grafana Dashboard CI Validation

## Overview

The `scripts/validate_grafana_dashboard.py` script validates that all Grafana dashboard JSON
files are structurally correct before they reach production.

## Validation Rules

1. File is valid JSON (no syntax errors).
2. Contains required top-level keys: `title`, `panels`, `schemaVersion`.
3. `schemaVersion` is a positive integer.
4. Every panel with a `targets` array has at least one target containing an `expr`
   (Prometheus) or `query`/`rawSql` (SQL/datasource) key.

## Usage

```bash
# Validate a single file
python scripts/validate_grafana_dashboard.py docs/grafana/dashboards/risk_status.json

# Validate all dashboards (glob supported)
python scripts/validate_grafana_dashboard.py docs/grafana/dashboards/*.json
python scripts/validate_grafana_dashboard.py configs/grafana/*.json

# Run both in CI
python scripts/validate_grafana_dashboard.py docs/grafana/dashboards/*.json configs/grafana/*.json
```

## CI Integration

Add this step to `.github/workflows/ci.yml`:

```yaml
- name: Validate Grafana dashboards
  run: python scripts/validate_grafana_dashboard.py docs/grafana/dashboards/*.json
```

## Expected Output

```
OK: OrcaAlgo — Risk Status
OK: OrcaAlgo — Equity Curve
OK: OrcaAlgo — Regime Gauge
OK: OrcaAlgo — Broker Health
OK: OrcaAlgo — Trade Log
OK: CI/CD Health Dashboard

Total: 0 failure(s)
```

## Adding New Dashboards

When a new dashboard is added to `docs/grafana/dashboards/`:
1. Ensure it has a `title`, `panels`, and `schemaVersion` at the top level.
2. Every data panel's target must include an `expr` key pointing to a valid Prometheus metric.
3. Run the validation script locally before committing.
4. The CI job will automatically catch missing fields.
