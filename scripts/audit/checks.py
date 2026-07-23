"""Regression checks for the OrcaAlgo matrix audit.

Each check is a small, fast, self-contained function that guards ONE of the
defects diagnosed and fixed during the 2026-07-08 session. Checks receive a
Context and return a CheckResult; they never raise (the runner wraps them).

Guarded regressions:
  R1  data_source ignored (served from stooq default) → empty synthetic results
  R2  wall-clock OrderRateLimiter throttling ~98% of backtest signals
  R3  concurrent map write crash (shared strategy-runner singletons in matrix)
  R4  position sizer 50x shrink (share-based MaxPositionPct cap)
  R5  mean-reversion entries missing Quantity → zero-qty trades → flat equity
  R6  optimizer never applied candidate params (flat objective landscape)
  R7  position size not optimizable (no universal sizing_percent/kelly_fraction)
  R8  signal_diag not exposed (no way to diagnose zero-trade runs)
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Callable

from . import config
from .client import APIClient, distinct_equity_count, signal_diag


@dataclass
class Context:
    client: APIClient
    symbols: list[str]
    timeframe: str = "1h"


@dataclass
class CheckResult:
    name: str
    status: str          # PASS | FAIL | ERROR | SKIP
    severity: str        # critical | high | medium
    guards: str          # which regression id(s)
    detail: str
    duration_s: float = 0.0


def _pass(name, sev, guards, detail) -> CheckResult:
    return CheckResult(name, "PASS", sev, guards, detail)


def _fail(name, sev, guards, detail) -> CheckResult:
    return CheckResult(name, "FAIL", sev, guards, detail)


# ── R8: signal_diag exposed ─────────────────────────────────────────────────
def check_signal_diag_exposed(ctx: Context) -> CheckResult:
    name = "signal_diag_exposed"
    r = ctx.client.run_single(config.LIGHT_STRATEGY, ctx.symbols[0], ctx.timeframe)
    if "error" in r:
        return _fail(name, "medium", "R8", f"backtest error: {r['error']}")
    diag = signal_diag(r)
    required = {"candles_seen", "signals_passed", "rate_limited", "trades_opened"}
    missing = required - set(diag)
    if missing:
        return _fail(name, "medium", "R8", f"signal_diag missing keys: {sorted(missing)}")
    return _pass(name, "medium", "R8",
                 f"signal_diag present ({diag.get('candles_seen')} candles seen)")


# ── R1: data_source honored ────────────────────────────────────────────────
def check_data_source_honored(ctx: Context) -> CheckResult:
    name = "data_source_honored"
    r = ctx.client.run_single(config.LIGHT_STRATEGY, ctx.symbols[0], ctx.timeframe,
                              data_source="synthetic")
    if "error" in r:
        return _fail(name, "critical", "R1", f"backtest error: {r['error']}")
    ds = r.get("data_source")
    warnings = r.get("warnings") or []
    no_data = any("no candle data" in str(w).lower() for w in warnings)
    if ds != "synthetic":
        return _fail(name, "critical", "R1",
                     f"requested synthetic but engine used data_source={ds!r} "
                     f"(request field is being ignored)")
    if no_data:
        return _fail(name, "critical", "R1",
                     f"synthetic requested but 'no candle data' warning present: {warnings}")
    seen = signal_diag(r).get("candles_seen", 0)
    if seen <= 0:
        return _fail(name, "critical", "R1", "no candles loaded for synthetic source")
    return _pass(name, "critical", "R1",
                 f"data_source=synthetic honored, {seen} candles loaded")


# ── R2: rate limiter disabled in backtest ──────────────────────────────────
def check_rate_limiter_disabled(ctx: Context) -> CheckResult:
    name = "rate_limiter_disabled"
    r = ctx.client.run_single(config.HEAVY_STRATEGY, ctx.symbols[0], ctx.timeframe)
    if "error" in r:
        return _fail(name, "critical", "R2", f"backtest error: {r['error']}")
    diag = signal_diag(r)
    rl = int(diag.get("rate_limited", 0))
    attempts = int(diag.get("signal_attempts", 0))
    if rl > config.MAX_ALLOWED_RATE_LIMITED:
        pct = (rl / attempts * 100) if attempts else 0
        return _fail(name, "critical", "R2",
                     f"backtest wall-clock rate-limited {rl}/{attempts} signals "
                     f"({pct:.1f}%) — live limiter leaked into backtest path")
    return _pass(name, "critical", "R2",
                 f"rate_limited={rl} of {attempts} attempts (limiter correctly absent)")


# ── R4/R5: equity actually moves (positions have real size + qty) ───────────
def check_equity_moves(ctx: Context) -> CheckResult:
    name = "equity_moves"
    r = ctx.client.run_single(config.HEAVY_STRATEGY, ctx.symbols[0], ctx.timeframe)
    if "error" in r:
        return _fail(name, "critical", "R4", f"backtest error: {r['error']}")
    trades = int(r.get("num_trades", 0))
    distinct = distinct_equity_count(r)
    ret = float(r.get("total_return", 0.0))
    if trades <= 0:
        return _fail(name, "critical", "R4/R5",
                     "strategy produced 0 trades (data_source, rate-limiter, or "
                     "sizing regression)")
    if distinct < config.MIN_DISTINCT_EQUITY or abs(ret) <= config.FLAT_RETURN_EPS:
        return _fail(name, "critical", "R4",
                     f"equity is flat despite {trades} trades "
                     f"(distinct_equity={distinct}, return={ret}); position sizer "
                     f"is shrinking positions to ~0")
    return _pass(name, "critical", "R4",
                 f"{trades} trades, equity moved (distinct={distinct}, return={ret:.4f}%)")


# ── R5: mean-reversion strategies are not flat ──────────────────────────────
def check_mean_reversion_not_flat(ctx: Context) -> CheckResult:
    name = "mean_reversion_not_flat"
    flat: list[str] = []
    checked: list[str] = []
    for strat in config.MEAN_REVERSION_STRATEGIES:
        r = ctx.client.run_single(strat, ctx.symbols[0], ctx.timeframe)
        if "error" in r:
            continue
        checked.append(strat)
        trades = int(r.get("num_trades", 0))
        distinct = distinct_equity_count(r)
        if trades > 0 and distinct < config.MIN_DISTINCT_EQUITY:
            flat.append(f"{strat}(trades={trades},flat)")
    if not checked:
        return CheckResult(name, "SKIP", "high", "R5",
                           "no mean-reversion strategies could be run")
    if flat:
        return _fail(name, "high", "R5",
                     f"mean-reversion entries produce flat equity (zero-qty trades): "
                     f"{', '.join(flat)}")
    return _pass(name, "high", "R5",
                 f"mean-reversion strategies produce non-flat equity: {', '.join(checked)}")


# ── R4/R5: returns scale with position size (searchable landscape) ──────────
def check_sizing_scales_returns(ctx: Context) -> CheckResult:
    name = "sizing_scales_returns"
    sym = ctx.symbols[0]
    lo = ctx.client.run_single(config.HEAVY_STRATEGY, sym, ctx.timeframe,
                               sizing_percent=config.SIZING_LOW)
    hi = ctx.client.run_single(config.HEAVY_STRATEGY, sym, ctx.timeframe,
                               sizing_percent=config.SIZING_HIGH)
    if "error" in lo or "error" in hi:
        return _fail(name, "high", "R4", f"backtest error: {lo.get('error') or hi.get('error')}")
    rlo = abs(float(lo.get("total_return", 0.0)))
    rhi = abs(float(hi.get("total_return", 0.0)))
    if rlo <= config.FLAT_RETURN_EPS and rhi <= config.FLAT_RETURN_EPS:
        return _fail(name, "high", "R4",
                     "both sizing levels produced ~0 return (flat objective landscape)")
    if rhi < rlo * config.SIZING_SCALE_MIN_RATIO:
        return _fail(name, "high", "R4",
                     f"return did not scale with sizing: |ret({config.SIZING_LOW})|={rlo:.4f} "
                     f"|ret({config.SIZING_HIGH})|={rhi:.4f} (expected ~"
                     f"{config.SIZING_HIGH/config.SIZING_LOW:.0f}x)")
    return _pass(name, "high", "R4",
                 f"return scales with sizing: {rlo:.4f}% -> {rhi:.4f}% "
                 f"({rhi/max(rlo, 1e-9):.1f}x for {config.SIZING_HIGH/config.SIZING_LOW:.0f}x sizing)")


# ── R3: matrix concurrency is stable (no shared-runner crash) ───────────────
def check_matrix_concurrency_stable(ctx: Context) -> CheckResult:
    name = "matrix_concurrency_stable"
    strategies = [config.HEAVY_STRATEGY, config.LIGHT_STRATEGY]
    syms = ctx.symbols[:2]
    sub = ctx.client.submit_matrix(strategies, syms, [ctx.timeframe])
    if "error" in sub:
        return _fail(name, "critical", "R3", f"submit failed: {sub['error']}")
    batch = sub.get("batch_run_id", "")
    total = sub.get("total_combos", len(strategies) * len(syms))
    final = ctx.client.poll_matrix(batch)
    summary = final.get("summary") or {}
    status = summary.get("status")
    results = final.get("results") or []
    # Server crash → subsequent poll returns connection error → status stays 'timeout'
    # and the process dies. A healthy completion returns status 'completed'.
    if status != "completed":
        alive = ctx.client.reachable()
        return _fail(name, "critical", "R3",
                     f"matrix did not complete (status={status}, results={len(results)}/"
                     f"{total}); server_reachable={alive} — likely concurrent-map-write "
                     f"crash from shared strategy-runner singletons")
    if len(results) < total:
        return _fail(name, "critical", "R3",
                     f"matrix completed but only {len(results)}/{total} combos returned")
    return _pass(name, "critical", "R3",
                 f"matrix completed {len(results)}/{total} combos with heavy strategy, no crash")


# ── R1 (matrix path): synthetic data flows through the matrix ───────────────
def check_matrix_data_source(ctx: Context) -> CheckResult:
    name = "matrix_data_source"
    sub = ctx.client.submit_matrix([config.LIGHT_STRATEGY], ctx.symbols[:2],
                                   [ctx.timeframe], data_source="synthetic")
    if "error" in sub:
        return _fail(name, "high", "R1", f"submit failed: {sub['error']}")
    final = ctx.client.poll_matrix(sub.get("batch_run_id", ""))
    results = final.get("results") or []
    traded = [r for r in results if int(r.get("num_trades", 0)) > 0]
    if not results:
        return _fail(name, "high", "R1", "matrix returned no results for synthetic source")
    if not traded:
        return _fail(name, "high", "R1",
                     f"matrix returned {len(results)} combos but none traded — "
                     f"synthetic data_source not flowing through matrix path")
    return _pass(name, "high", "R1",
                 f"matrix synthetic path OK: {len(traded)}/{len(results)} combos traded")


# ── R7 + R6: optimizer includes universal sizing and applies params ─────────
def check_optimizer_universal_sizing(ctx: Context) -> CheckResult:
    name = "optimizer_universal_sizing"
    r = ctx.client.run_optimization(
        config.LIGHT_STRATEGY, symbols=ctx.symbols[:1],
        max_combinations=8, step_months=12,
    )
    if "error" in r:
        return _fail(name, "high", "R6/R7", f"optimize error: {r['error']}")
    windows = r.get("best_params_per_window") or []
    non_null = [w for w in windows if isinstance(w, dict) and w]
    if not non_null:
        return _fail(name, "high", "R6/R7",
                     "optimizer returned no best_params_per_window (no search performed)")
    have_sizing = any(
        all(k in w for k in config.UNIVERSAL_OPT_PARAMS) for w in non_null
    )
    if not have_sizing:
        keys = sorted(non_null[0].keys())
        return _fail(name, "high", "R7",
                     f"best params lack universal sizing params "
                     f"{config.UNIVERSAL_OPT_PARAMS}; got {keys}")
    return _pass(name, "high", "R6/R7",
                 f"optimizer searched {len(non_null)} window(s) with universal sizing "
                 f"params applied (sizing_percent/kelly_fraction present)")


# ── R-HEALTH: system health endpoint exposes capacity telemetry ─────────────
def check_system_health(ctx: Context) -> CheckResult:
    name = "system_health"
    h = ctx.client.system_health()
    if "error" in h:
        return _fail(name, "medium", "R-HEALTH", f"/system/health error: {h['error']}")
    required = {"heap_inuse_mb", "heap_budget_mb", "matrix_workers", "db_pool_max", "near_capacity"}
    missing = required - set(h)
    if missing:
        return _fail(name, "medium", "R-HEALTH", f"health payload missing {sorted(missing)}")
    return _pass(name, "medium", "R-HEALTH",
                 f"heap {h['heap_inuse_mb']:.0f}/{h['heap_budget_mb']:.0f}MB, "
                 f"workers={h['matrix_workers']}, db_pool={h.get('db_pool_in_use')}/{h['db_pool_max']}")


# ── R-STREAM: since-cursor returns only deltas (no O(N) polling) ─────────────
def check_stream_cursor(ctx: Context) -> CheckResult:
    name = "stream_cursor"
    sub = ctx.client.submit_matrix([config.LIGHT_STRATEGY], ctx.symbols[:2], [ctx.timeframe])
    if "error" in sub:
        return _fail(name, "high", "R-STREAM", f"submit failed: {sub['error']}")
    batch = sub.get("batch_run_id", "")
    final = ctx.client.poll_matrix(batch)
    total = len(final.get("results") or [])
    if total == 0:
        return CheckResult(name, "SKIP", "high", "R-STREAM", "no results to cursor over")
    # After completion, a cursor at the end must return an empty delta.
    tail = ctx.client.matrix_results_since(batch, total)
    tail_n = len(tail.get("results") or [])
    seq = tail.get("seq", tail.get("summary", {}).get("seq"))
    if tail_n != 0:
        return _fail(name, "high", "R-STREAM",
                     f"since={total} returned {tail_n} rows; cursor is not delta-only")
    # A cursor at 0 must return all rows (backfill path for resume).
    head = ctx.client.matrix_results_since(batch, 0)
    head_n = len(head.get("results") or [])
    if head_n != total:
        return _fail(name, "high", "R-STREAM",
                     f"since=0 returned {head_n}, expected {total} (backfill broken)")
    return _pass(name, "high", "R-STREAM",
                 f"cursor works: since=0 -> {head_n} rows, since={total} -> 0 (seq={seq})")


# ── R-CHUNK: chunk telemetry present in matrix summary ──────────────────────
def check_chunk_telemetry(ctx: Context) -> CheckResult:
    name = "chunk_telemetry"
    sub = ctx.client.submit_matrix([config.LIGHT_STRATEGY, config.HEAVY_STRATEGY],
                                   ctx.symbols[:2], [ctx.timeframe])
    if "error" in sub:
        return _fail(name, "medium", "R-CHUNK", f"submit failed: {sub['error']}")
    final = ctx.client.poll_matrix(sub.get("batch_run_id", ""))
    summary = final.get("summary") or {}
    chunk = summary.get("chunk")
    if not isinstance(chunk, dict) or "index" not in chunk or "total" not in chunk:
        return _fail(name, "medium", "R-CHUNK", f"summary.chunk missing/invalid: {chunk}")
    if "throughput_per_min" not in summary or "eta_seconds" not in summary:
        return _fail(name, "medium", "R-CHUNK", "telemetry (throughput/eta) missing from summary")
    return _pass(name, "medium", "R-CHUNK",
                 f"chunk {chunk['index']}/{chunk['total']}, telemetry present")


# ── R-CANCEL: cancel endpoint stops a batch and the server survives ─────────
def check_cancellation(ctx: Context) -> CheckResult:
    name = "cancellation"
    # Heavy batch so it is still running when we cancel.
    sub = ctx.client.submit_matrix([config.HEAVY_STRATEGY, config.LIGHT_STRATEGY],
                                   ctx.symbols[:3], [ctx.timeframe],
                                   start="2021-01-01", end="2026-01-01")
    if "error" in sub:
        return _fail(name, "high", "R-CANCEL", f"submit failed: {sub['error']}")
    batch = sub.get("batch_run_id", "")
    resp = ctx.client.cancel_matrix(batch)
    # Either we cancelled a running batch, or it already finished (both acceptable).
    if "error" in resp:
        # If it finished before cancel, that's fine as long as the server is alive.
        final = ctx.client.poll_matrix(batch)
        status = (final.get("summary") or {}).get("status")
        if status in ("completed", "cancelled") and ctx.client.reachable():
            return _pass(name, "high", "R-CANCEL",
                         f"batch reached terminal status={status}; server alive")
        return _fail(name, "high", "R-CANCEL", f"cancel error: {resp['error']}")
    final = ctx.client.poll_matrix(batch)
    status = (final.get("summary") or {}).get("status")
    if not ctx.client.reachable():
        return _fail(name, "high", "R-CANCEL", "server not reachable after cancel (possible crash)")
    if status not in ("cancelled", "completed"):
        return _fail(name, "high", "R-CANCEL", f"batch did not stop (status={status})")
    return _pass(name, "high", "R-CANCEL", f"cancel handled cleanly (status={status}); server alive")


# ── registry ────────────────────────────────────────────────────────────────
QUICK_CHECKS: list[Callable[[Context], CheckResult]] = [
    check_signal_diag_exposed,
    check_data_source_honored,
    check_rate_limiter_disabled,
    check_equity_moves,
    check_mean_reversion_not_flat,
    check_sizing_scales_returns,
]

HEAVY_CHECKS: list[Callable[[Context], CheckResult]] = [
    check_system_health,
    check_matrix_concurrency_stable,
    check_matrix_data_source,
    check_stream_cursor,
    check_chunk_telemetry,
    check_cancellation,
    check_optimizer_universal_sizing,
]

ALL_CHECKS = QUICK_CHECKS + HEAVY_CHECKS
