"""Shared universe config loader.

Loads the canonical trading universe from configs/universe.json — the single
source of truth for symbols, strategies, timeframes, and data sources shared
between Go (internal/config) and Python (orca/universe_config.py).
"""

from __future__ import annotations

import json
import os
from pathlib import Path
from typing import Any

CONFIG_PATH_ENV = "ORCA_UNIVERSE_CONFIG"
DEFAULT_CONFIG_PATH = Path("configs") / "universe.json"

_cache: dict[str, Any] | None = None


def config_path() -> Path:
    """Return the universe config path.

    Resolution order: ORCA_UNIVERSE_CONFIG env var, then configs/universe.json
    relative to the working directory and each ancestor, then relative to the
    project root (derived from this module's location).
    """
    if p := os.environ.get(CONFIG_PATH_ENV):
        return Path(p)

    # Search upward from cwd.
    cwd = Path.cwd()
    for directory in [cwd, *cwd.parents]:
        candidate = directory / "configs" / "universe.json"
        if candidate.exists():
            return candidate

    # Fall back to project root derived from this module's location.
    project_root = Path(__file__).resolve().parent.parent
    return project_root / "configs" / "universe.json"


def load_universe() -> dict[str, Any]:
    """Load the universe config, caching the result."""
    global _cache
    if _cache is not None:
        return _cache
    path = config_path()
    data = json.loads(path.read_text(encoding="utf-8"))
    _cache = data
    return data


def get_tickers() -> list[str]:
    """Return the canonical ticker list."""
    u = load_universe()
    return [s["ticker"] for s in u["symbols"]]


def get_strategies() -> list[str]:
    """Return the strategy id list."""
    return load_universe()["strategies"]


def get_timeframes() -> list[str]:
    """Return the timeframe list."""
    return load_universe()["timeframes"]


def get_data_sources() -> list[str]:
    """Return the data source list."""
    return load_universe()["data_sources"]


def symbol_by_ticker(ticker: str) -> dict[str, Any] | None:
    """Return the full symbol definition for a canonical ticker."""
    u = load_universe()
    for s in u["symbols"]:
        if s["ticker"] == ticker:
            return s
    return None
