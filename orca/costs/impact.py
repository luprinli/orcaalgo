"""Market-impact calibration from OHLCV data.

- ``fit_sqrt_impact`` calibrates the square-root law ``|r| = sigma * eta *
  sqrt(V / ADV)`` via OLS through the origin, recovering ``eta`` — the
  ``VolumeImpactFactor`` coefficient in ``SlippageModel``.
- ``kyle_lambda`` calibrates Kyle's lambda ``dP = lambda * signed_flow``.
"""

from __future__ import annotations

import numpy as np


def fit_sqrt_impact(
    returns: np.ndarray,
    volume: np.ndarray,
    adv: float | None = None,
    sigma: float | None = None,
) -> float:
    """Fit the square-root impact coefficient ``eta``.

    Regresses ``|r| / sigma`` on ``sqrt(V / ADV)`` through the origin. ``eta`` is
    dimensionless and maps directly to ``SlippageModel.VolumeImpactFactor``.
    ``sigma`` is the ex-ante volatility; when omitted it is estimated as the
    sample standard deviation of ``returns``. Returns 0.0 when there is
    insufficient signal.
    """
    returns = np.asarray(returns, dtype=np.float64)
    volume = np.asarray(volume, dtype=np.float64)
    n = min(returns.size, volume.size)
    if n < 5:
        return 0.0
    r = returns[:n]
    v = volume[:n]
    if sigma is None:
        sigma = float(np.std(r, ddof=1))
    sigma = float(sigma)
    if sigma < 1e-12:
        return 0.0
    adv = float(adv) if adv is not None else float(np.mean(v))
    if adv <= 0:
        return 0.0

    x = np.sqrt(np.maximum(v, 0.0) / adv)
    y = np.abs(r) / sigma
    mask = np.isfinite(x) & np.isfinite(y) & (x > 0)
    if int(mask.sum()) < 2:
        return 0.0
    eta, *_ = np.linalg.lstsq(x[mask, None], y[mask], rcond=None)
    return float(np.clip(eta[0], 0.0, 100.0))


def kyle_lambda(price_change: np.ndarray, signed_volume: np.ndarray) -> float:
    """Kyle's lambda: slope of ``dP = lambda * signed_flow`` through the origin.

    ``signed_volume`` should carry the trade direction sign (buy positive, sell
    negative). Returns price impact per unit of signed volume, or 0.0 when
    undefined.
    """
    price_change = np.asarray(price_change, dtype=np.float64)
    signed_volume = np.asarray(signed_volume, dtype=np.float64)
    n = min(price_change.size, signed_volume.size)
    if n < 3:
        return 0.0
    x = signed_volume[:n]
    y = price_change[:n]
    mask = np.isfinite(x) & np.isfinite(y) & (x != 0)
    if int(mask.sum()) < 2:
        return 0.0
    lam, *_ = np.linalg.lstsq(x[mask, None], y[mask], rcond=None)
    return float(lam[0])


__all__ = ["fit_sqrt_impact", "kyle_lambda"]
