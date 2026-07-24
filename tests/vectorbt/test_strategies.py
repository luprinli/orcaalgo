"""Tests for orca.vectorbt.strategies — VectorBT strategy wrappers with numpy fallback."""

import numpy as np
import pandas as pd

from orca.vectorbt.strategies import (
    STRATEGY_MAP,
    _compute_adx_numpy,
    _compute_atr_numpy,
    _compute_rsi_numpy,
    grid_trading,
    intraday_mr,
    opening_range_breakout,
    session_scalp,
    trend_following,
)


def _make_ohlcv(n: int = 200, seed: int = 42) -> pd.DataFrame:
    """Generate synthetic OHLCV data."""
    np.random.seed(seed)
    close = 100 + np.cumsum(np.random.randn(n) * 0.1)
    high = close + np.abs(np.random.randn(n)) * 0.5
    low = close - np.abs(np.random.randn(n)) * 0.5
    return pd.DataFrame({
        "open": np.roll(close, 1),
        "high": high,
        "low": low,
        "close": close,
        "volume": np.random.randint(1000, 5000, n),
    })


class TestRSI:
    def test_rsi_shape(self):
        np.random.seed(42)
        prices = 100 + np.cumsum(np.random.randn(100) * 0.1)
        result = _compute_rsi_numpy(prices, 14)
        assert len(result) == len(prices)
        assert np.isnan(result[:14]).all()
        assert not np.isnan(result[20:]).all()

    def test_rsi_bounds(self):
        np.random.seed(123)
        prices = 100 + np.cumsum(np.random.randn(500) * 0.05)
        result = _compute_rsi_numpy(prices, 14)
        valid = result[~np.isnan(result)]
        assert (valid >= 0).all()
        assert (valid <= 100).all()

    def test_short_series(self):
        result = _compute_rsi_numpy(np.array([1.0, 2.0, 3.0]), 14)
        assert np.isnan(result).all()


class TestATR:
    def test_atr_shape(self):
        df = _make_ohlcv(100)
        result = _compute_atr_numpy(df["high"].values, df["low"].values, df["close"].values, 14)
        assert len(result) == 100

    def test_atr_positive(self):
        np.random.seed(77)
        close = 100 + np.cumsum(np.random.randn(200) * 0.1)
        high = close + 0.5
        low = close - 0.5
        result = _compute_atr_numpy(high, low, close, 14)
        valid = result[~np.isnan(result)]
        assert (valid >= 0).all()


class TestADX:
    def test_adx_shape(self):
        df = _make_ohlcv(100)
        result = _compute_adx_numpy(df["high"].values, df["low"].values, df["close"].values, 14)
        assert len(result) == 100

    def test_adx_short(self):
        result = _compute_adx_numpy(np.array([1.0]), np.array([1.0]), np.array([1.0]), 14)
        assert np.allclose(result, 25.0)


class TestIntradayMR:
    def test_returns_series(self):
        df = _make_ohlcv(500)
        result = intraday_mr(df["close"], rsi_period=20, entry_threshold=30, exit_threshold=50)
        assert isinstance(result, pd.Series)
        assert len(result) == 500

    def test_values_in_set(self):
        df = _make_ohlcv(500)
        result = intraday_mr(df["close"], rsi_period=20, entry_threshold=30, exit_threshold=50)
        assert set(result.unique()).issubset({-1, 0, 1})

    def test_default_thresholds(self):
        df = _make_ohlcv(300)
        result = intraday_mr(df["close"])
        assert isinstance(result, pd.Series)


class TestTrendFollowing:
    def test_returns_series(self):
        df = _make_ohlcv(300)
        result = trend_following(
            df["close"], df["high"], df["low"],
            ema_fast=20, ema_slow=50, adx_threshold=22.0,
        )
        assert isinstance(result, pd.Series)
        assert len(result) == 300

    def test_values_in_set(self):
        df = _make_ohlcv(300)
        result = trend_following(
            df["close"], df["high"], df["low"],
            ema_fast=10, ema_slow=30, adx_threshold=15.0,
        )
        assert set(result.unique()).issubset({-1, 0, 1})


class TestOpeningRangeBreakout:
    def test_returns_series(self):
        df = _make_ohlcv(100)
        result = opening_range_breakout(
            df["open"], df["high"], df["low"], df["close"],
            range_minutes=5, atr_mult=2.0, volume_mult=1.5,
        )
        assert isinstance(result, pd.Series)
        assert len(result) == 100

    def test_short_series_returns_zero(self):
        df = _make_ohlcv(3)
        result = opening_range_breakout(
            df["open"], df["high"], df["low"], df["close"],
            range_minutes=5,
        )
        assert (result == 0).all()


class TestGridTrading:
    def test_returns_series(self):
        df = _make_ohlcv(100)
        result = grid_trading(df["close"], grid_levels=5, grid_spacing_pct=1.0, max_open=10)
        assert isinstance(result, pd.Series)
        assert len(result) == 100

    def test_short_series(self):
        df = _make_ohlcv(1)
        result = grid_trading(df["close"])
        assert (result == 0).all()


class TestSessionScalp:
    def test_returns_series(self):
        df = _make_ohlcv(100)
        result = session_scalp(df["close"], df["high"], df["low"], df["volume"])
        assert isinstance(result, pd.Series)
        assert len(result) == 100

    def test_short_series(self):
        df = _make_ohlcv(5)
        result = session_scalp(df["close"], df["high"], df["low"], df["volume"])
        assert (result == 0).all()


class TestStrategyMap:
    def test_all_strategies_registered(self):
        assert "intraday_mr" in STRATEGY_MAP
        assert "trend_following" in STRATEGY_MAP
        assert "opening_range_breakout" in STRATEGY_MAP
        assert "grid_trading" in STRATEGY_MAP
        assert "session_scalp" in STRATEGY_MAP

    def test_param_name_contract(self):
        import inspect

        from orca.optimize.indicator_factory import STRATEGY_INDICATORS

        for sid in STRATEGY_INDICATORS:
            assert sid in STRATEGY_MAP, f"Missing vectorbt strategy: {sid}"

        factory_params = set(STRATEGY_INDICATORS["intraday_mr"]["default_params"].keys())
        vbt_params = set(inspect.signature(intraday_mr).parameters.keys()) - {"close"}
        assert factory_params == vbt_params, f"Param mismatch: {factory_params} != {vbt_params}"
