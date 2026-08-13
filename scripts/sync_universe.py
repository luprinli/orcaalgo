#!/usr/bin/env python3
"""Sync a database's symbols table to the canonical universe config.

Reads configs/universe.json, deactivates any active symbol not in the config,
and upserts the config symbols as active. Idempotent — safe to re-run.

Usage:
  python scripts/sync_universe.py                 # uses ORCA_DB_URL
  python scripts/sync_universe.py --dsn postgresql://user:pass@host:port/db
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(PROJECT_ROOT))

from orca.universe_config import load_universe  # noqa: E402


def main() -> None:
    parser = argparse.ArgumentParser(description="Sync symbols table to universe config")
    parser.add_argument("--dsn", help="PostgreSQL DSN (default: ORCA_DB_URL)")
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()

    dsn = args.dsn or os.environ.get("ORCA_DB_URL")
    if not dsn:
        print("ERROR: no DSN. Set ORCA_DB_URL or pass --dsn.")
        sys.exit(1)

    import psycopg2
    import psycopg2.extras

    u = load_universe()
    canonical = {s["ticker"]: s for s in u["symbols"]}

    conn = psycopg2.connect(dsn)
    cur = conn.cursor()

    # 1. Deactivate ALL symbols (old rows with non-canonical exchanges included).
    cur.execute("UPDATE symbols SET is_active = FALSE WHERE is_active = TRUE")
    deactivated = cur.rowcount

    # 2. Upsert canonical symbols as active.
    rows = [
        (s["ticker"], s["exchange"], s["asset_class"], s["tick_size"], s["lot_size"], True)
        for s in canonical.values()
    ]
    psycopg2.extras.execute_values(
        cur,
        """
        INSERT INTO symbols (ticker, exchange, asset_type, tick_size, lot_size, is_active)
        VALUES %s
        ON CONFLICT (ticker, exchange) DO UPDATE SET
            asset_type = EXCLUDED.asset_type,
            tick_size = EXCLUDED.tick_size,
            lot_size = EXCLUDED.lot_size,
            is_active = TRUE
        """,
        rows,
        page_size=100,
    )
    upserted = cur.rowcount

    if args.dry_run:
        conn.rollback()
        print(f"DRY-RUN: would deactivate {deactivated}, upsert {upserted}")
    else:
        conn.commit()
        print(f"Deactivated {deactivated} non-canonical symbols, upserted {upserted} canonical symbols")

    # 3. Verify.
    cur.execute("SELECT ticker FROM symbols WHERE is_active = TRUE ORDER BY ticker")
    active = [r[0] for r in cur.fetchall()]
    print(f"Active symbols ({len(active)}): {active}")

    conn.close()


if __name__ == "__main__":
    main()
