from __future__ import annotations

import pytest

from orca.math.wilson import wilson_ci


class TestWilsonCI:
    def test_known_value_60_100(self):
        lower, upper = wilson_ci(wins=60, n=100)
        assert lower == pytest.approx(0.50, abs=0.02)
        assert upper == pytest.approx(0.70, abs=0.02)

    def test_known_value_50_100_95ci(self):
        lower, upper = wilson_ci(wins=50, n=100, z=1.96)
        assert lower == pytest.approx(0.40, abs=0.02)
        assert upper == pytest.approx(0.60, abs=0.02)

    def test_99ci_wider_than_95ci(self):
        lower_95, upper_95 = wilson_ci(wins=60, n=100, z=1.96)
        lower_99, upper_99 = wilson_ci(wins=60, n=100, z=2.576)
        assert (upper_95 - lower_95) < (upper_99 - lower_99)

    def test_n_zero_returns_full_interval(self):
        lower, upper = wilson_ci(wins=0, n=0)
        assert lower == 0.0
        assert upper == 1.0

    def test_all_wins(self):
        lower, upper = wilson_ci(wins=100, n=100)
        assert lower > 0.0
        assert upper == pytest.approx(1.0)

    def test_all_losses(self):
        lower, upper = wilson_ci(wins=0, n=100)
        assert lower == 0.0
        assert upper < 1.0

    def test_bounds_within_zero_one(self):
        for n in [1, 5, 10, 50, 100]:
            for wins in range(n + 1):
                lower, upper = wilson_ci(wins=wins, n=n)
                assert 0.0 <= lower <= 1.0
                assert 0.0 <= upper <= 1.0
                assert lower <= upper

    def test_center_pulls_toward_prior(self):
        lower, upper = wilson_ci(wins=1, n=2)
        assert upper - lower < 1.0

    def test_symmetry_at_50_percent(self):
        lower, upper = wilson_ci(wins=50, n=100)
        center = (lower + upper) / 2
        assert center == pytest.approx(0.50, abs=0.01)

    def test_large_n_tight_interval(self):
        lower, upper = wilson_ci(wins=6000, n=10000)
        assert (upper - lower) < 0.05

    def test_check_precision(self):
        lower, upper = wilson_ci(wins=60, n=100)
        p_hat = 60 / 100
        center = (lower + upper) / 2
        assert center > p_hat - 0.02
        assert center < p_hat + 0.05
