#!/usr/bin/env python3
"""P&L attribution slicer.

Slices a settled trade ledger by side, price bucket, edge bucket, cohort,
and side × price bucket. Reports n, ROI on cost, hit rate, net P&L,
and a Wilson confidence interval on the hit rate.

Expects a SQLite DB or a CSV with at minimum these columns:
    side          'yes' or 'no'
    fill_price    in [0, 1]
    forecast_p    in [0, 1]
    outcome       0 or 1
    cost          dollars at risk
    pnl           realized P&L in dollars
    cohort        optional categorical
    placed_at     optional ISO timestamp

Usage:
    python attribute.py --csv trades.csv
    python attribute.py --db trades.sqlite --table trades --since 2026-01-01
"""
from __future__ import annotations

import argparse
import csv
import math
import sqlite3
from collections import defaultdict
from dataclasses import dataclass
from typing import Iterable


@dataclass
class Trade:
    side: str
    fill_price: float
    forecast_p: float
    outcome: int
    cost: float
    pnl: float
    cohort: str = ""
    placed_at: str = ""


def wilson_ci(wins: int, n: int, z: float = 1.96) -> tuple[float, float]:
    if n == 0:
        return (0.0, 1.0)
    p = wins / n
    denom = 1 + z * z / n
    center = (p + z * z / (2 * n)) / denom
    half = (z * math.sqrt(p * (1 - p) / n + z * z / (4 * n * n))) / denom
    return (max(center - half, 0.0), min(center + half, 1.0))


def price_bucket(p: float) -> str:
    if p < 0.30:
        return "0-30c"
    if p < 0.50:
        return "30-50c"
    if p < 0.70:
        return "50-70c"
    return "70-100c"


def edge_bucket(forecast_p: float, fill_price: float, side: str) -> str:
    if side == "yes":
        edge = forecast_p - fill_price
    else:
        edge = (1 - forecast_p) - fill_price
    if edge < 0.05:
        return "0-5%"
    if edge < 0.10:
        return "5-10%"
    if edge < 0.20:
        return "10-20%"
    return "20%+"


@dataclass
class SliceStats:
    label: str
    n: int
    wins: int
    cost: float
    pnl: float
    roi: float
    hit_rate: float
    ci_low: float
    ci_high: float


def slice_by(trades: Iterable[Trade], key) -> list[SliceStats]:
    groups: dict[str, list[Trade]] = defaultdict(list)
    for t in trades:
        groups[key(t)].append(t)
    out = []
    for label, rows in groups.items():
        n = len(rows)
        wins = sum(1 for r in rows if r.pnl > 0)
        cost = sum(r.cost for r in rows)
        pnl = sum(r.pnl for r in rows)
        roi = pnl / cost if cost > 0 else 0.0
        hit = wins / n if n > 0 else 0.0
        lo, hi = wilson_ci(wins, n)
        out.append(SliceStats(str(label), n, wins, cost, pnl, roi, hit, lo, hi))
    out.sort(key=lambda s: s.label)
    return out


def print_slice(title: str, stats: list[SliceStats]) -> None:
    print(f"\n=== {title} ===")
    print(f"  {'slice':<25} {'n':>5}  {'wins':>5}  {'cost':>10}  {'pnl':>10}  {'ROI':>8}  {'hit':>7}   {'CI':>15}")
    for s in stats:
        flag = " *" if s.n < 30 else ""
        print(f"  {s.label:<25} {s.n:>5}  {s.wins:>5}  ${s.cost:>9.2f}  ${s.pnl:>+9.2f}  {s.roi*100:>+7.2f}%  {s.hit*100:>5.1f}%   [{s.ci_low*100:>4.1f}-{s.ci_high*100:>4.1f}%]{flag}")
    print("  (* = n < 30, treat as exploratory)")


def load_csv(path: str, since: str | None) -> list[Trade]:
    out = []
    with open(path) as f:
        reader = csv.DictReader(f)
        for row in reader:
            if since and row.get("placed_at", "") < since:
                continue
            out.append(Trade(
                side=row["side"],
                fill_price=float(row["fill_price"]),
                forecast_p=float(row["forecast_p"]),
                outcome=int(row["outcome"]),
                cost=float(row["cost"]),
                pnl=float(row["pnl"]),
                cohort=row.get("cohort", ""),
                placed_at=row.get("placed_at", ""),
            ))
    return out


def load_sqlite(path: str, table: str, since: str | None) -> list[Trade]:
    conn = sqlite3.connect(path)
    conn.row_factory = sqlite3.Row
    where = ""
    params: tuple = ()
    if since:
        where = " WHERE placed_at >= ?"
        params = (since,)
    rows = conn.execute(f"SELECT side, fill_price, forecast_p, outcome, cost, pnl, cohort, placed_at FROM {table}{where}", params).fetchall()
    conn.close()
    return [Trade(
        side=r["side"],
        fill_price=r["fill_price"],
        forecast_p=r["forecast_p"],
        outcome=r["outcome"],
        cost=r["cost"],
        pnl=r["pnl"],
        cohort=r["cohort"] or "",
        placed_at=r["placed_at"] or "",
    ) for r in rows]


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    src = parser.add_mutually_exclusive_group(required=True)
    src.add_argument("--csv")
    src.add_argument("--db")
    parser.add_argument("--table", default="trades")
    parser.add_argument("--since", help="ISO date — filter trades placed_at >= since")
    args = parser.parse_args()

    if args.csv:
        trades = load_csv(args.csv, args.since)
    else:
        trades = load_sqlite(args.db, args.table, args.since)

    if not trades:
        print("no trades match the filter")
        return

    print(f"loaded {len(trades)} trades")
    if args.since:
        print(f"  filtered since {args.since}")

    print_slice("by side", slice_by(trades, lambda t: t.side))
    print_slice("by entry price bucket", slice_by(trades, lambda t: price_bucket(t.fill_price)))
    print_slice("by edge bucket", slice_by(trades, lambda t: edge_bucket(t.forecast_p, t.fill_price, t.side)))
    print_slice("by side x price bucket", slice_by(trades, lambda t: f"{t.side}/{price_bucket(t.fill_price)}"))
    if any(t.cohort for t in trades):
        print_slice("by cohort", slice_by(trades, lambda t: t.cohort or "(none)"))


if __name__ == "__main__":
    main()
