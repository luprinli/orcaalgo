from __future__ import annotations

from datetime import UTC, datetime, timedelta


def resolve_since(since: str, default_days: int | None = None) -> datetime:
    """Parse 'since' string like '30d', '90d', '2w' into absolute UTC datetime."""
    if since.endswith("d"):
        days = int(since[:-1])
        return datetime.now(UTC) - timedelta(days=days)
    if since.endswith("w"):
        weeks = int(since[:-1])
        return datetime.now(UTC) - timedelta(weeks=weeks)
    if default_days is not None:
        return datetime.now(UTC) - timedelta(days=default_days)
    return datetime.fromisoformat(since)
