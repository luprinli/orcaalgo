# `scripts/` — Build, Launch & Utility Scripts

Scripts for orchestrating the full OrcaAlgo stack, building components, running tests, and managing ports.

---

## Orchestration Scripts

### `orchestrate.py` — Multi-Service Startup Orchestrator (Cross-Platform)

Python-based orchestrator that manages the full initialization sequence with health verification.

```bash
# Native mode: Docker for postgres/redis, native Go server + React
python scripts/orchestrate.py

# Docker mode: everything via Docker Compose
python scripts/orchestrate.py --docker

# Skip React frontend
python scripts/orchestrate.py --no-react

# Force-kill processes on occupied ports
python scripts/orchestrate.py --force

# Dry run — show what would happen without executing
python scripts/orchestrate.py --dry-run

# Write structured log to file
python scripts/orchestrate.py --log-file logs/startup.log
```

**Features:**
- Identifies and terminates processes occupying required ports (with `--force`)
- Launches dependency services in correct order: PostgreSQL → Redis → Go API → React
- Health-checks each service (TCP connect for DB, HTTP for API) before proceeding
- Supports both native and Docker Compose modes
- Structured colored logging with timestamps
- Graceful shutdown on Ctrl+C (SIGINT/SIGTERM)
- Optional monitoring services (Prometheus, Grafana)

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
