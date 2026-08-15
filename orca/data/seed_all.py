"""Unified data seeding pipeline (legacy Yahoo candle path — retired).

The canonical candle ingestion now runs through the stooq pipeline
(scripts/stooq_discovery.py → stooq_seed.py → stooq_resample.py →
stooq_synthetic.py), which writes `source='stooq'` / `'stooq-resampled'` /
`'stooq-calibrated'` with a deterministic generation_id. The orchestrator
(scripts/orchestrate.py --reset-reseed) drives that pipeline and calls the
dedicated `ingest-vix`, `build-regime-logs`, and `backfill-sentiment` commands.

This module is retained for backwards-compatible standalone use (`orca
seed-all`): it fetches Yahoo candles (source='yahoo'), VIX, regimes, and
sentiment. Its Yahoo candle fetch is deprecated — prefer the stooq pipeline
for research/backtest data. VIX is the one remaining Yahoo-sourced series
(stooq carries no ^vix index).
"""

from __future__ import annotations

import hashlib
import json
import time as _time
from datetime import date, datetime, timedelta

import numpy as np
import pandas as pd

# Default symbols derive from configs/universe.json (single source of truth).
# seed_all fetches from Yahoo, so it uses each symbol's Yahoo ticker.
try:
    from orca.universe_config import load_universe

    _cfg = load_universe()
    DEFAULT_SYMBOLS = [s["yahoo_ticker"] for s in _cfg["symbols"] if s.get("yahoo_ticker")]
except Exception:
    DEFAULT_SYMBOLS = [
        "SPY",
        "QQQ",
        "AAPL",
        "MSFT",
        "NVDA",
        "TSLA",
        "IWM",
        "GLD",
        "TLT",
        "EURUSD=X",
        "GBPUSD=X",
        "USDJPY=X",
        "AUDUSD=X",
        "USDCAD=X",
        "BTC-USD",
        "ETH-USD",
        "^GSPC",
        "^GDAXI",
    ]

TIMEFRAMES_SOURCE = ["5m", "1d"]
TIMEFRAMES_RESAMPLE = ["15m", "30m", "1h", "4h"]


def compute_generation_id(config: dict) -> str:
    """Deterministic generation ID from config dict.

    The hash covers only the inputs that define the data itself (symbols,
    date range, source, and generation parameters). Wall-clock timestamps are
    intentionally excluded so that re-running with identical inputs yields the
    same generation ID — a precondition for reproducible lineage.
    """
    raw = json.dumps(config, sort_keys=True, default=str)
    return hashlib.sha256(raw.encode()).hexdigest()[:16]


def fetch_candles_yahoo(
    symbol: str,
    start: date,
    end: date,
    interval: str = "5m",
) -> pd.DataFrame:
    """Fetch OHLCV candles from Yahoo Finance.

    Args:
        symbol: Yahoo ticker (e.g. 'SPY', 'EURUSD=X', 'BTC-USD')
        start: Start date (inclusive)
        end: End date (inclusive)
        interval: '1m', '5m', '15m', '30m', '1h', '1d'

    Returns:
        DataFrame with columns [time, open, high, low, close, volume].
        Empty DataFrame if fetch fails.
    """
    try:
        import yfinance as yf
    except ImportError as err:
        raise ImportError("yfinance required. Install with: pip install yfinance") from err

    ticker = yf.Ticker(symbol)
    df = ticker.history(
        start=str(start),
        end=str(end + timedelta(days=1)),
        interval=interval,
        auto_adjust=True,
    )

    if df.empty:
        return pd.DataFrame(columns=["time", "open", "high", "low", "close", "volume"])

    df = df.reset_index()
    df = df.rename(
        columns={
            "Datetime": "time",
            "Date": "time",
            "Open": "open",
            "High": "high",
            "Low": "low",
            "Close": "close",
            "Volume": "volume",
        }
    )
    keep_cols = [c for c in ["time", "open", "high", "low", "close", "volume"] if c in df.columns]
    df = df[keep_cols]
    df["time"] = pd.to_datetime(df["time"])

    return df


def fetch_vix_yahoo(start: date, end: date) -> pd.DataFrame:
    """Fetch ^VIX historical data from Yahoo Finance."""
    try:
        import yfinance as yf
    except ImportError as err:
        raise ImportError("yfinance required. Install with: pip install yfinance") from err

    ticker = yf.Ticker("^VIX")
    df = ticker.history(
        start=str(start),
        end=str(end + timedelta(days=1)),
        interval="1d",
        auto_adjust=True,
    )

    if df.empty:
        return pd.DataFrame(columns=["time", "vix_value", "vix_change"])

    result = pd.DataFrame()
    result["time"] = df.index
    result["vix_value"] = df["Close"].values
    result["vix_change"] = result["vix_value"].diff().fillna(0.0)
    return result


def generate_sentiment_from_candles(
    closes: np.ndarray,
    timestamps: np.ndarray,
    lookback: int = 21,
) -> pd.DataFrame:
    """Generate synthetic sentiment from candle returns.

    High positive returns → Greed, high negative returns → Fear.
    This is a fallback when Alternative.me data is unavailable.
    """
    if len(closes) < lookback + 1:
        return pd.DataFrame(columns=["time", "score", "label"])

    returns = np.diff(np.log(closes))
    n = len(closes)

    scores = np.full(n, 50, dtype=int)
    labels = np.full(n, "Neutral", dtype=object)

    for i in range(lookback, n):
        window = returns[i - lookback : i]
        total_return = (closes[i] / closes[i - lookback] - 1.0) * 100.0
        ann_vol = np.std(window) * np.sqrt(252)

        base_score = 50 + total_return * 3.0
        if ann_vol > 0.3:
            base_score -= 10.0

        score = int(np.clip(base_score, 5, 95))
        scores[i] = score

        if score <= 25:
            labels[i] = "Extreme Fear"
        elif score <= 45:
            labels[i] = "Fear"
        elif score <= 55:
            labels[i] = "Neutral"
        elif score <= 75:
            labels[i] = "Greed"
        else:
            labels[i] = "Extreme Greed"

    return pd.DataFrame(
        {
            "time": timestamps,
            "score": scores,
            "label": labels,
        }
    )


def seed_all(
    symbols: list[str] | None = None,
    start: date = date(2026, 6, 12),
    end: date = date(2026, 8, 12),
    reset: bool = False,
    verbose: bool = True,
) -> dict:
    """Unified data seeding: fetch → resample → VIX → regime → sentiment → DB.

    Args:
        symbols: List of Yahoo tickers. Defaults to DEFAULT_SYMBOLS.
        start: Start date for data pull.
        end: End date for data pull.
        reset: If True, truncate existing data for the target period before seeding.
        verbose: Print progress messages.

    Returns:
        Dict with stats: {rows_candles, rows_vix, rows_regime, rows_sentiment,
                          generation_id, elapsed_seconds}
    """
    from orca.data.db_integration import get_connection, insert_regime_logs, upsert_candles
    from orca.data.resample import resample_ohlc
    from orca.data.validate_resample import compute_effective_bpd

    t0 = _time.monotonic()

    if symbols is None:
        symbols = DEFAULT_SYMBOLS

    gen_id = compute_generation_id(
        {
            "symbols": sorted(symbols),
            "start": str(start),
            "end": str(end),
            "source": "yahoo",
        }
    )

    stats = {
        "generation_id": gen_id,
        "rows_candles": 0,
        "rows_vix": 0,
        "rows_regime": 0,
        "rows_sentiment": 0,
        "elapsed_seconds": 0.0,
        "errors": [],
    }

    if reset:
        _reset_scope(conn_source="yahoo", symbols=symbols, start=start, end=end)

    all_1d_closes = {}
    all_1d_times = {}

    for sym in symbols:
        try:
            if verbose:
                print(f"  {sym}: fetching 5m + 1d...", end=" ", flush=True)

            candles_5m = fetch_candles_yahoo(sym, start, end, "5m")
            candles_1d = fetch_candles_yahoo(sym, start, end, "1d")

            if candles_5m.empty and candles_1d.empty:
                if verbose:
                    print("no data")
                continue

            if not candles_1d.empty and len(candles_1d) >= 2:
                _t = pd.to_datetime(candles_1d["time"]).values
                all_1d_closes[sym] = np.array(candles_1d["close"].values, dtype=np.float64)
                all_1d_times[sym] = _t.astype("datetime64[us]")

            inserted = 0
            if not candles_5m.empty:
                inserted += upsert_candles(
                    sym, "5m", candles_5m.set_index("time"), source="yahoo", generation_id=gen_id
                )

                for tf in TIMEFRAMES_RESAMPLE:
                    derived = resample_ohlc(candles_5m.copy(), tf)
                    if not derived.empty:
                        inserted += upsert_candles(
                            sym, tf, derived, source="yahoo", generation_id=gen_id
                        )

            if not candles_1d.empty:
                inserted += upsert_candles(
                    sym, "1d", candles_1d.set_index("time"), source="yahoo", generation_id=gen_id
                )

            stats["rows_candles"] += inserted

            if verbose:
                bpd_5m = (
                    compute_effective_bpd(candles_5m.set_index("time"))
                    if not candles_5m.empty
                    else 0
                )
                print(f"{inserted} bars (5m BPD={bpd_5m:.0f})")

        except Exception as e:
            msg = f"{sym}: {e}"
            stats["errors"].append(msg)
            if verbose:
                print(f"ERROR: {msg}")

    if verbose:
        print("  VIX: fetching...", end=" ", flush=True)

    try:
        vix_df = fetch_vix_yahoo(start, end)
        if not vix_df.empty:
            import psycopg2

            conn = get_connection()
            try:
                with conn.cursor() as cur:
                    rows = [
                        (
                            r["time"].to_pydatetime(),
                            int(float(r["vix_value"]) * 10000),
                            int(float(r["vix_change"]) * 10000),
                        )
                        for _, r in vix_df.iterrows()
                    ]
                    psycopg2.extras.execute_values(
                        cur,
                        """
                        INSERT INTO vix_logs (timestamp, vix_value, vix_change)
                        VALUES %s
                        ON CONFLICT DO NOTHING
                    """,
                        rows,
                        page_size=500,
                    )
                    stats["rows_vix"] = cur.rowcount
                conn.commit()
            finally:
                conn.close()
        if verbose:
            print(f"{len(vix_df)} rows")
    except Exception as e:
        if verbose:
            print(f"VIX skipped: {e}")

    if verbose:
        print("  Regime: inferring...", end=" ", flush=True)

    try:
        from orca.data.regime_inference import build_regime_logs

        regime_data = {
            sym: (closes, times)
            for sym, (closes, times) in zip(
                all_1d_closes.keys(),
                zip(all_1d_closes.values(), all_1d_times.values(), strict=False),
                strict=False,
            )
            if len(closes) >= 21
        }
        logs = build_regime_logs(regime_data)
        if logs:
            stats["rows_regime"] = insert_regime_logs(logs)
        if verbose:
            print(f"{stats['rows_regime']} rows")
    except Exception as e:
        stats["errors"].append(f"Regime: {e}")
        if verbose:
            print(f"ERROR: {e}")

    if verbose:
        print("  Sentiment: generating...", end=" ", flush=True)

    try:
        import psycopg2

        conn = get_connection()
        total_sent = 0
        for sym, closes in all_1d_closes.items():
            if len(closes) < 22:
                continue
            times_dict = {
                s: t for s, t in zip(all_1d_closes.keys(), all_1d_times.values(), strict=False)
            }
            times = times_dict.get(sym)
            if times is None:
                continue
            sentiment_df = generate_sentiment_from_candles(closes, times)
            if not sentiment_df.empty:
                rows = [
                    (t.to_pydatetime(), int(s), str(lbl))
                    for t, s, lbl in zip(
                        sentiment_df["time"],
                        sentiment_df["score"],
                        sentiment_df["label"],
                        strict=False,
                    )
                ]
                with conn.cursor() as cur:
                    psycopg2.extras.execute_values(
                        cur,
                        """
                        INSERT INTO sentiment_logs (timestamp, score, label)
                        VALUES %s
                        ON CONFLICT DO NOTHING
                    """,
                        rows,
                        page_size=500,
                    )
                    total_sent += cur.rowcount
        conn.commit()
        conn.close()
        stats["rows_sentiment"] = total_sent
        if verbose:
            print(f"{total_sent} rows")
    except Exception as e:
        if verbose:
            print(f"Sentiment skipped: {e}")

    stats["elapsed_seconds"] = _time.monotonic() - t0

    _write_manifest(stats, symbols if symbols else DEFAULT_SYMBOLS)

    return stats


def _reset_scope(
    conn_source: str,
    symbols: list[str],
    start: date,
    end: date,
) -> None:
    """Scoped reset: delete only this writer's source for the given symbols/range.

    Prior behavior issued `DELETE FROM candles WHERE time BETWEEN ...`, which
    wiped bars from every source and left stale rows for untouched symbols.
    Scoping the delete to (source, symbol, time-range) makes reseeds idempotent
    and prevents one ingestion path from destroying another's data.
    """
    from orca.data.db_integration import get_connection

    conn = get_connection()
    try:
        with conn.cursor() as cur:
            # Re-activate canonical symbols before (re)writing them, so partial
            # reseeds never leave orphaned/inactive symbols rows behind.
            cur.execute(
                "UPDATE symbols SET is_active = TRUE WHERE ticker = ANY(%s)",
                (list(symbols),),
            )
            cur.execute(
                """
                DELETE FROM candles c
                USING symbols s
                WHERE c.symbol_id = s.id
                  AND s.ticker = ANY(%s)
                  AND c.source = %s
                  AND c.time >= %s AND c.time <= %s
                """,
                (list(symbols), conn_source, start, end),
            )
        for table in ["vix_logs", "regime_logs", "sentiment_logs"]:
            try:
                with conn.cursor() as cur:
                    cur.execute(
                        f"DELETE FROM {table} WHERE timestamp >= %s AND timestamp <= %s",
                        (start, end),
                    )
            except Exception:
                pass
        conn.commit()
    finally:
        conn.close()


def _write_manifest(stats: dict, symbols: list[str]) -> None:
    """Write data/.manifest.json with generation_id and data signature."""
    import json as _json
    from pathlib import Path as _Path

    manifest_path = _Path("data") / ".manifest.json"
    _Path("data").mkdir(parents=True, exist_ok=True)

    manifest = {
        "generation_id": stats["generation_id"],
        "generated_at": datetime.utcnow().isoformat(),
        "symbols": sorted(symbols),
        "symbol_count": len(symbols),
        "bar_count": stats["rows_candles"],
        "vix_count": stats["rows_vix"],
        "regime_count": stats["rows_regime"],
        "sentiment_count": stats["rows_sentiment"],
        "elapsed_seconds": round(stats["elapsed_seconds"], 2),
        "errors": stats.get("errors", []),
    }
    manifest_path.write_text(_json.dumps(manifest, indent=2, default=str) + "\n")
