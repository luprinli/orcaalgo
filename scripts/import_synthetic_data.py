"""Aggregate synthetic 1m regime-aware parquet data into timeframe candles and import to PostgreSQL."""

import os
import sys
import pyarrow.parquet as pq
import pyarrow as pa
import psycopg2
from psycopg2.extras import execute_values
from datetime import datetime, timedelta
import pandas as pd
import numpy as np

DATA_DIR = "data/synthetic/regime_batch"
PG_DSN = "host=localhost port=5432 dbname=orca_core user=orca password=change_me"
PRICE_SCALE = 100000
CHUNK_SIZE = 25000

START_DATE = "2023-01-01"
END_DATE = "2024-12-31"
TIMEFRAMES = ["1d", "1h", "15m"]

SYMBOLS = [
    ("usdeur", "forex", 0.00001), ("usdgbp", "forex", 0.00001), ("usdjpy", "forex", 0.001), ("usdchf", "forex", 0.00001), ("usdcad", "forex", 0.00001),
    ("usdaud", "forex", 0.00001), ("xauusd", "commodity", 0.01), ("xagusd", "commodity", 0.001), ("xpdusd", "commodity", 0.01), ("xptusd", "commodity", 0.01),
    ("btc.v", "crypto", 0.01), ("eth.v", "crypto", 0.01), ("xrp.v", "crypto", 0.0001), ("ada.v", "crypto", 0.0001), ("sol.v", "crypto", 0.01),
    ("doge.v", "crypto", 0.0001), ("dot.v", "crypto", 0.01), ("link.v", "crypto", 0.01), ("uni.v", "crypto", 0.01),
    ("spx", "index", 0.01), ("ndq", "index", 0.01), ("nkx", "index", 0.01), ("tsx", "index", 0.01), ("dax", "index", 0.01), ("ukx", "index", 0.01), ("hsi", "index", 0.01),
    ("nok_i", "index", 0.01), ("nokars", "commodity", 0.01), ("noknad", "forex", 0.00001), ("nokusd", "forex", 0.00001),
]

TF_MINUTES = {"1m": 1, "5m": 5, "15m": 15, "1h": 60, "1d": 1440}


def find_parquet(symbol):
    for f in os.listdir(DATA_DIR):
        if f.endswith("_1m.parquet") and (f.startswith(symbol + "_") or f == f"{symbol}_1m.parquet"):
            return os.path.join(DATA_DIR, f)
    return None


def connect():
    return psycopg2.connect(PG_DSN)


def ensure_symbols(conn):
    cur = conn.cursor()
    cur.execute("SELECT ticker, id FROM symbols")
    existing = {row[0]: row[1] for row in cur.fetchall()}
    ids = {}
    for (ticker, asset_type, tick_size) in SYMBOLS:
        tu = ticker.upper()
        if tu in existing:
            ids[ticker] = existing[tu]
        else:
            cur.execute(
                "INSERT INTO symbols (ticker, exchange, asset_type, tick_size) VALUES (%s, %s, %s, %s) RETURNING id",
                (tu, "SYNTHETIC", asset_type, str(tick_size)),
            )
            sid = cur.fetchone()[0]
            existing[tu] = sid
            ids[ticker] = sid
            print(f"  Created symbol {tu} -> id={sid}")
    conn.commit()
    return ids


def aggregate_timeframe(df, tf, generator):
    """Aggregate 1m candles to target timeframe. regime_label = majority vote."""
    rule = {"1m": "1min", "5m": "5min", "15m": "15min", "1h": "1h", "1d": "1D"}[tf]
    df = df.set_index("timestamp").sort_index()

    ohlc = df["close"].resample(rule, label="right", closed="right").agg(["first", "max", "min", "last"])
    ohlc.columns = ["open", "high", "low", "close"]
    ohlc = ohlc.dropna()

    vol = df["volume"].resample(rule, label="right", closed="right").sum()

    def majority_regime(x):
        if x.empty:
            return -1
        vals = x[x >= 0]
        if len(vals) == 0:
            return -1
        return int(vals.mode().iloc[0]) if len(vals.mode()) > 0 else int(vals.iloc[-1])

    regime = df["regime_label"].resample(rule, label="right", closed="right").agg(majority_regime)

    result = ohlc.copy()
    result["volume"] = vol
    result["regime_label"] = regime.fillna(-1).astype(int)
    result["generation_id"] = generator
    result = result.reset_index()
    result.columns = ["timestamp", "open", "high", "low", "close", "volume", "regime_label", "generation_id"]
    return result


def import_symbol_data(conn, sid, symbol, path):
    print(f"  Reading {symbol} parquet...")
    tbl = pq.read_table(path)
    pdf = tbl.to_pandas()
    pdf["timestamp"] = pd.to_datetime(pdf["timestamp"])
    pdf["regime_label"] = pdf["regime_label"].fillna(-1).astype(int)
    start_ts = pd.Timestamp(START_DATE)
    end_ts = pd.Timestamp(END_DATE) + pd.Timedelta(days=1)
    pdf = pdf[(pdf["timestamp"] >= start_ts) & (pdf["timestamp"] < end_ts)]
    if pdf.empty:
        print(f"    No data in range {START_DATE}..{END_DATE}")
        return 0

    generator = pdf["generation_id"].iloc[0] if "generation_id" in pdf.columns and not pdf["generation_id"].isna().all() else None

    total = 0
    for tf in TIMEFRAMES:
        if tf == "1m":
            agg = pdf[["timestamp", "open", "high", "low", "close", "volume", "regime_label"]].copy()
            agg["generation_id"] = generator
        else:
            agg = aggregate_timeframe(pdf, tf, generator)
        if agg.empty:
            continue

        rows = []
        for _, r in agg.iterrows():
            rows.append((
                r["timestamp"].isoformat(),
                sid,
                tf,
                int(r["open"] * PRICE_SCALE),
                int(r["high"] * PRICE_SCALE),
                int(r["low"] * PRICE_SCALE),
                int(r["close"] * PRICE_SCALE),
                int(r["volume"]),
                "synthetic",
                int(r["regime_label"]),
                str(r["generation_id"]) if pd.notna(r["generation_id"]) else None,
            ))

        cur = conn.cursor()
        inserted = 0
        for i in range(0, len(rows), CHUNK_SIZE):
            chunk = rows[i:i + CHUNK_SIZE]
            try:
                execute_values(
                    cur,
                    """INSERT INTO candles (time, symbol_id, timeframe, open_raw, high_raw, low_raw, close_raw, volume, source, regime_label, generation_id)
                       VALUES %s ON CONFLICT (symbol_id, timeframe, time) DO NOTHING""",
                    chunk,
                    template="(%s::timestamptz, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)"
                )
                conn.commit()
                inserted += len(chunk)
            except Exception as e:
                conn.rollback()
                print(f"    ERROR tf={tf}: {e}")
                return total
        print(f"    {tf}: {inserted} candles")
        total += inserted
    return total


def main():
    conn = connect()
    try:
        print("Ensuring symbols...")
        symbol_ids = ensure_symbols(conn)
        total_all = 0
        for (symbol, _, _) in SYMBOLS:
            path = find_parquet(symbol)
            if not path:
                print(f"  {symbol}: parquet not found, skipping")
                continue
            n = import_symbol_data(conn, symbol_ids[symbol], symbol, path)
            total_all += n
            print(f"  {symbol}: {n} total rows")
        print(f"\n=== DONE: {total_all} rows across {len(SYMBOLS)} symbols ===")
    finally:
        conn.close()


if __name__ == "__main__":
    main()
