"""Feature computation specification — shared semantic between Python training and Go inference.

Every feature defined here must have an equivalent computation in
internal/ml/feature_store.go. This prevents train/serve skew.

Feature indices are fixed and must match the order in config.FEATURE_NAMES.
"""

from __future__ import annotations

import math
from datetime import datetime

import numpy as np

from orca.ml.config import FEATURE_NAMES


def compute_price_features(
    closes: np.ndarray,
    highs: np.ndarray,
    lows: np.ndarray,
) -> dict[int, float]:
    """Compute price-based features at index -1 (most recent bar).

    Args:
        closes: Array of close prices, [0] = oldest, [-1] = current.
        highs: Array of high prices.
        lows: Array of low prices.

    Returns:
        Dict mapping feature index to value.
    """
    n = len(closes)
    if n < 21:
        raise ValueError(f"Need at least 21 bars, got {n}")

    current = closes[-1]

    features: dict[int, float] = {}

    # 0: ret1
    features[0] = math.log(current / closes[-2]) if closes[-2] > 0 else 0.0

    # 1: ret5
    features[1] = math.log(current / closes[-6]) if closes[-6] > 0 else 0.0

    # 2: ret20
    features[2] = math.log(current / closes[-21]) if closes[-21] > 0 else 0.0

    # 3: volatility20 — 20-period EWMA volatility of log returns
    log_returns = np.diff(np.log(closes[-21:]))
    alpha = 2.0 / 21.0  # span = 20
    ewma_var = log_returns[0] ** 2
    for r in log_returns[1:]:
        ewma_var = alpha * (r**2) + (1 - alpha) * ewma_var
    features[3] = float(math.sqrt(max(ewma_var, 1e-12)))

    # 4: atr_ratio — ATR / Close
    tr = np.maximum(
        highs[-15:] - lows[-15:],
        np.abs(highs[-15:] - np.roll(closes[-15:], 1)),
    )
    tr[0] = highs[-15] - lows[-15]
    atr = float(np.mean(tr[-14:]))
    features[4] = atr / current if current > 0 else 0.0

    return features


def compute_indicator_features(
    closes: np.ndarray,
    volumes: np.ndarray,
    highs: np.ndarray,
    lows: np.ndarray,
    cvd_divergence: float = 0.0,
    spread_pct: float = 0.0,
) -> dict[int, float]:
    """Compute indicator-based features.

    Args:
        closes: Close prices, [-1] = current.
        volumes: Volume data.
        highs: High prices.
        lows: Low prices.
        cvd_divergence: Pre-computed CVD divergence flag.
        spread_pct: Pre-computed (ask - bid) / close.

    Returns:
        Dict mapping feature index to value.
    """
    features: dict[int, float] = {}

    # 5: rsi14
    features[5] = float(_compute_rsi(closes, 14))

    # 6: macd_hist — MACD histogram = MACD line - signal line
    ema12 = _ema(closes, 12)
    ema26 = _ema(closes, 26)
    macd_line = ema12[-1] - ema26[-1]
    macd_series = np.array([ema12[i] - ema26[i] for i in range(len(ema12))])
    signal_line = float(np.mean(macd_series[-9:])) if len(macd_series) >= 9 else macd_line
    features[6] = macd_line - signal_line

    # 7: adx14
    features[7] = float(_compute_adx(highs, lows, closes, 14))

    # 8: bb_percent_b — Bollinger %B
    sma20 = float(np.mean(closes[-20:]))
    std20 = float(np.std(closes[-20:], ddof=0))
    if std20 > 0:
        features[8] = (closes[-1] - (sma20 - 2 * std20)) / (4 * std20)
    else:
        features[8] = 0.5

    # 9: volume_ratio — Volume / 20-period average volume
    avg_volume = float(np.mean(volumes[-21:-1])) if len(volumes) >= 21 else volumes[-1]
    features[9] = volumes[-1] / avg_volume if avg_volume > 0 else 1.0

    # 10: cvd_divergence
    features[10] = cvd_divergence

    # 11: spread_pct
    features[11] = spread_pct

    return features


def compute_regime_features(
    hmm_alpha: tuple[float, float, float, float] | None,
    hmm_confidence: float = 0.0,
) -> dict[int, float]:
    """Compute HMM regime features.

    Args:
        hmm_alpha: 4-element alpha vector from HMM forward algorithm.
        hmm_confidence: HMM confidence value.

    Returns:
        Dict mapping feature index to value.
    """
    features: dict[int, float] = {}

    if hmm_alpha is not None and len(hmm_alpha) == 4:
        for i, val in enumerate(hmm_alpha):
            features[12 + i] = float(val)
    else:
        for i in range(4):
            features[12 + i] = 0.0

    features[16] = hmm_confidence
    return features


def compute_signal_features(
    signal_type: int,
    signal_strength: float,
) -> dict[int, float]:
    """Compute signal-specific features.

    Args:
        signal_type: Integer enum for strategy type (0-9).
        signal_strength: Conviction measure (z-score distance, EMA distance, etc.).

    Returns:
        Dict mapping feature index to value.
    """
    return {
        17: float(signal_type),
        18: signal_strength,
    }


def compute_time_features(ts: datetime) -> dict[int, float]:
    """Compute cyclic time features.

    Args:
        ts: Timestamp of the current bar.

    Returns:
        Dict mapping feature index to value.
    """
    hour = ts.hour + ts.minute / 60.0 + ts.second / 3600.0
    return {
        19: float(math.sin(2 * math.pi * hour / 24.0)),
        20: float(math.cos(2 * math.pi * hour / 24.0)),
    }


def compute_full_feature_vector(
    closes: np.ndarray,
    highs: np.ndarray,
    lows: np.ndarray,
    volumes: np.ndarray,
    ts: datetime,
    hmm_alpha: tuple[float, float, float, float] | None = None,
    hmm_confidence: float = 0.0,
    signal_type: int = 0,
    signal_strength: float = 0.0,
    cvd_divergence: float = 0.0,
    spread_pct: float = 0.0,
) -> np.ndarray:
    """Compute the full 21-dim feature vector.

    This is the canonical implementation. Go's feature_store.go must produce
    bit-identical results for the same inputs.

    Returns:
        numpy array of shape (21,), float64.
    """
    features = np.zeros(len(FEATURE_NAMES), dtype=np.float64)

    price_f = compute_price_features(closes, highs, lows)
    for idx, val in price_f.items():
        features[idx] = val

    indicator_f = compute_indicator_features(
        closes, volumes, highs, lows, cvd_divergence, spread_pct,
    )
    for idx, val in indicator_f.items():
        features[idx] = val

    regime_f = compute_regime_features(hmm_alpha, hmm_confidence)
    for idx, val in regime_f.items():
        features[idx] = val

    signal_f = compute_signal_features(signal_type, signal_strength)
    for idx, val in signal_f.items():
        features[idx] = val

    time_f = compute_time_features(ts)
    for idx, val in time_f.items():
        features[idx] = val

    return features


def feature_vector_to_dict(fv: np.ndarray) -> dict[str, float]:
    """Convert a feature vector to a named dict for logging/debugging."""
    return {FEATURE_NAMES[i]: float(fv[i]) for i in range(len(FEATURE_NAMES))}


# ── Internal indicator helpers ────────────────────────────────────────────────

def _compute_rsi(closes: np.ndarray, period: int = 14) -> float:
    """Compute RSI for the most recent bar."""
    if len(closes) < period + 1:
        return 50.0
    deltas = np.diff(closes[-(period + 1):])
    gains = np.where(deltas > 0, deltas, 0.0)
    losses = np.where(deltas < 0, -deltas, 0.0)
    avg_gain = float(np.mean(gains))
    avg_loss = float(np.mean(losses))
    if avg_loss < 1e-12:
        return 100.0
    rs = avg_gain / avg_loss
    return 100.0 - 100.0 / (1.0 + rs)


def _ema(series: np.ndarray, period: int) -> np.ndarray:
    """Compute EMA of a series."""
    alpha = 2.0 / (period + 1.0)
    result = np.zeros_like(series)
    result[0] = series[0]
    for i in range(1, len(series)):
        result[i] = alpha * series[i] + (1 - alpha) * result[i - 1]
    return result


def _compute_adx(
    highs: np.ndarray,
    lows: np.ndarray,
    closes: np.ndarray,
    period: int = 14,
) -> float:
    """Compute ADX for the most recent bar."""
    n = len(highs)
    if n < period * 2:
        return 25.0

    tr = np.zeros(n)
    plus_dm = np.zeros(n)
    minus_dm = np.zeros(n)

    for i in range(1, n):
        tr[i] = max(
            highs[i] - lows[i],
            abs(highs[i] - closes[i - 1]),
            abs(lows[i] - closes[i - 1]),
        )
        up_move = highs[i] - highs[i - 1]
        down_move = lows[i - 1] - lows[i]
        plus_dm[i] = up_move if up_move > down_move and up_move > 0 else 0.0
        minus_dm[i] = down_move if down_move > up_move and down_move > 0 else 0.0

    atr = _ema(tr, period)
    smoothed_plus = _ema(plus_dm, period)
    smoothed_minus = _ema(minus_dm, period)

    plus_di = 100.0 * smoothed_plus[-1] / atr[-1] if atr[-1] > 0 else 0.0
    minus_di = 100.0 * smoothed_minus[-1] / atr[-1] if atr[-1] > 0 else 0.0

    dx_sum = abs(plus_di - minus_di) / (plus_di + minus_di) if (plus_di + minus_di) > 0 else 0.0
    return float(dx_sum * 100.0)
