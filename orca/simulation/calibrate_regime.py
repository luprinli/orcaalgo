"""Regime-aware parameter calibration.

Estimates model parameters separately for each market regime
using real historical data segmented by a rule-based classifier.
"""

from __future__ import annotations

import json
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

import numpy as np

from orca.simulation.calibrate import load_real_candles
from orca.simulation.regime import (
    DEFAULT_REGIME_PARAMS,
    REGIME_CALM,
    REGIME_CRISIS,
    REGIME_HIGH_VOL,
    REGIME_NAMES,
    REGIME_TRENDING,
)


def classify_regime_rule_based(
    returns: np.ndarray,
    vix_values: np.ndarray | None = None,
    window: int = 21,
    vix_threshold_high: float = 25.0,
    vix_threshold_crisis: float = 35.0,
    ret_std_high: float = 0.015,
    ret_std_crisis: float = 0.030,
    trend_threshold: float = 0.005,
) -> np.ndarray:
    """Classify each day into a regime using rule-based thresholds.

    Returns:
        Array of regime labels (0-3) with same length as returns.
    """
    n = len(returns)
    labels = np.full(n, REGIME_CALM, dtype=np.int32)

    rolling_vol = np.zeros(n)
    for i in range(window, n):
        rolling_vol[i] = np.std(returns[max(0, i - window) : i])

    for i in range(n):
        vol = rolling_vol[i]
        ret = returns[i]

        if vol >= ret_std_crisis or (
            vix_values is not None and vix_values[i] >= vix_threshold_crisis
        ):
            labels[i] = REGIME_CRISIS
        elif vol >= ret_std_high or (
            vix_values is not None and vix_values[i] >= vix_threshold_high
        ):
            labels[i] = REGIME_HIGH_VOL
        elif abs(ret) > trend_threshold and vol >= 0.008:
            labels[i] = REGIME_TRENDING

    non_calm = labels != REGIME_CALM
    if non_calm.any():
        for i in range(1, n):
            if labels[i] == REGIME_CALM and labels[i - 1] == REGIME_TRENDING:
                labels[i] = REGIME_TRENDING

    return labels


def calibrate_per_regime(
    symbol: str,
    start: datetime | None = None,
    end: datetime | None = None,
    timeframe: str = "1d",
) -> dict[int, dict[str, Any]]:
    """Calibrate model parameters separately for each regime.

    Returns:
        Dict mapping regime_id -> parameter dict.
    """
    try:
        prices, _volumes, _timestamps = load_real_candles(symbol, start, end, timeframe)
    except Exception:
        return {k: dict(v) for k, v in DEFAULT_REGIME_PARAMS.items()}

    if len(prices) < 20:
        return {k: dict(v) for k, v in DEFAULT_REGIME_PARAMS.items()}

    closes = prices[:, 3].astype(np.float64)
    returns = np.diff(np.log(closes))
    returns = returns[np.isfinite(returns)]

    labels = classify_regime_rule_based(returns)

    regime_params: dict[int, dict[str, Any]] = {}
    for regime_id in [REGIME_CALM, REGIME_TRENDING, REGIME_HIGH_VOL, REGIME_CRISIS]:
        mask = labels[1:] == regime_id
        regime_returns = returns[mask]

        defaults = dict(DEFAULT_REGIME_PARAMS[regime_id])

        if len(regime_returns) < 5:
            regime_params[regime_id] = defaults
            continue

        mu = float(np.mean(regime_returns))
        sigma = float(np.std(regime_returns, ddof=1))

        big_moves = regime_returns[np.abs(regime_returns) > 3 * sigma]
        jump_intensity = len(big_moves) / max(1, len(regime_returns))
        jump_mean = float(np.mean(big_moves)) if len(big_moves) > 0 else 0.0
        jump_std = float(np.std(big_moves)) if len(big_moves) > 1 else 0.0

        regime_params[regime_id] = {
            **defaults,
            "mu": mu,
            "sigma": sigma,
            "jump_intensity": min(jump_intensity, 0.5),
            "jump_mean": jump_mean,
            "jump_std": jump_std,
            "trend_bias": defaults["trend_bias"] if abs(mu) < 0.002 else np.sign(mu) * 0.6,
            "n_observations": int(mask.sum()),
        }

    return regime_params


def save_regime_params(
    params: dict[int, dict[str, Any]],
    symbol: str,
    output_dir: str | Path,
) -> Path:
    """Save calibrated regime parameters to JSON."""
    output_dir = Path(output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    output = {
        "symbol": symbol,
        "calibrated_at": datetime.now(UTC).isoformat(),
        "calibration_method": "rule_based_segmentation",
        "regimes": {REGIME_NAMES[k]: {"regime_id": k, **v} for k, v in sorted(params.items())},
    }

    path = output_dir / f"regime_params_{symbol}.json"
    with open(path, "w") as f:
        json.dump(output, f, indent=2, default=str)
    return path


def load_regime_params(symbol: str, config_dir: str | Path) -> dict[int, dict[str, Any]]:
    """Load previously saved regime parameters."""
    config_dir = Path(config_dir)
    path = config_dir / f"regime_params_{symbol}.json"
    if not path.exists():
        return {k: dict(v) for k, v in DEFAULT_REGIME_PARAMS.items()}

    with open(path) as f:
        data = json.load(f)

    params: dict[int, dict[str, Any]] = {}
    for _name, entry in data.get("regimes", {}).items():
        regime_id = entry.get("regime_id", 0)
        params[regime_id] = {k: v for k, v in entry.items() if k in DEFAULT_REGIME_PARAMS[0]}
        for k, v in DEFAULT_REGIME_PARAMS.get(regime_id, {}).items():
            params[regime_id].setdefault(k, v)

    return params
