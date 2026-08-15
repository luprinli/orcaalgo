"""Per-symbol cost calibration combining spread and impact estimators.

Produces the ``{spread_bps, roll_spread_bps, impact_eta}`` coefficients that seed
``SlippageModel`` (``internal/backtest/slippage.go``). Spreads are converted from
price units to basis points using the mean close.
"""

from __future__ import annotations

from typing import Any

import numpy as np

from orca.costs.impact import fit_sqrt_impact
from orca.costs.spread import corwin_schultz, roll_spread


def calibrate_symbol_costs(
    high: np.ndarray,
    low: np.ndarray,
    close: np.ndarray,
    volume: np.ndarray,
    timeframe: str = "1d",
) -> dict[str, Any]:
    """Calibrate spread and impact coefficients for a single symbol.

    Args:
        high, low, close, volume: OHLCV arrays aligned in time (ascending).
        timeframe: Bar timeframe label (informational only).

    Returns:
        Dict with ``spread_bps`` (Corwin-Schultz median), ``roll_spread_bps``,
        ``impact_eta``, plus the raw per-bar spread series and observation count.
    """
    high = np.asarray(high, dtype=np.float64)
    low = np.asarray(low, dtype=np.float64)
    close = np.asarray(close, dtype=np.float64)
    volume = np.asarray(volume, dtype=np.float64)

    close_ref = float(np.median(close)) if close.size else float("nan")
    per_bar_spread = corwin_schultz(high, low)
    valid_spread = per_bar_spread[np.isfinite(per_bar_spread)]
    spread_median = float(np.median(valid_spread)) if valid_spread.size else float("nan")

    spread_bps = (
        spread_median / close_ref * 1e4
        if (np.isfinite(spread_median) and close_ref > 0)
        else float("nan")
    )
    roll = roll_spread(close)
    roll_bps = roll / close_ref * 1e4 if (np.isfinite(roll) and close_ref > 0) else float("nan")

    log_close = np.log(np.maximum(close, 1e-12))
    returns = np.diff(log_close)
    bar_volume = volume[1:] if volume.size > 1 else volume
    eta = fit_sqrt_impact(returns, bar_volume, adv=float(np.mean(volume)))

    return {
        "timeframe": timeframe,
        "n_bars": int(close.size),
        "spread_bps": spread_bps,
        "roll_spread_bps": roll_bps,
        "impact_eta": float(eta),
        "mean_close": close_ref,
        "per_bar_spread": [float(x) if np.isfinite(x) else None for x in per_bar_spread],
    }


__all__ = ["calibrate_symbol_costs"]
