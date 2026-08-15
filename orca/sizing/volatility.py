from __future__ import annotations

import numpy as np

from orca.math.ewma import ewma_volatility

__all__ = ["diversification_scaling", "ewma_volatility", "vol_adjusted_size"]


def vol_adjusted_size(
    kelly_fraction: float, vol: float, baseline_vol: float, max_size: float
) -> float:
    if baseline_vol <= 0:
        return min(kelly_fraction, max_size)
    vol_ratio = vol / baseline_vol
    adj = 1.0 / vol_ratio if vol_ratio > 0 else 1.0
    adj = max(0.5, min(2.0, adj))
    return min(max_size, kelly_fraction * adj)


def diversification_scaling(num_positions: int, avg_correlation: float) -> float:
    if num_positions <= 1:
        return 1.0
    effective_n = num_positions * (1 - avg_correlation) + avg_correlation
    if effective_n <= 0:
        return 0.25
    scaling = 1.0 / np.sqrt(effective_n)
    return max(0.25, min(1.0, float(scaling)))
