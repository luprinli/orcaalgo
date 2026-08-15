"""Deflated Sharpe Ratio and related selection-bias controls.

The probability that a backtested Sharpe ratio survives scrutiny shrinks with the
number of strategies/parameter combinations that were tried. This module provides
the canonical corrections from Bailey & López de Prado:

- ``probabilistic_sharpe_ratio`` (PSR) — probability that the true Sharpe exceeds
  a benchmark given a finite sample.
- ``deflated_sharpe_ratio`` (DSR) — PSR against the *expected maximum* Sharpe
  under ``n_trials``, correcting for selection bias.
- ``expected_max_sharpe`` — the Sharpe expected from the best of ``n_trials``
  independent null strategies.
- ``minimum_track_record_length`` — sample size needed to achieve a target PSR.
- ``cscv_pbo`` — Combinatorially Symmetric Cross-Validation probability of
  backtest overfitting (Bailey, Borwein, López de Prado & Zhu 2017).

All math is pure numpy/scipy (no reimplementation of canonical sizing logic in
Go — HP #1).
"""

from __future__ import annotations

import math
from typing import Any

import numpy as np
from scipy import stats

_EULER_MASCHERONI = 0.5772156649015329


def _expected_max_standard_normal(n_trials: int) -> float:
    """Expected maximum of ``n_trials`` iid standard normals.

    Uses the Bailey & López de Prado approximation with the Euler-Mascheroni
    constant. Returns 0 for a single trial.
    """
    if n_trials <= 1:
        return 0.0
    inv_1 = stats.norm.ppf(1.0 - 1.0 / n_trials)
    inv_e = stats.norm.ppf(1.0 - 1.0 / (n_trials * math.e))
    return (1.0 - _EULER_MASCHERONI) * inv_1 + _EULER_MASCHERONI * inv_e


def expected_max_sharpe(
    n_trials: int,
    n_obs: int,
    skew: float = 0.0,
    excess_kurtosis: float = 0.0,
) -> float:
    """Expected maximum Sharpe of ``n_trials`` null strategies on ``n_obs`` samples.

    Under the null (true Sharpe zero), the standard error is ``1/sqrt(n_obs - 1)``,
    so the expected maximum is the expected-max of a standard normal scaled by that
    standard error. The skew/kurtosis terms are retained for signature symmetry
    with the PSR but do not affect the null benchmark.
    """
    del skew, excess_kurtosis
    if n_obs < 3:
        return float("nan")
    return _expected_max_standard_normal(n_trials) / math.sqrt(n_obs - 1)


def probabilistic_sharpe_ratio(
    sharpe: float,
    benchmark: float,
    n_obs: int,
    skew: float = 0.0,
    excess_kurtosis: float = 0.0,
) -> float:
    """PSR: probability the true Sharpe exceeds ``benchmark``.

    ``sharpe`` is at the observation frequency (annualized Sharpe divided by
    ``sqrt(periods_per_year)``).
    """
    if n_obs < 3:
        return float("nan")
    var_num = 1.0 - skew * sharpe + (excess_kurtosis + 2.0) / 4.0 * sharpe**2
    var_num = max(var_num, 1e-12)
    z = (sharpe - benchmark) * math.sqrt(n_obs - 1) / math.sqrt(var_num)
    return float(stats.norm.cdf(z))


def deflated_sharpe_ratio(
    sharpe: float,
    n_obs: int,
    n_trials: int,
    skew: float = 0.0,
    excess_kurtosis: float = 0.0,
) -> dict[str, Any]:
    """Deflated Sharpe Ratio: PSR against the expected-max Sharpe over trials.

    Returns a dict with the DSR, the deflation benchmark (expected max Sharpe),
    and the undeflated PSR against zero for reference.
    """
    sr_star = expected_max_sharpe(n_trials, n_obs, skew, excess_kurtosis)
    dsr = probabilistic_sharpe_ratio(sharpe, sr_star, n_obs, skew, excess_kurtosis)
    psr_zero = probabilistic_sharpe_ratio(sharpe, 0.0, n_obs, skew, excess_kurtosis)
    return {
        "deflated_sharpe_ratio": float(dsr),
        "expected_max_sharpe": float(sr_star),
        "probabilistic_sharpe_ratio": float(psr_zero),
    }


def minimum_track_record_length(
    sharpe: float,
    benchmark: float = 0.0,
    skew: float = 0.0,
    excess_kurtosis: float = 0.0,
    target_psr: float = 0.95,
) -> float:
    """Minimum number of observations for PSR(sharpe, benchmark) >= target_psr.

    Classic MinTRL (Bailey & López de Prado). To deflate for multiple testing,
    pass ``benchmark=expected_max_sharpe(n_trials, n_obs)``. Returns ``inf`` when
    ``sharpe <= benchmark`` (no sample size can rescue a non-positive edge).
    """
    num = 1.0 - skew * sharpe + (excess_kurtosis + 2.0) / 4.0 * sharpe**2
    num = max(num, 1e-12)
    denom = sharpe - benchmark
    if denom <= 0:
        return float("inf")
    z_star = stats.norm.ppf(target_psr)
    return float((z_star * math.sqrt(num) / denom) ** 2 + 1)


def cscv_pbo(
    returns: np.ndarray,
    n_splits: int = 16,
    seed: int | None = None,
) -> dict[str, Any]:
    """Combinatorially Symmetric Cross-Validation probability of backtest overfitting.

    ``returns`` is a (T, S) matrix of T observations for S strategies/trials (a
    single trial is rejected: PBO is a cross-trial quantity). Returns the fraction
    of IS/OOS combinations where the IS-optimal strategy underperforms the OOS
    median.
    """
    matrix = np.asarray(returns, dtype=np.float64)
    if matrix.ndim != 2 or matrix.shape[1] < 2:
        return {
            "pbo": float("nan"),
            "n_combinations": 0,
            "error": "needs (T, S) matrix with S >= 2",
        }

    t = matrix.shape[0]
    n_splits = max(2, min(int(n_splits), t))
    if t < 2 * n_splits:
        n_splits = max(2, t // 2)
    if n_splits < 2:
        return {"pbo": float("nan"), "n_combinations": 0, "error": "insufficient observations"}

    rng = np.random.default_rng(seed)
    # Split the T rows into n_splits contiguous, equal-size blocks.
    block_len = t // n_splits
    blocks = [matrix[i * block_len : (i + 1) * block_len] for i in range(n_splits)]
    split_idx = np.arange(n_splits)

    n_half = n_splits // 2
    overfit_count = 0
    n_combinations = 0

    # Sample up to min(C(n_splits, n_half), 1000) IS/OOS partitions.
    max_combos = min(math.comb(n_splits, n_half), 1000)
    for _ in range(max_combos):
        is_idx = rng.choice(split_idx, size=n_half, replace=False)
        oos_idx = np.setdiff1d(split_idx, is_idx, assume_unique=True)

        is_returns = np.vstack([blocks[i] for i in is_idx])
        oos_returns = np.vstack([blocks[i] for i in oos_idx])

        is_sharpe = _sharpe_per_strategy(is_returns)
        oos_sharpe = _sharpe_per_strategy(oos_returns)
        if np.all(~np.isfinite(is_sharpe)) or np.all(~np.isfinite(oos_sharpe)):
            continue

        best_is = int(np.nanargmax(is_sharpe))
        oos_median = float(np.nanmedian(oos_sharpe))
        n_combinations += 1
        if oos_sharpe[best_is] < oos_median:
            overfit_count += 1

    if n_combinations == 0:
        return {"pbo": float("nan"), "n_combinations": 0, "error": "no valid combinations"}

    return {"pbo": float(overfit_count / n_combinations), "n_combinations": n_combinations}


def _sharpe_per_strategy(matrix: np.ndarray) -> np.ndarray:
    """Sharpe (mean/std) per column of a returns matrix."""
    means = np.mean(matrix, axis=0)
    stds = np.std(matrix, axis=0, ddof=1)
    out = np.full(matrix.shape[1], np.nan)
    mask = stds > 1e-12
    out[mask] = means[mask] / stds[mask]
    return out


__all__ = [
    "cscv_pbo",
    "deflated_sharpe_ratio",
    "expected_max_sharpe",
    "minimum_track_record_length",
    "probabilistic_sharpe_ratio",
]
