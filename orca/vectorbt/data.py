"""Data connector between Orca's data sources and VectorBT.

Supports two backends with identical output:
  1. TimescaleDB — query candles hypertable (preferred, psycopg2 required)
  2. File-based — read CSV or legacy Stooq .txt files from data/

Output is always a pandas DataFrame with index=timestamp, columns:
  open, high, low, close, volume
"""

import os
from pathlib import Path

import pandas as pd

REQUIRED_COLUMNS = ["open", "high", "low", "close", "volume"]

_TIMEFRAME_DIR: dict[str, str] = {
    "1d": "daily",
    "4h": "hourly",
    "1h": "hourly",
    "15m": "5 min",
    "5m": "5 min",
    "1m": "1 min",
}


def load_candles(
    symbol: str,
    start: str | None = None,
    end: str | None = None,
    timeframe: str = "1h",
    backend: str = "auto",
) -> pd.DataFrame:
    """Load OHLCV data into a pandas DataFrame.

    Args:
        symbol: Ticker symbol (e.g. "SPY", "EURUSD")
        start: Start date string ("2023-01-01") or None for all data
        end: End date string ("2024-12-31") or None for all data
        timeframe: "1m" | "5m" | "15m" | "30m" | "1h" | "4h" | "1d"
        backend: "auto" (try TimescaleDB, fall back to file),
                 "timescaledb", or "file"

    Returns:
        DataFrame with datetime index and columns: open, high, low, close, volume
    """
    if backend == "auto":
        try:
            return _load_from_timescaledb(symbol, start, end, timeframe)
        except (ImportError, Exception):
            return _load_from_file(symbol, start, end, timeframe)
    elif backend == "timescaledb":
        return _load_from_timescaledb(symbol, start, end, timeframe)
    else:
        return _load_from_file(symbol, start, end, timeframe)


def _load_from_timescaledb(
    symbol: str,
    start: str | None,
    end: str | None,
    timeframe: str,
) -> pd.DataFrame:
    """Query candles from TimescaleDB hypertable."""
    import psycopg2

    conn = psycopg2.connect(
        host=os.getenv("ORCA_DB_HOST", "localhost"),
        port=os.getenv("ORCA_DB_PORT", "5432"),
        dbname=os.getenv("ORCA_DB_NAME", "orca_core"),
        user=os.getenv("ORCA_DB_USER", "postgres"),
        password=os.getenv("ORCA_DB_PASSWORD", ""),
    )
    query = """
        SELECT timestamp, open, high, low, close, volume
        FROM candles
        WHERE symbol = %s AND timeframe = %s
          AND timestamp >= COALESCE(%s::timestamptz, '-infinity'::timestamptz)
          AND timestamp <= COALESCE(%s::timestamptz, 'infinity'::timestamptz)
        ORDER BY timestamp
    """
    params = [symbol, timeframe, start, end]
    df = pd.read_sql(query, conn, params=params)
    conn.close()

    if df.empty:
        raise ValueError(
            f"No data for {symbol} ({timeframe}) in TimescaleDB between {start} and {end}"
        )

    df.set_index("timestamp", inplace=True)
    df.columns = [c.lower() for c in df.columns]
    return df


def _load_from_file(
    symbol: str,
    start: str | None,
    end: str | None,
    timeframe: str,
) -> pd.DataFrame:
    """Read from data/ directory — CSV first, then legacy Stooq .txt.

    Same resolution logic as the Go CLI data loader and sweeper.py.
    """
    dir_name = _TIMEFRAME_DIR.get(timeframe, "hourly")
    data_root = Path(os.getenv("ORCA_DATA_DIR", "data"))
    symbol_lower = symbol.lower()

    # 1. Try modern CSV path
    csv_path = data_root / f"{symbol_lower}_{timeframe}.csv"
    if csv_path.exists():
        df = pd.read_csv(csv_path)
        date_col = _find_date_column(df)
        if date_col:
            df[date_col] = pd.to_datetime(df[date_col])
            df.set_index(date_col, inplace=True)
    else:
        # 2. Try legacy Stooq path: data/{timeframe}/world/{subdir}/{symbol}.txt
        stooq_path = _resolve_stooq_path(data_root, dir_name, symbol_lower)
        if stooq_path is None:
            raise FileNotFoundError(
                f"No data found for {symbol} at paths: {csv_path}, "
                f"data/{dir_name}/world/currencies/major/{symbol_lower}.txt, "
                f"data/{dir_name}/world/indices/{symbol_lower}.txt, "
                f"data/{dir_name}/world/cryptocurrencies/{symbol_lower}.txt"
            )
        df = pd.read_csv(
            stooq_path,
            names=[
                "ticker",
                "per",
                "date",
                "time",
                "open",
                "high",
                "low",
                "close",
                "vol",
                "openint",
            ],
            header=0,
        )
        df["date"] = pd.to_datetime(df["date"].astype(str), format="%Y%m%d")
        df.set_index("date", inplace=True)

    _normalize_columns(df)

    if start:
        df = df[df.index >= start]
    if end:
        df = df[df.index <= end]

    return df.sort_index()


def _resolve_stooq_path(data_root: Path, dir_name: str, symbol_lower: str) -> Path | None:
    """Try known Stooq subdirectories in order."""
    candidates = [
        data_root / dir_name / "world" / "currencies" / "major" / f"{symbol_lower}.txt",
        data_root / dir_name / "world" / "indices" / f"{symbol_lower}.txt",
        data_root / dir_name / "world" / "cryptocurrencies" / f"{symbol_lower}.txt",
    ]
    for path in candidates:
        if path.exists():
            return path
    return None


def _find_date_column(df: pd.DataFrame) -> str | None:
    """Find the date column in a DataFrame (case-insensitive)."""
    for col in df.columns:
        if col.lower() in ("date", "timestamp", "datetime"):
            return col
    return df.columns[0] if len(df.columns) > 0 else None


def _normalize_columns(df: pd.DataFrame) -> None:
    """Normalize column names to lowercase: open, high, low, close, volume."""
    df.columns = [c.lower().strip() for c in df.columns]

    rename = {}
    for col in df.columns:
        if col in ("open", "high", "low", "close"):
            rename[col] = col
        elif col in ("volume", "vol"):
            rename[col] = "volume"

    if rename:
        df.rename(columns=rename, inplace=True)


def load_asset_classes() -> dict[str, str]:
    """Return mapping of symbol → asset class for filtering."""
    return {
        "EURUSD": "forex",
        "GBPUSD": "forex",
        "USDJPY": "forex",
        "AUDUSD": "forex",
        "SPY": "equity",
        "QQQ": "equity",
        "AAPL": "equity",
        "BTCUSD": "crypto",
        "ETHUSD": "crypto",
        "CL": "commodity",
        "GC": "commodity",
    }
