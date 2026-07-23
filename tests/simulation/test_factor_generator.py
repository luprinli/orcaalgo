"""Unit tests for multi-factor regime generator."""

import numpy as np
import pytest

from orca.simulation.factor_generator import REGIME_FACTORS, FactorConfig, FactorGenerator


def test_regime_factors_defined() -> None:
    """All 4 regimes must have configs."""
    assert len(REGIME_FACTORS) == 4, f"Expected 4 regimes, got {len(REGIME_FACTORS)}"
    for r in range(4):
        assert r in REGIME_FACTORS, f"Regime {r} missing"
        cfg = REGIME_FACTORS[r]
        assert isinstance(cfg, FactorConfig)
        assert isinstance(cfg.trend_phi, float)
        assert isinstance(cfg.mr_theta, float)
        assert isinstance(cfg.vol_sigma, float)
        assert isinstance(cfg.drift, float)


def test_factor_generator_returns_correct_length() -> None:
    labels = np.array([0, 0, 1, 1, 2, 2, 3, 3] * 25, dtype=int)
    fg = FactorGenerator(labels)
    log_rets = fg.generate_log_returns()
    assert len(log_rets) == len(labels), f"Expected {len(labels)}, got {len(log_rets)}"


def test_factor_generator_produces_varied_returns() -> None:
    labels = np.array([0, 0, 1, 1, 2, 2, 3, 3] * 25, dtype=int)
    fg = FactorGenerator(labels)
    log_rets = fg.generate_log_returns()
    assert float(np.std(log_rets)) > 0, "Returns must have non-zero variance"


def test_factor_generator_generates_prices() -> None:
    labels = np.array([0, 0, 1, 1, 2, 2, 3, 3] * 25, dtype=int)
    fg = FactorGenerator(labels)
    prices = fg.generate_prices(start_price=100.0)
    assert len(prices) == len(labels)
    assert float(prices[0]) == pytest.approx(100.0, abs=1e-9)
    assert np.all(prices > 0), "All prices must be positive"


def test_regime_dynamics_differ() -> None:
    """Calm regime should have lower vol than Crisis."""
    calm = REGIME_FACTORS[0]
    crisis = REGIME_FACTORS[3]
    assert calm.vol_sigma < crisis.vol_sigma, "Crisis vol must be higher than Calm"
    assert abs(calm.trend_phi) < abs(crisis.trend_phi), "Crisis should have stronger trend reversal"


def test_custom_config() -> None:
    labels = np.array([0] * 50, dtype=int)
    custom = {0: FactorConfig(trend_phi=0.5, mr_theta=0.1, vol_sigma=0.01, drift=0.0)}
    fg = FactorGenerator(labels, config=custom)
    log_rets = fg.generate_log_returns()
    assert len(log_rets) == 50
