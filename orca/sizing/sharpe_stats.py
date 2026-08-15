"""Closed-form standard errors for the Sharpe ratio and mean return.

Implements the Lo (2002) / López de Prado (2025) non-normal, non-IID closed-form
standard error of the annualized Sharpe ratio, plus a Newey-West HAC standard
error for the mean return. These complement the bootstrap confidence intervals in
``orca/sizing/block_bootstrap.py`` with cheap, deterministic point estimates.

The closed form is (per-observation Sharpe ``SR``, skewness ``gamma3``, excess
kurtosis ``kappa``, ``n`` observations)::

    var(SR) = (1 - gamma3*SR + (kappa + 2)/4 * SR**2) / (n - 1)

For a normal distribution (gamma3=0, kappa=0) this reduces to the familiar
``(1 + SR**2/2) / (n - 1)``. The per-observation standard error is annualized by
``sqrt(periods_per_year)``.
"""

from __future__ import annotations

import numpy as np


def _clean(returns: np.ndarray) -> np.ndarray:
    arr = np.asarray(returns, dtype=np.float64)
    arr = arr[~np.isnan(arr)]
    return arr[np.isfinite(arr)]


def _annualized_sharpe(returns: np.ndarray, periods_per_year: float) -> float:
    if returns.size < 2:
        return 0.0
    std = np.std(returns, ddof=1)
    if std < 1e-12:
        return 0.0
    return float(np.mean(returns) / std * np.sqrt(periods_per_year))


def skewness(returns: np.ndarray) -> float:
    """Population skewness of a return series."""
    r = _clean(returns)
    n = r.size
    if n < 3:
        return 0.0
    mu = np.mean(r)
    std = np.std(r, ddof=1)
    if std < 1e-12:
        return 0.0
    m3 = np.mean((r - mu) ** 3)
    return float(m3 / std**3)


def excess_kurtosis(returns: np.ndarray) -> float:
    """Population excess kurtosis of a return series."""
    r = _clean(returns)
    n = r.size
    if n < 4:
        return 0.0
    mu = np.mean(r)
    std = np.std(r, ddof=1)
    if std < 1e-12:
        return 0.0
    m4 = np.mean((r - mu) ** 4)
    return float(m4 / std**4 - 3.0)


def sharpe_se_from_stats(
    sharpe: float,
    n_obs: int,
    skew: float = 0.0,
    excess_kurtosis: float = 0.0,
) -> float:
    """Closed-form standard error of a (per-observation) Sharpe ratio.

    ``sharpe`` is the Sharpe ratio at the observation frequency (i.e. annualized
    Sharpe divided by ``sqrt(periods_per_year)``). The caller is responsible for
    annualizing the result with ``sqrt(periods_per_year)`` when needed.

    Returns ``nan`` when there are too few observations to estimate.
    """
    if n_obs < 3:
        return float("nan")
    var = (1.0 - skew * sharpe + (excess_kurtosis + 2.0) / 4.0 * sharpe**2) / (n_obs - 1)
    var = max(var, 0.0)
    return float(np.sqrt(var))


def sharpe_se(returns: np.ndarray, periods_per_year: float = 252.0) -> float:
    """Annualized closed-form standard error of the Sharpe ratio from returns."""
    r = _clean(returns)
    if r.size < 3:
        return float("nan")
    per_period_sr = _annualized_sharpe(r, 1.0)
    per_se = sharpe_se_from_stats(per_period_sr, r.size, skewness(r), excess_kurtosis(r))
    return float(per_se * np.sqrt(periods_per_year))


def sharpe_variance(returns: np.ndarray, periods_per_year: float = 252.0) -> float:
    """Variance of the annualized Sharpe estimator (square of ``sharpe_se``)."""
    se = sharpe_se(returns, periods_per_year)
    return float(se * se) if not np.isnan(se) else float("nan")


def newey_west_se(returns: np.ndarray, max_lag: int | None = None) -> float:
    """Newey-West HAC standard error of the mean of a return series.

    Uses the Bartlett kernel with the standard automatic lag length
    ``4 * (n/100) ** (2/9)`` when ``max_lag`` is not supplied.
    """
    r = _clean(returns)
    n = r.size
    if n < 2:
        return float("nan")
    if max_lag is None:
        max_lag = round(4.0 * (n / 100.0) ** (2.0 / 9.0))
    max_lag = max(1, min(int(max_lag), n - 1))

    centered = r - np.mean(r)
    gamma0 = float(np.dot(centered, centered)) / n
    s2 = gamma0
    for lag in range(1, max_lag + 1):
        weight = 1.0 - lag / (max_lag + 1.0)
        gamma_lag = float(np.dot(centered[lag:], centered[:-lag])) / n
        s2 += 2.0 * weight * gamma_lag
    if s2 < 0.0:
        s2 = gamma0
    return float(np.sqrt(s2 / n))


__all__ = [
    "excess_kurtosis",
    "newey_west_se",
    "sharpe_se",
    "sharpe_se_from_stats",
    "sharpe_variance",
    "skewness",
]
