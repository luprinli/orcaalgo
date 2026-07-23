"""Tests for orca.optimize.sweeper — hyperparameter sweeper."""

import tempfile
from pathlib import Path

import pandas as pd
import pytest

from orca.optimize.sweeper import (
    _compute_rsi,
    _evaluate_params,
    _generate_signals,
    _grid_search,
    _random_search,
    _trade_signals,
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


class TestRSI:
    def test_rsi_returns_array(self):
        import numpy as np
        prices = np.linspace(100, 110, 50)
        result = _compute_rsi(prices, 14)
        assert len(result) == 50

    def test_rsi_bounded(self):
        import numpy as np
        np.random.seed(123)
        prices = 100 + np.cumsum(np.random.randn(300) * 0.05)
        result = _compute_rsi(prices, 14)
        valid = result[~np.isnan(result)]
        assert (valid >= 0).all()
        assert (valid <= 100).all()


class TestGenerateSignals:
    def test_returns_array(self):
        import numpy as np
        df = pd.DataFrame({
            "Close": 100 + np.cumsum(np.random.randn(200) * 0.1),
        })
        params = {"rsi_period": 20, "entry_threshold": 30, "exit_threshold": 50}
        result = _generate_signals(df, params)
        assert result.ndim == 1

    def test_signal_values_in_set(self):
        import numpy as np
        df = pd.DataFrame({
            "Close": 100 + np.cumsum(np.random.randn(200) * 0.1),
        })
        params = {"rsi_period": 20, "entry_threshold": 30, "exit_threshold": 50}
        result = _generate_signals(df, params)
        assert set(result).issubset({-1, 0, 1})


class TestEvaluateParams:
    def test_returns_metrics_dict(self, tmp_path):
        csv_path = _make_ohlcv_csv(tmp_path / "test.csv", n=300)
        df = pd.read_csv(csv_path, parse_dates=["Date"], index_col="Date")
        params = {"rsi_period": 20, "entry_threshold": 30, "exit_threshold": 50}
        metrics = _evaluate_params(df, params)
        assert "sharpe_ratio" in metrics
        assert "max_drawdown" in metrics
        assert "total_return" in metrics
        assert "win_rate" in metrics
        assert "num_trades" in metrics

    def test_short_data_returns_zeros(self):
        import numpy as np
        df = pd.DataFrame({"Close": np.array([100.0])})
        metrics = _evaluate_params(df, {"rsi_period": 20})
        assert metrics["sharpe_ratio"] == 0
        assert metrics["num_trades"] == 0


class TestGridSearch:
    def test_returns_best(self, tmp_path):
        csv_path = _make_ohlcv_csv(tmp_path / "test.csv", n=200)
        df = pd.read_csv(csv_path, parse_dates=["Date"], index_col="Date")
        param_grid = {"rsi_period": [14, 20], "entry_threshold": [30], "exit_threshold": [50]}
        results, best = _grid_search(df, param_grid)
        assert len(results) > 0
        assert "params" in best
        assert "sharpe_ratio" in best


class TestRandomSearch:
    def test_returns_results(self, tmp_path):
        csv_path = _make_ohlcv_csv(tmp_path / "test.csv", n=200)
        df = pd.read_csv(csv_path, parse_dates=["Date"], index_col="Date")
        param_grid = {"rsi_period": [14, 20], "entry_threshold": [25, 30]}
        results, best = _random_search(df, param_grid, n=10)
        assert len(results) <= 10
        assert "params" in best


class TestSweepStrategy:
    def test_grid_sweep(self, tmp_path):
        csv_path = _make_ohlcv_csv(tmp_path / "test.csv", n=250)
        param_grid = {"rsi_period": [14, 20], "entry_threshold": [30], "exit_threshold": [50]}
        result = sweep_strategy("intraday_mr", str(csv_path), param_grid, method="grid")
        assert result["strategy_id"] == "intraday_mr"
        assert result["method"] == "grid"
        assert result["n_trials"] > 0
        assert "best_params" in result
        assert "best_metrics" in result

    def test_random_sweep(self, tmp_path):
        csv_path = _make_ohlcv_csv(tmp_path / "test.csv", n=200)
        param_grid = {"rsi_period": [14, 20], "entry_threshold": [25, 30]}
        result = sweep_strategy("intraday_mr", str(csv_path), param_grid, method="random", n_random=5)
        assert result["n_trials"] <= 5
        assert result["method"] == "random"


class TestTradeSignals:
    def test_no_signals(self):
        close = [100, 101, 102]
        signals = [0, 0, 0]
        returns = _trade_signals(close, signals)
        assert returns == []

    def test_single_trade(self):
        close = [100, 105]
        signals = [1, -1]
        returns = _trade_signals(close, signals)
        assert len(returns) == 1
        assert abs(returns[0] - 0.05) < 1e-6
