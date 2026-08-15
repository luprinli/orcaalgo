"""TimescaleDB integration for data pipeline outputs.

Provides upsert logic for resampled candles and bulk-insert for regime logs.
Follows the same pattern as orca/simulation/calibrate.py for DB connectivity.
"""

from __future__ import annotations

import os


def _to_py_datetime(val):
    """Convert numpy datetime64 or pandas Timestamp to Python datetime."""
    import numpy as np
    import pandas as pd

    if isinstance(val, (np.datetime64, pd.Timestamp)):
        return val.astype("datetime64[us]").astype("M8[us]").astype(object)
    if hasattr(val, "to_pydatetime"):
        return val.to_pydatetime()
    if hasattr(val, "astype"):
        try:
            return val.astype("datetime64[us]").astype("M8[us]").astype(object)
        except Exception:
            pass
    return val


def _get_db_url() -> str:
    return os.environ.get("ORCA_DB_URL", "postgresql://orca:orca@localhost:5432/orca_core")


def get_connection():
    """Return a psycopg2 connection. Import is lazy to keep CLI startup fast."""
    try:
        import psycopg2
    except ImportError as e:
        raise ImportError(
            "psycopg2 is required for DB operations. Install with: pip install psycopg2-binary"
        ) from e
    return psycopg2.connect(_get_db_url())


def upsert_candles(
    symbol: str,
    timeframe: str,
    df,
    batch_size: int = 5000,
    source: str = "yahoo",
    generation_id: str | None = None,
) -> int:
    """Upsert OHLCV bars into the candles table with provenance.

    Uses ON CONFLICT (symbol_id, timeframe, time, source) DO UPDATE to support
    idempotent re-runs while preserving per-source bar identity. Prices are
    stored as BIGINT with 100000x scale factor.

    Args:
        symbol: Trading instrument ticker (e.g. 'SPY', 'EURUSD')
        timeframe: Bar resolution label ('15m', '1h', '1d', etc.)
        df: DataFrame with columns [open, high, low, close, volume] indexed by datetime
        source: provenance label ('yahoo', 'stooq', 'stooq-resampled', ...)
        generation_id: deterministic data-generation identifier for lineage
        batch_size: Rows per INSERT batch

    Returns:
        Number of rows upserted
    """
    import psycopg2
    import psycopg2.extras

    conn = get_connection()
    inserted = 0

    try:
        with conn.cursor() as cur:
            cur.execute("SELECT id FROM symbols WHERE ticker = %s LIMIT 1", (symbol,))
            row = cur.fetchone()
            if row is None:
                cur.execute(
                    "INSERT INTO symbols (ticker, asset_type, is_active) "
                    "VALUES (%s,%s,TRUE) ON CONFLICT (ticker) DO NOTHING RETURNING id",
                    (symbol, "unknown"),
                )
                row = cur.fetchone()
                if row is None:
                    cur.execute("SELECT id FROM symbols WHERE ticker = %s LIMIT 1", (symbol,))
                    row = cur.fetchone()
            symbol_id = row[0]

            rows = []
            for idx in df.index:
                bar = df.loc[idx]
                rows.append(
                    (
                        symbol_id,
                        timeframe,
                        idx.to_pydatetime(),
                        round(float(bar["open"]) * 100000),
                        round(float(bar["high"]) * 100000),
                        round(float(bar["low"]) * 100000),
                        round(float(bar["close"]) * 100000),
                        round(float(bar["volume"])),
                        source,
                        generation_id,
                    )
                )

            psycopg2.extras.execute_values(
                cur,
                """
                INSERT INTO candles
                    (symbol_id, timeframe, time, open_raw, high_raw, low_raw, close_raw, volume, source, generation_id)
                VALUES %s
                ON CONFLICT (symbol_id, timeframe, time, source) DO UPDATE SET
                    open_raw = EXCLUDED.open_raw,
                    high_raw = EXCLUDED.high_raw,
                    low_raw = EXCLUDED.low_raw,
                    close_raw = EXCLUDED.close_raw,
                    volume = EXCLUDED.volume,
                    generation_id = EXCLUDED.generation_id
            """,
                rows,
                page_size=batch_size,
            )
            inserted = cur.rowcount
        conn.commit()
    finally:
        conn.close()

    return inserted


def insert_regime_logs(logs: list[dict], batch_size: int = 500) -> int:
    """Insert regime classification results into regime_logs table.

    Args:
        logs: List of dicts with keys: timestamp, symbol, hmm_state, confidence
        batch_size: Rows per INSERT batch

    Returns:
        Number of rows inserted
    """
    import psycopg2
    import psycopg2.extras

    conn = get_connection()
    inserted = 0

    try:
        with conn.cursor() as cur:
            rows = [
                (
                    _to_py_datetime(log["timestamp"]),
                    log["symbol"],
                    int(log["hmm_state"]),
                    float(log["confidence"]),
                )
                for log in logs
            ]
            psycopg2.extras.execute_values(
                cur,
                """
                INSERT INTO regime_logs (timestamp, symbol, hmm_state, confidence)
                VALUES %s
                ON CONFLICT DO NOTHING
            """,
                rows,
                page_size=batch_size,
            )
            inserted = cur.rowcount
        conn.commit()
    finally:
        conn.close()

    return inserted
