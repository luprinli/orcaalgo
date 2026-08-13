#!/usr/bin/env python3
"""Phase 1: Stooq Symbol Discovery & Manifest Builder.

Walks the data/stooq/ directory tree for the 18-symbol universe, verifies
file existence, extracts metadata (row count, date range, size), and
outputs data/stooq/manifest.json — the machine-readable index that all
subsequent phases consume.

Does NOT load any CSV content — only reads headers and boundary lines.
"""

from __future__ import annotations

import json
import os
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

PROJECT_ROOT = Path(__file__).resolve().parent.parent
STOOQ_ROOT = PROJECT_ROOT / "data" / "stooq"
MANIFEST_PATH = STOOQ_ROOT / "manifest.json"

# The 18-symbol universe is loaded from configs/universe.json (single source of
# truth shared with Go). Fall back to an empty list if the config is missing.
sys.path.insert(0, str(PROJECT_ROOT))
try:
    from orca.universe_config import load_universe
    _cfg = load_universe()
    UNIVERSE: list[dict[str, str]] = [
        {
            "symbol": s["ticker"],
            "stooq_ticker": s.get("stooq_ticker", ""),
            "asset_class": s["asset_class"],
        }
        for s in _cfg.get("symbols", [])
    ]
except Exception:
    UNIVERSE: list[dict[str, str]] = []


def _find_file(ticker: str, freq: str) -> Path | None:
    """Find a stooq CSV file by ticker and frequency (1d, 5m or 1h).

    Searches both US and world trees. The sharding scheme places files
    in subdirectories by first letter or exchange category.
    """
    filename = f"{ticker}.txt"

    if freq == "5m":
        base_paths = [
            STOOQ_ROOT / "5_us_txt" / "data" / "5 min" / "us",
            STOOQ_ROOT / "5_world_txt" / "data" / "5 min" / "world",
        ]
    elif freq == "1d":
        base_paths = [
            STOOQ_ROOT / "d_us_txt" / "data" / "daily" / "us",
            STOOQ_ROOT / "d_world_txt" / "data" / "daily" / "world",
        ]
    else:
        base_paths = [
            STOOQ_ROOT / "h_us_txt" / "data" / "hourly" / "us",
            STOOQ_ROOT / "h_world_txt" / "data" / "hourly" / "world",
        ]

    for base in base_paths:
        if not base.exists():
            continue
        for root, _dirs, files in os.walk(str(base)):
            if filename in files:
                return Path(root) / filename

    return None


def _extract_metadata(filepath: Path, freq: str) -> dict[str, Any]:
    """Extract metadata from a stooq CSV file without loading all data.

    Returns row count, first/last bar timestamps, file size, and an
    indicator of whether the file is empty.
    """
    size = filepath.stat().st_size
    if size == 0:
        return {
            "exists": True,
            "size_bytes": 0,
            "row_count": 0,
            "first_bar": None,
            "last_bar": None,
            "is_empty": True,
            "error": None,
        }

    try:
        with open(filepath, "r", encoding="utf-8") as f:
            header = f.readline()
            if not header.startswith("<TICKER>"):
                return {
                    "exists": True, "size_bytes": size, "error": "bad header",
                    "row_count": 0, "first_bar": None, "last_bar": None,
                    "is_empty": False,
                }

            # Count lines via fast read — only the header + last line matter
            line_count = 1  # header
            data_lines = f.readlines()
            line_count += len(data_lines)

            if len(data_lines) == 0:
                return {
                    "exists": True, "size_bytes": size,
                    "row_count": 0, "first_bar": None, "last_bar": None,
                    "is_empty": True, "error": None,
                }

            first_line = data_lines[0].strip()
            last_line = data_lines[-1].strip()

            first_bar = _parse_bar_ts(first_line)
            last_bar = _parse_bar_ts(last_line)

        return {
            "exists": True,
            "size_bytes": size,
            "row_count": line_count - 1,  # exclude header
            "first_bar": first_bar,
            "last_bar": last_bar,
            "is_empty": False,
            "error": None,
            "bars_per_day_approx": round(
                (line_count - 1) / max((_days_between(first_bar, last_bar) or 1), 1), 1
            ) if first_bar and last_bar else None,
        }
    except Exception as e:
        return {
            "exists": True, "size_bytes": size,
            "row_count": 0, "first_bar": None, "last_bar": None,
            "is_empty": False, "error": str(e)[:200],
        }


def _parse_bar_ts(line: str) -> str | None:
    """Parse DATE+TIME from a stooq CSV line into ISO 8601 UTC."""
    parts = line.split(",")
    if len(parts) < 4:
        return None
    date_str = parts[2].strip()   # YYYYMMDD
    time_str = parts[3].strip() or "000000"   # HHMMSS (empty for daily bars)
    if len(date_str) != 8 or len(time_str) not in (6, 8):
        return None
    try:
        dt = datetime.strptime(f"{date_str}{time_str}", "%Y%m%d%H%M%S")
        return dt.replace(tzinfo=timezone.utc).isoformat()
    except ValueError:
        return None


def _days_between(start_iso: str | None, end_iso: str | None) -> int | None:
    """Compute calendar days between two ISO 8601 timestamps."""
    if not start_iso or not end_iso:
        return None
    try:
        start = datetime.fromisoformat(start_iso)
        end = datetime.fromisoformat(end_iso)
        return max((end - start).days, 1)
    except ValueError:
        return None


def build_manifest() -> dict[str, Any]:
    """Walk the stooq tree for all 18 symbols and build the manifest."""
    entries: list[dict[str, Any]] = []
    stats = {
        "total": len(UNIVERSE),
        "found_1d": 0, "found_5m": 0, "found_1h": 0,
        "missing_1d": 0, "missing_5m": 0, "missing_1h": 0,
    }

    for item in UNIVERSE:
        entry = dict(item)
        for freq, key in [("1d", "stooq_1d"), ("5m", "stooq_5m"), ("1h", "stooq_1h")]:
            filepath = _find_file(item["stooq_ticker"], freq)
            if filepath:
                rel = str(filepath.relative_to(STOOQ_ROOT))
                meta = _extract_metadata(filepath, freq)
                entry[key] = {"path": rel, **meta}
                stats[f"found_{freq}"] += 1
            else:
                entry[key] = None
                stats[f"missing_{freq}"] += 1

        entries.append(entry)

    manifest = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "stooq_root": str(STOOQ_ROOT),
        "universe_size": len(UNIVERSE),
        "stats": stats,
        "symbols": entries,
    }
    return manifest


def main() -> None:
    manifest = build_manifest()
    MANIFEST_PATH.parent.mkdir(parents=True, exist_ok=True)
    MANIFEST_PATH.write_text(json.dumps(manifest, indent=2, default=str) + "\n", encoding="utf-8")

    stats = manifest["stats"]
    print(f"Manifest written to {MANIFEST_PATH}")
    print(f"  Symbols: {stats['total']}")
    print(f"  1d: {stats['found_1d']} found, {stats['missing_1d']} missing")
    print(f"  5m: {stats['found_5m']} found, {stats['missing_5m']} missing")
    print(f"  1h: {stats['found_1h']} found, {stats['missing_1h']} missing")

    if stats["missing_1d"] > 0 or stats["missing_5m"] > 0 or stats["missing_1h"] > 0:
        print("\n  Missing:")
        for entry in manifest["symbols"]:
            for freq in ["stooq_1d", "stooq_5m", "stooq_1h"]:
                if entry[freq] is None:
                    print(f"    {entry['symbol']} ({entry['asset_class']}): {freq} not found")


if __name__ == "__main__":
    main()
