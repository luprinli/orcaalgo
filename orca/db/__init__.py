"""Thin DB access layer that delegates to the Go backend via subprocess.

The canonical trade record is stored in the Go backend's TimescaleDB.
This module provides fetch functions used by the attribution and calibration
CLI pipelines. If the DB is unreachable, callers fall back to synthetic data.
"""

from __future__ import annotations

import json
import os
import subprocess
from typing import Optional


def fetch_trades(
    strategy_id: str | None = None,
    since: str | None = None,
    limit: int = 500,
) -> list[dict]:
    """Fetch trade executions from the Go backend via subprocess.

    Returns a list of trade dicts. Returns an empty list if the backend is
    unreachable or if the orca-fetch CLI is not available.
    """
    try:
        exec_path = os.environ.get("ORCA_FETCH_PATH", "orca-fetch")
        args = [exec_path, "trades"]
        if strategy_id:
            args.extend(["--strategy", strategy_id])
        if since:
            args.extend(["--since", since])
        args.extend(["--limit", str(limit), "--format", "json"])

        result = subprocess.run(args, capture_output=True, text=True, timeout=15)
        if result.returncode != 0:
            return []

        trades = json.loads(result.stdout)
        if isinstance(trades, list):
            return trades
        return []
    except Exception:
        return []


__all__ = ["fetch_trades"]
