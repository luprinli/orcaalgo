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
    for r in returns[seed_window:]:
        ewmv = alpha * r * r + (1 - alpha) * ewmv
    return float(np.sqrt(max(ewmv, 0.0)))
