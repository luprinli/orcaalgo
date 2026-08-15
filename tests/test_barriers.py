"""Tests for triple-barrier labeling."""

import numpy as np
import pytest

from orca.ml.barriers import (
    BarrierConfig,
    BarrierLabel,
    batch_triple_barrier_labels,
    compute_sigma_from_prices,
    label_to_binary,
    triple_barrier_label,
)


class TestBarrierConfig:
    def test_default_config(self):
        cfg = BarrierConfig()
        assert cfg.profit_factor == 2.0
        assert cfg.stop_factor == 1.0
        assert cfg.time_horizon == 20

    def test_custom_config(self):
        cfg = BarrierConfig(profit_factor=3.0, stop_factor=1.5, time_horizon=10)
        assert cfg.profit_factor == 3.0


class TestTripleBarrierLabel:
    def test_upper_hit(self):
        """When price rises above upper barrier, label is +1."""
        prices = np.array([100.0, 101.0, 102.0, 103.5, 104.0])
        result = triple_barrier_label(100.0, prices, 0)
        assert result.label == BarrierLabel.UPPER_HIT
        assert result.return_pct > 0
        assert result.hit_barrier == "upper"

    def test_lower_hit(self):
        """When price falls below lower barrier, label is -1."""
        prices = np.array([100.0, 99.0, 98.0, 97.0, 96.5])
        result = triple_barrier_label(100.0, prices, 0)
        assert result.label == BarrierLabel.LOWER_HIT
        assert result.return_pct < 0
        assert result.hit_barrier == "lower"

    def test_time_hit_neutral(self):
        """When neither barrier is hit, time exit determines label by return."""
        prices = np.full(30, 100.0)
        prices[25] = 100.001  # tiny positive return
        result = triple_barrier_label(100.0, prices, 0, BarrierConfig(time_horizon=20))
        assert result.hit_barrier == "time"
        # With tiny positive return and min_return=0.001, should be neutral
        assert result.label in (BarrierLabel.TIME_HIT, BarrierLabel.UPPER_HIT)

    def test_time_hit_negative_return(self):
        """Time barrier with negative return should yield loss."""
        prices = np.full(30, 100.0)
        prices[25] = 99.0
        result = triple_barrier_label(
            100.0,
            prices,
            0,
            BarrierConfig(time_horizon=20, min_return=0.0),
        )
        assert result.hit_barrier == "time"
        assert result.label == BarrierLabel.LOWER_HIT

    def test_entry_at_index(self):
        """Entry price at non-zero index."""
        prices = np.array([95.0, 96.0, 100.0, 102.0, 104.0, 106.0])
        result = triple_barrier_label(100.0, prices, 2)
        assert result.label == BarrierLabel.UPPER_HIT

    def test_batch_labels(self):
        """Batch labeling returns results for multiple trades."""
        prices = np.array([100.0, 102.0, 104.0, 106.0, 108.0, 110.0])
        entry_prices = np.array([100.0, 102.0])
        entry_indices = np.array([0, 1])
        results = batch_triple_barrier_labels(entry_prices, entry_indices, prices)
        assert len(results) == 2
        assert all(r.label == BarrierLabel.UPPER_HIT for r in results)


class TestLabelToBinary:
    def test_upper_hit_to_one(self):
        assert label_to_binary(BarrierLabel.UPPER_HIT) == 1

    def test_lower_hit_to_zero(self):
        assert label_to_binary(BarrierLabel.LOWER_HIT) == 0

    def test_time_hit_to_zero(self):
        assert label_to_binary(BarrierLabel.TIME_HIT) == 0


class TestSigmaFromPrices:
    def test_computes_rolling_sigma(self):
        rng = np.random.default_rng(42)
        prices = 100.0 + np.cumsum(rng.normal(0, 0.5, 100))
        sigma = compute_sigma_from_prices(prices, lookback=20)
        assert np.isnan(sigma[0])
        assert np.isnan(sigma[19])
        assert sigma[20] > 0
        assert not np.isnan(sigma[20])

    def test_constant_prices_zero_sigma(self):
        prices = np.full(50, 100.0)
        sigma = compute_sigma_from_prices(prices, lookback=20)
        assert sigma[20] == pytest.approx(0.0, abs=1e-10)
