"""Entry point for python -m orca.data_quality."""

from __future__ import annotations

import json as _json

from orca.data_quality import load_candles


def main():
    candles = load_candles()
    summary = {
        "total_symbols": len(candles),
        "total_candles": sum(len(v) for v in candles.values()),
        "symbols": sorted(candles.keys()),
    }
    print(_json.dumps(summary, indent=2, default=str))


if __name__ == "__main__":
    main()
