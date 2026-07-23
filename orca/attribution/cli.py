from __future__ import annotations

from datetime import UTC, datetime, timedelta

from orca.common.trade_loader import load_trades_from_db_or_synthetic


def _resolve_since(since: str) -> datetime:
    """Parse a lookback string like '30d', '90d' into a UTC datetime."""
    if since.endswith("d"):
        days = int(since[:-1])
        return datetime.now(UTC) - timedelta(days=days)
    if since.endswith("w"):
        weeks = int(since[:-1])
        return datetime.now(UTC) - timedelta(weeks=weeks)
    return datetime.now(UTC) - timedelta(days=90)


def _load_trades_for_attribution(since_dt: datetime) -> list[dict]:
    return load_trades_from_db_or_synthetic(
        since_dt,
        synthetic_count=50,
        logger_name="orca.attribution",
        synthetic_generator="attribution",
    )
