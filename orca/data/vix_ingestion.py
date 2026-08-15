"""Historical VIX data ingestion from Yahoo Finance.

Fetches ^VIX daily data from Yahoo Finance and inserts into the vix_logs table.
Uses yfinance for reliable access without rate limiting issues.
"""

from __future__ import annotations

from datetime import date, timedelta


def fetch_vix_historical(
    start: date,
    end: date,
) -> list[dict]:
    """Fetch historical VIX values from Yahoo Finance.

    Args:
        start: Start date (inclusive)
        end: End date (inclusive)

    Returns:
        List of dicts with keys: timestamp, vix_value, vix_change, source
    """
    try:
        import yfinance as yf
    except ImportError as err:
        raise ImportError(
            "yfinance is required for VIX ingestion. Install with: pip install yfinance"
        ) from err

    ticker = yf.Ticker("^VIX")
    df = ticker.history(start=str(start), end=str(end) + timedelta(days=1))

    if df.empty:
        return []

    logs = []
    prev_close = None

    for idx in df.index:
        close_val = float(df.loc[idx, "Close"])
        change = 0.0
        if prev_close is not None and prev_close > 0:
            change = close_val - prev_close
        prev_close = close_val

        logs.append(
            {
                "timestamp": idx.to_pydatetime(),
                "vix_value": close_val,
                "vix_change": change,
                "source": "yahoo",
            }
        )

    return logs
