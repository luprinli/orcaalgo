"""Unit tests for signal injection module."""

import numpy as np
import pytest

from orca.simulation.signal_injector import (
    BaseInjector,
    BreakoutInjector,
    MeanReversionInjector,
    TrendInjector,
)


def _make_price_path(length=1000) -> np.ndarray:
    rng = np.random.default_rng(42)
    returns = rng.normal(0, 0.01, length)
    return 100 * np.exp(np.cumsum(returns))


def test_trend_injector_modifies_prices() -> None:
    prices = _make_price_path()
    injector = TrendInjector(strength=0.5, phi=0.8)
    result = injector.inject(prices)
    assert len(result) == len(prices), f"Length mismatch: {len(result)} != {len(prices)}"
    assert np.all(result > 0), "All prices must be positive"
    assert not np.allclose(result, prices), "Prices must be modified"


def test_trend_injector_zero_strength() -> None:
    prices = _make_price_path()
    injector = TrendInjector(strength=0.0)
    result = injector.inject(prices)
    assert np.allclose(result, prices), "Zero strength must preserve prices"


def test_mean_reversion_injector_modifies_prices() -> None:
    prices = _make_price_path()
    injector = MeanReversionInjector(strength=0.5, theta=0.2)
    result = injector.inject(prices)
    assert len(result) == len(prices)
    assert np.all(result > 0)
    assert not np.allclose(result, prices)


def test_mean_reversion_injector_zero_strength() -> None:
    prices = _make_price_path()
    injector = MeanReversionInjector(strength=0.0)
    result = injector.inject(prices)
    assert np.allclose(result, prices)


def test_breakout_injector_modifies_prices() -> None:
    prices = _make_price_path()
    injector = BreakoutInjector(strength=0.5, lookback=20, drift_bars=5)
    result = injector.inject(prices)
    assert len(result) == len(prices)
    assert np.all(result > 0)
    assert not np.allclose(result, prices)


def test_breakout_injector_zero_strength() -> None:
    prices = _make_price_path()
    injector = BreakoutInjector(strength=0.0)
    result = injector.inject(prices)
    assert np.allclose(result, prices)


def test_base_injector_raises() -> None:
    prices = _make_price_path()
    try:
        BaseInjector().inject(prices)
        pytest.fail("BaseInjector must raise NotImplementedError")
    except NotImplementedError:
        pass
