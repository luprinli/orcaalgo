#!/usr/bin/env python3
"""Phase 2: Stooq Data Ingestion into TimescaleDB.

Streams real 5-min and hourly CSV files from the stooq tree into the
candles table. Reads manifest.json for symbol-to-file mapping. Never
loads a full CSV into memory — line-by-line streaming with 2000-row
batch inserts. Ingest hourly bars first (2-year coverage, Phase 2a)
then 5-min bars (5-month coverage, Phase 2b).

Usage:
  python scripts/stooq_seed.py                     # Both timeframes
  python scripts/stooq_seed.py --freq 1h           # Hourly only
  python scripts/stooq_seed.py --freq 5m           # 5-min only
  python scripts/stooq_seed.py --symbols SPY AAPL  # Specific symbols only
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import sys
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any
from zoneinfo import ZoneInfo

import psycopg2
import psycopg2.extras

PROJECT_ROOT = Path(__file__).resolve().parent.parent
STOOQ_ROOT = PROJECT_ROOT / "data" / "stooq"
MANIFEST_PATH = STOOQ_ROOT / "manifest.json"
PRICE_SCALE = 100_000
BATCH_SIZE = 2000

# Make `orca` importable when run standalone (e.g. `python scripts/stooq_seed.py`).
sys.path.insert(0, str(PROJECT_ROOT))
from orca.data.timezones import timezone_for_symbol  # noqa: E402


def get_db_url() -> str:
    return os.environ.get("ORCA_DB_URL", "postgresql://artisan:@localhost:5432/artisan")


def build_cli() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Stooq Data Ingestion")
    parser.add_argument("--freq", choices=["1d", "1h", "5m"], help="Frequency to ingest (default: all)")
    parser.add_argument("--symbols", nargs="*", help="Specific symbols to ingest (default: all 18)")
    parser.add_argument("--dry-run", action="store_true", help="Validate without inserting")
    return parser


def compute_generation_id(symbols: list[str], freq: str) -> str:
    """Deterministic generation ID for this ingestion run (no wall-clock term)."""
    raw = json.dumps({"symbols": sorted(symbols), "freq": freq, "source": "stooq"}, sort_keys=True)
    return hashlib.sha256(raw.encode()).hexdigest()[:16]


def ensure_symbol(conn: Any, ticker: str, info: dict[str, str]) -> int:
    """Look up symbol_id in the symbols table. Insert if not found."""
    cur = conn.cursor()
    cur.execute("SELECT id FROM symbols WHERE ticker = %s", (ticker,))
    row = cur.fetchone()
    if row:
        return int(row[0])

    cur.execute(
        "INSERT INTO symbols (ticker, exchange, asset_type, is_active) "
        "VALUES (%s, %s, %s, TRUE) RETURNING id",
        (ticker, info.get("exchange", "STOOQ"), info.get("asset_class", "unknown")),
    )
    row = cur.fetchone()
    if row is None:
        raise RuntimeError(f"Failed to insert symbol {ticker}")
    conn.commit()
    print(f"  Created symbol: {ticker} (id={row[0]}, type={info['asset_class']})")
    return int(row[0])


def _parse_stooq_line(line: str, tz: ZoneInfo) -> tuple[datetime, int, int, int, int, int] | None:
    """Parse a single stooq CSV data line into (UTC timestamp, o, h, l, c, v).

    Format: TICKER,PER,DATE,TIME,OPEN,HIGH,LOW,CLOSE,VOL,OPENINT
    Daily files carry an empty TIME field, which is normalized to 00:00:00.
    Timestamps are exchange-local and are converted to UTC using the symbol's
    exchange timezone (R14).
    """
    parts = line.strip().split(",")
    if len(parts) < 9:
        return None

    try:
        date_str = parts[2].strip()
        time_str = parts[3].strip() or "000000"
        dt = datetime.strptime(f"{date_str}{time_str}", "%Y%m%d%H%M%S")
        ts = dt.replace(tzinfo=tz).astimezone(timezone.utc)

        o_settled = int(round(float(parts[4]) * PRICE_SCALE))
        h_settled = int(round(float(parts[5]) * PRICE_SCALE))
        l_settled = int(round(float(parts[6]) * PRICE_SCALE))
        c_settled = int(round(float(parts[7]) * PRICE_SCALE))
        v_settled = int(round(float(parts[8])))
    except (ValueError, IndexError):
        return None

    return (ts, o_settled, h_settled, l_settled, c_settled, v_settled)


def ingest_file(
    conn: Any, symbol_id: int, filepath: Path, timeframe: str, source: str,
    generation_id: str, tz: ZoneInfo, dry_run: bool,
) -> int:
    """Stream a stooq CSV file and batch-INSERT into candles table."""
    cur = conn.cursor()
    batch: list[tuple] = []
    inserted = 0

    with open(filepath, "r", encoding="utf-8") as f:
        header = f.readline()
        if not header.startswith("<TICKER>"):
            print(f"  WARN: bad header in {filepath.name}")
            return 0

        for line in f:
            parsed = _parse_stooq_line(line, tz)
            if parsed is None:
                continue
            ts, o_s, h_s, l_s, c_s, v_s = parsed
            batch.append((symbol_id, timeframe, ts, o_s, h_s, l_s, c_s, v_s, source, generation_id))

            if len(batch) >= BATCH_SIZE:
                if not dry_run:
                    _insert_batch(cur, batch)
                inserted += len(batch)
                batch.clear()

    # Final batch
    if batch and not dry_run:
        _insert_batch(cur, batch)
    inserted += len(batch)

    return inserted


def _insert_batch(cur: Any, batch: list[tuple]) -> None:
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
    conn_commit(cur)


def conn_commit(cur: Any) -> None:
    cur.connection.commit()


def main() -> None:
    args = build_cli().parse_args()

    if not MANIFEST_PATH.exists():
        print(f"ERROR: {MANIFEST_PATH} not found. Run stooq_discovery.py first.")
        sys.exit(1)

    manifest = json.loads(MANIFEST_PATH.read_text(encoding="utf-8"))
    freq_filter = {args.freq} if args.freq else {"1d", "1h", "5m"}
    sym_filter = set(args.symbols) if args.symbols else None

    conn = psycopg2.connect(get_db_url())
    total_rows = 0
    t0 = time.monotonic()

    for freq in sorted(freq_filter, reverse=True):  # 1d first, then 1h, then 5m
        key = f"stooq_{freq}"
        timeframe = freq if freq == "5m" else ("1h" if freq == "1h" else "1d")
        gen_id = compute_generation_id(
            [e["symbol"] for e in manifest["symbols"] if not sym_filter or e["symbol"] in sym_filter],
            freq,
        )
        print(f"\n--- Ingesting {freq} ({timeframe}) [generation_id={gen_id}] ---")

        for entry in manifest["symbols"]:
            sym = entry["symbol"]
            if sym_filter and sym not in sym_filter:
                continue

            file_info = entry.get(key)
            if not file_info or not file_info.get("exists"):
                print(f"  {sym}: SKIP (file not found or empty)")
                continue
            if file_info.get("is_empty"):
                print(f"  {sym}: SKIP (empty file)")
                continue

            filepath = STOOQ_ROOT / file_info["path"]
            if not filepath.exists():
                print(f"  {sym}: SKIP (path {filepath} does not exist)")
                continue

            symbol_id = ensure_symbol(conn, sym, entry)
            tz = timezone_for_symbol(sym, entry.get("asset_class", "unknown"))
            n = ingest_file(conn, symbol_id, filepath, timeframe, "stooq", gen_id, tz, args.dry_run)
            duration = time.monotonic() - t0
            print(f"  {sym:<8} {timeframe:<3} {n:>7} rows  ({duration:.1f}s)")
            total_rows += n

    conn.close()
    print(f"\nDone: {total_rows} rows in {time.monotonic() - t0:.1f}s")


if __name__ == "__main__":
    main()
