from __future__ import annotations

from datetime import UTC, datetime, timedelta


def resolve_since(since: str) -> datetime:
    """Parse 'since' string like '30d', '90d' into absolute UTC datetime."""
    if since.endswith("d"):
        days = int(since[:-1])
        return datetime.now(UTC) - timedelta(days=days)
    return datetime.fromisoformat(since)
