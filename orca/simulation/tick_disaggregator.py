"""Tick disaggregation module.

Converts synthetic 1-minute candles into realistic tick-level data
using Brownian Bridge intra-minute price paths, bid-ask spread,
and volume distribution.
"""

from __future__ import annotations

from pathlib import Path
from typing import Any

import numpy as np
import pandas as pd


def _brownian_bridge(
    open_price: float,
    high: float,
    low: float,
    close: float,
    n_ticks: int,
    rng: np.random.Generator,
) -> np.ndarray:
    """Generate intra-minute price path using Brownian Bridge.

    The path starts at open, ends at close, and is constrained to
    stay within [low * (1-eps), high * (1+eps)].

    Args:
        open_price: Opening price at start of minute.
        high: High of the 1-minute candle.
        low: Low of the 1-minute candle.
        close: Closing price at end of minute.
        n_ticks: Number of intermediate ticks.
        rng: Numpy random generator.

    Returns:
        Array of n_ticks prices.
    """
    if n_ticks < 2:
        return np.array([close])

    t = np.linspace(0, 1, n_ticks)
    dt = t[1] - t[0]

    dW = rng.normal(0, np.sqrt(dt), n_ticks - 1)
    W = np.zeros(n_ticks)
    W[1:] = np.cumsum(dW)

    bridge = open_price + (close - open_price) * t + W - t * W[-1]

    scale = 0.02 * open_price
    bridge += rng.normal(0, scale * 0.01, n_ticks)

    floor = low * 0.9995
    ceil = high * 1.0005
    bridge = np.clip(bridge, floor, ceil)

    bridge[-1] = close
    bridge[0] = open_price

    return bridge


def _distribute_volume(
    total_volume: int,
    n_ticks: int,
    profile: str = "sine",
    rng: np.random.Generator | None = None,
) -> np.ndarray:
    """Distribute total volume across ticks using a profile.

    sine profile = higher volume at open/close (U-shaped).
    """
    if rng is None:
        rng = np.random.default_rng()
    t = np.linspace(0, 1, n_ticks)
    if profile == "sine":
        weights = np.cos(np.pi * t) ** 2 + 0.1
    elif profile == "flat":
        weights = np.ones(n_ticks)
    elif profile == "u_shaped":
        weights = np.cos(np.pi * t) ** 2 + 0.3
    else:
        weights = np.ones(n_ticks)

    weights = weights / weights.sum()
    volumes_f = weights * total_volume
    volumes = np.floor(volumes_f).astype(np.int64)

    remainder = int(total_volume - volumes.sum())
    if remainder > 0:
        idx = rng.choice(n_ticks, size=min(remainder, n_ticks), replace=False)
        np.add.at(volumes, idx, 1)

    return volumes


def disaggregate_1m_to_ticks(
    candle_df: pd.DataFrame,
    ticks_per_minute: int = 60,
    spread_bps: float = 0.5,
    volume_profile: str = "sine",
    seed: int | None = None,
) -> pd.DataFrame:
    """Convert a DataFrame of 1-minute candles into tick-level data.

    Args:
        candle_df: DataFrame with columns [time, open, high, low, close, volume, symbol].
        ticks_per_minute: Number of ticks to generate per 1-minute candle.
        spread_bps: Bid-ask spread in basis points (default 0.5 bps = 0.005%).
        volume_profile: Volume distribution shape ('sine', 'flat', 'u_shaped').
        seed: Random seed for reproducibility.

    Returns:
        DataFrame with columns: timestamp_ms, price, bid, ask, volume, symbol, generation_id.
    """
    rng = np.random.default_rng(seed)

    all_rows: list[dict[str, Any]] = []

    required_cols = {"time", "open", "high", "low", "close", "volume", "symbol"}
    if not required_cols.issubset(set(candle_df.columns)):
        missing = required_cols - set(candle_df.columns)
        raise ValueError(f"Missing required columns: {missing}")

    spread_fraction = spread_bps / 10000.0

    for _, row in candle_df.iterrows():
        candle_time = pd.Timestamp(row["time"])
        open_p = float(row["open"])
        high_p = float(row["high"])
        low_p = float(row["low"])
        close_p = float(row["close"])
        vol = int(row["volume"])
        symbol = str(row["symbol"])

        prices = _brownian_bridge(open_p, high_p, low_p, close_p, ticks_per_minute, rng)
        volumes = _distribute_volume(vol, ticks_per_minute, volume_profile, rng)

        for tick_idx in range(ticks_per_minute):
            ts_offset = int((tick_idx / ticks_per_minute) * 60_000)
            price = float(prices[tick_idx])

            half_spread = spread_fraction / 2.0
            bid = price * (1.0 - half_spread)
            ask = price * (1.0 + half_spread)

            tick_vol = int(volumes[tick_idx])
            if tick_vol <= 0:
                continue

            all_rows.append({
                "timestamp_ms": int(candle_time.timestamp() * 1000) + ts_offset,
                "price": price,
                "bid": bid,
                "ask": ask,
                "volume": tick_vol,
                "symbol": symbol,
            })

    if not all_rows:
        return pd.DataFrame(columns=["timestamp_ms", "price", "bid", "ask", "volume", "symbol"])

    df = pd.DataFrame(all_rows)
    df = df.sort_values("timestamp_ms").reset_index(drop=True)
    return df


def disaggregate_and_save(
    candle_df: pd.DataFrame,
    generation_id: str,
    symbol: str,
    ticks_per_minute: int = 60,
    spread_bps: float = 0.5,
    volume_profile: str = "sine",
    seed: int | None = None,
    output_dir: str = "data/synthetic/ticks",
) -> pd.DataFrame:
    """Disaggregate candles to ticks and save as Parquet.

    Args:
        candle_df: 1-minute candle DataFrame.
        generation_id: Generation identifier for tracking.
        symbol: Ticker symbol.
        ticks_per_minute: Number of ticks per minute.
        spread_bps: Bid-ask spread in basis points.
        volume_profile: Volume distribution shape.
        seed: Random seed.
        output_dir: Output directory for Parquet files.

    Returns:
        Tick DataFrame.
    """
    ticks_df = disaggregate_1m_to_ticks(
        candle_df=candle_df,
        ticks_per_minute=ticks_per_minute,
        spread_bps=spread_bps,
        volume_profile=volume_profile,
        seed=seed,
    )

    ticks_df["generation_id"] = generation_id

    out_path = Path(output_dir) / symbol / generation_id
    out_path.mkdir(parents=True, exist_ok=True)

    ticks_df.to_parquet(out_path / "ticks.parquet", index=False)

    jsonl_path = out_path / "ticks.jsonl"
    ticks_df.to_json(jsonl_path, orient="records", lines=True)

    return ticks_df


def load_candle_parquet(input_path: str | Path) -> pd.DataFrame:
    """Load 1-minute candles from a Parquet directory or file."""
    path = Path(input_path)
    if path.is_dir():
        return pd.read_parquet(path)
    return pd.read_parquet(path)
