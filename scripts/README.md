# `scripts/` — Build, Launch & Utility Scripts

Scripts for orchestrating the full OrcaAlgo stack, building components, running tests, and managing ports.

---

## Orchestration Scripts

### `orchestrate.py` — Multi-Service Startup Orchestrator + Data Pipeline

Python-based orchestrator that manages the full initialization sequence with health verification, plus a 5-step data regeneration pipeline.

```bash
# Native mode with data regeneration (5-year history, all timeframes)
python scripts/orchestrate.py --local --reset-reseed

# With custom symbols and date range
python scripts/orchestrate.py --local --reset-reseed --reseed-symbols SPY QQQ NVDA --reseed-start 2021-07-01

# Skip validation step
python scripts/orchestrate.py --local --reset-reseed --reseed-skip-validate

# Force-kill processes on occupied ports
python scripts/orchestrate.py --force
```

**Data Pipeline (5 steps):**
1. `seed-all --reset` — Real Yahoo 1d data + VIX + regime + sentiment (5 years)
2. `_generate_synthetic_intraday` — Constrained random walk from 1d OHLC → 5m/15m/30m/1h/4h
3. `_fetch_recent_intraday` — Real Yahoo 5m data for recent 59-day window (overwrites synthetic)
4. `build-regime-logs` — Regime inference from all candle data
5. `validate-data-integrity` — Cross-pipeline validation checks

**Features:**
- 38-symbol default list (`FULL_SYMBOL_LIST`: equities, ETFs, FX, crypto, indices, futures)
- Synthetic intraday generation preserves daily O/H/L/C via constrained random walk
- Real Yahoo 5m data replaces synthetic for most recent 59 days
- Produces 5.7M+ bars across 6 timeframes for 30 symbols

**Configuration:** Service definitions with ports, health URLs, dependencies, and startup commands are declared at the top of the script. Edit the `SERVICES` list to customize.

---

### `launch.ps1` — Full Launch Sequence (PowerShell)

PowerShell orchestrator with native and Docker modes. Declarative service definitions mirror `orchestrate.py`.

```powershell
# Docker mode (default): build and run via Docker Compose
.\scripts\launch.ps1

# Native mode: Docker for postgres/redis, native Go server
.\scripts\launch.ps1 -Native

# Skip React frontend
.\scripts\launch.ps1 -Native -SkipReact

# Skip Prometheus + Grafana
.\scripts\launch.ps1 -SkipMonitoring

# Force-kill processes on ports
.\scripts\launch.ps1 -Native -ForceCleanup

# Dry run
.\scripts\launch.ps1 -Native -DryRun

# Custom health timeout (seconds)
.\scripts\launch.ps1 -Native -HealthTimeout 120

# Write structured log
.\scripts\launch.ps1 -Native -LogFile "logs\startup.log"
```

**Features:**
- Same service ordering and health verification as Python script
- Uses `System.Net.Sockets.TcpClient` for TCP health checks
- Uses `System.Net.WebRequest` for HTTP health checks
- Auto-generates required env vars (`ORCA_JWT_SECRET`, `ORCA_ADMIN_PASSWORD`) for dev mode
- Monitors Go server process and shuts down if it exits unexpectedly

---

## Port Management

### `cleanup-ports.ps1` — Port Process Killer

Identifies and terminates processes occupying specified ports. Used internally by `launch.ps1` and `orchestrate.py`.

```powershell
# Check which processes are on default ports (dry run)
.\scripts\cleanup-ports.ps1 -DryRun

# Force-kill processes on default ports
.\scripts\cleanup-ports.ps1 -Force

# Kill processes on specific ports
.\scripts\cleanup-ports.ps1 -Force -Ports @(8080, 5432, 3000)
```

Ports managed: 8080 (API), 9091 (Prometheus metrics), 5173 (React), 3000 (Grafana), 5432 (PostgreSQL), 6379 (Redis).

Docker daemon processes (`com.docker.backend`, `vpnkit`) are automatically detected and skipped to avoid breaking Docker.

---

## Build Scripts

### `build_all.ps1` — Full Build Pipeline

```
.\scripts\build_all.ps1
```

Builds all components: Python lint + tests, Go build + tests, React build.

---

## Development Scripts

### `dev.ps1` — Hot-Reload Development Mode

```
.\scripts\dev.ps1
```

Starts Go server with `air` (hot-reload) and monitors Odin source files for automatic recompilation.

---

## Testing & Maintenance

### `test_related.py` — Targeted Test Runner

```bash
python scripts/test_related.py
```

Detects changed files vs main branch and runs only the relevant tests (Python + Go). Supports `--language python|go|all`.

### `anti_pattern_scan.py` — Hard Prohibition Scanner

```bash
python scripts/anti_pattern_scan.py
```

Scans the codebase for violations of the 18 hard prohibitions defined in `AGENTS.md`. Enforces HP #17 (Rule 11: RiskPipeline bypass detection) via CI.

### Stooq Data Pipeline — Real Intraday Ingestion

Four stream-based scripts replace synthetic intraday data with real Stooq bars (18-symbol prop-firm universe). All read/write the TimescaleDB `candles` table via `ORCA_DB_URL` and never load full CSVs into memory.

```bash
python scripts/stooq_discovery.py     # Walk stooq tree → data/stooq/manifest.json
python scripts/stooq_seed.py          # Stream real 1h + 5m CSVs → candles (source='stooq')
python scripts/stooq_resample.py      # 1H→4H + 5m→15m/30m (source='stooq-resampled')
python scripts/stooq_synthetic.py     # Unconstrained-GBM gap-fill (source='stooq-calibrated')
```

| Script | Purpose | Source Label |
|--------|---------|-------------|
| `stooq_discovery.py` | Maps 18 symbols to stooq files, outputs manifest | — |
| `stooq_seed.py` | Streams real 1h (2yr) + 5m (5mo) CSVs | `stooq` |
| `stooq_resample.py` | Resamples 1H→4H, 5m→15m/30m | `stooq-resampled` |
| `stooq_synthetic.py` | Unconstrained GBM gap-fill calibrated from stooq σ (Close-to-Close + H-L range) | `stooq-calibrated` |

**Synthetic generator**: Pure Geometric Brownian Motion with per-symbol volatility (EWMA λ=0.94), soft blend toward daily Close in final 50% of session. Paths free to break daily H/L — no clipping.

### `migrate.ps1` — Database Migration Runner

```
.\scripts\migrate.ps1
```

Applies TimescaleDB migrations from `internal/db/migrations/`.

### `run_backtest.ps1` — Backtest Runner

```
.\scripts\run_backtest.ps1
```

Executes a backtest using the Go backtest engine with sample OHLCV data.

### `seed_data.sql` — Database Seed SQL

Raw SQL fixture for seeding the database outside of the Go seeder pipeline.

---

## Pre-Commit Hooks

### `pre-commit-scope.ps1` — Language-Aware Pre-Commit

```
.\scripts\pre-commit-scope.ps1
```

Detects which languages were changed in staged files and runs only relevant linters (ruff + mypy for Python, go vet for Go, eslint + tsc for React). Used by `.pre-commit-config.yaml`.
