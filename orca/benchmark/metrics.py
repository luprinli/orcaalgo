"""Benchmark-relative performance metrics (HP #1: canonical math in Python).

All functions operate on per-period (aligned) decimal returns. Alignment (e.g.
trading-calendar intersection) is the caller's responsibility; this module
requires equal-length, finite inputs and raises on mismatch.
"""

from __future__ import annotations

from typing import Any

import numpy as np

_EPS = 1e-12


def _clean(returns: np.ndarray) -> np.ndarray:
    arr = np.asarray(returns, dtype=np.float64)
    arr = arr[np.isfinite(arr)]
    return arr


def _annualize(per_period: float, periods_per_year: float) -> float:
    return float(per_period * np.sqrt(periods_per_year))


def _total_return(returns: np.ndarray) -> float:
    if returns.size == 0:
        return 0.0
    return float(np.exp(np.sum(np.log1p(returns))) - 1.0)


def _cagr(returns: np.ndarray, periods_per_year: float) -> float:
    if returns.size == 0:
        return 0.0
    total = _total_return(returns)
    if total <= -1.0:
        return -1.0
    return float((1.0 + total) ** (periods_per_year / returns.size) - 1.0)


def _max_drawdown(equity: np.ndarray) -> float:
    if equity.size < 2:
        return 0.0
    peak = np.maximum.accumulate(equity)
    return float(np.min((equity - peak) / peak))


def compute_benchmark_metrics(
    strategy: np.ndarray,
    benchmark: np.ndarray,
    periods_per_year: float = 252.0,
) -> dict[str, Any]:
    """Compute benchmark-relative metrics for aligned per-period return series.

    Args:
        strategy: Strategy per-period (decimal) returns.
        benchmark: Benchmark per-period (decimal) returns, same length.
        periods_per_year: Annualization factor.

    Returns:
        Dict with beta, alpha_annualized, information_ratio (== annualized
        active Sharpe), tracking_error, correlation, up/down capture,
        win_rate_vs_benchmark, relative_max_drawdown, excess_cagr,
        excess_total_return, n_periods.
    """
    s = _clean(strategy)
    b = _clean(benchmark)
    if s.size != b.size:
        raise ValueError(
            f"strategy and benchmark must be aligned (same length): got {s.size} vs {b.size}"
        )
    n = s.size
    if n < 3:
        raise ValueError("need at least 3 aligned observations")

    active = s - b

    var_b = float(np.var(b, ddof=1))
    beta = float(np.cov(s, b, ddof=1)[0, 1] / var_b) if var_b > _EPS else 0.0
    alpha_annualized = float((np.mean(s) - beta * np.mean(b)) * periods_per_year)

    std_active = float(np.std(active, ddof=1))
    ir = (
        _annualize(float(np.mean(active)) / std_active, periods_per_year)
        if std_active > _EPS
        else 0.0
    )
    tracking_error = _annualize(std_active, periods_per_year)

    corr = 0.0
    std_s = float(np.std(s, ddof=1))
    std_b = float(np.std(b, ddof=1))
    if std_s > _EPS and std_b > _EPS:
        corr = float(np.corrcoef(s, b)[0, 1])

    up_mask = b > 0
    down_mask = b < 0
    up_capture = None
    down_capture = None
    if up_mask.any() and float(np.mean(b[up_mask])) > _EPS:
        up_capture = float(np.mean(s[up_mask]) / np.mean(b[up_mask]))
    if down_mask.any() and float(np.mean(b[down_mask])) < -_EPS:
        down_capture = float(np.mean(s[down_mask]) / np.mean(b[down_mask]))

    win_rate_vs_benchmark = float(np.mean(s > b))

    # Multiplicative active book (long strategy / short benchmark) starting at 1,
    # so the relative drawdown is the standard positive-curve drawdown (no div-by-zero).
    active_equity = np.exp(np.cumsum(active))
    relative_max_drawdown = _max_drawdown(active_equity)

    excess_cagr = _cagr(s, periods_per_year) - _cagr(b, periods_per_year)
    excess_total_return = _total_return(s) - _total_return(b)

    return {
        "n_periods": int(n),
        "beta": round(beta, 6),
        "alpha_annualized": round(alpha_annualized, 6),
        "information_ratio": round(ir, 6),
        "active_sharpe": round(ir, 6),
        "tracking_error": round(tracking_error, 6),
        "correlation": round(corr, 6),
        "up_capture": round(up_capture, 6) if up_capture is not None else None,
        "down_capture": round(down_capture, 6) if down_capture is not None else None,
        "win_rate_vs_benchmark": round(win_rate_vs_benchmark, 6),
        "relative_max_drawdown": round(relative_max_drawdown, 6),
        "excess_cagr": round(excess_cagr, 6),
        "excess_total_return": round(excess_total_return, 6),
    }


__all__ = ["compute_benchmark_metrics"]
