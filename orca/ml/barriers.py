"""Triple-barrier labeling for supervised financial ML.

Implements the Lopez de Prado (2018) triple-barrier method for generating
labels from price paths. Each trade entry produces a label:

  +1 : upper barrier hit first (win)
  -1 : lower barrier hit first (loss)
   0 : time barrier hit first (time-based exit, use return sign)

For meta-labeling, labels are binarized: target = 1 if label == +1 else 0.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass
from enum import IntEnum

import numpy as np

from orca.ml.config import (
    BARRIER_MIN_RETURN,
    BARRIER_PROFIT_FACTOR,
    BARRIER_STOP_FACTOR,
    BARRIER_TIME_HORIZON,
)

logger = logging.getLogger("orca.ml.barriers")


class BarrierLabel(IntEnum):
    UPPER_HIT = 1  # win — upper (profit) barrier hit first
    LOWER_HIT = -1  # loss — lower (stop) barrier hit first
    TIME_HIT = 0  # time barrier — label determined by return at barrier


@dataclass(frozen=True)
class BarrierConfig:
    """Configuration for triple-barrier labeling."""

    profit_factor: float = BARRIER_PROFIT_FACTOR
    stop_factor: float = BARRIER_STOP_FACTOR
    time_horizon: int = BARRIER_TIME_HORIZON
    min_return: float = BARRIER_MIN_RETURN


@dataclass(frozen=True)
class BarrierResult:
    """Result of triple-barrier labeling for a single trade."""

    label: BarrierLabel
    exit_bar: int  # bar index where barrier was hit (relative to entry)
    exit_price: float  # price at exit
    return_pct: float  # percentage return at exit
    hit_barrier: str  # "upper", "lower", or "time"


def compute_barriers(
    entry_price: float,
    sigma: float,
    config: BarrierConfig | None = None,
) -> tuple[float, float]:
    """Compute upper and lower barrier prices.

    Returns:
        (upper_barrier, lower_barrier)
    """
    if config is None:
        config = BarrierConfig()
    upper = entry_price * (1.0 + config.profit_factor * sigma)
    lower = entry_price * (1.0 - config.stop_factor * sigma)
    return upper, lower


def triple_barrier_label(
    entry_price: float,
    prices: np.ndarray,
    entry_idx: int,
    config: BarrierConfig | None = None,
) -> BarrierResult:
    """Apply triple-barrier labeling to a single trade entry.

    Args:
            entry_price: Price at trade entry.
        prices: Array of prices following entry. prices[entry_idx] = entry_price.
        entry_idx: Index of the entry bar in the prices array.
        config: Barrier configuration.

    Returns:
        BarrierResult with label, exit bar, exit price, return, and hit barrier.
    """
    if config is None:
        config = BarrierConfig()
    if entry_price <= 0:
        raise ValueError(f"entry_price must be positive, got {entry_price}")

    # Estimate sigma from recent returns (20-bar lookback)
    if entry_idx >= 20:
        lookback = prices[max(0, entry_idx - 20) : entry_idx + 1]
        log_returns = np.diff(np.log(lookback[lookback > 0]))
        sigma = float(np.std(log_returns)) if len(log_returns) > 0 else 0.01
    else:
        sigma = 0.01

    upper, lower = compute_barriers(entry_price, sigma, config)
    time_barrier_idx = entry_idx + config.time_horizon
    max_idx = min(time_barrier_idx, len(prices) - 1)

    for i in range(entry_idx + 1, max_idx + 1):
        price = prices[i]
        if price >= upper:
            ret = (price - entry_price) / entry_price
            return BarrierResult(
                label=BarrierLabel.UPPER_HIT,
                exit_bar=i - entry_idx,
                exit_price=price,
                return_pct=float(ret),
                hit_barrier="upper",
            )
        if price <= lower:
            ret = (price - entry_price) / entry_price
            return BarrierResult(
                label=BarrierLabel.LOWER_HIT,
                exit_bar=i - entry_idx,
                exit_price=price,
                return_pct=float(ret),
                hit_barrier="lower",
            )

    # Time barrier hit — label by return sign
    final_price = prices[max_idx]
    ret = (final_price - entry_price) / entry_price

    if abs(ret) < config.min_return:
        label = BarrierLabel.TIME_HIT  # flat — neutral
    elif ret > 0:
        label = BarrierLabel.UPPER_HIT
    else:
        label = BarrierLabel.LOWER_HIT

    return BarrierResult(
        label=label,
        exit_bar=max_idx - entry_idx,
        exit_price=final_price,
        return_pct=float(ret),
        hit_barrier="time",
    )


def batch_triple_barrier_labels(
    entry_prices: np.ndarray,
    entry_indices: np.ndarray,
    prices: np.ndarray,
    config: BarrierConfig | None = None,
) -> list[BarrierResult]:
    """Apply triple-barrier labeling to a batch of trade entries.

    Args:
        entry_prices: Array of entry prices, shape (n_trades,).
        entry_indices: Array of entry bar indices, shape (n_trades,).
        prices: Full price series, shape (n_bars,).
        config: Barrier configuration.

    Returns:
        List of BarrierResult, one per trade.
    """
    results = []
    for i in range(len(entry_prices)):
        result = triple_barrier_label(
            entry_prices[i],
            prices,
            int(entry_indices[i]),
            config,
        )
        results.append(result)
    return results


def label_to_binary(label: BarrierLabel) -> int:
    """Convert barrier label to binary target for meta-labeling.

    +1 → 1 (win), -1 → 0 (loss), 0 → 0 (neutral/time → loss)
    """
    return 1 if label == BarrierLabel.UPPER_HIT else 0


def compute_sigma_from_prices(
    prices: np.ndarray,
    lookback: int = 20,
) -> np.ndarray:
    """Compute rolling sigma (standard deviation of log returns).

    Returns:
        Array of sigma values, shape same as prices. First lookback values are NaN.
    """
    log_returns = np.diff(np.log(np.maximum(prices, 1e-12)))
    sigma = np.full(len(prices), np.nan)

    for i in range(lookback, len(prices)):
        window = log_returns[i - lookback : i]
        sigma[i] = float(np.std(window))

    return sigma
