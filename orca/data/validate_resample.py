"""Validates OHLCV resampling invariants.

For each bar in a derived (coarser) timeframe, verifies that its OHLCV values
match the constituent bars from the source (finer) timeframe. Zero violations
indicates the resampling was performed correctly.
"""

from __future__ import annotations

import pandas as pd


def validate_resampling(
    source: pd.DataFrame,
    derived: pd.DataFrame,
    timeframe: str,
    tolerance: float = 1e-8,
) -> list[str]:
    """Validate that derived bars are correct aggregations of source bars.

    Args:
        source: DataFrame with finer-resolution bars [open, high, low, close, volume]
                indexed by datetime.
        derived: DataFrame with coarser-resolution bars, also datetime-indexed.
        timeframe: pandas frequency string of the derived bars ('15min', '1h', '1D').
        tolerance: Acceptable floating-point difference for equality checks.

    Returns:
        List of error messages. Empty list means all invariants pass.
    """
    errors = []

    for idx in derived.index:
        window_end = idx
        window_start = window_end - pd.Timedelta(timeframe.replace("min", "T").replace("h", "H").replace("1D", "1d"))
        constituents = source[(source.index >= window_start) & (source.index <= window_end)]

        if len(constituents) == 0:
            continue

        bar = derived.loc[idx]

        actual_open = constituents["open"].iloc[0]
        actual_high = constituents["high"].max()
        actual_low = constituents["low"].min()
        actual_close = constituents["close"].iloc[-1]
        actual_volume = constituents["volume"].sum()

        if abs(bar["open"] - actual_open) > tolerance:
            errors.append(f"{idx}: open mismatch ({bar['open']} vs {actual_open})")
        if abs(bar["high"] - actual_high) > tolerance:
            errors.append(f"{idx}: high mismatch ({bar['high']} vs {actual_high})")
        if abs(bar["low"] - actual_low) > tolerance:
            errors.append(f"{idx}: low mismatch ({bar['low']} vs {actual_low})")
        if abs(bar["close"] - actual_close) > tolerance:
            errors.append(f"{idx}: close mismatch ({bar['close']} vs {actual_close})")
        if abs(bar["volume"] - actual_volume) > tolerance:
            errors.append(f"{idx}: volume mismatch ({bar['volume']} vs {actual_volume})")

    return errors


def compute_effective_bpd(candles: pd.DataFrame) -> float:
    """Compute effective bars-per-day for a candle dataset.

    Args:
        candles: DataFrame with datetime index.

    Returns:
        Average number of bars per calendar day.
    """
    if len(candles) < 2:
        return 0.0
    total_days = (candles.index[-1] - candles.index[0]).total_seconds() / 86400.0
    if total_days <= 0:
        return 0.0
    return len(candles) / total_days
