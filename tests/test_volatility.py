from __future__ import annotations

import numpy as np
import pytest

from orca.sizing.volatility import (
    diversification_scaling,
    ewma_volatility,
    vol_adjusted_size,
)


class TestEWMAVolatility:
    def test_constant_returns_zero_vol(self):
        returns = np.array([0.01] * 100)
        vol = ewma_volatility(returns, span=20)
        assert vol < 0.01

    def test_alternating_returns(self):
        returns = np.array([0.02, -0.02] * 50)
        vol = ewma_volatility(returns, span=10)
        assert vol > 0.01

    def test_single_return(self):
        returns = np.array([0.01])
        vol = ewma_volatility(returns, span=20)
        assert vol == 0.0

    def test_two_returns(self):
        returns = np.array([0.01, -0.02])
        vol = ewma_volatility(returns, span=20)
        assert vol >= 0.0

    def test_high_volatility(self):
        returns = np.array([0.10, -0.08, 0.12, -0.09, 0.11, -0.10] * 10)
        vol = ewma_volatility(returns, span=10)
        assert vol > 0.05

    def test_shorter_span_reacts_faster(self):
        returns = np.array([0.001] * 19 + [0.10])
        vol_long = ewma_volatility(returns, span=50)
        vol_short = ewma_volatility(returns, span=5)
        assert vol_short > vol_long

    def test_positive_output(self):
        returns = np.random.RandomState(42).randn(100) * 0.01
        vol = ewma_volatility(returns, span=20)
        assert vol >= 0.0


class TestVolAdjustedSize:
    def test_high_vol_reduces_size(self):
        size = vol_adjusted_size(kelly_fraction=0.10, vol=0.30, baseline_vol=0.15, max_size=0.20)
        assert size < 0.10

    def test_low_vol_increases_size(self):
        size = vol_adjusted_size(kelly_fraction=0.10, vol=0.10, baseline_vol=0.20, max_size=0.20)
        assert size > 0.10

    def test_respects_max_size(self):
        size = vol_adjusted_size(kelly_fraction=0.50, vol=0.01, baseline_vol=0.10, max_size=0.20)
        assert size <= 0.20

    def test_zero_baseline_vol(self):
        size = vol_adjusted_size(kelly_fraction=0.10, vol=0.10, baseline_vol=0.0, max_size=0.20)
        assert size == pytest.approx(min(0.10, 0.20))

    def test_negative_baseline_vol(self):
        size = vol_adjusted_size(kelly_fraction=0.10, vol=0.10, baseline_vol=-0.05, max_size=0.20)
        assert size == pytest.approx(min(0.10, 0.20))

    def test_equal_vol_no_adjustment(self):
        size = vol_adjusted_size(kelly_fraction=0.10, vol=0.20, baseline_vol=0.20, max_size=0.20)
        assert size == pytest.approx(min(0.20, 0.10 * 1.0))


class TestDiversificationScaling:
    def test_single_position(self):
        scale = diversification_scaling(num_positions=1, avg_correlation=0.5)
        assert scale == 1.0

    def test_zero_positions(self):
        scale = diversification_scaling(num_positions=0, avg_correlation=0.5)
        assert scale == 1.0

    def test_negative_positions(self):
        scale = diversification_scaling(num_positions=-1, avg_correlation=0.5)
        assert scale == 1.0

    def test_zero_correlation_reduces_with_sqrt(self):
        scale_2 = diversification_scaling(num_positions=2, avg_correlation=0.0)
        scale_4 = diversification_scaling(num_positions=4, avg_correlation=0.0)
        assert scale_2 == pytest.approx(1.0 / np.sqrt(2))
        assert scale_4 == pytest.approx(0.5)

    def test_perfect_correlation(self):
        scale = diversification_scaling(num_positions=5, avg_correlation=1.0)
        assert scale == pytest.approx(1.0)

    def test_scaling_between_bounds(self):
        for n in [2, 5, 10, 20]:
            for corr in [0.0, 0.25, 0.5, 0.75]:
                scale = diversification_scaling(num_positions=n, avg_correlation=corr)
                assert 0.25 <= scale <= 1.0

    def test_high_correlation_high_positions(self):
        scale = diversification_scaling(num_positions=100, avg_correlation=0.9)
        assert scale < 1.0
        assert scale >= 0.25

    def test_effective_n_zero(self):
        scale = diversification_scaling(num_positions=2, avg_correlation=-0.999)
        assert scale >= 0.25

    def test_more_positions_reduces_scaling(self):
        scale_2 = diversification_scaling(num_positions=2, avg_correlation=0.2)
        scale_10 = diversification_scaling(num_positions=10, avg_correlation=0.2)
        assert scale_10 < scale_2
