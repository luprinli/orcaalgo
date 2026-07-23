from __future__ import annotations

import numpy as np


def ewma_volatility(returns: np.ndarray, span: int = 20) -> float:
    returns = np.asarray(returns, dtype=np.float64)
    nan_mask = np.isnan(returns)
    if nan_mask.any():
        returns = returns[~nan_mask]
    if len(returns) < 2:
        return 0.0
    alpha = 2.0 / (span + 1)
    seed_window = min(len(returns), max(span, 5))
    ewmv = float(np.var(returns[:seed_window]))
    for r in returns[1:]:
        ewmv = alpha * r * r + (1 - alpha) * ewmv  # RiskMetrics: no mean correction
    return float(np.sqrt(max(ewmv, 0.0)))


def vol_adjusted_size(kelly_fraction: float, vol: float, baseline_vol: float, max_size: float) -> float:
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
