from __future__ import annotations

import numpy as np
import pytest

from orca.math.ewma import ewma_volatility


class TestEWMAVolatility:
    def test_basic_normal_returns(self):
        np.random.seed(42)
        returns = np.random.normal(0, 0.01, 100)
        vol = ewma_volatility(returns)
        assert vol > 0, f"Expected positive volatility, got {vol}"
        assert vol < 0.05, f"Expected reasonable volatility, got {vol}"

    def test_span_parameter(self):
        np.random.seed(42)
        returns = np.random.normal(0, 0.02, 200)
        vol_long = ewma_volatility(returns, span=60)
        vol_short = ewma_volatility(returns, span=5)
        assert vol_long > 0 and vol_short > 0

    def test_constant_returns(self):
        returns = np.array([0.01] * 50)
        vol = ewma_volatility(returns)
        assert vol >= 0.0

    def test_single_return(self):
        returns = np.array([0.05])
        vol = ewma_volatility(returns)
        assert vol == 0.0

    def test_empty_array(self):
        returns = np.array([])
        vol = ewma_volatility(returns)
        assert vol == 0.0

    def test_two_returns(self):
        returns = np.array([0.01, 0.03])
        vol = ewma_volatility(returns)
        assert vol > 0

    def test_nan_handling(self):
        returns = np.array([0.01, np.nan, 0.02, 0.03, np.nan, 0.01])
        vol = ewma_volatility(returns)
        assert vol > 0
        assert not np.isnan(vol)

    def test_all_nan(self):
        returns = np.full(10, np.nan)
        vol = ewma_volatility(returns)
        assert vol == 0.0

    def test_negative_returns(self):
        np.random.seed(7)
        returns = np.random.normal(0, 0.02, 100)
        vol = ewma_volatility(returns)
        assert vol > 0

    def test_increasing_volatility(self):
        np.random.seed(7)
        returns_low = np.random.normal(0, 0.003, 100)
        np.random.seed(7)
        returns_high = np.random.normal(0, 0.03, 100)
        vol_low = ewma_volatility(returns_low)
        vol_high = ewma_volatility(returns_high)
        assert vol_high > vol_low, f"High-vol returns should give higher EWMA vol: {vol_high} > {vol_low}"

    def test_large_spike_increases_vol(self):
        np.random.seed(7)
        base = np.random.normal(0, 0.005, 50).tolist()
        spiked = np.array(base + [0.10] + base)
        flat = np.array(base + [0.001] + base)
        vol_spiked = ewma_volatility(spiked)
        vol_flat = ewma_volatility(flat)
        assert vol_spiked > vol_flat, f"Spike should increase vol: {vol_spiked} vs {vol_flat}"

    def test_list_input(self):
        returns = [0.01, -0.02, 0.015, -0.005, 0.02]
        vol = ewma_volatility(returns)
        assert vol > 0

    def test_seed_window_fallback(self):
        returns = np.array([0.01, 0.02])
        vol = ewma_volatility(returns, span=20)
        assert vol > 0

    def test_alpha_formula(self):
        span = 10
        expected_alpha = 2.0 / (span + 1)
        assert expected_alpha == pytest.approx(0.1818, abs=0.001)
