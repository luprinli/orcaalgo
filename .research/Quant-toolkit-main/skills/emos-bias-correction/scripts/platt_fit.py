#!/usr/bin/env python3
"""Fit two-parameter Platt scaling per cohort.

Reads a training CSV and an out-of-time validation CSV with columns:
    raw_p, outcome[, cohort]
Fits A, B in: calibrated_p = sigmoid(A · logit(raw_p) + B) per cohort,
evaluates on validation, and writes coefficients.json with one block per
cohort plus a verdict (USE_COHORT_FIT / USE_GLOBAL_FIT / USE_RAW).

Requires numpy + scipy.

Usage:
    python platt_fit.py --train train.csv --val val.csv --by cohort --out coefficients.json
"""
from __future__ import annotations

import argparse
import csv
import json
import math
import sys
from collections import defaultdict
from dataclasses import dataclass, asdict

try:
    import numpy as np
    from scipy.optimize import minimize
except ImportError:
    sys.exit("requires numpy + scipy. install with: pip install numpy scipy")


EPS = 1e-6


def logit(p: np.ndarray) -> np.ndarray:
    p = np.clip(p, EPS, 1 - EPS)
    return np.log(p / (1 - p))


def sigmoid(x: np.ndarray) -> np.ndarray:
    return 1.0 / (1.0 + np.exp(-x))


def brier(p: np.ndarray, y: np.ndarray) -> float:
    return float(np.mean((p - y) ** 2))


def fit_platt(raw_p: np.ndarray, y: np.ndarray) -> tuple[float, float]:
    z = logit(raw_p)

    def neg_loglik(params):
        a, b = params
        p = sigmoid(a * z + b)
        p = np.clip(p, EPS, 1 - EPS)
        return -float(np.mean(y * np.log(p) + (1 - y) * np.log(1 - p)))

    result = minimize(neg_loglik, x0=[1.0, 0.0], method="Nelder-Mead")
    a, b = float(result.x[0]), float(result.x[1])
    return a, b


def apply_platt(raw_p: np.ndarray, a: float, b: float) -> np.ndarray:
    return sigmoid(a * logit(raw_p) + b)


@dataclass
class CohortFit:
    cohort: str
    n_train: int
    n_val: int
    a: float
    b: float
    val_brier_raw: float
    val_brier_global: float
    val_brier_cohort: float
    verdict: str


def load_csv(path: str, by: str | None) -> dict[str, tuple[np.ndarray, np.ndarray]]:
    groups: dict[str, list[tuple[float, int]]] = defaultdict(list)
    with open(path) as f:
        reader = csv.DictReader(f)
        for row in reader:
            try:
                p = float(row["raw_p"])
                y = int(row["outcome"])
            except (KeyError, ValueError) as exc:
                raise SystemExit(f"bad row in {path}: {row} ({exc})")
            key = row.get(by, "all") if by else "all"
            groups[key].append((p, y))

    out: dict[str, tuple[np.ndarray, np.ndarray]] = {}
    for k, rows in groups.items():
        ps = np.array([r[0] for r in rows], dtype=float)
        ys = np.array([r[1] for r in rows], dtype=float)
        out[k] = (ps, ys)
    return out


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--train", required=True)
    parser.add_argument("--val", required=True)
    parser.add_argument("--by", help="Column name to fit per cohort (e.g., 'cohort')")
    parser.add_argument("--out", default="coefficients.json")
    parser.add_argument("--min-cohort-n", type=int, default=200)
    parser.add_argument("--min-improvement-pct", type=float, default=5.0,
                        help="Minimum Brier improvement (%) for promotion")
    args = parser.parse_args()

    train = load_csv(args.train, args.by)
    val = load_csv(args.val, args.by)

    # Fit global Platt on union of training data.
    all_train_p = np.concatenate([p for p, _ in train.values()])
    all_train_y = np.concatenate([y for _, y in train.values()])
    g_a, g_b = fit_platt(all_train_p, all_train_y)

    fits: list[CohortFit] = []
    for cohort, (vp, vy) in sorted(val.items()):
        if vp.size == 0:
            continue
        raw_brier = brier(vp, vy)
        global_brier = brier(apply_platt(vp, g_a, g_b), vy)

        tp, ty = train.get(cohort, (np.array([]), np.array([])))
        if tp.size >= args.min_cohort_n:
            c_a, c_b = fit_platt(tp, ty)
            cohort_brier = brier(apply_platt(vp, c_a, c_b), vy)
        else:
            c_a, c_b = g_a, g_b
            cohort_brier = global_brier

        candidates = [
            ("USE_RAW", raw_brier),
            ("USE_GLOBAL_FIT", global_brier),
            ("USE_COHORT_FIT", cohort_brier),
        ]
        verdict, best_brier = min(candidates, key=lambda kv: kv[1])

        # Apply minimum-improvement rule: only promote a fit if it beats raw
        # by at least min-improvement-pct.
        improvement_pct = (raw_brier - best_brier) / raw_brier * 100 if raw_brier > 0 else 0
        if verdict != "USE_RAW" and improvement_pct < args.min_improvement_pct:
            verdict = "USE_RAW"

        fits.append(CohortFit(
            cohort=cohort,
            n_train=int(tp.size),
            n_val=int(vp.size),
            a=c_a, b=c_b,
            val_brier_raw=raw_brier,
            val_brier_global=global_brier,
            val_brier_cohort=cohort_brier,
            verdict=verdict,
        ))

    # Print summary
    print(f"  global Platt: a={g_a:+.4f}  b={g_b:+.4f}")
    print()
    print(f"  {'cohort':<20} {'n_train':>8} {'n_val':>6} {'raw':>8} {'global':>8} {'cohort':>8}  verdict")
    for f in fits:
        print(f"  {f.cohort:<20} {f.n_train:>8} {f.n_val:>6} {f.val_brier_raw:>8.4f} {f.val_brier_global:>8.4f} {f.val_brier_cohort:>8.4f}  {f.verdict}")

    payload = {
        "global": {"a": g_a, "b": g_b},
        "cohorts": {f.cohort: asdict(f) for f in fits},
    }
    with open(args.out, "w") as f:
        json.dump(payload, f, indent=2)
    print(f"\n  written to {args.out}")


if __name__ == "__main__":
    main()
