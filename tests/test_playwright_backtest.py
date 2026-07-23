"""Unit tests for playwright_backtest_matrix.py logic functions.

These tests validate the analysis, polling, and report-generation
logic without requiring a running server or browser.
"""

from __future__ import annotations

import sys
from pathlib import Path
from typing import Any

# Ensure the scripts directory is importable
sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))

from playwright_backtest_matrix import _analyze_results

# ── _analyze_results ──────────────────────────────────────────────

def test_analyze_results_with_trades() -> None:
    """Results with num_trades > 0 should appear in analysis."""
    completed: dict[str, Any] = {
        "results": [
            {"strategy_id": "grid_trading", "symbol": "USDEUR", "timeframe": "1d",
             "sharpe_ratio": 0.85, "total_return": 0.02, "num_trades": 9},
            {"strategy_id": "intraday_mr", "symbol": "USDJPY", "timeframe": "1d",
             "sharpe_ratio": 0.0, "total_return": 0.0, "num_trades": 0},
        ],
        "failed": 0,
        "status": "completed",
    }
    analysis = _analyze_results(completed, "test-batch")
    assert analysis["combos_with_trades"] == 1
    assert analysis["combos_no_trades"] == 1
    assert "1d" in analysis["timeframes"]
    assert "grid_trading" in analysis["strategies"]
    assert analysis["strategies"]["grid_trading"]["total_trades"] == 9


def test_analyze_results_empty() -> None:
    """Empty results should produce zeroes, not crash."""
    analysis = _analyze_results(
        {"results": [], "failed": 0, "status": "completed"}, "empty",
    )
    assert analysis["total_combos"] == 0
    assert analysis["combos_with_trades"] == 0
    assert analysis["timeframes"] == {}


def test_analyze_results_missing_fields() -> None:
    """Results missing num_trades should not crash."""
    completed: dict[str, Any] = {
        "results": [
            {"strategy_id": "grid_trading", "symbol": "USDEUR", "timeframe": "1d",
             "sharpe_ratio": 0.5, "total_return": 0.01},
        ],
        "failed": 0,
        "status": "completed",
    }
    analysis = _analyze_results(completed, "missing")
    assert analysis["combos_with_trades"] == 0
    assert analysis["combos_no_trades"] == 1


def test_analyze_results_multi_timeframe() -> None:
    """Multiple timeframes should each appear in analysis."""
    completed: dict[str, Any] = {
        "results": [
            {"strategy_id": "grid_trading", "symbol": "USDEUR", "timeframe": "1d",
             "sharpe_ratio": 0.8, "total_return": 0.02, "num_trades": 5},
            {"strategy_id": "grid_trading", "symbol": "USDEUR", "timeframe": "1h",
             "sharpe_ratio": 0.3, "total_return": 0.01, "num_trades": 3},
        ],
        "failed": 0,
        "status": "completed",
    }
    analysis = _analyze_results(completed, "multi-tf")
    assert "1d" in analysis["timeframes"]
    assert "1h" in analysis["timeframes"]
    assert analysis["timeframes"]["1d"]["total_trades"] == 5
    assert analysis["timeframes"]["1h"]["total_trades"] == 3


def test_analyze_results_top_performers_sorted() -> None:
    """Top performers should be sorted by sharpe_ratio descending."""
    completed: dict[str, Any] = {
        "results": [
            {"strategy_id": "low", "symbol": "A", "timeframe": "1d",
             "sharpe_ratio": -0.5, "total_return": -0.01, "num_trades": 5},
            {"strategy_id": "high", "symbol": "B", "timeframe": "1d",
             "sharpe_ratio": 2.0, "total_return": 0.05, "num_trades": 10},
            {"strategy_id": "mid", "symbol": "C", "timeframe": "1d",
             "sharpe_ratio": 0.5, "total_return": 0.02, "num_trades": 7},
        ],
        "failed": 0,
        "status": "completed",
    }
    analysis = _analyze_results(completed, "sort-test")
    top = analysis["top_performers"]
    assert len(top) == 3
    assert top[0]["sharpe"] == 2.0
    assert top[1]["sharpe"] == 0.5
    assert top[2]["sharpe"] == -0.5
