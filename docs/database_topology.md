# Database Topology & the Single Source of Truth

**Date:** 2026-07-08
**Status:** Implemented

## Summary

OrcaAlgo runs on **one** database: **TimescaleDB**, provisioned by `docker-compose`
(`timescale/timescaledb:latest-pg16`) and reachable on **host port `5433`**. There
must be exactly one instance. This note explains the incident that produced two,
the fixes that prevent recurrence, and the operating rules.

## The incident: two Postgres instances

Two Postgres servers were running simultaneously:

| Instance | What | TimescaleDB | Role |
|----------|------|-------------|------|
| **docker `orca_algo-postgres-1`** on host **:5433** | `timescale/timescaledb:pg16`, `pgdata` volume | **Yes (2.27.2)**, hypertables present | **Authoritative** — the mandated DB |
| **native Windows Postgres 18** on host **:5432** | a host install (PID) | **No** | Rogue — the app was accidentally using it |
| orphaned `orca-db` (`postgres:17-alpine`) | exited 12 days | — | Dead, removed |

**Why the app used the wrong one:** three layers of config drift all pointed at
`:5432` instead of `:5433`:

1. `.env` had `ORCA_DB_PORT=5432` (its own comment said "5433"), and defined
   `ORCA_DB_URL=…:5432…`.
2. `internal/db.DefaultConfig()` read the discrete `ORCA_DB_*` vars but **ignored
   `ORCA_DB_URL`** — so setting the URL had no effect (a silent footgun).
3. `scripts/orchestrate.py` docker mode started the container on `:5433` but left
   the natively-run Go server's DB env **empty**, so it fell back to `.env`'s
   `:5432`.

The consequences were severe but silent: the app ran against a **plain Postgres
without TimescaleDB**, so `create_hypertable`, compression (7 d) and retention
(30 d) policies — mandated by the stack — never existed there, and migrations
`000017`–`000019` (engine-version + retention) had never been applied to it.

## The fixes (prevent recurrence)

1. **`ORCA_DB_URL` is authoritative** (`internal/db/repository.go`). When set, it is
   the single connection string and overrides the discrete vars; pool sizing is
   appended automatically. Setting the URL can no longer be silently ignored.
2. **Startup guard + provenance log.** On connect the server logs the resolved
   target with a **redacted** DSN, the server address, and the TimescaleDB version;
   if the extension is **absent** it logs a loud `DB WARNING` that it is pointed at
   the wrong (non-Timescale) database. Drift is now impossible to miss:
   ```
   DB connected: postgresql://orca:****@localhost:5433/orca_core?... (db=orca_core server=…:5432 timescaledb=2.27.2)
   ```
3. **Config aligned to `:5433`.** `.env` and `.env.example` now use
   `ORCA_DB_PORT=5433` and `ORCA_DB_URL=…:5433…`.
4. **One variable set.** The legacy `DB_USER`/`DB_PASSWORD`/`DB_NAME` compose vars
   were removed; both the app and `docker-compose` now read the same `ORCA_DB_*`
   variables (single source of truth for credentials).
5. **Orchestrator pins docker mode to `:5433`.** `scripts/orchestrate.py` sets the
   Go server's DB env explicitly to the compose TimescaleDB (host `5433`) in docker
   mode, and runs a **`verify_timescaledb`** check after the DB is healthy —
   warning loudly if the extension is missing.
6. **Orphan removed.** The dead `orca-db` container was deleted.

## Operating rules (best practice)

- **The docker-compose TimescaleDB (`:5433`) is the only project database.** Do not
  point the app at a native/host Postgres — it cannot satisfy the hypertable /
  compression / retention mandate.
- **Provision & migrate via the compose DB.** `docker compose up -d postgres`, then
  `scripts/migrate.ps1` (idempotent; applies `000001`…`000019`).
- **Configure via `ORCA_DB_URL`** (or the `ORCA_DB_*` set) — never a bare
  `ORCA_DB_PORT` that disagrees with compose. In-container services use
  `ORCA_DB_HOST=postgres` / `:5432` (the compose network); host processes use
  `localhost:5433`.
- **Trust the startup line.** If you don't see `timescaledb=<version>` in the
  `DB connected:` log, you're on the wrong database — stop and fix the config.

## Recommendation: drop the native `:5432` Postgres

The native Windows PostgreSQL 18 on `:5432` is **not used by this project** and was
the source of the drift. If it is not needed by anything else on the host, stop and
remove it:

```powershell
# Inspect the service first
Get-Service | Where-Object { $_.Name -like 'postgres*' }
# Stop and disable (adjust the service name)
Stop-Service postgresql-x64-18 ; Set-Service postgresql-x64-18 -StartupType Disabled
# (Optional) uninstall via the PostgreSQL uninstaller if truly unused.
```

If you must keep it (used by other work), that's fine — the guards above ensure
OrcaAlgo will never silently use it again, and it can coexist on `:5432` while the
project owns `:5433`.

## Files touched

| Area | File |
|------|------|
| Authoritative URL + startup guard | `internal/db/repository.go` |
| Config alignment (5433, single var set) | `.env`, `.env.example` |
| Unified compose vars | `docker-compose.yml` |
| Docker-mode DB pin + TimescaleDB verify | `scripts/orchestrate.py` |
