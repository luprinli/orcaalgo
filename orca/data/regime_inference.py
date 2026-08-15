"""Regime inference from candle data.

Classifies each trading day into one of 4 HMM regimes (Calm/Trending/HighVol/Crisis)
using return-based features (volatility, trend strength, drawdown) and inserts results
into the regime_logs table for use by the RiskPipeline regime gating system.
"""

from __future__ import annotations

from typing import Any

import numpy as np


def infer_regimes(
    close_prices: np.ndarray,
    timestamps: np.ndarray,
    lookback: int = 20,
) -> tuple[np.ndarray, np.ndarray]:
    """Classify each day into a regime based on return characteristics.

    Args:
        close_prices: 1D array of daily close prices
        timestamps: 1D array of corresponding datetime64 timestamps
        lookback: rolling window for volatility and trend computation

    Returns:
        (regime_labels, confidences) — both same length as inputs
        regime_labels: int8 array of regime states (0=Calm,1=Trending,2=HighVol,3=Crisis)
        confidences: float64 array of classification confidence [0,1]
    """
    n = len(close_prices)
    if n < lookback + 1:
        return np.zeros(n, dtype=np.int8), np.zeros(n, dtype=np.float64)

    log_returns = np.diff(np.log(close_prices))
    regimes = np.zeros(n, dtype=np.int8)
    confidences = np.zeros(n, dtype=np.float64)

    for i in range(lookback, n):
        window_returns = log_returns[i - lookback:i]
        ann_vol = np.std(window_returns) * np.sqrt(252)
        total_return = close_prices[i] / close_prices[i - lookback] - 1.0

        # Trend strength is the window-scaled information ratio: the lookback-day
        # return divided by the lookback-day volatility. Dividing by ann_vol
        # instead understated it by sqrt(252/lookback) (~3.5x for lookback=20),
        # which made the "Trending" (state 1) regime effectively unreachable and
        # regime-gated strategies (trend_following, dragon_trend) fully dormant.
        window_vol = np.std(window_returns) * np.sqrt(lookback)
        trend_strength = total_return / max(window_vol, 0.01)

        if ann_vol > 0.45:
            regimes[i], confidences[i] = 3, min(0.95, ann_vol / 0.60)
        elif ann_vol > 0.25:
            regimes[i], confidences[i] = 2, min(0.95, ann_vol / 0.40)
        elif abs(trend_strength) > 1.5:
            regimes[i], confidences[i] = 1, min(0.95, abs(trend_strength) / 3.0)
        else:
            regimes[i], confidences[i] = 0, 0.75 + max(0.0, 1.0 - ann_vol / 0.15) * 0.2

    return regimes, confidences


def _max_drawdown(prices: np.ndarray) -> float:
    """Compute max drawdown of a price series."""
    peak = np.maximum.accumulate(prices)
    return float(np.max((peak - prices) / peak))


def build_regime_logs(
    prices_by_symbol: dict[str, tuple[np.ndarray, np.ndarray]],
    lookback: int = 20,
) -> list[dict[str, Any]]:
    """Build regime log records for all symbols.

    Args:
        prices_by_symbol: {symbol: (close_prices, timestamps)}
        lookback: rolling window for computation

    Returns:
        List of dicts with keys: timestamp, symbol, hmm_state, confidence
    """
    logs = []
    for symbol, (prices, timestamps) in prices_by_symbol.items():
        labels, confs = infer_regimes(prices, timestamps, lookback)
        for i in range(len(labels)):
            if confs[i] > 0:
                logs.append({
                    "timestamp": timestamps[i],
                    "symbol": symbol,
                    "hmm_state": int(labels[i]),
                    "confidence": float(confs[i]),
                })
    return logs
