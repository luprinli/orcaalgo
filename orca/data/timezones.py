"""Exchange-local timezone mapping for candle ingestion (R14).

stooq intraday timestamps are exchange-local; we attach the correct zone and
normalize to UTC on ingest so session-window strategies (ORB / session_scalp)
see the true session time rather than a UTC-shifted time.
"""

from __future__ import annotations

from zoneinfo import ZoneInfo

# Asset class -> exchange-local IANA timezone.
ASSET_CLASS_TIMEZONES: dict[str, str] = {
    "equity": "America/New_York",
    "equity_etf": "America/New_York",
    "commodity_etf": "America/New_York",
    "bond_etf": "America/New_York",
    "forex_major": "UTC",
    "forex_minor": "UTC",
    "forex_exotic": "UTC",
    "crypto": "UTC",
    "index_agg": "America/New_York",
    "index_eu": "Europe/Berlin",
}

# Default fallback when a symbol's asset class is unknown.
DEFAULT_TIMEZONE = "UTC"


def timezone_for_symbol(ticker: str, asset_class: str) -> ZoneInfo:
    """Resolve the exchange-local timezone for a symbol.

    Falls back to UTC for unknown asset classes (never raises).
    """
    tz_name = ASSET_CLASS_TIMEZONES.get(asset_class, DEFAULT_TIMEZONE)
    try:
        return ZoneInfo(tz_name)
    except Exception:
        return ZoneInfo(DEFAULT_TIMEZONE)


__all__ = ["ASSET_CLASS_TIMEZONES", "timezone_for_symbol"]
