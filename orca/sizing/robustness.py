"""Backtest statistical-robustness summary (R1/R5 wiring).

Composes the closed-form Sharpe SE (``sharpe_stats``) and the deflated-Sharpe
selection-bias controls (``deflated_sharpe``) into a single dict for a daily
return series. This is the canonical Python surface the Go API shells out to for
the ``/backtests/:id/robustness`` endpoint (HP #1: math stays in Python).
"""

from __future__ import annotations

from typing import Any

import numpy as np

from orca.sizing.deflated_sharpe import (
    deflated_sharpe_ratio,
    minimum_track_record_length,
)
from orca.sizing.sharpe_stats import (
    excess_kurtosis,
    sharpe_se,
    skewness,
)

_CI_Z = 1.96


def backtest_robustness_stats(
    returns: np.ndarray,
    n_trials: int = 1,
    periods_per_year: float = 252.0,
) -> dict[str, Any]:
    """Compute the statistical-robustness summary for a return series.

    Args:
        returns: Array of per-period (daily) returns.
        n_trials: Number of strategies/combinations tried (deflation trials).
        periods_per_year: Annualization factor (252 for daily).

    Returns:
        Dict with ``sharpe`` (annualized), ``sharpe_se``, ``sharpe_ci_low`` /
        ``sharpe_ci_high`` (95%), ``deflated_sharpe_ratio``, ``min_trl`` and
        ``n_returns``. Empty/insufficient input yields an ``error`` key.
    """
    r = np.asarray(returns, dtype=np.float64)
    r = r[np.isfinite(r)]
    n = r.size
    if n < 3:
        return {"error": "insufficient returns", "n_returns": int(n)}

    mean = float(np.mean(r))
    std = float(np.std(r, ddof=1))
    sharpe_ann = mean / std * np.sqrt(periods_per_year) if std > 1e-12 else 0.0
    se_ann = sharpe_se(r, periods_per_year)

    per_period_sr = mean / std if std > 1e-12 else 0.0
    skew = skewness(r)
    kurt = excess_kurtosis(r)
    dsr = deflated_sharpe_ratio(per_period_sr, n, max(n_trials, 1), skew, kurt)
    min_trl = minimum_track_record_length(
        per_period_sr,
        benchmark=float(dsr["expected_max_sharpe"]),
        skew=skew,
        excess_kurtosis=kurt,
    )

    return {
        "n_returns": int(n),
        "sharpe": round(sharpe_ann, 4),
        "sharpe_se": round(se_ann, 4),
        "sharpe_ci_low": round(sharpe_ann - _CI_Z * se_ann, 4),
        "sharpe_ci_high": round(sharpe_ann + _CI_Z * se_ann, 4),
        "deflated_sharpe_ratio": round(float(dsr["deflated_sharpe_ratio"]), 4),
        "expected_max_sharpe": round(float(dsr["expected_max_sharpe"]), 4),
        "min_trl": round(min_trl, 2) if np.isfinite(min_trl) else None,
        "n_trials": int(n_trials),
    }


__all__ = ["backtest_robustness_stats"]
