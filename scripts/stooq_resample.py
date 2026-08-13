#!/usr/bin/env python3
"""Phase 3: Stooq Resampling — 1H→4H and 5m→15m/30m.

Reads real stooq bars from the candles table (source='stooq'), resamples
to higher timeframes using standard OHLC aggregation, and inserts with
source='stooq-resampled'.

Timeframe mapping:
  1H → 4H:  aggregate 4 consecutive 1H bars (2-year real coverage)
  5m → 15m: aggregate 3 consecutive 5m bars (5-month real coverage)
  5m → 30m: aggregate 6 consecutive 5m bars (5-month real coverage)
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import sys
import time
from typing import Any

import psycopg2
import psycopg2.extras

# Windows consoles default to cp1252, which cannot encode the "\u2192" arrow in
# resample labels. Reconfigure to UTF-8 so logging never raises UnicodeEncodeError.
for _stream in (sys.stdout, sys.stderr):
    try:
        _stream.reconfigure(encoding="utf-8", errors="replace")
    except (AttributeError, ValueError):
        pass

PRICE_SCALE = 100_000
BATCH_SIZE = 2000

RESAMPLE_RULES = [
    {"source_tf": "1h", "target_tf": "4h", "bars_per_group": 4, "label": "1H→4H"},
    {"source_tf": "5m", "target_tf": "15m", "bars_per_group": 3, "label": "5m→15m"},
    {"source_tf": "5m", "target_tf": "30m", "bars_per_group": 6, "label": "5m→30m"},
]


def get_db_url() -> str:
    return os.environ.get("ORCA_DB_URL", "postgresql://artisan:@localhost:5432/artisan")


def build_cli() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Stooq Resampling")
    parser.add_argument("--symbols", nargs="*", help="Specific symbols (default: all with stooq data)")
    parser.add_argument("--dry-run", action="store_true")
    return parser


def compute_generation_id(symbols: list[str] | None, source_tf: str, target_tf: str) -> str:
    """Deterministic generation ID for this resample step (no wall-clock term)."""
    raw = json.dumps({
        "symbols": sorted(symbols) if symbols else None,
        "source_tf": source_tf, "target_tf": target_tf, "source": "stooq-resampled",
    }, sort_keys=True)
    return hashlib.sha256(raw.encode()).hexdigest()[:16]


def _resample_and_insert(
    conn: Any,
    source_tf: str,
    target_tf: str,
    bars_per_group: int,
    symbol_ids: list[int] | None,
    generation_id: str,
    dry_run: bool,
) -> int:
    """Read source bars, aggregate into target bars, batch INSERT."""
    cur = conn.cursor()

    query = """
        SELECT symbol_id, time, open_raw, high_raw, low_raw, close_raw, volume
        FROM candles
        WHERE source = 'stooq' AND timeframe = %s
    """
    params: list[Any] = [source_tf]
    if symbol_ids:
        query += " AND symbol_id = ANY(%s)"
        params.append(symbol_ids)

    query += " ORDER BY symbol_id, time ASC"

    cur.execute(query, params)
    total_inserted = 0
    current_sym: int | None = None
    batch: list[tuple] = []
    group_bars: list[tuple] = []

    for row in cur:
        sym_id, bar_time, o_r, h_r, l_r, c_r, vol = row
        if current_sym is not None and sym_id != current_sym:
            _flush_group(group_bars, batch, current_sym, target_tf, generation_id)
            group_bars = []

        current_sym = sym_id
        group_bars.append(row)

        if len(group_bars) >= bars_per_group:
            _flush_group(group_bars, batch, current_sym, target_tf, generation_id)
            group_bars = []

        if len(batch) >= BATCH_SIZE:
            if not dry_run:
                _insert_batch(conn, batch)
            total_inserted += len(batch)
            batch.clear()

    if group_bars:
        _flush_group(group_bars, batch, current_sym, target_tf, generation_id)
    if batch:
        if not dry_run:
            _insert_batch(conn, batch)
        total_inserted += len(batch)

    conn.commit()
    return total_inserted


def _flush_group(
    group: list[tuple], batch: list[tuple], sym_id: int, target_tf: str, generation_id: str,
) -> None:
    """Aggregate a group of source bars into one target bar and add to batch."""
    if not group:
        return

    # Row tuple: (symbol_id, time, open_raw, high_raw, low_raw, close_raw, volume)
    # Indices:        0        1       2         3         4        5         6
    opens  = [r[2] for r in group]
    highs  = [r[3] for r in group]
    lows   = [r[4] for r in group]
    closes = [r[5] for r in group]
    volumes = [r[6] for r in group]

    batch.append((
        sym_id,
        target_tf,
        group[0][1],      # bucket time = first source bar's timestamp
        opens[0],          # open = first bar's open
        max(highs),         # high = max of highs
        min(lows),          # low = min of lows
        closes[-1],         # close = last bar's close
        sum(volumes),       # volume = sum
        "stooq-resampled",
        generation_id,
    ))


def _insert_batch(conn: Any, batch: list[tuple]) -> None:
    cur = conn.cursor()
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
    """, batch, page_size=BATCH_SIZE)


def main() -> None:
    args = build_cli().parse_args()
    conn = psycopg2.connect(get_db_url())

    # Disable parallel query: the large source-bar SELECT otherwise spawns
    # parallel workers that exhaust shared-memory locks (max_locks_per_transaction).
    with conn.cursor() as cur:
        cur.execute("SET max_parallel_workers_per_gather = 0")
        cur.execute("SET max_parallel_workers = 0")

    # Get symbol IDs for requested symbols
    symbol_ids = None
    if args.symbols:
        cur = conn.cursor()
        cur.execute("SELECT id FROM symbols WHERE ticker = ANY(%s)", (args.symbols,))
        symbol_ids = [r[0] for r in cur.fetchall()]

    total = 0
    t0 = time.monotonic()
    for rule in RESAMPLE_RULES:
        label = rule["label"]
        gen_id = compute_generation_id(args.symbols, rule["source_tf"], rule["target_tf"])
        print(f"\n--- {label} [generation_id={gen_id}] ---")
        n = _resample_and_insert(
            conn,
            rule["source_tf"],
            rule["target_tf"],
            rule["bars_per_group"],
            symbol_ids,
            gen_id,
            args.dry_run,
        )
        elapsed = time.monotonic() - t0
        print(f"  {label}: {n} bars inserted ({elapsed:.1f}s)")
        total += n

    conn.close()
    print(f"\nDone: {total} total bars in {time.monotonic() - t0:.1f}s")


if __name__ == "__main__":
    main()
