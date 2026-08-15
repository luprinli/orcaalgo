"""Risk-free rate ingestion from Yahoo Finance (^IRX, 13-week T-bill yield).

Mirrors ``orca/data/vix_ingestion.py``. Fetches the ^IRX daily yield and stores
it as a fractional annualized yield (5.2% -> 0.052) in ``benchmark_series`` with
name ``risk_free_3m``. The benchmark filter's ``risk_free`` kind converts this
to a daily return (``value / 252``) at consumption time.
"""

from __future__ import annotations

from datetime import date, timedelta


def fetch_risk_free_yield(start: date, end: date) -> list[dict]:
    """Fetch ^IRX historical yield from Yahoo Finance.

    Returns:
        List of dicts with keys: timestamp, value (fractional annualized yield),
        source.
    """
    try:
        import yfinance as yf
    except ImportError as err:
        raise ImportError(
            "yfinance is required for risk-free ingestion. Install with: pip install yfinance"
        ) from err

    ticker = yf.Ticker("^IRX")
    df = ticker.history(start=str(start), end=str(end) + timedelta(days=1))
    if df.empty:
        return []

    logs = []
    for idx in df.index:
        close = float(df.loc[idx, "Close"])
        logs.append(
            {
                "timestamp": idx.to_pydatetime(),
                "value": close / 100.0,
                "source": "yahoo",
            }
        )
    return logs
