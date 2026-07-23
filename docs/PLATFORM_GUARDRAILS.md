# OrcaAlgo Platform Guardrails — Operations Documentation

**Version:** 1.0
**Date:** 2026-07-22
**Scope:** Regression prevention, AI agent overreach protection, system integrity validation

---

## 1. Architecture Overview

```
┌──────────────────────────────────────────────────────────────┐
│                    GIT PRE-COMMIT HOOK                        │
│              .githooks/pre-commit                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ 1. change_audit.py --staged   — blocks destructive deltas│ │
│  │ 2. anti_pattern_scan.py      — hard prohibition scan    │ │
│  │ 3. env_guard.py              — environment safety check │ │
│  └─────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
                               │
    ┌──────────────────────────┼──────────────────────────────┐
    │    COMMAND LAYER          │    AGENT LAYER               │
    │    .kilo/command/         │    .kilo/agent/              │
    │ ┌──────────────────────┐  │ ┌──────────────────────────┐ │
    │ │ Prerequisites step 0 │  │ │ ENVIRONMENT GUARDRAIL    │ │
    │ │ - change_audit       │  │ │ - env_guard check first  │ │
    │ │ - env_guard          │  │ │ - no live account ops    │ │
    │ │ - anti_pattern_scan  │  │ │ - post-fix test req      │ │
    │ └──────────────────────┘  │ └──────────────────────────┘ │
    └──────────────────────────┴──────────────────────────────┘
                               │
                    ┌──────────┴──────────┐
                    │   SCRIPTS LAYER      │
                    │  scripts/            │
                    │  change_audit.py     │  File delta magnitude detection
                    │  env_guard.py        │  Live account protection
                    │  anti_pattern_scan   │  10 hard prohibition enforcement
                    │  test_related.py     │  Change-aware test execution
                    └─────────────────────┘
```

---

## 2. Guardrail Catalog

### 2.1 Change Audit (`scripts/change_audit.py`)

**What it protects:** Prevents another "radical rebuild" that deletes 75 files and replaces with 10.

**Thresholds** (configurable in `config/change-threshold.yaml`):
- `max_changed_files: 50` — Block commits exceeding this file count
- `max_deletion_pct: 30` — Block commits where >30% of changed lines are deletions
- `test_file_guard: true` — Block commits that delete test files without replacement

**Critical files** (in `config/critical-paths.json`):
Deletion or degradation of any of these files ALWAYS blocks:
- `orca/preflight/checklist.py` (pre-deployment checks)
- `internal/risk/kill_switch.go` (kill-switch)
- `internal/strategy/registry.go` (strategy factory)
- `internal/risk/position_sizer.go` (sizing)
- `internal/backtest/engine.go` (ML fields)
- `orca/simulation/__init__.py` (synthetic data)
- `orca/models/strategy.py` (RiskProfile)
- `orca/cli.py` (CLI subcommands)

**Usage:**
```bash
# Pre-commit (staged only)
python scripts/change_audit.py --staged

# Full last-commit audit
python scripts/change_audit.py

# JSON output for automation
python scripts/change_audit.py --json

# Override (audit logged)
python scripts/change_audit.py --staged --bypass-guard
# ^ requires: log entry in reports/change_audit_YYYYMMDDHHMMSS.json
```

**Exit codes:** 0 = safe, 1 = blocked, 2 = config error

---

### 2.2 Environment Guard (`scripts/env_guard.py`)

**What it protects:** Prevents AI agents from connecting to live brokerage accounts during `/preflight`, `/deploy-gate`, kill-switch E2E tests, or balance reconciliation.

**Protected operations:**
- `kill_switch_e2e` — Cancels ALL open orders on target account
- `balance_reconciliation` — Connects to exchange API
- `preflight_strict` — Full pre-deployment checklist
- `deploy_gate` — Pre-deployment verification
- `orca_calibrate` — Could connect to broker for data
- `orca_simulate` — Simulation with broker configs

**Environment detection:**
| Variable | Value | Result |
|----------|-------|--------|
| `PAPER_TRADING=true` | Safe | All operations allowed |
| `ALPACA_LIVE=true` + no auth | BLOCKED | Must set `ALLOW_LIVE_GUARD=explicit` |
| `ORCA_ENV=production` + no auth | BLOCKED | Same as above |

**Usage:**
```bash
# Check before running a command
python scripts/env_guard.py --check kill_switch_e2e

# Explicit live authorization
ALLOW_LIVE_GUARD=explicit python scripts/env_guard.py --check deploy_gate --allow-live
```

**Exit codes:** 0 = safe, 1 = blocked, 2 = error

---

### 2.3 Anti-Pattern Scanner (`scripts/anti_pattern_scan.py`)

**What it protects:** Enforces all 10 hard prohibitions from `AGENTS.md`.

**Hardenings applied (2026-07-22):**
- Rule 2: Expanded to match camelCase price fields (`limitPrice`, `stopPrice`, etc.)
- Rule 5: Checks actual calibration recency (>90 days = violation), not just directory existence
- Rule 7: Detects `@dataclass` without `frozen=True`; reduces Go false-positives to domain dirs only
- Rule 8: Verifies `_isLocked` AND `_killSwitchInFlight` appear inside `Trigger()` specifically
- SARIF output for CI dashboard integration
- Severity levels: CRITICAL (merge-blocking), HIGH (deploy-blocking), MEDIUM (advisory)

**Usage:**
```bash
# Full scan (reports HIGH+ by default)
python scripts/anti_pattern_scan.py

# Changed files only (fast pre-commit)
python scripts/anti_pattern_scan.py --changed-only

# SARIF output for CI dashboards
python scripts/anti_pattern_scan.py --format sarif

# Lower severity threshold
python scripts/anti_pattern_scan.py --min-severity MEDIUM
```

---

### 2.4 Test-Related Runner (`scripts/test_related.py`)

**What it protects:** Ensures that changed code is always tested. Blocks merges when changed files map to zero tests.

**Complete mappings (2026-07-22):**
- Go: All `internal/` packages mapped + `cmd/`
- Python: All `orca/` subdirectories mapped
- Web: `tsc` + `vitest` for `web/src/`, `playwright` for `web/e2e/`
- GKR: `orca validate` for `configs/strategies/` changes

**Zero-test guard (NEW):**
If changed files map to zero tests, exit code **2** (not 0). This is **BLOCKING**.

**Usage:**
```bash
# All languages
python scripts/test_related.py

# Specific language
python scripts/test_related.py --language go
python scripts/test_related.py --language web

# Custom base branch
python scripts/test_related.py --base origin/develop
```

**Exit codes:** 0 = PASS, 1 = FAILURES, 2 = UNTESTED (blocking)

---

## 3. Installation

### 3.1 Pre-Commit Hook

```bash
# Install (once per clone)
git config core.hooksPath .githooks

# Verify
ls .githooks/pre-commit  # Should exist and be executable

# Uninstall (if needed)
git config --unset core.hooksPath
```

### 3.2 Verify Guardrails Are Active

```bash
# 1. Change audit detects staged deletions
echo "test" > /tmp/test_delete.py && git add /tmp/test_delete.py
python scripts/change_audit.py --staged  # Should report file count

# 2. Environment guard blocks live ops
ALPACA_LIVE=true python scripts/env_guard.py --check kill_switch_e2e
# ^ Should exit 1 with BLOCKED message

# 3. Anti-pattern scan runs clean
python scripts/anti_pattern_scan.py
# ^ Should exit 0

# 4. Test-related detects zero-test changes
echo "test" > /tmp/untested.go && git add /tmp/untested.go
git diff --cached --name-only | python scripts/test_related.py
# ^ Should exit 2 (UNTESTED)
```

---

## 4. Override Procedures

### 4.1 Bypassing Change Audit (Large Refactors)

Large intentional refactors that exceed thresholds:

```bash
# 1. Review the audit report
python scripts/change_audit.py --staged --json > reports/change_audit_pre.json

# 2. Get approval (document in commit message)
# Commit message must include: "GUARDRAIL OVERRIDE: <reason>"

# 3. Bypass with audit log
python scripts/change_audit.py --staged --bypass-guard
```

The `--bypass-guard` flag writes an audit entry to `reports/change_audit_bypass_*.json`.

### 4.2 Allowing Live Operations

```bash
# Double-gate: env var + flag
ALLOW_LIVE_GUARD=explicit orca preflight --strict --allow-live
```

Both `ALLOW_LIVE_GUARD=explicit` AND `--allow-live` must be present.

---

## 5. Incident Response

### 5.1 Guardrail Blocks Legitimate Operation

1. Read the error message — it includes the exact reason for the block
2. Check the guardrail configuration file:
   - `config/change-threshold.yaml` for threshold adjustments
   - `config/env_guard.json` for environment restrictions
   - `config/critical-paths.json` for protected file lists
3. Adjust config if thresholds are too strict
4. Re-run the operation

### 5.2 Guardrail Failed to Block (False Negative)

1. Check if the guardrail is installed and active:
   ```bash
   git config core.hooksPath  # Should output ".githooks"
   ```
2. Verify the script exists and is executable
3. Review the guardrail test suite:
   ```bash
   pytest tests/guardian/test_guardrails.py -v
   ```
4. If a bug is found in a guardrail, fix the script and add a regression test

### 5.3 Environment Guard Blocks in Paper Trading Mode

1. Verify environment variables:
   ```bash
   echo $PAPER_TRADING  # Must be "true"
   echo $ALPACA_LIVE    # Must be empty or "false"
   ```
2. If using a `.env` file, verify it's loaded before the command
3. Check `config/env_guard.json` for the correct allowed environments

---

## 6. Configuration Reference

### `config/change-threshold.yaml`
```yaml
max_changed_files: 50
max_deletion_pct: 30
test_file_guard: true
```

### `config/critical-paths.json`
Declares files that must never be deleted or degraded. Each entry includes:
- `files`: Glob patterns for protected files
- `required_patterns`: Substrings that must exist in the file
- `check_command` / `test_command`: Validation command to verify integrity

### `config/env_guard.json`
Declares which operations are protected and which environments are allowed.

### `config/quality-gates.yaml`
Central thresholds for CI gates (referenced by `/ci` command).

### `config/ci-presets.yaml`
CI run modes: `quick`, `standard`, `full`, `pre-deploy`.

---

## 7. Self-Validation

Run the guardrail test suite to verify all protections are working:

```bash
pytest tests/guardian/test_guardrails.py -v
```

This validates:
- Change audit configuration loading
- Environment guard safety/block logic
- Anti-pattern scanner severity and SARIF output
- Test-related Go package mapping completeness
- Critical paths JSON validity and file existence
- Preflight check count (>=12)
- Kill-switch re-entrancy guards
- Strategy registry factory pattern
- Engine ML field presence

---

*Maintained by the platform team. Last updated 2026-07-22.*
