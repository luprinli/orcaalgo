"""Tests for orca.vectorbt.optimize — parameter sweep with fallback."""

import os
import tempfile
from pathlib import Path

import pandas as pd
import pytest

from orca.vectorbt.optimize import (
    _extract_vbt_metrics,
    _sweep_native,
    sweep_strategy,
)


def _make_ohlcv_csv(path: Path, n: int = 200) -> Path:
    import numpy as np

    np.random.seed(42)
    dates = pd.date_range("2023-01-02", periods=n)
    close = 100 + np.cumsum(np.random.randn(n) * 0.1)
    df = pd.DataFrame({
        "Date": dates,
        "Open": np.roll(close, 1),
        "High": close + np.abs(np.random.randn(n)) * 0.5,
        "Low": close - np.abs(np.random.randn(n)) * 0.5,
        "Close": close,
        "Volume": np.random.randint(1000, 5000, n),
    })
    df.iloc[0, df.columns.get_loc("Open")] = 100.0
    df.to_csv(path, index=False)
    return path


class TestSweepStrategy:
    def test_sweep_native_backend(self, monkeypatch, tmp_path):
        csv_path = _make_ohlcv_csv(tmp_path / "test_1d.csv", n=300)
        param_grid = {"rsi_period": [14, 20], "entry_threshold": [25, 30], "exit_threshold": [50, 55]}

        result = _sweep_native(
            pd.read_csv(csv_path, parse_dates=["Date"], index_col="Date"),
            "intraday_mr",
            param_grid,
        )

        assert result["strategy_id"] == "intraday_mr"
        assert result["method"] == "grid"
        assert "best_params" in result
        assert "best_metrics" in result
        assert "sharpe_ratio" in result["best_metrics"]

    def test_unknown_strategy_raises(self, tmp_path):
        csv_path = _make_ohlcv_csv(tmp_path / "test.csv", n=100)
        with pytest.raises(ValueError, match="Unknown strategy"):
            sweep_strategy(
                "test", "2023-01-01", "2023-12-31",
                "nonexistent_strategy", {"a": [1]},
                timeframe="1d", fallback="native",
            )

    def test_sweep_with_file_data_source(self, monkeypatch, tmp_path):
        csv_path = _make_ohlcv_csv(tmp_path / "test_1d.csv", n=200)
        monkeypatch.setenv("ORCA_DATA_DIR", str(tmp_path))

        result = sweep_strategy(
            "test", "2023-01-01", "2023-12-31",
            "intraday_mr", {"rsi_period": [14, 20], "entry_threshold": [30], "exit_threshold": [50]},
            timeframe="1d", fallback="native",
        )

        assert result["n_trials"] > 0
        assert "best_params" in result


class TestExtractMetrics:
    def test_missing_metrics_returns_zeros(self):
        metrics = _extract_vbt_metrics(pd.Series(), (0,))
        assert metrics["sharpe_ratio"] == 0.0
        assert metrics["num_trades"] == 0
