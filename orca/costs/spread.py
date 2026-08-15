"""Bid-ask spread estimators from OHLC data.

Both estimators recover the spread in price units from daily (or bar-level)
high/low/close series, using no order-book data. They feed ``SpreadBps`` in the
slippage model (HP #9).

- Corwin & Schultz (2012): high-low range decomposition across two adjacent bars.
- Roll (1984): negative first-order autocovariance of price changes.
"""

from __future__ import annotations

import numpy as np


def corwin_schultz(high: np.ndarray, low: np.ndarray) -> np.ndarray:
    """Corwin-Schultz (2012) high-low spread estimator.

    Returns a per-bar spread estimate (price units). Bars where the estimator is
    undefined (negative ``alpha``) yield ``nan``. The caller typically reduces
    with a robust location statistic (e.g. median).
    """
    high = np.asarray(high, dtype=np.float64)
    low = np.asarray(low, dtype=np.float64)
    n = min(high.size, low.size)
    if n < 2:
        return np.full(n, np.nan)

    log_hl = np.log(high[:n] / low[:n])
    beta = log_hl**2

    # Two-bar high/low ratio.
    two_high = np.maximum(high[1:n], high[: n - 1])
    two_low = np.minimum(low[1:n], low[: n - 1])
    gamma = np.log(two_high / two_low) ** 2

    const = 3.0 - 2.0 * np.sqrt(2.0)
    alpha = (np.sqrt(2.0 * beta[1:n]) - np.sqrt(beta[1:n])) / const - np.sqrt(gamma / const)

    spread = np.full(n, np.nan)
    valid = alpha > 0
    a = alpha[valid]
    spread[1:n][valid] = 2.0 * (np.exp(a) - 1.0) / (1.0 + np.exp(a))
    return spread


def roll_spread(close: np.ndarray) -> float:
    """Roll (1984) spread estimator from serial covariance of price changes.

    Returns the spread in price units, or ``nan`` when the first-order
    autocovariance is non-negative (no mean reversion, estimator undefined).
    """
    close = np.asarray(close, dtype=np.float64)
    if close.size < 3:
        return float("nan")
    dp = np.diff(close)
    cov = np.mean((dp[1:] - np.mean(dp[1:])) * (dp[:-1] - np.mean(dp[:-1])))
    if cov >= 0:
        return float("nan")
    return float(2.0 * np.sqrt(-cov))


__all__ = ["corwin_schultz", "roll_spread"]
