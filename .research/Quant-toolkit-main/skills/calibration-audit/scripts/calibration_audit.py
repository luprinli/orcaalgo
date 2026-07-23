#!/usr/bin/env python3
"""Calibration audit for a probabilistic forecast.

Reads a CSV of (forecast_p, outcome) pairs and prints:
  - Reliability diagram (per-bin table)
  - Brier score with the Murphy decomposition
  - Worst-3 bin diagnostics
  - Optional grouping by `side` or `cohort`

Usage:
    python calibration_audit.py --csv predictions.csv --bins 10
    python calibration_audit.py --csv predictions.csv --group-by side
"""
from __future__ import annotations

import argparse
import csv
from collections import defaultdict
from dataclasses import dataclass


@dataclass
class BinStats:
    lo: float
    hi: float
    n: int
    mean_forecast: float
    empirical_rate: float
    gap: float  # empirical - mean_forecast


@dataclass
class AuditResult:
    n: int
    base_rate: float
    brier: float
    reliability: float
    resolution: float
    uncertainty: float
    bins: list[BinStats]


def audit(pairs: list[tuple[float, int]], n_bins: int = 10) -> AuditResult:
    """Audit calibration on (forecast_p, outcome) pairs.

    outcome must be 0 or 1.
    """
    if not pairs:
        raise ValueError("no data")
    forecasts = [p for p, _ in pairs]
    outcomes = [o for _, o in pairs]
    n = len(pairs)
    base_rate = sum(outcomes) / n

    # Brier score
    brier = sum((p - o) ** 2 for p, o in pairs) / n

    # Bin assignments
    bin_data: dict[int, list[tuple[float, int]]] = defaultdict(list)
    for p, o in pairs:
        idx = min(int(p * n_bins), n_bins - 1)
        bin_data[idx].append((p, o))

    bins: list[BinStats] = []
    reliability = 0.0
    resolution = 0.0
    for idx in range(n_bins):
        rows = bin_data.get(idx, [])
        if not rows:
            continue
        n_bin = len(rows)
        mean_p = sum(p for p, _ in rows) / n_bin
        rate = sum(o for _, o in rows) / n_bin
        bins.append(BinStats(
            lo=idx / n_bins,
            hi=(idx + 1) / n_bins,
            n=n_bin,
            mean_forecast=mean_p,
            empirical_rate=rate,
            gap=rate - mean_p,
        ))
        reliability += (n_bin / n) * (mean_p - rate) ** 2
        resolution += (n_bin / n) * (rate - base_rate) ** 2

    uncertainty = base_rate * (1 - base_rate)

    return AuditResult(
        n=n,
        base_rate=base_rate,
        brier=brier,
        reliability=reliability,
        resolution=resolution,
        uncertainty=uncertainty,
        bins=bins,
    )


def print_audit(label: str, r: AuditResult, small_bin_threshold: int = 20) -> None:
    print(f"\n=== {label} ===")
    print(f"  n                  {r.n}")
    print(f"  base rate          {r.base_rate:.4f}")
    print(f"  Brier              {r.brier:.4f}")
    print(f"    reliability      {r.reliability:.4f}  (lower better)")
    print(f"    resolution       {r.resolution:.4f}  (higher better)")
    print(f"    uncertainty      {r.uncertainty:.4f}")
    decomp_check = r.reliability - r.resolution + r.uncertainty
    print(f"    decomp check     {decomp_check:.4f}  (should equal Brier)")
    print()
    print(f"  {'bin':>10}  {'n':>6}  {'mean_p':>8}  {'rate':>8}  {'gap':>8}  flag")
    for b in r.bins:
        flag = "small-n" if b.n < small_bin_threshold else ""
        print(f"  [{b.lo:.2f}-{b.hi:.2f}]  {b.n:>6}  {b.mean_forecast:>8.4f}  {b.empirical_rate:>8.4f}  {b.gap:>+8.4f}  {flag}")

    # Worst 3 bins (largest |gap|, ignoring small-n)
    sortable = [b for b in r.bins if b.n >= small_bin_threshold]
    sortable.sort(key=lambda b: abs(b.gap), reverse=True)
    if sortable:
        print(f"\n  worst-3 (n >= {small_bin_threshold}):")
        for b in sortable[:3]:
            direction = "over-confident" if b.gap < 0 else "under-confident"
            print(f"    [{b.lo:.2f}-{b.hi:.2f}]  gap={b.gap:+.4f}  ({direction})")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--csv", required=True, help="CSV with columns forecast_p,outcome (optional: side, cohort)")
    parser.add_argument("--bins", type=int, default=10)
    parser.add_argument("--group-by", help="Column name to group rows by (e.g. side, cohort)")
    parser.add_argument("--small-bin-threshold", type=int, default=20)
    args = parser.parse_args()

    groups: dict[str, list[tuple[float, int]]] = defaultdict(list)
    with open(args.csv) as f:
        reader = csv.DictReader(f)
        for row in reader:
            try:
                p = float(row["forecast_p"])
                o = int(row["outcome"])
            except (KeyError, ValueError) as exc:
                raise SystemExit(f"bad row: {row} ({exc})")
            key = row.get(args.group_by, "all") if args.group_by else "all"
            groups[key].append((p, o))

    for label, pairs in sorted(groups.items()):
        result = audit(pairs, n_bins=args.bins)
        print_audit(label, result, small_bin_threshold=args.small_bin_threshold)


if __name__ == "__main__":
    main()
