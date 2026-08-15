#!/usr/bin/env python3
"""Phase 4: Stooq-Calibrated Synthetic Gap-Fill.

Replaces all old synthetic intraday bars (source='yahoo', non-1d timeframes)
with new synthetic bars calibrated from real stooq data. Per-symbol σ and μ
are extracted from stooq 1H and 5m returns (EWMA λ=0.94), replacing the
fixed 0.015 multiplier that caused long/short asymmetry.

Gaps filled:
  - 1H:   2021-07-01 → 2024-07-23  (~3 years, calib from stooq 1H returns)
  - 4H:   2021-07-01 → 2024-07-23  (from filled 1H resample)
  - 5m:   2021-07-01 → 2026-03-14  (~4.7 years, calib from stooq 5m returns)
  - 15m:  2021-07-01 → 2026-03-14  (from filled 5m resample)
  - 30m:  2021-07-01 → 2026-03-14  (from filled 5m resample)
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import sys
import time
from collections import defaultdict
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any

import numpy as np
import psycopg2
import psycopg2.extras

# Windows consoles default to cp1252, which cannot encode the σ/μ characters in
# calibration logs. Reconfigure to UTF-8 so logging never raises UnicodeEncodeError.
for _stream in (sys.stdout, sys.stderr):
    try:
        _stream.reconfigure(encoding="utf-8", errors="replace")
    except (AttributeError, ValueError):
        pass

PRICE_SCALE = 100_000
BATCH_SIZE = 2000
N_MINUTES = 390
EWMA_LAMBDA = 0.94
PROJECT_ROOT = Path(__file__).resolve().parent.parent


def _generate_intraday_path(
    rng: np.random.Generator,
    open_price: float,
    close_price: float,
    sigma_bar: float,
    mu_bar: float = 0.0,
    steps_per_bar: int = 60,
    n_steps: int = 390,
) -> np.ndarray:
    """Generate an unconstrained intraday price path.

    Uses Geometric Brownian Motion with per-symbol σ and μ calibrated from
    real stooq returns, then conditions the path to end at the actual daily
    Close via a Brownian bridge. The path is free to break through daily
    High/Low — no artificial clipping.

    The bridge (B2 fix) rescales the WHOLE path with a constant per-step
    log-drift so it ends at Close, preserving the random shape (breakouts stay
    breakouts). This replaces the previous soft-close blend, whose last-half
    linear pull toward Close injected a deterministic mean-reversion pattern
    that favored mean-reversion and suppressed breakouts.

    Args:
        rng: NumPy random generator (seeded for reproducibility).
        open_price: Daily Open price.
        close_price: Daily Close price (bridge endpoint).
        sigma_bar: Per-bar volatility from stooq calibration (e.g. per-1h σ).
        mu_bar: Per-bar drift from stooq calibration (e.g. per-1h mean return).
        steps_per_bar: Number of 1-minute steps per target bar (60 for 1h, 5 for 5m).
        n_steps: Number of intraday steps (default 390 for 1-minute).

    Returns:
        Array of n_steps prices representing the intraday path.
    """
    # Per-step (1-minute) vol and drift so that a bar of `steps_per_bar`
    # minutes has exactly sigma_bar volatility and ~mu_bar drift. This fixes
    # the previous units mismatch (B11) where per-bar sigma was applied as if
    # it were daily sigma (understating intraday vol ~2.5x for 1h, ~8.8x for
    # 5m), and adds the calibrated drift (B1) so trend-following sees trend.
    sigma_step = sigma_bar / np.sqrt(steps_per_bar)
    mu_step = mu_bar / steps_per_bar
    shocks = rng.normal(0, 1, n_steps)
    log_returns = (mu_step - 0.5 * sigma_step**2) + sigma_step * shocks

    # Pure GBM — unconstrained, free to break through any level
    path = open_price * np.exp(np.cumsum(log_returns))

    # Brownian-bridge conditioning (B2): rescale the whole path so it ends at
    # Close, preserving the random shape with a constant per-step log-drift
    # (no last-half mean-reversion pull).
    if close_price > 0 and path[-1] > 0:
        t = np.linspace(0.0, 1.0, n_steps)
        path = path * np.power(close_price / path[-1], t)

    # Floor at near-zero to prevent negative prices in extreme σ scenarios
    path = np.maximum(path, open_price * 0.01)

    return path
SYNTHETIC_START = datetime(2021, 7, 1, tzinfo=timezone.utc)

# Boundary dates where stooq real data begins for each calibration tier
STOOQ_1H_START = datetime(2024, 7, 24, tzinfo=timezone.utc)
STOOQ_5M_START = datetime(2026, 3, 16, tzinfo=timezone.utc)

TARGETS_TIER1 = [("1h", 7, 60)]     # Tier 1 (calibrated from 1H returns): fills to STOOQ_1H_START
TARGETS_TIER2 = [("5m", 78, 5),      # Tier 2 (calibrated from 5m returns): fills to STOOQ_5M_START
                  ("15m", 26, 15), ("30m", 13, 30)]


def get_db_url() -> str:
    return os.environ.get("ORCA_DB_URL", "postgresql://artisan:@localhost:5432/artisan")


def compute_generation_id(symbols: list[str] | None) -> str:
    """Deterministic generation ID for this synthetic gap-fill (no wall-clock term)."""
    raw = json.dumps({
        "symbols": sorted(symbols) if symbols else None,
        "source": "stooq-calibrated",
        "algo": "gbm-bridge-v2",  # B1 drift + B11 sigma-units + B2 Brownian bridge
    }, sort_keys=True)
    return hashlib.sha256(raw.encode()).hexdigest()[:16]


def _insert_rows(cur: Any, rows: list[tuple]) -> None:
    """Batch-insert synthetic bars with provenance, idempotently (DO UPDATE)."""
    if not rows:
        return
    psycopg2.extras.execute_values(cur, """
        INSERT INTO candles
            (symbol_id, timeframe, time, open_raw, high_raw, low_raw, close_raw, volume, source, generation_id)
        VALUES %s
        ON CONFLICT (symbol_id, timeframe, time, source) DO UPDATE SET
            open_raw = EXCLUDED.open_raw,
            high_raw = EXCLUDED.high_raw,
            low_raw = EXCLUDED.low_raw,
            close_raw = EXCLUDED.close_raw,
            volume = EXCLUDED.volume,
            generation_id = EXCLUDED.generation_id
    """, rows, page_size=BATCH_SIZE)


def build_cli() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description="Stooq-Calibrated Synthetic Gap-Fill")
    p.add_argument("--symbols", nargs="*", help="Specific symbols (default: all)")
    p.add_argument("--dry-run", action="store_true")
    return p


def _compute_ewma_vol(returns: np.ndarray, lam: float = EWMA_LAMBDA) -> tuple[float, float, float]:
    """Compute EWMA volatility, mean, and annualized vol from return series."""
    if len(returns) < 5:
        return 0.01, 0.0, 0.1
    mu = float(np.mean(returns))
    residual = returns - mu
    variance = float(np.mean(residual ** 2))
    # EWMA decay
    w = lam ** np.arange(len(residual) - 1, -1, -1)
    ewma_var = float(np.average(residual ** 2, weights=w) if len(w) > 0 else variance)
    sigma = float(np.sqrt(max(ewma_var, 1e-12)))
    ann_sigma = sigma * np.sqrt(252 * 7 if len(returns) > 100 else 252)
    return sigma, mu, ann_sigma


def calibrate_all_symbols(conn: Any) -> dict[int, dict[str, Any]]:
    """Compute per-symbol σ and μ from real stooq bars.

    Uses two complementary volatility sources:
      1. Close-to-Close returns (inter-bar volatility)
      2. High-Low range / Close (intra-bar range, scaled to daily equivalent)

    The final sigma is the maximum of both — this ensures the synthetic path
    has realistic bar-level spread, not just inter-bar drift.
    """
    cur = conn.cursor()
    calib: dict[int, dict[str, Any]] = {}

    # Tier 1: 1H — Close-to-Close returns + H-L range
    cur.execute("""
        SELECT symbol_id, open_raw, high_raw, low_raw, close_raw
        FROM candles
        WHERE source = 'stooq' AND timeframe = '1h'
        ORDER BY symbol_id, time ASC
    """)
    sym_ret_cc_1h: dict[int, list[float]] = defaultdict(list)
    sym_ret_hl_1h: dict[int, list[float]] = defaultdict(list)
    prev_close: dict[int, float] = {}
    for sym_id, o_r, h_r, l_r, c_r in cur:
        # Close-to-Close return (inter-bar drift)
        if sym_id in prev_close and prev_close[sym_id] > 0:
            sym_ret_cc_1h[sym_id].append(float(c_r - prev_close[sym_id]) / float(prev_close[sym_id]))
        if c_r > 0:
            prev_close[sym_id] = float(c_r)
            # H-L range / Close (intra-bar spread)
            sym_ret_hl_1h[sym_id].append(float(h_r - l_r) / float(c_r))

    # Tier 2: 5m — same calculation
    cur.execute("""
        SELECT symbol_id, open_raw, high_raw, low_raw, close_raw
        FROM candles
        WHERE source = 'stooq' AND timeframe = '5m'
        ORDER BY symbol_id, time ASC
    """)
    sym_ret_cc_5m: dict[int, list[float]] = defaultdict(list)
    sym_ret_hl_5m: dict[int, list[float]] = defaultdict(list)
    prev_close = {}
    for sym_id, o_r, h_r, l_r, c_r in cur:
        if sym_id in prev_close and prev_close[sym_id] > 0:
            sym_ret_cc_5m[sym_id].append(float(c_r - prev_close[sym_id]) / float(prev_close[sym_id]))
        if c_r > 0:
            prev_close[sym_id] = float(c_r)
            sym_ret_hl_5m[sym_id].append(float(h_r - l_r) / float(c_r))

    cur.execute("SELECT id, ticker FROM symbols")
    ticker_map = {r[0]: r[1] for r in cur.fetchall()}

    for sym_id in set(list(sym_ret_cc_1h.keys()) + list(sym_ret_cc_5m.keys())):
        # Close-to-Close vol + drift
        sigma_cc_1h, mu_1h, _ = _compute_ewma_vol(np.array(sym_ret_cc_1h.get(sym_id, [])))
        sigma_cc_5m, mu_5m, _ = _compute_ewma_vol(np.array(sym_ret_cc_5m.get(sym_id, [])))

        # H-L range vol (converted to daily equivalent)
        hl_1h_arr = np.array(sym_ret_hl_1h.get(sym_id, []))
        hl_5m_arr = np.array(sym_ret_hl_5m.get(sym_id, []))
        sigma_hl_1h = float(np.mean(hl_1h_arr)) if len(hl_1h_arr) > 0 else 0.0
        sigma_hl_5m = float(np.mean(hl_5m_arr)) if len(hl_5m_arr) > 0 else 0.0

        # Use the maximum of CC-vol and HL-vol for realistic bar spread
        sigma_1h = max(sigma_cc_1h, sigma_hl_1h * 2.0)
        sigma_5m = max(sigma_cc_5m, sigma_hl_5m * 3.0)

        ticker = ticker_map.get(sym_id, f"id_{sym_id}")
        calib[sym_id] = {
            "ticker": ticker,
            "sigma_1h": sigma_1h, "sigma_5m": sigma_5m,
            "mu_1h": mu_1h, "mu_5m": mu_5m,
            "n_1h": len(sym_ret_cc_1h.get(sym_id, [])),
            "n_5m": len(sym_ret_cc_5m.get(sym_id, [])),
        }
        print(f"  {ticker:<8} σ_1h={sigma_1h:.4f} σ_5m={sigma_5m:.4f} μ_1h={mu_1h:.6f} μ_5m={mu_5m:.6f}  "
              f"n_1h={calib[sym_id]['n_1h']} n_5m={calib[sym_id]['n_5m']}")

    return calib


def delete_old_synthetic(conn: Any, dry_run: bool) -> int:
    """Remove all legacy synthetic and previous stooq-calibrated intraday bars.

    Only bars with source='stooq' and source='stooq-resampled' are preserved
    (real and resampled-from-real data). Everything else in intraday timeframes
    is purged so the fixed generator can replace it cleanly.
    """
    if dry_run:
        cur = conn.cursor()
        cur.execute("""
            SELECT COUNT(1) FROM candles
            WHERE timeframe IN ('5m','15m','30m','1h','4h')
              AND source NOT IN ('stooq', 'stooq-resampled')
        """)
        return cur.fetchone()[0]

    cur = conn.cursor()
    cur.execute("""
        DELETE FROM candles
        WHERE timeframe IN ('5m','15m','30m','1h','4h')
          AND source NOT IN ('stooq', 'stooq-resampled')
    """)
    deleted = cur.rowcount
    conn.commit()
    return deleted


def generate_for_symbol(
    conn: Any, sym_id: int, calib: dict[str, Any],
    tier1_end: datetime, tier2_end: datetime, generation_id: str, dry_run: bool,
) -> dict[str, int]:
    """Generate synthetic intraday bars for one symbol's gap periods."""
    cur = conn.cursor()
    # Per-symbol deterministic seed (B3): a shared seed 42 made every symbol's
    # synthetic shock sequence identical (scaled only by per-symbol sigma),
    # injecting spurious cross-symbol correlation into pairs/cross-sectional
    # strategies. Deriving the seed from sym_id + generation_id keeps runs
    # reproducible while de-correlating symbols.
    rng = np.random.default_rng(
        int(hashlib.sha256(f"{sym_id}:{generation_id}".encode()).hexdigest()[:8], 16)
    )
    sigma_h = calib["sigma_1h"]
    sigma_m = calib["sigma_5m"]
    mu_h = calib["mu_1h"]
    mu_m = calib["mu_5m"]
    stats: dict[str, int] = {}

    # Read 1d bars from start to tier2_end for this symbol (stooq daily when
    # available, falling back to yahoo 1d for the historical window).
    cur.execute("""
        SELECT time, open_raw, high_raw, low_raw, close_raw, volume
        FROM candles
        WHERE symbol_id = %s AND timeframe = '1d'
          AND source IN ('stooq', 'yahoo')
          AND time >= %s AND time < %s
        ORDER BY time ASC
    """, (sym_id, SYNTHETIC_START, tier2_end))
    daily_bars = cur.fetchall()

    if not daily_bars:
        return stats

    # Deduplicate by calendar date. A prior stooq_seed run left duplicate 1d
    # bars per date (two rows with different timestamps for the same session),
    # which would otherwise generate duplicate intraday timestamps and violate
    # the (symbol_id, timeframe, time, source) unique index on candles.
    seen_dates: set[Any] = set()
    deduped: list[tuple] = []
    for bar in daily_bars:
        d = bar[0].date()
        if d in seen_dates:
            continue
        seen_dates.add(d)
        deduped.append(bar)
    daily_bars = deduped

    # Tier 1 (1H): calibrate from 1H returns, fill to tier1_end
    all_rows_1h = []
    for bar_time, o_r, h_r, l_r, c_r, vol in daily_bars:
        if bar_time >= tier1_end:
            continue
        o = o_r / PRICE_SCALE
        c = c_r / PRICE_SCALE

        path = _generate_intraday_path(rng, o, c, sigma_h, mu_h, 60)

        ts_base = bar_time.replace(hour=13, minute=30, second=0, microsecond=0)
        for i in range(7):
            si = i * 60; ei = si + 60; seg = path[si:ei]
            if len(seg) == 0: continue
            all_rows_1h.append((sym_id, "1h", ts_base + timedelta(minutes=si),
                int(round(seg[0]*PRICE_SCALE)), int(round(seg.max()*PRICE_SCALE)),
                int(round(seg.min()*PRICE_SCALE)), int(round(seg[-1]*PRICE_SCALE)),
                int(round(float(vol or 0) / 7)), "stooq-calibrated", generation_id))

    if all_rows_1h and not dry_run:
        _insert_rows(cur, all_rows_1h)
        conn.commit()
    stats["1h"] = len(all_rows_1h)

    # Tier 2 (5m/15m/30m): calibrate from 5m returns, fill to tier2_end
    for tf_name, bpd, mpb in TARGETS_TIER2:
        all_rows = []
        for bar_time, o_r, h_r, l_r, c_r, vol in daily_bars:
            if bar_time >= tier2_end:
                continue
            o = o_r / PRICE_SCALE
            c = c_r / PRICE_SCALE

            path = _generate_intraday_path(rng, o, c, sigma_m, mu_m, 5)

            ts_base = bar_time.replace(hour=13, minute=30, second=0, microsecond=0)
            for i in range(bpd):
                si = i * mpb; ei = si + mpb; seg = path[si:ei]
                if len(seg) == 0: continue
                all_rows.append((sym_id, tf_name, ts_base + timedelta(minutes=si),
                    int(round(seg[0]*PRICE_SCALE)), int(round(seg.max()*PRICE_SCALE)),
                    int(round(seg.min()*PRICE_SCALE)), int(round(seg[-1]*PRICE_SCALE)),
                    int(round(float(vol or 0) / bpd)), "stooq-calibrated", generation_id))

        if all_rows and not dry_run:
            _insert_rows(cur, all_rows)
            conn.commit()
        stats[tf_name] = len(all_rows)

    # Tier 1 (4H): resample from just-generated 1H bars for the gap period
    if stats.get("1h", 0) > 0 and not dry_run:
        cur.execute("""
            SELECT symbol_id, time, open_raw, high_raw, low_raw, close_raw, volume
            FROM candles WHERE symbol_id = %s AND timeframe = '1h'
            AND source = 'stooq-calibrated' ORDER BY time ASC
        """, (sym_id,))
        bars_1h = cur.fetchall()
        all_rows_4h = []
        for i in range(0, len(bars_1h) - 3, 4):
            group = bars_1h[i:i+4]
            sym_id_g = group[0][0]
            all_rows_4h.append((sym_id_g, "4h", group[0][1],
                group[0][2], max(r[3] for r in group), min(r[4] for r in group),
                group[-1][5], sum(r[6] for r in group), "stooq-calibrated", generation_id))
        if all_rows_4h and not dry_run:
            _insert_rows(cur, all_rows_4h)
            conn.commit()
        stats["4h"] = len(all_rows_4h)

    return stats


def main() -> None:
    args = build_cli().parse_args()
    conn = psycopg2.connect(get_db_url())

    # Disable parallel query: the calibration SELECT otherwise spawns parallel
    # workers that exhaust shared-memory locks (max_locks_per_transaction).
    with conn.cursor() as cur:
        cur.execute("SET max_parallel_workers_per_gather = 0")
        cur.execute("SET max_parallel_workers = 0")
    t0 = time.monotonic()

    print("Calibrating per-symbol σ/μ from real stooq data...")
    calib = calibrate_all_symbols(conn)

    print("\nDeleting old synthetic intraday bars...")
    deleted = delete_old_synthetic(conn, args.dry_run)
    print(f"  {deleted} old bars marked for deletion" + (" (dry-run)" if args.dry_run else ""))

    if args.symbols:
        cur = conn.cursor()
        cur.execute("SELECT id FROM symbols WHERE ticker = ANY(%s)", (args.symbols,))
        symbol_ids = [r[0] for r in cur.fetchall()]
        calib = {k: v for k, v in calib.items() if k in symbol_ids}

    print(f"\nGenerating synthetic bars for {len(calib)} symbols...")
    print(f"  Tier 1 (1H/4H): fills to {STOOQ_1H_START.date()} (calib from stooq 1H returns)")
    print(f"  Tier 2 (5m/15m/30m): fills to {STOOQ_5M_START.date()} (calib from stooq 5m returns)")

    gen_id = compute_generation_id(args.symbols)
    print(f"  generation_id={gen_id}")

    grand_total: dict[str, int] = defaultdict(int)
    for sym_id, params in sorted(calib.items(), key=lambda x: x[1]["ticker"]):
        stats = generate_for_symbol(conn, sym_id, params, STOOQ_1H_START, STOOQ_5M_START, gen_id, args.dry_run)
        for k, v in stats.items():
            grand_total[k] += v
        elapsed = time.monotonic() - t0
        print(f"  {params['ticker']:<8} "
              + " ".join(f"{k}={v}" for k, v in sorted(stats.items())) + f"  ({elapsed:.0f}s)")

    conn.close()
    print(f"\nDone: {sum(grand_total.values())} synthetic bars in {time.monotonic() - t0:.0f}s")
    for k, v in sorted(grand_total.items()):
        print(f"  {k}: {v}")

    _write_calibration_sidecar(calib, gen_id, grand_total, args.dry_run)


def _write_calibration_sidecar(
    calib: dict[int, dict[str, Any]],
    gen_id: str,
    grand_total: dict[str, int],
    dry_run: bool,
) -> None:
    """Persist synthetic-generation calibration parameters (R13).

    Records per-symbol σ (1h/5m), EWMA decay λ, and the generation_id so the
    synthetic bars are reproducible and auditable. Written as a sidecar JSON
    rather than per-bar (would bloat 2.6M+ rows).
    """
    if dry_run:
        return

    out_path = PROJECT_ROOT / "data" / "stooq" / "synthetic_calibration.json"
    payload = {
        "generation_id": gen_id,
        "ewma_lambda": EWMA_LAMBDA,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "n_symbols": len(calib),
        "bars": {k: v for k, v in sorted(grand_total.items())},
        "symbols": [
            {
                "ticker": v["ticker"],
                "sigma_1h": v["sigma_1h"],
                "sigma_5m": v["sigma_5m"],
                "mu_1h": v["mu_1h"],
                "mu_5m": v["mu_5m"],
                "n_1h": v["n_1h"],
                "n_5m": v["n_5m"],
            }
            for v in sorted(calib.values(), key=lambda x: x["ticker"])
        ],
    }
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(json.dumps(payload, indent=2, default=str) + "\n", encoding="utf-8")
    print(f"\nCalibration sidecar written to {out_path}")


if __name__ == "__main__":
    main()
