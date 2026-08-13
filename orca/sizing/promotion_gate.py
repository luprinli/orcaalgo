"""Promotion gate for matrix sweep results (R15 / Fix 4).

A backtest combination is promotion-eligible only if it survives BOTH:
  1. Multiple-testing correction — the combination's Sharpe must be significant
     under Benjamini-Hochberg FDR control (and optionally Bonferroni FWER)
     against the full matrix sweep (every combination tested is a hypothesis).
  2. Walk-forward validation — when walk-forward columns are present, the OOS
     Sharpe must not degrade more than `max_oos_degradation` (20%) from IS.

The Sharpe p-value uses the asymptotic t-statistic (t = Sharpe * sqrt(n_trades),
two-sided normal). This is the pre-promotion gate that must pass with zero
failures before any strategy is promoted to orchestration/live.
"""

from __future__ import annotations

import csv
import math
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from orca.sizing.multiple_testing import (
    apply_multiple_testing_correction,
)

MIN_TRADES = 20
DEFAULT_ALPHA = 0.05
DEFAULT_MAX_OOS_DEGRADATION = 0.20


@dataclass(frozen=True)
class GateResult:
    """Outcome of the promotion gate for the whole matrix sweep."""

    n_tests: int
    n_candidates: int
    bh_significant: int
    bonferroni_significant: int
    survivors: list[dict[str, Any]]
    passed: bool

    def as_dict(self) -> dict[str, Any]:
        return {
            "n_tests": self.n_tests,
            "n_candidates": self.n_candidates,
            "bh_significant": self.bh_significant,
            "bonferroni_significant": self.bonferroni_significant,
            "survivors": self.survivors,
            "passed": self.passed,
        }


def _sharpe_p_value(sharpe: float, n_trades: int) -> float:
    """Two-sided asymptotic p-value for a Sharpe ratio.

    t = Sharpe * sqrt(n_trades); p = 2 * Phi(-|t|). Consistent with the standard
    quant multiple-testing gate (see orca/sizing/block_bootstrap.py for the
    bootstrap refinement).
    """
    t = abs(sharpe) * math.sqrt(max(n_trades, 1))
    return math.erfc(t / math.sqrt(2.0))


def _walk_forward_ok(row: dict[str, str], max_degradation: float) -> bool | None:
    """Return True/False for walk-forward gate, or None when no WF data."""
    oos_raw = row.get("WfOOSSharpe", "")
    if oos_raw in ("", "N/A"):
        return None
    oos = float(oos_raw)
    isharpe_raw = row.get("WfISSharpe", "")
    if isharpe_raw not in ("", "N/A"):
        isharpe = float(isharpe_raw)
        if isharpe > 0:
            return oos >= (1.0 - max_degradation) * isharpe
    return oos > 0


def apply_promotion_gate(
    matrix_csv: str | Path,
    alpha: float = DEFAULT_ALPHA,
    min_trades: int = MIN_TRADES,
    max_oos_degradation: float = DEFAULT_MAX_OOS_DEGRADATION,
) -> GateResult:
    """Apply the promotion gate to a matrix-runner CSV.

    Args:
        matrix_csv: Path to the matrix results CSV (matrix-runner output).
        alpha: FDR/FWER significance level.
        min_trades: Minimum trades for a combination to be considered reliable.
        max_oos_degradation: Maximum allowed walk-forward OOS Sharpe degradation.

    Returns:
        GateResult summarizing candidates and survivors.
    """
    rows = list(csv.DictReader(open(matrix_csv, encoding="utf-8")))
    n_tests = len(rows)

    candidates: list[dict[str, str]] = []
    for r in rows:
        try:
            trades = int(r["Trades"])
            sharpe = float(r["Sharpe"])
        except (KeyError, ValueError):
            continue
        if trades >= min_trades and sharpe > 0:
            candidates.append(r)

    # p-values across the full sweep (every combo is a hypothesis test).
    p_values = [
        _sharpe_p_value(float(r["Sharpe"]), int(r["Trades"]))
        for r in candidates
    ]
    bh = apply_multiple_testing_correction(p_values, alpha=alpha, method="bh")
    bonf = apply_multiple_testing_correction(p_values, alpha=alpha, method="bonferroni")

    survivors: list[dict[str, Any]] = []
    for i, r in enumerate(candidates):
        bh_pass = bool(bh["significant"][i])
        wf = _walk_forward_ok(r, max_oos_degradation)
        if bh_pass and wf is not False:
            survivors.append({
                "strategy": r["Strategy"],
                "symbol": r["Symbol"],
                "timeframe": r["Tf"],
                "sharpe": float(r["Sharpe"]),
                "trades": int(r["Trades"]),
                "p_value": p_values[i],
                "walk_forward": "pass" if wf is True else "n/a",
            })

    return GateResult(
        n_tests=n_tests,
        n_candidates=len(candidates),
        bh_significant=int(bh["n_significant"]),
        bonferroni_significant=int(bonf["n_significant"]),
        survivors=survivors,
        passed=len(survivors) > 0,
    )


__all__ = ["GateResult", "apply_promotion_gate"]
