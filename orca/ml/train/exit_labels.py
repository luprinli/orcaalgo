"""Path-dependent labeling for exit optimization.

For each exited trade, generates training labels by looking forward k bars
to determine whether holding would have been better or exiting would have
been better. This produces a regression target for the exit urgency model.

Labels:
  0.0 = should have held (price moved favorably after the exit bar)
  0.5 = neutral (price was flat)
  1.0 = should have exited (price moved adversely after the exit bar)

The exit model learns to predict urgency: when to tighten stops early
vs when to widen them to ride trends longer.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass

import numpy as np

logger = logging.getLogger("orca.ml.train.exit_labels")


@dataclass(frozen=True)
class ExitLabel:
    urgency: float          # 0.0 (hold) to 1.0 (exit now)
    forward_return: float   # return over the look-forward window
    exit_bar: int           # bar index of the exit
    label_type: str         # "favorable", "adverse", "neutral"


def path_dependent_label(
    prices: np.ndarray,
    exit_idx: int,
    exit_price: float,
    entry_price: float,
    look_forward: int = 5,
    adverse_threshold: float = -0.002,
    favorable_threshold: float = 0.002,
) -> ExitLabel:
    """Generate a path-dependent exit label.

    Looks forward from the exit bar to determine if exiting was correct.

    Args:
        prices: Full price series.
        exit_idx: Index of the exit bar.
        exit_price: Price at exit.
        entry_price: Price at trade entry.
        look_forward: Bars to look ahead.
        adverse_threshold: Return below this = adverse (should have exited).
        favorable_threshold: Return above this = favorable (should have held).

    Returns:
        ExitLabel with urgency score.
    """
    if exit_idx + look_forward >= len(prices):
        look_forward = len(prices) - exit_idx - 1
    if look_forward <= 0:
        return ExitLabel(urgency=0.5, forward_return=0.0, exit_bar=exit_idx, label_type="neutral")

    forward_price = prices[exit_idx + look_forward]
    if exit_price > 0:
        forward_return = (forward_price - exit_price) / exit_price
    else:
        forward_return = 0.0

    if forward_return <= adverse_threshold:
        urgency = 1.0
        label_type = "adverse"
    elif forward_return >= favorable_threshold:
        urgency = 0.0
        label_type = "favorable"
    else:
        urgency = 0.5
        label_type = "neutral"

    return ExitLabel(
        urgency=urgency,
        forward_return=forward_return,
        exit_bar=exit_idx,
        label_type=label_type,
    )


def build_exit_features(
    entry_price: float,
    current_price: float,
    current_stop: float,
    high_since_entry: float,
    low_since_entry: float,
    bars_since_entry: int,
    atr: float,
    vol_at_entry: float,
    vol_current: float,
    hmm_state: int,
    cvd_trend: float,
    volume_trend: float,
    adx: float,
    hour: float,
    signal_confidence: float,
) -> np.ndarray:
    """Build 12-dim feature vector for exit optimization.

    Returns:
        Feature vector, shape (12,).
    """
    features = np.zeros(12)
    if entry_price > 0 and atr > 0:
        pnl = (current_price - entry_price) / entry_price
        features[0] = pnl / max(atr / entry_price, 1e-6)  # PnL in ATR units
        features[1] = (current_stop - entry_price) / entry_price / max(atr / entry_price, 1e-6)
    features[2] = float(bars_since_entry) / 100.0
    features[3] = vol_current / max(vol_at_entry, 1e-6) - 1.0
    features[4] = float(hmm_state) / 3.0
    features[5] = cvd_trend
    features[6] = volume_trend
    features[7] = adx / 50.0
    if entry_price > 0:
        mae = (entry_price - low_since_entry) / entry_price  # long
        mfe = (high_since_entry - entry_price) / entry_price
        features[8] = mae / max(atr / entry_price, 1e-6)
        features[9] = mfe / max(atr / entry_price, 1e-6)
    features[10] = np.sin(2 * np.pi * hour / 24.0)
    features[11] = signal_confidence
    return features


def batch_exit_labels(
    trades: list[dict],
    prices_map: dict[str, np.ndarray],
    look_forward: int = 5,
) -> tuple[np.ndarray, np.ndarray]:
    """Generate labeled training data for exit optimization from trade history.

    Args:
        trades: List of trade dicts with keys: symbol, entry_price, exit_price,
                entry_time, exit_time, pnl, stop_loss, hmm_regime.
        prices_map: Dict of {symbol: price_series}.
        look_forward: Bars to look ahead after exit.

    Returns:
        (X, y) — features (n_samples, 12) and labels (n_samples,).
    """
    X_list = []
    y_list = []

    for trade in trades:
        symbol = trade.get("symbol", "")
        prices = prices_map.get(symbol)
        if prices is None:
            continue

        entry_price = trade.get("entry_price", 0)
        exit_price = trade.get("exit_price", 0)
        if entry_price <= 0 or exit_price <= 0:
            continue

        label_result = path_dependent_label(
            prices, len(prices) // 2, exit_price, entry_price, look_forward,
        )

        features = build_exit_features(
            entry_price=entry_price,
            current_price=exit_price,
            current_stop=trade.get("stop_loss", entry_price * 0.95),
            high_since_entry=entry_price * 1.05,
            low_since_entry=entry_price * 0.95,
            bars_since_entry=trade.get("bars_held", 10),
            atr=trade.get("atr", 0.01 * entry_price),
            vol_at_entry=0.01,
            vol_current=0.01,
            hmm_state=trade.get("hmm_regime", 1),
            cvd_trend=0.0,
            volume_trend=0.0,
            adx=trade.get("adx", 25.0),
            hour=12.0,
            signal_confidence=trade.get("confidence", 0.5),
        )

        X_list.append(features)
        y_list.append(label_result.urgency)

    if not X_list:
        return np.zeros((0, 12)), np.zeros(0)

    return np.array(X_list), np.array(y_list)
