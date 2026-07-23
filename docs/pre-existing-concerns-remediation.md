# OrcaAlgo Pre-Existing Concerns — Remediation Plan

Generated 2026-07-23. Covers all pre-existing issues identified during the post-implementation
cleanup of the frontend transfer catalog work. Issues are categorized by stack, severity, and
remediation effort.

---

## Frontend (`web/`) — ALL RESOLVED

All 6 pre-existing issues were fixed in-place during the cleanup pass. Verification confirms
zero TypeScript errors, zero ESLint errors, and 154/154 passing tests.

### Resolved Issues

| # | File:Line | Issue | Root Cause | Fix |
|---|-----------|-------|-----------|-----|
| 1 | `__tests__/auth.test.ts:4` | `beforeEach` not found by TS | `vitest` globals not imported in tsconfig | Added `beforeEach` to vitest import |
| 2 | `pages/StatusPage.tsx:8` | `Record<string, unknown>` vs `SystemHealth` | `useState` generic didn't match API return type | Changed to `SystemHealth`, imported from `types/api.ts` |
| 3 | `App.tsx:233` | Ternary expression used as statement | ESLint `no-unused-expressions` | Converted ternary to `if/else` block |
| 4 | `charts/EquityCurveChart.tsx:4` | `LineData` imported but unused | Dead import | Removed from import |
| 5 | `charts/EquityCurveChart.tsx:287` | `i` defined but unused in `.map()` | Shadowed unused var | Renamed to `_i` |
| 6 | `pages/LiveTrading.tsx:16` | `wsTicks` assigned but never read | State value never consumed | Discarded with `[, setWsTicks]` |

---

## Go Backend (`internal/`, `tests/`)

### G1 — SHADOW TEST BUILD FAILURE (Blocks CI)

**Severity:** CRITICAL (P0) — halts `go vet ./...` in CI backend job
**Effort:** 10 minutes

**Files:**
- `tests/shadow/shadow_mode_test.go:62`, `:102`, `:115`

**Root cause:** Two methods (`SetStopLossConfig`, `ActivePositions`) were removed from
`*engine.LiveEngine` after the test was written. `SetStopLossConfig` became internal
(stop-loss is now handled by `CheckOpenStops()` inside `ProcessTick`). `ActivePositions`
was removed without a replacement getter.

**Fix:**
```
Line 62:  Remove `liveEng.SetStopLossConfig(nil, nil, 14, false)`
Line 102: Remove `liveEng.SetStopLossConfig(slCfg, tpCfg, 14, false)`
Line 115: Either expose `ActivePositions() int` on LiveEngine (preferred), or
          remove the log line and test `liveEng.Halted` only.
```

**Dependency ordering:** None. This test is isolated — no DB, no network, no external processes.

**Verification gate:** `go vet ./... && go test ./tests/shadow/ -count=1 -v`

---

### G2 — ENGINE REPLAY PARITY TEST (Passes but Weak)

**Severity:** LOW (P3) — test passes but never exercises real code paths
**Effort:** 1 hour

**File:** `internal/engine/replay_parity_test.go:59-98`

**Root cause:** `TestReplayParity_WithML` never registers a strategy in
`strategy.GlobalRegistry()` and never injects a mock `metaLabeler`. The feed loop
produces zero signals every run, making the parity check trivially `0 == 0`. The
"ML gate" path (`metaLabeler.IsHealthy()` → `EvaluateSignal`) is never reached.

**Fix:**
```go
// Before replayEngine.Replay():
strategy.GlobalRegistry().Register("mean_reversion",
    strategy.NewMeanReversionStrategy(strategy.MeanReversionParams{Lookback: 20, EntryZ: 1.5}))
```

**Verification gate:** `go test ./internal/engine/ -run TestReplayParity -count=1 -v`

---

## Python (`orca/`)

### P1 — PYFLAKES CRITICAL ERRORS (12 violations, 5 files)

**Severity:** HIGH (P1) — undefined names cause `ImportError` or `NameError` at runtime
**Effort:** 15 minutes

| # | File:Line | Code | Issue | Fix |
|---|-----------|------|-------|-----|
| 1 | `orca/optimize/sweeper.py:154` | F821 | `"np.ndarray"` before `import numpy` | Move `import numpy as np` to top |
| 2 | `orca/optimize/sweeper.py:180` | F821 | `"np.ndarray"` before `import numpy` | Move `import numpy as np` to top |
| 3 | `orca/simulation/regime.py:233` | F821 | `Path` used without import | Add `from pathlib import Path` |
| 4 | `orca/simulation/regime.py:236` | F821 | `Path` used without import | Add `from pathlib import Path` |
| 5 | `orca/attribution/slicer.py:44` | F841 | `p = wins / n` dead assignment | Remove line |
| 6 | `orca/simulation/calibrate.py:313` | F841 | `date_str` dead assignment | Remove line |
| 7 | `orca/simulation/generate_1m.py:245` | F841 | `minute_prices` dead assignment | Remove line |
| 8 | `orca/simulation/regime_generator.py:94` | F841 | `n_minutes` dead assignment | Remove line |
| 9 | `orca/cli.py:718` | F401 | `DEFAULT_TRANSITION_MATRIX` unused | Remove import |
| 10 | `orca/db/__init__.py:13` | F401 | `typing.Optional` unused | Remove import |
| 11 | `orca/preflight/checklist.py:89` | F401 | `attribute_pnl` import-only existence check | Replace with `importlib.util.find_spec` |
| 12 | `orca/cli.py:698` | F401 | (Additional unused import) | Remove import |

**Dependency ordering:** None — all fixes are single-line in isolated files.

**Verification gate:** `python -m ruff check orca/ --select F`

---

### P2 — RAISE-WITHOUT-FROM IN CLI (17 violations in 1 file)

**Severity:** MEDIUM (P1) — masks exception chains; debugging harder
**Effort:** 5 minutes

**File:** `orca/cli.py` (lines 88, 196, 230, 295, 317, 358, 413, 463, 494, 565, 633, 666, 703, 790, 827, 852; plus 1 more)

**Pattern:**
```python
# Current:
except ImportError as e:
    typer.echo(f"Module not available: {e}", err=True)
    raise typer.Exit(code=1)

# Fixed:
except ImportError as e:
    typer.echo(f"Module not available: {e}", err=True)
    raise typer.Exit(code=1) from e
```

**Verification gate:** `python -m ruff check orca/cli.py --select B904`

---

### P3 — MYPY GENERIC TYPE ARGUMENTS (59 violations, ~20 files)

**Severity:** MEDIUM (P2) — reduces type safety of public API contracts
**Effort:** 15 minutes

**Pattern:**
```python
# Current:
candles_by_symbol: dict = {}
samples: list[dict] = []

# Fixed:
candles_by_symbol: dict[str, list[Candle]] = {}
samples: list[dict[str, Any]] = []
```

**Top files:**
- `orca/optimize/sweeper.py` (16 errors)
- `orca/cli.py` (13 errors)
- `orca/data_quality/__init__.py` (11 errors)
- `orca/ml/train/hierarchical.py` (10 errors)

**Verification gate:** Run `python -m mypy orca/ --ignore-missing-imports` before/after;
expect reduction from 173 → ~114 errors.

---

### P4 — MYPY UNTYPED FUNCTIONS (24 violations, ~15 files)

**Severity:** LOW (P3) — affects type inference in callers
**Effort:** 30 minutes

**Pattern:** Add return type annotations and parameter types to functions currently
lacking them. Typical pattern in `cli.py` Typer commands.

**Verification gate:** `python -m mypy orca/ --ignore-missing-imports --disallow-untyped-defs`

---

### P5 — STYLE / NAMING (190 violations, ~30 files)

**Severity:** LOW (P3) — cosmetic; does not affect functionality or safety
**Effort:** 45 minutes

| Category | Count | Typical Fix |
|----------|-------|-------------|
| E501 Line too long | 104 | Extract help strings to constants; implicit concatenation |
| N806 Non-lowercase var | 31 | Rename `DEFAULT_*` to use `_DEFAULT_*` or add `# noqa: N806` |
| S311 Non-crypto random | 22 | Add `# noqa: S311` — acceptable for simulation/backtest |
| B904 Raise without from | 17 | **(Covered in P2 above)** |
| N803 Invalid arg name | 10 | Rename to snake_case |
| S105 Hardcoded password | 9 | Audit for false positives; suppress or extract to env |
| RUF059 Unused unpacked var | 5 | Replace with `_` |
| B905 zip without strict | 4 | Add `strict=True` |
| B007 Unused loop variable | 3 | Rename to `_` |
| S110 Bare try-except-pass | 4 | Add specific exception or logging |
| S603 Subprocess without shell | 3 | Add explicit `shell=False` |

---

## CI/CD Impact Assessment

| Issue | Blocks CI? | CI Job Affected | Immediate Action Required? |
|-------|-----------|-----------------|---------------------------|
| G1 — Shadow test build error | **YES** | `backend` (`go vet ./...`) | **Fix immediately** — blocks all Go CI |
| G2 — Weak replay test | No | `backend` (passes) | Defer to P3 |
| P1 — Pyflakes F errors | No | `python` (ruff check) | Fix in current sprint |
| P2-P5 — Style/type issues | No | `python` (warnings only) | Fix incrementally |

---

## Dependency Conflict Remediation

### NPM (Frontend)

| Concern | Status | Action |
|---------|--------|--------|
| esbuild ≤0.24.2 (moderate) | Dev-only; no production impact | Defer — requires vite 8.x which breaks React 18 |
| 7 major-version bumps available | All are breaking (React 19, vite 8, TS 7, eslint 10, router 7) | Plan React 19 migration for Q4 2026 |
| lucide-react 1.17→1.26 | Minor bump, safe | Applied via `npm update` |

### Python

No dependency conflicts detected. `ruff`, `mypy`, `pytest` are all at current versions.
Third-party ML libraries (numpy, xgboost, vectorbt, psycopg) are NOT pinned to exact
versions — consider adding `requirements.lock` or `constraints.txt` for reproducible CI.

### Go

No dependency conflicts detected. `go.sum` is clean. `golangci-lint` binary is not
installed on the Windows development machine — install via `go install` or chocolatey.

---

## Execution Order

```
Phase 1 (P0, 10 min):  G1 — Fix shadow test build error
Phase 2 (P1, 15 min):  P1 — Fix pyflakes critical errors (F821, F841, F401)
Phase 3 (P1,  5 min):  P2 — Fix raise-without-from in cli.py
Phase 4 (P2, 15 min):  P3 — Add generic type arguments (fixes 59 mypy errors)
Phase 5 (P2, 45 min):  P5 — Fix E501 line-too-long (104 violations)
Phase 6 (P3, 30 min):  P4 — Add missing type annotations
Phase 7 (P3,  1 hr):   G2 — Strengthen engine replay test
```

Each phase is independently verifiable via `go vet`, `ruff check`, or `mypy`.
No phase has inter-phase dependencies except tool availability (ruff, mypy must be
present in the Python environment).

---

## Pre-Commit Gate Additions (Recommended)

To prevent regression, add these to `.github/workflows/ci.yml` or `scripts/pre-commit.sh`:

```yaml
# In the python job:
- name: Critical ruff checks (must pass)
  run: python -m ruff check orca/ --select F --output-format concise

# In the backend job:
- name: Build all Go packages (including tests/)
  run: go build ./...  # catches build errors in tests/shadow/
```

The current `go vet ./...` in CI already catches G1. Adding `go build ./...` provides
defense-in-depth for all non-`internal/` Go packages.
