#!/usr/bin/env python3
"""Drawdown monitor.

Reads a settled-trade ledger (CSV or SQLite), computes running bankroll,
high-water-mark, and current drawdown, then emits the threshold verdict.

Expected schema:
    placed_at  ISO timestamp
    pnl        realized P&L for the trade in dollars

Usage:
    python drawdown.py --csv trades.csv --starting-bankroll 1000
    python drawdown.py --db trades.sqlite --table trades --hwm-window 30
"""
from __future__ import annotations

import argparse
import csv
import sqlite3
from dataclasses import dataclass
from datetime import datetime, timedelta


@dataclass
class Thresholds:
    warn_pct: float = 0.05
    derisk_pct: float = 0.10
    halt_pct: float = 0.20


@dataclass
class DrawdownStatus:
    current_bankroll: float
    hwm: float
    hwm_at: datetime | None
    current_dd: float
    dd_since: datetime | None
    dd_duration_days: int
    level: str   # 'clear', 'warn', 'derisk', 'halt'


def load_trades_csv(path: str) -> list[tuple[datetime, float]]:
    out = []
    with open(path) as f:
        reader = csv.DictReader(f)
        for row in reader:
            ts = datetime.fromisoformat(row["placed_at"])
            pnl = float(row["pnl"])
            out.append((ts, pnl))
    out.sort(key=lambda t: t[0])
    return out


def load_trades_sqlite(path: str, table: str) -> list[tuple[datetime, float]]:
    conn = sqlite3.connect(path)
    rows = conn.execute(f"SELECT placed_at, pnl FROM {table} ORDER BY placed_at").fetchall()
    conn.close()
    return [(datetime.fromisoformat(r[0]), float(r[1])) for r in rows if r[0]]


def compute_status(
    trades: list[tuple[datetime, float]],
    starting_bankroll: float,
    thresholds: Thresholds,
    hwm_window_days: int | None = None,  # None = all-time
) -> DrawdownStatus:
    bankroll = starting_bankroll
    hwm = starting_bankroll
    hwm_at: datetime | None = None
    dd_since: datetime | None = None

    now = datetime.now() if not trades else trades[-1][0]
    cutoff = now - timedelta(days=hwm_window_days) if hwm_window_days else None

    for ts, pnl in trades:
        bankroll += pnl
        if cutoff is None or ts >= cutoff:
            if bankroll > hwm:
                hwm = bankroll
                hwm_at = ts
                dd_since = None
            elif bankroll < hwm and dd_since is None:
                dd_since = ts

    current_dd = (hwm - bankroll) / hwm if hwm > 0 else 0.0
    dd_duration = (now - dd_since).days if dd_since else 0

    if current_dd >= thresholds.halt_pct:
        level = "halt"
    elif current_dd >= thresholds.derisk_pct:
        level = "derisk"
    elif current_dd >= thresholds.warn_pct:
        level = "warn"
    else:
        level = "clear"

    return DrawdownStatus(
        current_bankroll=bankroll,
        hwm=hwm,
        hwm_at=hwm_at,
        current_dd=current_dd,
        dd_since=dd_since,
        dd_duration_days=dd_duration,
        level=level,
    )


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    src = parser.add_mutually_exclusive_group(required=True)
    src.add_argument("--csv")
    src.add_argument("--db")
    parser.add_argument("--table", default="trades")
    parser.add_argument("--starting-bankroll", type=float, required=True)
    parser.add_argument("--hwm-window", default="all", help="'all' or integer days (e.g. 30)")
    parser.add_argument("--warn-pct", type=float, default=0.05)
    parser.add_argument("--derisk-pct", type=float, default=0.10)
    parser.add_argument("--halt-pct", type=float, default=0.20)
    args = parser.parse_args()

    if args.csv:
        trades = load_trades_csv(args.csv)
    else:
        trades = load_trades_sqlite(args.db, args.table)

    window = None if args.hwm_window == "all" else int(args.hwm_window)
    thresholds = Thresholds(args.warn_pct, args.derisk_pct, args.halt_pct)
    status = compute_status(trades, args.starting_bankroll, thresholds, window)

    print(f"  trades              {len(trades)}")
    print(f"  current bankroll    ${status.current_bankroll:.2f}")
    print(f"  high-water-mark     ${status.hwm:.2f}")
    print(f"    set at            {status.hwm_at.isoformat() if status.hwm_at else 'inception'}")
    print(f"  current drawdown    {status.current_dd*100:.2f}%")
    print(f"  dd duration         {status.dd_duration_days} days")
    print(f"  level               {status.level.upper()}")
    print()
    print(f"  thresholds: warn≥{thresholds.warn_pct*100:.0f}%  derisk≥{thresholds.derisk_pct*100:.0f}%  halt≥{thresholds.halt_pct*100:.0f}%")

    if status.level == "halt":
        print("\n  ACTION: stop placing new orders. cancel resting maker quotes.")
        print("           let open positions ride to settlement. user must re-arm explicitly.")
    elif status.level == "derisk":
        print("\n  ACTION: cut Kelly multiplier by half, tighten edge thresholds.")
        print("           notify user. continue at reduced size.")
    elif status.level == "warn":
        print("\n  ACTION: notify user. increase logging. force P&L attribution review.")


if __name__ == "__main__":
    main()
