"""Tests for orca.optimize.walk_forward — walk-forward validation."""

from pathlib import Path

import pandas as pd

from orca.optimize.walk_forward import walk_forward_validate


def _make_ohlcv_csv(path: Path, n: int = 500) -> Path:
    import numpy as np

    np.random.seed(42)
    dates = pd.date_range("2023-01-02", periods=n)
    close = 100 + np.cumsum(np.random.randn(n) * 0.1)
    df = pd.DataFrame(
        {
            "Date": dates,
            "Open": np.roll(close, 1),
            "High": close + np.abs(np.random.randn(n)) * 0.5,
            "Low": close - np.abs(np.random.randn(n)) * 0.5,
            "Close": close,
            "Volume": np.random.randint(1000, 5000, n),
        }
    )
    df.iloc[0, df.columns.get_loc("Open")] = 100.0
    df.to_csv(path, index=False)
    return path


class TestWalkForwardValidate:
    def test_not_enough_data_returns_error(self, tmp_path):
        csv_path = _make_ohlcv_csv(tmp_path / "short.csv", n=50)
        result = walk_forward_validate(
            str(csv_path),
            "intraday_mr",
            {"rsi_period": [14, 20], "entry_threshold": [30], "exit_threshold": [50]},
            window_size=252,
            step_size=63,
            oos_size=63,
        )
        assert "error" in result

    def test_sufficient_data_produces_windows(self, tmp_path):
        csv_path = _make_ohlcv_csv(tmp_path / "data.csv", n=400)
        param_grid = {"rsi_period": [14], "entry_threshold": [30], "exit_threshold": [50]}
        result = walk_forward_validate(
            str(csv_path),
            "intraday_mr",
            param_grid,
            window_size=100,
            step_size=50,
            oos_size=50,
        )
        assert "avg_oos_sharpe" in result
        assert "degradation_pct" in result
        assert "total_windows" in result
        assert result["total_windows"] > 0

    def test_metrics_keys_present(self, tmp_path):
        csv_path = _make_ohlcv_csv(tmp_path / "data.csv", n=400)
        param_grid = {"rsi_period": [14], "entry_threshold": [30], "exit_threshold": [50]}
        result = walk_forward_validate(
            str(csv_path),
            "intraday_mr",
            param_grid,
            window_size=100,
            step_size=50,
            oos_size=50,
        )
        if "error" not in result:
            assert "passed_windows" in result
            assert "avg_is_sharpe" in result
