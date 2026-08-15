"""CLI support for data quality validation."""

from __future__ import annotations

import random
from datetime import UTC, datetime

__all__ = [
    "DEFAULT_PRICE_SCALE",
    "Candle",
    "load_candles",
]

DEFAULT_PRICE_SCALE = 100000


class Candle:
    __slots__ = ("close", "time", "volume")

    def __init__(self, time, close, volume):
        self.time = time
        self.close = close
        self.volume = volume

    def __repr__(self):
        return f"Candle(time={self.time!r}, close={self.close!r}, volume={self.volume!r})"


_Candle = Candle


def _try_import_psycopg():
    try:
        import psycopg

        return ("psycopg3", psycopg)
    except ImportError:
        pass
    try:
        import psycopg2

        return ("psycopg2", psycopg2)
    except ImportError:
        pass
    return (None, None)


def _load_candles_from_db(
    price_scale: int = DEFAULT_PRICE_SCALE,
    db_url: str | None = None,
) -> dict:
    import os

    if db_url is None:
        db_url = os.environ.get("ORCA_DB_URL", "postgresql://orca:orca@localhost:5432/orca_core")

    version, pg = _try_import_psycopg()
    if pg is None:
        import logging

        logging.getLogger("orca.data_quality").debug("No psycopg available, using synthetic data")
        return _generate_sample_data()

    conn = None
    cur = None
    try:
        if version == "psycopg3":
            conn = pg.connect(db_url)
            cur = conn.cursor()
        else:
            conn = pg.connect(db_url)
            cur = conn.cursor()

        cur.execute("""
            SELECT s.ticker, c.time, c.close_raw, c.volume, c.timeframe
            FROM candles c
            JOIN symbols s ON c.symbol_id = s.id
            WHERE c.source != 'synthetic'
            ORDER BY s.ticker, c.timeframe, c.time
        """)
        rows = cur.fetchall()

        candles_by_symbol: dict = {}
        scale = float(price_scale)
        for ticker, t, close_raw, volume, tf in rows:
            key = f"{ticker}:{tf}"
            if key not in candles_by_symbol:
                candles_by_symbol[key] = []
            close_price = float(close_raw) / scale if close_raw else 0.0
            candles_by_symbol[key].append(Candle(t, close_price, volume))
        return candles_by_symbol
    except Exception:
        import logging

        logging.getLogger("orca.data_quality").debug(
            "Database query failed, using synthetic data", exc_info=True
        )
        return _generate_sample_data()
    finally:
        if cur is not None:
            cur.close()
        if conn is not None:
            conn.close()


def _generate_sample_data() -> dict:
    symbols = [
        "EURUSD:1d",
        "GBPUSD:1d",
        "USDJPY:1d",
        "USDCHF:1d",
        "AUDUSD:1d",
        "USDCAD:1d",
        "NZDUSD:1d",
        "SPX500:1d",
        "NAS100:1d",
        "XAUUSD:1d",
        "BTCUSD:1d",
        "ETHUSD:1d",
    ]
    candles_by_symbol: dict = {}
    base_prices = {
        "EURUSD": 1.08,
        "GBPUSD": 1.26,
        "USDJPY": 150.0,
        "USDCHF": 0.88,
        "AUDUSD": 0.66,
        "USDCAD": 1.35,
        "NZDUSD": 0.61,
        "SPX500": 5000.0,
        "NAS100": 17500.0,
        "XAUUSD": 2300.0,
        "BTCUSD": 60000.0,
        "ETHUSD": 3000.0,
    }
    for sym_tf in symbols:
        sym = sym_tf.split(":")[0]
        base = base_prices.get(sym, 100.0)
        candles = []
        price = base
        for i in range(500):
            t = datetime(2024, 1, 1, tzinfo=UTC).timestamp() + i * 86400
            price *= 1.0 + random.uniform(-0.01, 0.01)
            candles.append(Candle(t, price, random.randint(1000, 100000)))
        candles_by_symbol[sym_tf] = candles
    return candles_by_symbol


def load_candles(
    price_scale: int = DEFAULT_PRICE_SCALE,
    db_url: str | None = None,
) -> dict:
    return _load_candles_from_db(price_scale=price_scale, db_url=db_url)
