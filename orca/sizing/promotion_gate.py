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

from orca.sizing.deflated_sharpe import deflated_sharpe_ratio
from orca.sizing.multiple_testing import (
    apply_multiple_testing_correction,
)
from orca.sizing.sharpe_stats import sharpe_se_from_stats

MIN_TRADES = 20
DEFAULT_ALPHA = 0.05
DEFAULT_MAX_OOS_DEGRADATION = 0.20
DEFAULT_DSR_THRESHOLD = 0.95


@dataclass(frozen=True)
class GateResult:
    """Outcome of the promotion gate for the whole matrix sweep."""

    n_tests: int
    n_candidates: int
    bh_significant: int
    bonferroni_significant: int
    survivors: list[dict[str, Any]]
    passed: bool
    n_dsr_significant: int = 0
    n_benchmark_failed: int = 0

    def as_dict(self) -> dict[str, Any]:
        return {
            "n_tests": self.n_tests,
            "n_candidates": self.n_candidates,
            "bh_significant": self.bh_significant,
            "bonferroni_significant": self.bonferroni_significant,
            "n_dsr_significant": self.n_dsr_significant,
            "n_benchmark_failed": self.n_benchmark_failed,
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


def _trade_distribution_ok(row: dict[str, str]) -> bool | None:
    """Return True/False for the trade-distribution gate, or None when the
    columns are absent (the matrix runner has not exported them yet).

    Gate: the median trade PnL must be positive (a strategy whose "typical"
    trade loses money is not promotion-worthy regardless of Sharpe), and the
    candidate must span at least 2 unique tickers (breadth, not a single-name
    lucky run). Backward-compatible: absent columns are treated as n/a.
    """
    checks: list[bool] = []

    median_raw = row.get("MedianTradePnL", "")
    if median_raw not in ("", "N/A"):
        try:
            checks.append(float(median_raw) > 0)
        except ValueError:
            pass

    unique_raw = row.get("UniqueTickers", "")
    if unique_raw not in ("", "N/A"):
        try:
            checks.append(int(unique_raw) >= 2)
        except ValueError:
            pass

    if not checks:
        return None
    return all(checks)


def _benchmark_ok(row: dict[str, str]) -> bool | None:
    """Return True/False for the market-based benchmark filter gate, or None when
    the column is absent (the matrix runner has not exported benchmark verdicts).

    The matrix runner writes ``BenchmarkPass`` (true/false); the optional
    ``BenchmarkIR`` / ``BenchmarkAlpha`` columns are informational and not read
    here. Backward-compatible: absent column is treated as n/a.
    """
    raw = row.get("BenchmarkPass", "")
    if raw in ("", "N/A"):
        return None
    return raw.strip().lower() in ("true", "1", "pass", "yes")


def apply_promotion_gate(
    matrix_csv: str | Path,
    alpha: float = DEFAULT_ALPHA,
    min_trades: int = MIN_TRADES,
    max_oos_degradation: float = DEFAULT_MAX_OOS_DEGRADATION,
    require_dsr: bool = False,
    dsr_threshold: float = DEFAULT_DSR_THRESHOLD,
) -> GateResult:
    """Apply the promotion gate to a matrix-runner CSV.

    Args:
        matrix_csv: Path to the matrix results CSV (matrix-runner output).
        alpha: FDR/FWER significance level.
        min_trades: Minimum trades for a combination to be considered reliable.
        max_oos_degradation: Maximum allowed walk-forward OOS Sharpe degradation.
        require_dsr: When True, a survivor must also have Deflated Sharpe Ratio
            (DSR) >= ``dsr_threshold`` — an additive, opt-in selection-bias veto.
        dsr_threshold: DSR significance threshold (default 0.95).

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
    p_values = [_sharpe_p_value(float(r["Sharpe"]), int(r["Trades"])) for r in candidates]
    bh = apply_multiple_testing_correction(p_values, alpha=alpha, method="bh")
    bonf = apply_multiple_testing_correction(p_values, alpha=alpha, method="bonferroni")

    n_trials = max(len(candidates), 1)
    dsr_significant = 0
    benchmark_failed = 0
    survivors: list[dict[str, Any]] = []
    for i, r in enumerate(candidates):
        bh_pass = bool(bh["significant"][i])
        wf = _walk_forward_ok(r, max_oos_degradation)
        td = _trade_distribution_ok(r)
        bm = _benchmark_ok(r)
        if bm is False:
            benchmark_failed += 1

        sharpe = float(r["Sharpe"])
        trades = int(r["Trades"])
        skew, excess_kurtosis = _read_moment_columns(r)
        dsr = deflated_sharpe_ratio(
            sharpe, trades, n_trials, skew=skew, excess_kurtosis=excess_kurtosis
        )
        sharpe_se = sharpe_se_from_stats(sharpe, trades, skew, excess_kurtosis)
        dsr_value = float(dsr["deflated_sharpe_ratio"])
        if dsr_value >= dsr_threshold:
            dsr_significant += 1
        dsr_pass = (not require_dsr) or (dsr_value >= dsr_threshold)

        if bh_pass and dsr_pass and wf is not False and td is not False and bm is not False:
            survivors.append(
                {
                    "strategy": r["Strategy"],
                    "symbol": r["Symbol"],
                    # The API/CSV export writes "Timeframe"; the matrix-runner CLI
                    # writes "Tf". Accept both so the gate works on either format.
                    "timeframe": r.get("Timeframe", r.get("Tf", "")),
                    "sharpe": sharpe,
                    "trades": trades,
                    "p_value": p_values[i],
                    "sharpe_se": float(sharpe_se) if not math.isnan(sharpe_se) else None,
                    "deflated_sharpe_ratio": round(dsr_value, 4),
                    "expected_max_sharpe": round(float(dsr["expected_max_sharpe"]), 4),
                    "walk_forward": "pass" if wf is True else "n/a",
                    "trade_distribution": "pass" if td is True else "n/a",
                    "benchmark": "pass" if bm is True else "n/a",
                }
            )

    return GateResult(
        n_tests=n_tests,
        n_candidates=len(candidates),
        bh_significant=int(bh["n_significant"]),
        bonferroni_significant=int(bonf["n_significant"]),
        survivors=survivors,
        passed=len(survivors) > 0,
        n_dsr_significant=dsr_significant,
        n_benchmark_failed=benchmark_failed,
    )


def _read_moment_columns(row: dict[str, str]) -> tuple[float, float]:
    """Read optional Skew / Kurtosis (excess) columns, defaulting to normality."""
    skew = 0.0
    excess_kurtosis = 0.0
    for key, default in (("Skew", skew), ("Kurtosis", excess_kurtosis)):
        raw = row.get(key, "")
        if raw not in ("", "N/A"):
            try:
                value = float(raw)
            except ValueError:
                value = default
        else:
            value = default
        if key == "Skew":
            skew = value
        else:
            excess_kurtosis = value
    return skew, excess_kurtosis


__all__ = ["GateResult", "apply_promotion_gate"]
