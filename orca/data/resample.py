"""OHLCV candle resampling from fine to coarse timeframes.

Standard OHLC aggregation: Open=first, High=max, Low=min, Close=last, Volume=sum.
Pandas resample() is the industry-standard approach with 15+ years of battle-tested logic.
"""

from __future__ import annotations

import pandas as pd

TIMEFRAME_HIERARCHY: dict[str, str] = {
    "1m":  "1min",
    "5m":  "5min",
    "15m": "15min",
    "30m": "30min",
    "1h":  "1h",
    "4h":  "4h",
    "1d":  "1D",
}

REQUIRED_COLUMNS = {"time", "open", "high", "low", "close", "volume"}


def resample_ohlc(df: pd.DataFrame, timeframe: str) -> pd.DataFrame:
    """Resample OHLCV bars from a finer to a coarser timeframe.

    Args:
        df: DataFrame with columns [time, open, high, low, close, volume].
            time column must be datetime-typed and set as the index before calling,
            or the function will set it.
        timeframe: pandas frequency string ('15min', '1h', '4h', '1D')

    Returns:
        DataFrame with aggregated OHLCV bars at the requested timeframe.
        NaN bars (empty buckets) are dropped.
    
    Raises:
        ValueError: If required columns are missing.
    """
    missing = REQUIRED_COLUMNS - set(df.columns)
    if missing:
        raise ValueError(f"Missing required columns: {missing}")

    working = df.copy()
    if "time" in working.columns:
        working = working.set_index("time")

    tf_key = TIMEFRAME_HIERARCHY.get(timeframe, timeframe)

    return working.resample(tf_key).agg({
        "open":   "first",
        "high":   "max",
        "low":    "min",
        "close":  "last",
        "volume": "sum",
    }).dropna()
